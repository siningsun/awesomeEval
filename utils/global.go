package utils

import "time"

// BaseModel 定义通用字段
type BaseModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	CreatedBy int64
	UpdatedBy int64
}
