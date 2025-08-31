package eval

import (
	"awesomeEval/internal/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
	"gorm.io/gorm"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
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
	// todo: select only userId
	var user models.UserDB
	if err := s.DB.Model(models.UserDB{}).Where("user_id = ?", userId).First(&user).Error; err != nil {
		return err
	}
	if &user == nil || user.ID <= 0 {
		return errors.New("invalid user")
	}
	taskUuid := uuid.New().String()
	evalTask := &models.EvalBatchTask{
		DatasetId:             req.DatasetId,
		UserId:                userId,
		Status:                "pending",
		TaskType:              req.EvalTaskType,
		Name:                  req.Name,
		TaskUuid:              taskUuid,
		CandidateSystemPrompt: &req.CandidateSystemPrompt,
		CandidateUserPrompt:   &req.CandidateUserPrompt,
		JudgeSystemPrompt:     &req.JudgeSystemPrompt,
		ModelA:                req.ModelA,
		ModelB:                req.ModelB,
		ModelJudge:            req.ModelJudge,
		DatasetItem:           req.DatasetItem,
	}
	tx := s.DB.Session(&gorm.Session{}).Begin()
	if err := tx.Create(evalTask).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}
	// publish
	channel, err := s.Conn.Channel()
	if err != nil {
		log.Printf("RabbitMQ channel error: %v", err)
		return err
	}
	defer channel.Close()
	// 声明队列
	_, err = channel.QueueDeclare(
		"eval_jobs",
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(evalTask)
	if err != nil {
		return err
	}
	err = channel.Publish(
		"",
		"eval_task_jobs",
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         payload,
			DeliveryMode: amqp.Persistent,
			Expiration:   strconv.FormatInt(int64(2*time.Hour), 10),
		},
	)
	if err != nil {
		log.Printf("RabbitMQ publish error: %v", err)
		return err
	}
	return nil
}

func (s *Service) PreviewEval(ctx context.Context) error {

	return nil
}

// RunEvalTask 并发处理
func (s *Service) RunEvalTask(ctx context.Context, job *models.EvalBatchTask) error {
	// prepare sample
	samples, err := s.GetInputMessages(job.DatasetId, *job.CandidateSystemPrompt, *job.CandidateUserPrompt)
	if err != nil {
		fmt.Printf("GetInputMessages error: %v", err)
		return err
	}
	// batch
	concurrency := 15
	var wg sync.WaitGroup
	inputChan := make(chan models.Sample)
	outputChan := make(chan models.ModelResult, len(samples))
	// Workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for sample := range inputChan {
				res := s.ProcessSample(ctx, sample, job)
				outputChan <- res
			}
		}()
	}
	// Feed data
	go func() {
		for _, sample := range samples {
			inputChan <- sample
		}
		close(inputChan)
	}()
	// Wait for all to finish
	wg.Wait()
	close(outputChan)
	// Collect results
	var results []models.ModelResult
	for res := range outputChan {
		results = append(results, res)
	}
	// save to Db

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

func (s *Service) GetInputMessages(datasetId int, sysPrompt, userPrompt string) ([]models.Sample, error) {
	var datasetItems []*models.DatasetItem
	if err := s.DB.Model(models.DatasetItem{}).Where("dataset_id = ? && is_deleted = false", datasetId).Find(&datasetItems).Error; err != nil {
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
			Role:    "System",
			Content: handleSysPrompt,
		})
		prompt = append(prompt, &schema.Message{
			Role:    "User",
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
		AnswerA: answerA.Content,
		AnswerB: answerB.Content,
		Judge:   judgeAnswer.Content,
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
	return []*schema.Message{
		{
			Role:    "System",
			Content: sysPrompt,
		},
		{
			Role:    "User",
			Content: builder.String(),
		},
	}
}
