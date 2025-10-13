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
	"math"
	"time"
)

const (
	EvalQueueName        = "eval_jobs"
	EvalDLQName          = "eval_jobs_dlq"
	EvalTimeout          = 1 * time.Hour
	EvalMaxRetry         = 5
	EvalWorkerPoolSize   = 15
	EvalTaskRedisChannel = "eval_task_results"
)

type EvalWorker struct {
	Service *eval.Service
	PoolSem chan struct{}
}

func NewEvalWorker(conn *amqp.Connection, db *gorm.DB, minioClient *minio.Client, redisClient *redis.Client) *EvalWorker {
	return &EvalWorker{
		Service: eval.NewEvalService(conn, db, minioClient, redisClient),
		PoolSem: make(chan struct{}, EvalWorkerPoolSize),
	}
}

func (w *EvalWorker) Start(ctx context.Context) error {
	channel, err := w.Service.Conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	// 声明死信队列
	_, _ = channel.QueueDeclare(
		EvalDLQName,
		true,
		false,
		false,
		false,
		nil,
	)

	// 声明主队列并绑定死信队列
	_, err = channel.QueueDeclare(
		EvalQueueName,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": EvalDLQName,
		},
	)
	if err != nil {
		return err
	}

	msgs, err := channel.Consume(
		EvalQueueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	logger.Log.Info("EvalWorker started, waiting for jobs...",
		zap.String("queue", EvalQueueName),
		zap.Int("worker_pool_size", EvalWorkerPoolSize),
	)

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("EvalWorker context canceled, shutting down...")
			return nil
		case msg, ok := <-msgs:
			if !ok {
				logger.Log.Warn("Message channel closed, shutting down EvalWorker...")
				return nil
			}

			w.PoolSem <- struct{}{}
			go func(m amqp.Delivery) {
				defer func() { <-w.PoolSem }()

				if err := w.processMessage(ctx, channel, m); err != nil {
					log.Printf("Task failed: %v", err)
				}
			}(msg)
		}
	}
}

func (w *EvalWorker) processMessage(ctx context.Context, channel *amqp.Channel, msg amqp.Delivery) error {
	var task models.EvalBatchTask
	if err := json.Unmarshal(msg.Body, &task); err != nil {
		logger.Log.Error("Failed to unmarshal task", zap.Error(err))
		msg.Nack(false, false)
		return err
	}

	var taskDB models.EvalBatchTask
	if err := w.Service.DB.Where("task_uuid = ?", task.TaskUuid).First(&taskDB).Error; err != nil {
		logger.Log.Error("Failed to query task in DB", zap.Error(err))
		msg.Nack(false, false)
		return err
	}

	if taskDB.Status == models.TaskSuccess {
		msg.Ack(false)
		return nil
	}

	retryCount := 0
	if val, ok := msg.Headers["x-retry"]; ok {
		if floatVal, ok := val.(float64); ok { // amqp.Table 默认 float64
			retryCount = int(floatVal)
		}
	}

	taskCtx, cancel := context.WithTimeout(ctx, EvalTimeout)
	defer cancel()

	logger.Log.Info("Executing task",
		zap.String("task_uuid", task.TaskUuid),
		zap.Int("retry_count", retryCount))

	taskRes, err := w.Service.RunEvalTask(taskCtx, &task)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("Failed to run task, retry %d", retryCount), zap.Error(err))
		return w.handleFailure(channel, msg, task, retryCount)
	}

	taskDB.Status = models.TaskSuccess

	if err := w.Service.DB.Save(&taskDB).Error; err != nil {
		logger.Log.Error("Failed to save task in DB", zap.Error(err))
		msg.Nack(false, true)
		return err
	}

	logger.Log.Info("Updating Task status in DB succeeded",
		zap.String("task_uuid", task.TaskUuid),
		zap.Int("retry_count", retryCount))

	if err := w.Service.SaveEvalResults(&task, taskRes); err != nil {
		logger.Log.Error("Failed to save task results in DB", zap.Error(err))
		msg.Nack(false, true)
		return err
	}

	logger.Log.Info("Updating Task results in DB succeeded",
		zap.String("task_uuid", task.TaskUuid),
		zap.Int("retry_count", retryCount))

	if err := w.publishResult(ctx, task.TaskUuid, models.TaskSuccess, fmt.Sprintf("task: %s success", task.TaskUuid)); err != nil {
		log.Printf("Publish result error: %v", err)
	}

	logger.Log.Info("Publish result to Redis succeeded",
		zap.String("task_uuid", task.TaskUuid),
		zap.Int("retry_count", retryCount))

	msg.Ack(false)

	logger.Log.Info("Task finished successfully",
		zap.String("task_uuid", task.TaskUuid),
		zap.Int("retry_count", retryCount))

	return nil
}

func (w *EvalWorker) handleFailure(channel *amqp.Channel, msg amqp.Delivery, task models.EvalBatchTask, retryCount int) error {
	if retryCount >= EvalMaxRetry {
		logger.Log.Info("Retry limit exceeded, sending to DLQ", zap.String("task_uuid", task.TaskUuid))
		msg.Nack(false, false)
		return nil
	}

	// 指数退避，延迟重试
	delay := time.Duration(math.Pow(2, float64(retryCount))) * time.Second
	time.Sleep(delay)

	newHeaders := msg.Headers
	if newHeaders == nil {
		newHeaders = amqp.Table{}
	}
	newHeaders["x-retry"] = float64(retryCount + 1)

	if err := channel.Publish(
		"",
		msg.RoutingKey,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  msg.ContentType,
			Body:         msg.Body,
			Headers:      newHeaders,
		},
	); err != nil {
		logger.Log.Error("Failed to publish message", zap.Error(err))
		msg.Nack(false, true)
		return err
	}

	msg.Ack(false)
	logger.Log.Info(fmt.Sprintf("Task retrying: %d times", retryCount+1),
		zap.String("task_uuid", task.TaskUuid))
	return nil
}

// push任务的执行结果给前端
func (w *EvalWorker) publishResult(ctx context.Context, taskUuid string, status string, msg string) error {
	result := models.TaskResult{
		TaskUuid: taskUuid,
		Status:   status,
		Message:  msg,
	}
	body, _ := json.Marshal(result)
	channel := EvalTaskRedisChannel // 前端订阅频道
	return w.Service.RedisClient.Publish(ctx, channel, body).Err()
}
