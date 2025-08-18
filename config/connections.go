package config

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	amqp "github.com/streadway/amqp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"time"
)

type Connections struct {
	PostgreSQL *gorm.DB
	Redis      *redis.Client
	RabbitMQ   *amqp.Connection
}

var GlobalConnections *Connections

func InitializeConnections(cfg *Config) error {
	conns := &Connections{}

	// 初始化PostgreSQL连接
	if err := conns.initPostgreSQL(cfg.PostgreSQL); err != nil {
		return fmt.Errorf("初始化PostgreSQL失败: %w", err)
	}

	// 初始化Redis连接
	if err := conns.initRedis(cfg.Redis); err != nil {
		return fmt.Errorf("初始化Redis失败: %w", err)
	}

	// 初始化RabbitMQ连接
	if err := conns.initRabbitMQ(cfg.RabbitMQ); err != nil {
		return fmt.Errorf("初始化RabbitMQ失败: %w", err)
	}

	GlobalConnections = conns
	log.Println("所有数据库连接初始化成功")
	return nil
}

func (c *Connections) initPostgreSQL(cfg PostgreSQLConfig) error {
	dsn := cfg.GetDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接PostgreSQL失败: %w", err)
	}
	sqlDB, err := db.DB()
	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("PostgreSQL连接测试失败: %w", err)
	}
	// 设置连接池参数
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	// 赋值
	c.PostgreSQL = db
	log.Println("PostgreSQL连接成功")
	return nil
}

func (c *Connections) initRedis(cfg RedisConfig) error {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.GetAddr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis连接测试失败: %w", err)
	}

	c.Redis = rdb
	log.Println("Redis连接成功")
	return nil
}

func (c *Connections) initRabbitMQ(cfg RabbitMQConfig) error {
	url := cfg.GetURL()
	conn, err := amqp.Dial(url)
	if err != nil {
		return fmt.Errorf("连接RabbitMQ失败: %w", err)
	}

	c.RabbitMQ = conn
	log.Println("RabbitMQ连接成功")
	return nil
}

func (c *Connections) Close() {
	if c.PostgreSQL != nil {
		sqlDB, _ := c.PostgreSQL.DB()
		if err := sqlDB.Close(); err != nil {
			log.Printf("关闭PostgreSQL连接失败: %v", err)
		}
	}

	if c.Redis != nil {
		if err := c.Redis.Close(); err != nil {
			log.Printf("关闭Redis连接失败: %v", err)
		}
	}

	if c.RabbitMQ != nil {
		if err := c.RabbitMQ.Close(); err != nil {
			log.Printf("关闭RabbitMQ连接失败: %v", err)
		}
	}

	log.Println("所有数据库连接已关闭")
}

// GetPostgreSQL 获取PostgreSQL连接
func GetPostgreSQL() *gorm.DB {
	if GlobalConnections != nil {
		return GlobalConnections.PostgreSQL
	}
	return nil
}

// GetRedis 获取Redis连接
func GetRedis() *redis.Client {
	if GlobalConnections != nil {
		return GlobalConnections.Redis
	}
	return nil
}

// GetRabbitMQ 获取RabbitMQ连接
func GetRabbitMQ() *amqp.Connection {
	if GlobalConnections != nil {
		return GlobalConnections.RabbitMQ
	}
	return nil
}
