package models

type ModelConfig struct {
	ModelType   int     `json:"model_type"` // local or remote
	Model       string  `json:"model"`
	APIKey      string  `json:"api_key"`
	BaseURL     string  `json:"base_url"`
	TopP        float64 `json:"top_p"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
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
	JudgeUserPrompt       string      `json:"judge_user_prompt"`
	DatasetItem           int         `json:"dataset_item"`
}

type EvalBatchTaskDB struct {
	DatasetId             int         `json:"dataset_id" gorm:"column:dataset_id"`
	UserId                int         `json:"user_id" gorm:"column:user_id"`
	Status                string      `json:"status" gorm:"column:status"`
	TaskType              int         `json:"task_type" gorm:"column:task_type"`
	Name                  string      `json:"name" gorm:"column:name"`
	TaskUuid              string      `json:"task_uuid" gorm:"column:task_uuid"`
	CandidateSystemPrompt *string     `json:"candidate_system_prompt" gorm:"column:candidate_system_prompt"`
	CandidateUserPrompt   *string     `json:"candidate_user_prompt" gorm:"column:candidate_user_prompt"`
	JudgeSystemPrompt     *string     `json:"judge_system_prompt" gorm:"column:judge_system_prompt"`
	JudgeUserPrompt       *string     `json:"judge_user_prompt" gorm:"column:judge_user_prompt"`
	ModelA                ModelConfig `json:"model_a" gorm:"column:model_a"`
	ModelB                ModelConfig `json:"model_b" gorm:"model_b"`
	ModelJudge            ModelConfig `json:"model_judge" gorm:"model_judge"`
	DatasetItem           int         `json:"dataset_item" gorm:"dataset_item"`
}

func (*EvalBatchTaskDB) TableName() string {
	return "eval_batch_tasks"
}
