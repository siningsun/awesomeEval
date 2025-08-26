package models

import "awesomeEval/internal/utils"

type DatasetDB struct {
	utils.BaseModel
	Name        string `json:"name"`
	Description string `json:"description"`
	UserID      uint   `json:"user_id"`
	FilePath    string `json:"file_path"`
}
type DatasetIOJob struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	DatasetID uint   `json:"dataset_id"`
	JobType   string `json:"job_type"` // e.g., "import" or "export"
	Status    string `json:"status"`   // e.g., "pending", "in_progress", "completed", "failed"
	UserID    uint   `json:"user_id"`
	FilePath  string `json:"file_path"`
}

type DatasetItem struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	DatasetID  uint   `json:"dataset_id"`
	RawContent string `json:"raw_content"`
	UserID     uint   `json:"user_id"`
	IsDeleted  bool   `json:"is_deleted"`
}

type CreateDatasetRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
