package main

import (
	"awesomeEval/handler"
	"log"
	"os"
	"os/signal"
	"syscall"

	"awesomeEval/config"
	"github.com/gin-gonic/gin"
)

func init() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库连接
	if err := config.InitializeConnections(cfg); err != nil {
		log.Fatalf("初始化数据库连接失败: %v", err)
	}
}

func main() {
	router := gin.Default()

	// 添加健康检查路由
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "服务运行正常",
		})
	})

	// New UserHandler
	userHandler := handler.NewUserHandler(config.GlobalConnections.PostgreSQL, config.GlobalConnections.Redis)

	router.POST("/signup", userHandler.Signup)
	router.POST("/loginByPassword", userHandler.LoginByPassword)

	// 优雅关闭
	go func() {
		if err := router.Run(":8080"); err != nil {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭服务器...")

	// 关闭数据库连接
	if config.GlobalConnections != nil {
		config.GlobalConnections.Close()
	}

	log.Println("服务器已关闭")
}
