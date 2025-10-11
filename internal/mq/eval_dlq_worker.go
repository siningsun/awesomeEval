package mq

import (
	"awesomeEval/internal/models"
	"awesomeEval/internal/service/eval"
	"context"
	"encoding/json"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
	"gorm.io/gorm"
	"log"
)

type DLQWorker struct {
	Service *eval.Service
}

func NewDLQWorker(conn *amqp.Connection, db *gorm.DB, minioClient *minio.Client, redisClient *redis.Client) *DLQWorker {
	return &DLQWorker{
		Service: eval.NewEvalService(conn, db, minioClient, redisClient),
	}
}

func (w *DLQWorker) Start(ctx context.Context) error {
	channel, err := w.Service.Conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	_, err = channel.QueueDeclare(EvalDLQName, true, false, false, false, nil)
	if err != nil {
		return err
	}

	msgs, err := channel.Consume(EvalDLQName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return nil
			}
			w.handleDLQMessage(msg)
		}
	}
}

// 标记任务为失败
func (w *DLQWorker) handleDLQMessage(msg amqp.Delivery) {
	var task models.EvalBatchTask
	if err := json.Unmarshal(msg.Body, &task); err != nil {
		msg.Ack(false)
		return
	}

	// 更新任务状态为失败
	if err := w.Service.DB.Model(&models.EvalBatchTask{}).
		Where("task_uuid = ?", task.TaskUuid).
		Updates(map[string]interface{}{"status": models.TaskFailed, "updated_at": gorm.Expr("NOW()")}).Error; err != nil {
		log.Printf("Update task to failed error: %v", err)
		msg.Nack(false, true) // 失败重试
		return
	}

	msg.Ack(false)
}
