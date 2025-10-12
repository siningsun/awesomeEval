package mq

import (
	"awesomeEval/internal/logger"
	"awesomeEval/internal/models"
	"awesomeEval/internal/service/eval"
	"context"
	"encoding/json"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
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
			w.handleDLQMessage(ctx, msg)
		}
	}
}

// 标记任务为失败
func (w *DLQWorker) handleDLQMessage(ctx context.Context, msg amqp.Delivery) {
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

	// publish to redis channel
	result := models.TaskResult{
		TaskUuid: task.TaskUuid,
		Status:   models.TaskFailed,
		Message:  "Task failed due to processing error.",
	}
	body, _ := json.Marshal(result)
	channel := EvalTaskRedisChannel // 前端订阅频道
	if err := w.Service.RedisClient.Publish(ctx, channel, body).Err(); err != nil {
		logger.Log.Error(fmt.Sprintf("Publish task failed message error, %v", err),
			zap.String("task_uuid", task.TaskUuid),
		)
		msg.Nack(false, true) // 失败重试
		return
	}
	msg.Ack(false)
}
