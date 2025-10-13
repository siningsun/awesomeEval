package config

import (
	"errors"
	"fmt"
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	PostgreSQL PostgreSQLConfig `mapstructure:"postgresql"`
	Redis      RedisConfig      `mapstructure:"redis"`
	RabbitMQ   RabbitMQConfig   `mapstructure:"rabbitmq"`
	Minio      MinioConfig      `mapstructure:"minio"`
	Log        LogConfig        `mapstructure:"log"`
}

type PostgreSQLConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type RabbitMQConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	VHost    string `mapstructure:"vhost"`
}

type MinioConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"accessKeyID"`
	SecretAccessKey string `mapstructure:"secretAccessKey"`
	BucketName      string `mapstructure:"bucketName"`
	UseSSL          bool   `mapstructure:"useSSL"`
}

type LogConfig struct {
	Level            string   `mapstructure:"level"`
	Encoding         string   `mapstructure:"encoding"`
	OutputPaths      []string `mapstructure:"outputPaths"`
	ErrorOutputPaths []string `mapstructure:"errorOutputPaths"`
	Development      bool     `mapstructure:"development"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// 设置默认值
	viper.SetDefault("postgresql.host", "localhost")
	viper.SetDefault("postgresql.port", 5432)
	viper.SetDefault("postgresql.sslmode", "disable")

	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.db", 0)

	viper.SetDefault("rabbitmq.host", "localhost")
	viper.SetDefault("rabbitmq.port", 5672)
	viper.SetDefault("rabbitmq.vhost", "/")

	viper.SetDefault("minio.endpoint", "localhost:9000")
	viper.SetDefault("minio.useSSL", false)
	viper.SetDefault("minio.bucketName", "my-bucket")

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("error: %w", err)
		}
		log.Println("cannot find config file, using default values")
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return &config, nil
}

func (c *PostgreSQLConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

func (c *RedisConfig) GetAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *RabbitMQConfig) GetURL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		c.User, c.Password, c.Host, c.Port, c.VHost)
}

func (c *MinioConfig) GetDSN() string {
	return fmt.Sprintf("%s (AccessKeyID: %s, Bucket: %s)", c.Endpoint, c.AccessKeyID, c.BucketName)
}
