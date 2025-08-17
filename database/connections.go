package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"awesomeEval/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	amqp "github.com/streadway/amqp"
)

type Connections struct {
	PostgreSQL *sqlx.DB
	Redis      *redis.Client
	RabbitMQ   *amqp.Connection
}

var GlobalConnections *Connections

func InitializeConnections(cfg *config.Config) error {
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

func (c *Connections) initPostgreSQL(cfg config.PostgreSQLConfig) error {
	dsn := cfg.GetDSN()
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return fmt.Errorf("连接PostgreSQL失败: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 测试连接
	if err := db.Ping(); err != nil {
		return fmt.Errorf("PostgreSQL连接测试失败: %w", err)
	}

	c.PostgreSQL = db
	log.Println("PostgreSQL连接成功")
	return nil
}

func (c *Connections) initRedis(cfg config.RedisConfig) error {
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

func (c *Connections) initRabbitMQ(cfg config.RabbitMQConfig) error {
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
		if err := c.PostgreSQL.Close(); err != nil {
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

// 获取PostgreSQL连接
func GetPostgreSQL() *sqlx.DB {
	if GlobalConnections != nil {
		return GlobalConnections.PostgreSQL
	}
	return nil
}

// 获取Redis连接
func GetRedis() *redis.Client {
	if GlobalConnections != nil {
		return GlobalConnections.Redis
	}
	return nil
}

// 获取RabbitMQ连接
func GetRabbitMQ() *amqp.Connection {
	if GlobalConnections != nil {
		return GlobalConnections.RabbitMQ
	}
	return nil
}
