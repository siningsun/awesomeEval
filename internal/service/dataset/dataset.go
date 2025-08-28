package dataset

import (
	"awesomeEval/internal/models"
	"awesomeEval/internal/utils"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
	"gorm.io/gorm"
	"log"
	"mime/multipart"
	"path/filepath"
)

type Service struct {
	Conn        *amqp.Connection
	DB          *gorm.DB
	MinioClient *minio.Client
	RedisClient *redis.Client
}

func NewService(conn *amqp.Connection, db *gorm.DB, minioClient *minio.Client, redisClient *redis.Client) *Service {
	return &Service{
		Conn:        conn,
		DB:          db,
		MinioClient: minioClient,
		RedisClient: redisClient,
	}
}

func (s *Service) CreateDataset(file multipart.File, header *multipart.FileHeader, request *models.CreateDatasetRequest, userID int64, ctx context.Context) error {
	// 开启事务
	tx := s.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建 dataset 记录
	datasetDB := &models.DatasetDB{
		BaseModel:   utils.BaseModel{CreatedBy: userID},
		Name:        request.Name,
		Description: request.Description,
		UserID:      uint(userID),
		FilePath:    "",
	}

	if err := tx.Create(datasetDB).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 上传文件到 MinIO
	bucketName := "datasets"
	objectName := uuid.New().String() + filepath.Ext(header.Filename)

	// 检查桶是否存在
	found, err := s.MinioClient.BucketExists(ctx, bucketName)
	if err != nil {
		tx.Rollback()
		fmt.Printf("MinIO bucket check error: %v", err)
		return err
	}
	if !found {
		err = s.MinioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			tx.Rollback()
			log.Printf("MinIO bucket creation error: %v", err)
			return err
		}
	}

	// 上传文件
	_, err = s.MinioClient.PutObject(ctx, bucketName, objectName, file, header.Size, minio.PutObjectOptions{
		ContentType: header.Header.Get("Content-Type"),
	})
	if err != nil {
		tx.Rollback()
		log.Printf("MinIO upload error: %v", err)
		return err
	}

	// 上传成功，更新数据库文件路径
	datasetDB.FilePath = objectName
	if err := tx.Save(datasetDB).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 创建异步 IO Job
	job := &models.DatasetIOJob{
		DatasetID: uint(datasetDB.ID),
		JobType:   "import",
		Status:    "pending",
		UserID:    uint(userID),
		FilePath:  objectName,
	}
	if err := tx.Create(job).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return err
	}

	// 推送 RabbitMQ 异步任务
	channel, err := s.Conn.Channel()
	if err != nil {
		log.Printf("RabbitMQ channel error: %v", err)
		return err
	}
	defer channel.Close()

	// 声明队列
	_, err = channel.QueueDeclare(
		"dataset_jobs",
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	)
	if err != nil {
		return err
	}

	// 发布消息
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	err = channel.Publish(
		"",
		"dataset_jobs",
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         payload,
			DeliveryMode: amqp.Persistent,
		},
	)
	if err != nil {
		log.Printf("RabbitMQ publish error: %v", err)
		return err
	}

	return nil
}

// ProcessDatasetJob 处理异步任务 todo: 目前只实现了 import, 需要增加失败重试机制和进度条计算功能
func (s *Service) ProcessDatasetJob(ctx context.Context, m *models.DatasetIOJob) error {
	// 模拟处理时间
	log.Printf("Processing dataset job ID %d of type %s", m.ID, m.JobType)
	switch m.JobType {
	case "import":
		log.Printf("Importing dataset ID %d", m.DatasetID)
		err := s.RunInsertDatasetItemsJob(m)
		if err != nil {
			log.Printf("Failed to import dataset ID %d: %v", m.DatasetID, err)
			return err
		}
	case "export":
		log.Printf("Exporting dataset ID %d", m.DatasetID)
		//todo: 实现导出逻辑
	default:
		log.Printf("Unknown job type: %s", m.JobType)
	}
	// 更新 Job 状态为 completed
	tx := s.DB.Session(&gorm.Session{}).Begin()
	if err := tx.Model(&models.DatasetIOJob{}).Where("id = ?", m.ID).Updates(map[string]interface{}{
		"status":     "completed",
		"updated_at": gorm.Expr("NOW()"),
	}).Error; err != nil {
		log.Printf("Failed to update job status: %v", err)
		tx.Rollback()
		return err
	}
	if err := tx.Commit().Error; err != nil {
		log.Printf("Failed to commit job status update: %v", err)
	}
	log.Printf("Completed dataset job ID %d", m.ID)
	return nil
}

func (s *Service) RunInsertDatasetItemsJob(job *models.DatasetIOJob) error {
	// 从 MinIO 下载文件
	bucketName := "datasets"
	objectName := job.FilePath

	object, err := s.MinioClient.GetObject(context.Background(), bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		log.Printf("MinIO get object error: %v", err)
		return err
	}
	defer object.Close()
	// validate JSON keys
	isValid, errLine, keys, err := s.ValidateJsonKeys(object)
	if err != nil || !isValid {
		log.Printf("Dataset validation failed at line %d: %v", errLine, err)
		isValid = false
		return fmt.Errorf("dataset validation failed at line %d: %v", errLine, err)
	}
	log.Printf("Dataset JSON keys: %v", keys)
	// update valid status to dataset_metadata
	tx := s.DB.Session(&gorm.Session{}).Begin()
	if err := tx.Model(&models.DatasetDB{}).Where("id = ?", job.DatasetID).Updates(map[string]interface{}{
		"is_valid":     isValid,
		"dataset_keys": keys,
		"updated_at":   gorm.Expr("NOW()"),
	}).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to update dataset valid status: %v", err)
	}
	if err := tx.Commit().Error; err != nil {
		log.Printf("Failed to commit dataset valid status update: %v", err)
	}
	// 读取数据集，jsonl格式, 每行一个 JSON 对象
	// 重新打开对象，因为上面的扫描已经读完了
	object, err = s.MinioClient.GetObject(context.Background(), bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		log.Printf("MinIO get object error: %v", err)
		return err
	}
	// 逐行读取文件内容
	var items []models.DatasetItem
	scanner := bufio.NewScanner(object)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			items = append(items, models.DatasetItem{
				DatasetID:  job.DatasetID,
				RawContent: line,
				UserID:     job.UserID,
				IsDeleted:  false,
			})
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from object: %v", err)
		return err
	}

	// 批量插入数据项
	tx = s.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	batchSize := 500
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batch := items[i:end]
		if err := tx.Create(&batch).Error; err != nil {
			log.Printf("DB insert error: %v", err)
			tx.Rollback()
			return err
		}
	}
	// 提交事务
	if err := tx.Commit().Error; err != nil {
		log.Printf("DB commit error: %v", err)
		return err
	}
	return nil
}

func (s *Service) ValidateJsonKeys(object *minio.Object) (bool, int, map[string]struct{}, error) {
	scanner := bufio.NewScanner(object)
	var keys map[string]struct{}
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(line), &obj); err != nil {
				log.Printf("Invalid JSON line: %v", err)
				return false, lineCount, nil, fmt.Errorf("invalid JSON format")
			}
			if keys == nil {
				keys = make(map[string]struct{})
				for k := range obj {
					keys[k] = struct{}{}
				}
			} else {
				if len(keys) != len(obj) {
					return false, lineCount, nil, fmt.Errorf("inconsistent JSON keys")
				}
				for k := range obj {
					if _, exists := keys[k]; !exists {
						return false, lineCount, nil, fmt.Errorf("inconsistent JSON keys")
					}
				}
			}
			lineCount++
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from object: %v", err)
		return false, lineCount, nil, err
	}
	return true, lineCount, keys, nil
}

func (s *Service) ListDatasets(userId int64, ctx context.Context) ([]models.DatasetResponse, error) {
	var datasets []models.DatasetDB
	if err := s.DB.Where("user_id = ? AND is_deleted = false And is_valid = true", userId).Find(&datasets).Error; err != nil {
		return nil, err
	}
	var resp []models.DatasetResponse
	for _, dataset := range datasets {
		resp = append(resp, models.DatasetResponse{
			ID:          uint(dataset.ID),
			Name:        dataset.Name,
			Description: dataset.Description,
			UserID:      uint(userId),
			FilePath:    dataset.FilePath,
			CreatedAt:   dataset.CreatedAt.Unix(),
		})
	}
	//todo: 写入 redis
	s.RedisClient.Set(ctx, fmt.Sprintf("datasets:user:%d", userId), resp, utils.ExpireDuration)
	return resp, nil
}

func (s *Service) ListDatasetItems(datasetId int, userId int64, pageNum, pageSize int) ([]map[string]interface{}, error) {
	var items []models.DatasetItem
	if err := s.DB.Where("dataset_id = ? AND user_id = ? AND is_deleted = false", datasetId, userId).
		Offset((pageNum - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, err
	}
	var resp []map[string]interface{}
	for _, item := range items {
		var content map[string]interface{}
		if err := json.Unmarshal([]byte(item.RawContent), &content); err != nil {
			log.Printf("Failed to unmarshal dataset item ID %d: %v", item.ID, err)
			continue
		}
		content["id"] = item.ID
		content["created_at"] = item.CreatedAt.Unix()
		resp = append(resp, content)
	}
	return resp, nil
}
