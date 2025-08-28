package main

import (
	"awesomeEval/config"
	"awesomeEval/internal/mq"
	"context"
	"log"
)

var worker *mq.DatasetWorker

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
	worker = mq.NewDatasetWorker(config.GetRabbitMQ(), config.GetPostgreSQL(), config.GetMinio(), config.GetRedis())
}

func main() {
	err := worker.Start(context.Background())
	if err != nil {
		return
	}
}
