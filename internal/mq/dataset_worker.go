package mq

import (
	"awesomeEval/internal/models"
	"awesomeEval/internal/service/dataset"
	"context"
	"encoding/json"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
	"gorm.io/gorm"
	"log"
)

type DatasetWorker struct {
	Service *dataset.Service
}

func NewDatasetWorker(conn *amqp.Connection, db *gorm.DB, minioClient *minio.Client, redisClient *redis.Client) *DatasetWorker {
	return &DatasetWorker{
		Service: dataset.NewService(conn, db, minioClient, redisClient),
	}
}

func (w *DatasetWorker) Start(ctx context.Context) error {
	channel, err := w.Service.Conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	// 确保队列存在
	_, err = channel.QueueDeclare(
		"dataset_jobs", // 队列名称
		true,           // durable
		false,          // autoDelete
		false,          // exclusive
		false,          // noWait
		nil,            // arguments
	)
	if err != nil {
		return err
	}

	msgs, err := channel.Consume(
		"dataset_jobs", // 队列名称
		"",             // consumer tag
		false,          // auto-ack
		false,          // exclusive
		false,          // no-local
		false,          // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	log.Println("DatasetWorker started, waiting for jobs...")

	for msg := range msgs {
		go func(m amqp.Delivery) {
			if err := w.handleMessage(ctx, m); err != nil {
				log.Printf("Job error: %v", err)
				m.Nack(false, true) // 不确认并重新入队
			} else {
				m.Ack(false)
			}
		}(msg)
	}

	return nil
}

func (w *DatasetWorker) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	var jobMsg models.DatasetIOJob
	if err := json.Unmarshal(msg.Body, &jobMsg); err != nil {
		log.Printf("JSON decode error: %v", err)
		return err
	}
	// 从数据库中加载完整 Job（包括最新状态）
	var job models.DatasetIOJob
	if err := w.Service.DB.First(&job, jobMsg.ID).Error; err != nil {
		log.Printf("⚠️ DB fetch job error: %v", err)
		return err
	}

	log.Printf("Processing dataset job id=%d, dataset_id=%d", job.ID, job.DatasetID)
	return w.Service.ProcessDatasetJob(ctx, &job)
}
