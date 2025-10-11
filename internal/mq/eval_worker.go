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
	"sync"
	"time"
)

const (
	EvalQueueName      = "eval_jobs"
	EvalTimeout        = 1 * time.Hour
	EvalMaxRetry       = 5
	EvalWorkerPoolSize = 15
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

	_, err = channel.QueueDeclare(
		EvalQueueName, // 队列名称
		true,          // durable
		false,         // autoDelete
		false,         // exclusive
		false,         // noWait
		nil,           // arguments
	)
	if err != nil {
		return err
	}

	msgs, err := channel.Consume(
		EvalQueueName, // 队列名称
		"",            // consumer tag
		false,         // auto-ack
		false,         // exclusive
		false,         // no-local
		false,         // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	logger.Log.Info("EvalWorker started, waiting for jobs...",
		zap.String("queue", EvalQueueName),
		zap.Int("worker_pool_size", EvalWorkerPoolSize),
	)

	// revise, restrict worker pool size
	wg := sync.WaitGroup{}
	for msg := range msgs {
		select {
		case <-ctx.Done():
			log.Println("Worker context canceled")
			break
		case w.PoolSem <- struct{}{}:
			wg.Add(1)
			go func(m amqp.Delivery) {
				defer wg.Done()
				defer func() { <-w.PoolSem }()
				if err := w.processMessage(ctx, channel, m); err != nil {
					log.Printf("Task failed: %v", err)
				}
			}(msg)
		}
	}
	wg.Wait()
	return nil
}

// 处理单条消息
func (w *EvalWorker) processMessage(ctx context.Context, channel *amqp.Channel, msg amqp.Delivery) error {
	var task models.EvalBatchTask
	if err := json.Unmarshal(msg.Body, &task); err != nil {
		logger.Log.Error("Failed to unmarshal task", zap.Error(err),
			zap.String("body", string(msg.Body)),
			zap.String("queue", EvalQueueName),
		)
		msg.Nack(false, false)
		return err
	}

	var taskDB models.EvalBatchTask
	if err := w.Service.DB.Where("task_uuid = ?", task.TaskUuid).First(&taskDB).Error; err != nil {
		logger.Log.Error("Failed to query task in DB", zap.Error(err),
			zap.String("task_uuid", task.TaskUuid),
			zap.String("queue", EvalQueueName))
		msg.Nack(false, false)
		return err
	}

	if taskDB.Status == models.TaskSuccess {
		msg.Ack(false)
		return nil
	}

	// 获取当前重试次数
	retryCount := 0
	if val, ok := msg.Headers["x-retry"]; ok {
		if intVal, ok := val.(int32); ok {
			retryCount = int(intVal)
		}
	}

	// 超时 Context
	taskCtx, cancel := context.WithTimeout(ctx, EvalTimeout)
	defer cancel()

	// 执行任务
	err := w.Service.RunEvalTask(taskCtx, &task)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("Failed to run task, current retry: %d. ", retryCount)+err.Error(),
			zap.String("task_uuid", task.TaskUuid),
			zap.String("queue", EvalQueueName))
		return w.handleFailure(channel, msg, task, retryCount)
	}

	// 更新任务状态为成功
	taskDB.Status = models.TaskSuccess
	if err := w.Service.DB.Save(&taskDB).Error; err != nil {
		logger.Log.Error("Failed to save task in DB", zap.Error(err),
			zap.String("task_uuid", task.TaskUuid),
			zap.String("queue", EvalQueueName))
		msg.Nack(false, true)
		return err
	}

	// 推送消息给前端
	if err := w.publishResult(task.TaskUuid, models.TaskSuccess, "Task completed"); err != nil {
		log.Printf("Publish result error: %v", err)
	}

	msg.Ack(false)
	logger.Log.Info("Task finished successfully",
		zap.String("task_uuid", task.TaskUuid),
		zap.String("queue", EvalQueueName))
	return nil
}

// 处理失败任务
func (w *EvalWorker) handleFailure(channel *amqp.Channel, msg amqp.Delivery, task models.EvalBatchTask, retryCount int) error {
	if retryCount >= EvalMaxRetry {
		logger.Log.Info(fmt.Sprintf("Task %s retry limit exceeded", task.TaskUuid),
			zap.String("task_uuid", task.TaskUuid),
			zap.String("queue", EvalQueueName))
		msg.Nack(false, false) // DLQ, 不再重试
		return nil
	}

	// 重新投递
	newHeaders := msg.Headers
	if newHeaders == nil {
		newHeaders = amqp.Table{}
	}
	newHeaders["x-retry"] = int32(retryCount + 1)

	if err := channel.Publish(
		"",
		msg.RoutingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: msg.ContentType,
			Body:        msg.Body,
			Headers:     newHeaders,
		},
	); err != nil {
		logger.Log.Error("Failed to publish message", zap.Error(err),
			zap.String("task_uuid", task.TaskUuid),
			zap.String("queue", EvalQueueName))
		msg.Nack(false, true) // requeue, still trying
		return err
	}

	msg.Ack(false)
	logger.Log.Info(fmt.Sprintf("Task retrying: %d times", retryCount),
		zap.String("task_uuid", task.TaskUuid),
		zap.String("queue", EvalQueueName))
	return nil
}

// push任务的执行结果给前端
func (w *EvalWorker) publishResult(taskUuid string, status string, msg string) error {
	result := models.TaskResult{
		TaskUuid: taskUuid,
		Status:   status,
		Message:  msg,
	}
	body, _ := json.Marshal(result)
	channel := "eval_task_results" // 前端订阅频道
	return w.Service.RedisClient.Publish(context.Background(), channel, body).Err()
}
