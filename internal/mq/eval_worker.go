package mq

import (
	"awesomeEval/internal/service/eval"
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
	"gorm.io/gorm"
	"log"
)

type EvalWorker struct {
	Service *eval.Service
}

func NewEvalWorker(conn *amqp.Connection, db *gorm.DB, minioClient *minio.Client, redisClient *redis.Client) *EvalWorker {
	return &EvalWorker{
		Service: eval.NewEvalService(conn, db, minioClient, redisClient),
	}
}

func (w *EvalWorker) Start(ctx context.Context) error {
	channel, err := w.Service.Conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	// 确保队列存在
	_, err = channel.QueueDeclare(
		"eval_jobs", // 队列名称
		true,        // durable
		false,       // autoDelete
		false,       // exclusive
		false,       // noWait
		nil,         // arguments
	)
	if err != nil {
		return err
	}

	msgs, err := channel.Consume(
		"eval_jobs", // 队列名称
		"",          // consumer tag
		false,       // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
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

func (w *EvalWorker) handleMessage(ctx context.Context, msg amqp.Delivery) error {

	return nil
}
