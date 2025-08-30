package eval

import (
	"awesomeEval/internal/models"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
	"gorm.io/gorm"
	"log"
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
	evalTask := &models.EvalBatchTaskDB{
		DatasetId:             req.DatasetId,
		UserId:                userId,
		Status:                "pending",
		TaskType:              req.EvalTaskType,
		Name:                  req.Name,
		TaskUuid:              taskUuid,
		CandidateSystemPrompt: &req.CandidateSystemPrompt,
		CandidateUserPrompt:   &req.CandidateUserPrompt,
		JudgeSystemPrompt:     &req.JudgeSystemPrompt,
		JudgeUserPrompt:       &req.JudgeUserPrompt,
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

// RunEvalTask todo: 提供超时和失败重试机制
func (s *Service) RunEvalTask(job *models.EvalBatchTaskDB) error {
	return nil
}
