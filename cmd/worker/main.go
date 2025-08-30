package main

import (
	"awesomeEval/config"
	"awesomeEval/internal/mq"
	"context"
	"fmt"
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
	DatasetWorker = mq.NewDatasetWorker(config.GetRabbitMQ(), config.GetPostgreSQL(), config.GetMinio(), config.GetRedis())
	EvalWorker = mq.NewEvalWorker(config.GetRabbitMQ(), config.GetPostgreSQL(), config.GetMinio(), config.GetRedis())
}

func main() {
	err := DatasetWorker.Start(context.Background())
	if err != nil {
		return
	}
	fmt.Print("DatasetWorker starts!")
	err = EvalWorker.Start(context.Background())
	if err != nil {
		return
	}
	fmt.Print("EvalWorker starts!")
}
