package models

import (
	"awesomeEval/internal/utils"
)

type DatasetDB struct {
	utils.BaseModel
	Name        string `json:"name" gorm:"column:name;uniqueIndex"`
	Description string `json:"description" gorm:"column:description"`
	UserID      uint   `json:"user_id" gorm:"column:user_id"`
	FilePath    string `json:"file_path" gorm:"column:file_path"`
	IsDeleted   bool   `json:"is_deleted" gorm:"column:is_deleted;default:false"`
}

type DatasetResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	UserID      uint   `json:"user_id"`
	FilePath    string `json:"file_path"`
	CreatedAt   int64  `json:"created_at"`
}
type DatasetIOJob struct {
	utils.BaseModel
	DatasetID uint   `json:"dataset_id" gorm:"column:dataset_id"`
	JobType   string `json:"job_type" gorm:"column:job_type"`             // e.g., "import" or "export"
	Status    string `json:"status" gorm:"column:status;default:pending"` // e.g., "pending", "in_progress", "completed", "failed"
	UserID    uint   `json:"user_id" gorm:"column:user_id"`
	FilePath  string `json:"file_path" gorm:"column:file_path"`
	IsDeleted bool   `json:"is_deleted" gorm:"column:is_deleted;default:false"`
}

type DatasetItem struct {
	utils.BaseModel
	DatasetID  uint   `json:"dataset_id" gorm:"column:dataset_id;index"`
	RawContent string `json:"raw_content" gorm:"column:raw_content;type:jsonb"`
	UserID     uint   `json:"user_id" gorm:"column:user_id"`
	IsDeleted  bool   `json:"is_deleted" gorm:"column:is_deleted;default:false"`
}

type DatasetItemResponse struct {
	ID        uint `json:"id"`
	DatasetID uint `json:"dataset_id"`
	UserID    uint `json:"user_id"`
	// parse RawContent to map
	RawContent map[string]interface{} `json:"raw_content"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
}

type CreateDatasetRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (*DatasetDB) TableName() string {
	return "dataset_metadata"
}

func (*DatasetIOJob) TableName() string {
	return "dataset_io_jobs"
}

func (*DatasetItem) TableName() string {
	return "dataset_item"
}
