package eval

import (
	"awesomeEval/internal/logger"
	"awesomeEval/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

const (
	MaxConcurrency = 15
	EvalQueueName  = "eval_jobs"
)

type Service struct {
	Conn        *amqp.Connection
	DB          *gorm.DB
	MinioClient *minio.Client
	RedisClient *redis.Client
}

func NewEvalService(conn *amqp.Connection, db *gorm.DB, minioClient *minio.Client, redisClient *redis.Client) *Service {
	return &Service{
		Conn:        conn,
		DB:          db,
		MinioClient: minioClient,
		RedisClient: redisClient,
	}
}

func (s *Service) CreateBatchEvalJob(ctx context.Context, userId int, req *models.EvalBatchTaskRequest) error {
	logger.Log.Info("CreateBatchEvalJob started",
		zap.Int("userId", userId),
		zap.Int("datasetId", req.DatasetId))
	var user models.UserDB
	if err := s.DB.Model(models.UserDB{}).Where("id = ?", userId).First(&user).Error; err != nil {
		logger.Log.Error("CreateBatchEvalJob user not found, "+err.Error(),
			zap.Int("userId", userId),
		)
		return err
	}
	if &user == nil || user.ID <= 0 {
		log.Println("CreateBatchEvalJob user not found",
			zap.Int("userId", userId))
		return fmt.Errorf("CreateBatchEvalJob user not found")
	}
	taskUuid := uuid.New().String()
	evalTask := &models.EvalBatchTask{
		DatasetId:             req.DatasetId,
		UserId:                userId,
		Status:                "pending",
		TaskType:              req.EvalTaskType,
		Name:                  req.Name,
		TaskUuid:              taskUuid,
		CandidateSystemPrompt: req.CandidateSystemPrompt,
		CandidateUserPrompt:   req.CandidateUserPrompt,
		JudgeSystemPrompt:     req.JudgeSystemPrompt,
		ModelA:                req.ModelA,
		ModelB:                req.ModelB,
		ModelJudge:            req.ModelJudge,
		DatasetItem:           req.DatasetItem,
	}
	tx := s.DB.Session(&gorm.Session{}).Begin()
	if err := tx.Create(evalTask).Error; err != nil {
		tx.Rollback()
		logger.Log.Error("CreateBatchEvalJob tx err"+err.Error(),
			zap.Int("userId", userId),
		)
		return err
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		logger.Log.Error("CreateBatchEvalJob tx commit err"+err.Error(),
			zap.Int("userId", userId))
		return err
	}
	// publish
	channel, err := s.Conn.Channel()
	if err != nil {
		log.Printf("RabbitMQ channel error: %v", err)
		logger.Log.Error("CreateBatchEvalJob channel err"+err.Error(),
			zap.Int("userId", userId))
		return err
	}
	defer channel.Close()
	// 声明队列
	_, err = channel.QueueDeclare(
		EvalQueueName,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	)
	if err != nil {
		logger.Log.Error("CreateBatchEvalJob channel queue err"+err.Error(),
			zap.Int("userId", userId))
		log.Fatalf("CreateBatchEvalJob channel queue err " + err.Error())
		return err
	}
	payload, err := json.Marshal(evalTask)
	if err != nil {
		logger.Log.Error("CreateBatchEvalJob marshal err"+err.Error(),
			zap.Int("userId", userId))
		log.Fatalf("CreateBatchEvalJob marshal err " + err.Error())
		return err
	}
	err = channel.Publish(
		"",
		EvalQueueName,
		true,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         payload,
			DeliveryMode: amqp.Persistent,
		},
	)
	if err != nil {
		logger.Log.Error("CreateBatchEvalJob channel publish err"+err.Error(),
			zap.Int("userId", userId))
		log.Printf("RabbitMQ publish error: %v", err)
		return err
	}
	logger.Log.Info("CreateBatchEvalJob success",
		zap.Int("userId", userId))
	log.Printf("CreateBatchEvalJob publish rabbitmq success")
	return nil
}

func (s *Service) PreviewEval(w http.ResponseWriter, ctx context.Context, req *models.EvalBatchTaskRequest) {
	samples, err := s.GetInputMessages(req.DatasetItem, req.DatasetId, *req.CandidateSystemPrompt, *req.CandidateUserPrompt)
	if err != nil {
		fmt.Printf("error during processing prompt, err: %v", err)
		return
	}
	for _, sample := range samples {
		s.ProcessSampleStream(w, ctx, sample, req)
	}
	return
}

func (s *Service) RunEvalTask(ctx context.Context, job *models.EvalBatchTask) ([]models.ModelResult, error) {
	samples, err := s.GetInputMessages(job.DatasetItem, job.DatasetId, *job.CandidateSystemPrompt, *job.CandidateUserPrompt)
	if err != nil {
		logger.Log.Error("RunEvalTask GetInputMessages err"+err.Error(),
			zap.String("task_uuid", job.TaskUuid),
			zap.Int("userId", job.UserId))
		return nil, err
	}

	concurrency := MaxConcurrency
	inputChan := make(chan models.Sample)
	outputChan := make(chan models.ModelResult, len(samples))

	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for sample := range inputChan {
				select {
				case <-ctx.Done():
					return
				default:
					res := s.ProcessSample(ctx, sample, job)
					select {
					case outputChan <- res:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	go func() {
		for _, sample := range samples {
			select {
			case <-ctx.Done():
				break
			case inputChan <- sample:
			}
		}
		close(inputChan)
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(outputChan)
		close(done)
	}()

	var results []models.ModelResult
	for res := range outputChan {
		results = append(results, res)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
		return results, nil
	}
}

func (s *Service) SaveEvalResults(job *models.EvalBatchTask, results []models.ModelResult) error {
	var evalResults []models.EvalTaskResult
	for _, result := range results {
		evalResults = append(evalResults, models.EvalTaskResult{
			TaskUuid:              &job.TaskUuid,
			DatasetId:             job.DatasetId,
			UserId:                job.UserId,
			DatasetItemId:         result.ID,
			ResponseA:             &result.AnswerA,
			ResponseB:             &result.AnswerB,
			JudgeResponse:         &result.Judge,
			TaskType:              job.TaskType,
			CandidateSystemPrompt: &result.CandidateSystemPrompt,
			CandidateUserPrompt:   &result.CandidateUserPrompt,
			JudgeSystemPrompt:     &result.JudgeSystemPrompt,
			JudgeUserPrompt:       &result.JudgeUserPrompt,
		})
	}
	tx := s.DB.Session(&gorm.Session{}).Begin()
	if err := tx.Model(models.EvalTaskResult{}).Create(evalResults).Error; err != nil {
		logger.Log.Error("SaveEvalResults tx err" + err.Error())
		tx.Rollback()
	}
	if err := tx.Commit().Error; err != nil {
		logger.Log.Error("SaveEvalResults tx commit err" + err.Error())
		tx.Rollback()
	}
	return nil
}

func (s *Service) GetInputReplaceVariables(input string, values map[string]string) (string, error) {
	// replace {{}} variables with actual dataset contents
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)

	result := re.ReplaceAllStringFunc(input, func(match string) string {
		key := re.FindStringSubmatch(match)[1]
		if val, ok := values[key]; ok {
			return val
		}
		return match
	})
	return result, nil
}

func (s *Service) GetInputMessages(datasetItem int, datasetId int, sysPrompt, userPrompt string) ([]models.Sample, error) {
	var datasetItems []*models.DatasetItem
	if err := s.DB.Model(models.DatasetItem{}).Where("dataset_id = ? and is_deleted = false", datasetId).Limit(datasetItem).Find(&datasetItems).Error; err != nil {
		log.Printf("DatasetItems error: %v", err)
		return nil, err
	}

	var samples []models.Sample
	for _, item := range datasetItems {
		var values map[string]string
		err := json.Unmarshal([]byte(item.RawContent), &values)
		if err != nil {
			fmt.Printf("error during unmarshal raw contents from sql, err: %v", err)
			return nil, err
		}
		prompt := make([]*schema.Message, 0)
		handleSysPrompt, err := s.GetInputReplaceVariables(sysPrompt, values)
		if err != nil {
			fmt.Printf("GetInputReplaceVariables error: %v", err)
			return nil, err
		}
		handleUserPrompt, err := s.GetInputReplaceVariables(userPrompt, values)
		if err != nil {
			fmt.Printf("GetInputReplaceVariables error: %v", err)
		}
		prompt = append(prompt, &schema.Message{
			Role:    "system",
			Content: handleSysPrompt,
		})
		prompt = append(prompt, &schema.Message{
			Role:    "user",
			Content: handleUserPrompt,
		})
		samples = append(samples, models.Sample{
			ID:     int(item.ID),
			Prompt: prompt,
		},
		)
	}
	return samples, nil
}

func (s *Service) ProcessSample(ctx context.Context, sample models.Sample, req *models.EvalBatchTask) models.ModelResult {
	answerA, errA := s.CallModel(ctx, sample.Prompt, &req.ModelA)
	answerB, errB := s.CallModel(ctx, sample.Prompt, &req.ModelB)

	if errA != nil || errB != nil {
		return models.ModelResult{Error: fmt.Errorf("model error: %v | %v", errA, errB)}
	}

	judgePrompt := s.GenerateJudgePrompt(*req.JudgeSystemPrompt, &sample, answerA, answerB)
	judgeAnswer, errJ := s.CallModel(ctx, judgePrompt, &req.ModelJudge)
	if errJ != nil {
		return models.ModelResult{Error: fmt.Errorf("judge error: %v", errJ)}
	}

	return models.ModelResult{
		ID:                    sample.ID,
		AnswerA:               answerA.Content,
		AnswerB:               answerB.Content,
		Judge:                 judgeAnswer.Content,
		CandidateSystemPrompt: sample.Prompt[0].Content,
		CandidateUserPrompt:   sample.Prompt[1].Content,
		JudgeSystemPrompt:     *req.JudgeSystemPrompt,
		JudgeUserPrompt:       judgePrompt[1].Content,
	}
}

func (s *Service) CallModel(ctx context.Context, in []*schema.Message, config *models.ModelConfig) (out *schema.Message, err error) {
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:      config.APIKey,
		BaseURL:     config.BaseURL,
		Model:       config.Model,
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
		TopP:        config.TopP,
	})
	output, err := chatModel.Generate(ctx, in)
	if err != nil {
		fmt.Printf("callModel error: %v", err)
	}
	return output, nil
}

func (s *Service) GenerateJudgePrompt(sysPrompt string, sample *models.Sample, answerA *schema.Message, answerB *schema.Message) []*schema.Message {
	builder := strings.Builder{}
	builder.WriteString("Given the input: ")
	builder.WriteString(sample.Prompt[1].Content)
	builder.WriteString("response from model A:")
	builder.WriteString(answerA.Content)
	builder.WriteString("\n")
	builder.WriteString("response from model B:")
	builder.WriteString(answerB.Content)
	builder.WriteString("\n")
	return []*schema.Message{
		{
			Role:    "system",
			Content: sysPrompt,
		},
		{
			Role:    "user",
			Content: builder.String(),
		},
	}
}

func (s *Service) CallModelStream(ctx context.Context, in []*schema.Message, config *models.ModelConfig) (out *schema.StreamReader[*schema.Message], err error) {
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:      config.APIKey,
		BaseURL:     config.BaseURL,
		Model:       config.Model,
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
		TopP:        config.TopP,
	})
	outputStream, err := chatModel.Stream(ctx, in)
	if err != nil {
		fmt.Printf("callModel error: %v", err)
	}
	return outputStream, nil
}

func (s *Service) ProcessSampleStream(w http.ResponseWriter, ctx context.Context, sample models.Sample, req *models.EvalBatchTaskRequest) {
	answerStreamA, errA := s.CallModelStream(ctx, sample.Prompt, &req.ModelA)
	answerStreamB, errB := s.CallModelStream(ctx, sample.Prompt, &req.ModelB)
	if errA != nil || errB != nil {
		fmt.Printf("model error: %v | %v", errA, errB)
		return
	}
	answerA, err := s.ResponseStreamSSE(answerStreamA, w)
	if err != nil {
		fmt.Printf("model error during model A streaming: %v", err)
		return
	}
	answerB, err := s.ResponseStreamSSE(answerStreamB, w)
	if err != nil {
		fmt.Printf("model error during model B streaming: %v", err)
		return
	}
	judgePrompt := s.GenerateJudgePrompt(*req.JudgeSystemPrompt, &sample, answerA, answerB)
	judgeStream, errJ := s.CallModelStream(ctx, judgePrompt, &req.ModelJudge)
	if errJ != nil {
		fmt.Printf("judge error: %v", errJ)
		return
	}
	_, err = s.ResponseStreamSSE(judgeStream, w)
	if err != nil {
		fmt.Printf("model error during judge streaming: %v", err)
		return
	}
	return
}

func (s *Service) ResponseStreamSSE(sr *schema.StreamReader[*schema.Message], w http.ResponseWriter) (res *schema.Message, err error) {
	defer sr.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	var content strings.Builder
	i := 0

	for {
		message, err := sr.Recv()
		if err == io.EOF {
			// 最后 flush 一次
			endMessage := models.SSEMessage{
				Type:            "end",
				Content:         "",
				FinishingReason: true,
			}
			jsonBytes, err := json.Marshal(endMessage)
			if err != nil {
				log.Printf("json marshal failed: %v", err)
				return nil, err
			}
			// SSE 格式发送
			msg := fmt.Sprintf("data: %s\n\n", string(jsonBytes))
			_, err = w.Write([]byte(msg))
			if err != nil {
				return nil, err
			}
			flusher.Flush()
			break
		}
		if err != nil {
			log.Printf("recv failed: %v", err)
			return nil, err
		}

		// 包装成结构体
		sseMsg := models.SSEMessage{
			Type:            "update", // 可以根据业务动态设置
			Content:         message.Content,
			FinishingReason: false,
		}

		// 转为 JSON
		jsonBytes, err := json.Marshal(sseMsg)
		if err != nil {
			log.Printf("json marshal failed: %v", err)
			return nil, err
		}

		// SSE 格式发送
		msg := fmt.Sprintf("data: %s\n\n", string(jsonBytes))
		_, err = w.Write([]byte(msg))
		if err != nil {
			return nil, err
		}

		flusher.Flush()
		content.WriteString(message.Content)
		i++
	}

	res = &schema.Message{
		Role:    schema.Assistant,
		Content: content.String(),
	}
	return res, nil
}
