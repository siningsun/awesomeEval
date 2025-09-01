package models

import (
	"awesomeEval/internal/utils"
	"github.com/cloudwego/eino/schema"
)

type ModelConfig struct {
	ModelType   int      `json:"model_type"` // local or remote
	Model       string   `json:"model"`
	APIKey      string   `json:"api_key"`
	BaseURL     string   `json:"base_url"`
	TopP        *float32 `json:"top_p"`
	Temperature *float32 `json:"temperature"`
	MaxTokens   *int     `json:"max_tokens"`
}

type EvalBatchTaskRequest struct {
	EvalTaskType          int         `json:"eval_task_type"`
	Name                  string      `json:"name"`
	ModelA                ModelConfig `json:"model_a"`
	ModelB                ModelConfig `json:"model_b"`
	ModelJudge            ModelConfig `json:"model_judge"`
	DatasetId             int         `json:"dataset_id"`
	CandidateSystemPrompt string      `json:"candidate_system_prompt"`
	CandidateUserPrompt   string      `json:"candidate_user_prompt"`
	JudgeSystemPrompt     string      `json:"judge_system_prompt"`
	DatasetItem           int         `json:"dataset_item"`
}

type EvalBatchTask struct {
	utils.BaseModel
	DatasetId             int         `json:"dataset_id" gorm:"column:dataset_id"`
	UserId                int         `json:"user_id" gorm:"column:user_id"`
	Status                string      `json:"status" gorm:"column:status"`
	TaskType              int         `json:"task_type" gorm:"column:task_type"`
	Name                  string      `json:"name" gorm:"column:name"`
	TaskUuid              string      `json:"task_uuid" gorm:"column:task_uuid"`
	CandidateSystemPrompt *string     `json:"candidate_system_prompt" gorm:"column:candidate_system_prompt"`
	CandidateUserPrompt   *string     `json:"candidate_user_prompt" gorm:"column:candidate_user_prompt"`
	JudgeSystemPrompt     *string     `json:"judge_system_prompt" gorm:"column:judge_system_prompt"`
	ModelA                ModelConfig `json:"model_a" gorm:"column:model_a"`
	ModelB                ModelConfig `json:"model_b" gorm:"model_b"`
	ModelJudge            ModelConfig `json:"model_judge" gorm:"model_judge"`
	DatasetItem           int         `json:"dataset_item" gorm:"dataset_item"`
	IsDeleted             bool        `json:"is_deleted" gorm:"column:is_deleted"`
}

type EvalTaskResult struct {
	DatasetItemId         int     `json:"dataset_item_id" gorm:"column:dataset_item_id"`
	TaskUuid              *string `json:"task_uuid" gorm:"column:task_uuid"`
	DatasetId             int     `json:"dataset_id" gorm:"column:dataset_id"`
	UserId                int     `json:"user_id" gorm:"column:user_id"`
	CandidateSystemPrompt *string `json:"candidate_system_prompt" gorm:"column:candidate_system_prompt"`
	CandidateUserPrompt   *string `json:"candidate_user_prompt" gorm:"column:candidate_user_prompt"`
	JudgeSystemPrompt     *string `json:"judge_system_prompt" gorm:"column:judge_system_prompt"`
	JudgeUserPrompt       *string `json:"judge_user_prompt" gorm:"column:judge_user_prompt"`
	TaskType              int     `json:"task_type" gorm:"column:task_type"`
	ResponseA             *string `json:"response_a" gorm:"column:response_a"`
	ResponseB             *string `json:"response_b" gorm:"column:response_b"`
	JudgeResponse         *string `json:"judge_response" gorm:"column:judge_response"`
	IsDeleted             bool    `json:"is_deleted" gorm:"column:is_deleted"`
}

type Sample struct {
	ID     int               `json:"id"`
	Prompt []*schema.Message `json:"prompt"`
}

type ModelResult struct {
	ID                    int
	AnswerA               string
	AnswerB               string
	Judge                 string
	Error                 error
	CandidateSystemPrompt string
	CandidateUserPrompt   string
	JudgeSystemPrompt     string
	JudgeUserPrompt       string
}

func (*EvalBatchTask) TableName() string {
	return "eval_batch_tasks"
}

func (*EvalTaskResult) TableName() string {
	return "eval_task_results"
}
