package main

import (
	"awesomeEval/config"
	"awesomeEval/internal/logger"
	"awesomeEval/internal/mq"
	"fmt"
	"golang.org/x/net/context"
	"log"
)

var DatasetWorker *mq.DatasetWorker

var EvalWorker *mq.EvalWorker

func init() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	// 初始化连接
	if err := config.InitializeConnections(cfg); err != nil {
		log.Fatalf("初始化连接失败: %v", err)
	}
	// init logger
	if err := logger.InitLogger(cfg); err != nil {
		log.Fatal("初始化日志失败: ", err)
	}
	DatasetWorker = mq.NewDatasetWorker(config.GetRabbitMQ(), config.GetPostgreSQL(), config.GetMinio(), config.GetRedis())
	EvalWorker = mq.NewEvalWorker(config.GetRabbitMQ(), config.GetPostgreSQL(), config.GetMinio(), config.GetRedis())
}

func main() {
	ctx := context.Background()
	errCh := make(chan error, 2)

	go func() {
		if err := DatasetWorker.Start(ctx); err != nil {
			errCh <- fmt.Errorf("DatasetWorker: %w", err)
			return
		}
		errCh <- nil
	}()

	go func() {
		if err := EvalWorker.Start(ctx); err != nil {
			errCh <- fmt.Errorf("EvalWorker: %w", err)
			return
		}
		errCh <- nil
	}()

	// 等待两个 worker 启动结果
	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}
