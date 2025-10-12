package main

import (
	"awesomeEval/config"
	"awesomeEval/internal/handler"
	"awesomeEval/internal/logger"
	"awesomeEval/internal/service/dataset"
	"awesomeEval/internal/service/eval"
	"awesomeEval/internal/service/sms"
	"awesomeEval/internal/ws"
	"awesomeEval/middleware"
	"context"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func init() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始连接
	if err := config.InitializeConnections(cfg); err != nil {
		log.Fatalf("初始化连接失败: %v", err)
	}

	// init Logger
	if err := logger.InitLogger(cfg, "gin-api"); err != nil {
		log.Fatalf("init logger failed, error: %v", err)
	}
}

func main() {
	ctx := context.Background()

	// start WebSocket hub
	hub := ws.NewHub()
	go hub.Run()

	// start Redis subscriber
	ws.StartRedisSubscriber(ctx, config.GlobalConnections.Redis, hub)

	// start Gin server
	router := gin.Default()
	router.Use(ginzap.Ginzap(logger.Log, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(logger.Log, true))

	// 测试日志
	logger.Log.Info("server starts successfully")

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "message": "server is healthy"})
	})

	// WebSocket 路由必须在 Run 之前注册
	router.GET("/ws", ws.ServeWs(hub))

	// 其他业务路由
	authHandler := middleware.NewAuthHandler(config.GlobalConnections.Redis)
	userHandler := handler.NewUserHandler(
		config.GlobalConnections.PostgreSQL,
		config.GlobalConnections.Redis,
		&sms.MockSmsService{},
	)
	router.POST("/signup", userHandler.Signup)
	router.POST("/loginByPassword", userHandler.LoginByPassword)
	router.POST("/loginByCode", userHandler.LoginByMobile)

	authorized := router.Group("/", authHandler.AuthMiddleware())
	{
		datasetHandler := handler.NewDatasetHandler(&dataset.Service{
			DB:          config.GlobalConnections.PostgreSQL,
			Conn:        config.GlobalConnections.RabbitMQ,
			RedisClient: config.GlobalConnections.Redis,
			MinioClient: config.GlobalConnections.Minio,
		})
		evalHandler := handler.NewEvalHandler(&eval.Service{
			Conn:        config.GlobalConnections.RabbitMQ,
			DB:          config.GlobalConnections.PostgreSQL,
			RedisClient: config.GlobalConnections.Redis,
			MinioClient: config.GlobalConnections.Minio,
		})

		authorized.POST("/dataset/create", datasetHandler.CreateDataset)
		authorized.GET("/dataset/list", datasetHandler.ListDatasets)
		authorized.GET("/dataset/view", datasetHandler.ListDatasetItems)
		authorized.POST("/eval/task/create", evalHandler.CreateBatchEvalJob)
		authorized.POST("/eval/preview", evalHandler.PreviewEval)
	}

	// 优雅关闭
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("server shutting down...")

	// 优雅关闭 Gin
	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctxShutDown); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}

	// 关闭数据库连接
	if config.GlobalConnections != nil {
		config.GlobalConnections.Close()
	}

	log.Println("server closed.")
}
