package dataset

import (
	"awesomeEval/internal/models"
	"bufio"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
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
}

func NewService(conn *amqp.Connection, db *gorm.DB, minioClient *minio.Client) *Service {
	return &Service{
		Conn:        conn,
		DB:          db,
		MinioClient: minioClient,
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
		log.Printf("MinIO bucket check error: %v", err)
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

func (s *Service) ProcessDatasetJob(ctx context.Context, m *models.DatasetIOJob) error {
	// 模拟处理时间
	log.Printf("Processing dataset job ID %d of type %s", m.ID, m.JobType)
	switch m.JobType {
	case "import":
		log.Printf("Importing dataset ID %d", m.DatasetID)
		err := s.RunInsertDatasetItemsJob(m)
		if err != nil {
			return err
		}
	case "export":
		log.Printf("Exporting dataset ID %d", m.DatasetID)
		//todo: 实现导出逻辑
	default:
		log.Printf("Unknown job type: %s", m.JobType)
	}
	// 更新 Job 状态为 completed
	if err := s.DB.Model(&models.DatasetIOJob{}).Where("id = ?", m.ID).Updates(map[string]interface{}{
		"status":     "completed",
		"updated_at": gorm.Expr("NOW()"),
	}).Error; err != nil {
		log.Printf("Failed to update job status: %v", err)
		return err
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

	// 读取数据集，jsonl格式, 每行一个 JSON 对象
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
	tx := s.DB.Begin()
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
