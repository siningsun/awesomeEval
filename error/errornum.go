package error

const (
	CommonInvalidParamCode  = 1001
	CommonInternalErrorCode = 1002
	CommonDbErrorCode       = 1003
	UserAlreadyExistsCode   = 2001
)

var DefaultMessages = map[int]map[string]string{
	CommonInvalidParamCode: {
		"en": "Invalid parameters",
		"cn": "无效的参数",
	},
	CommonInternalErrorCode: {
		"en": "Internal server error",
		"cn": "服务器内部错误",
	},
	CommonDbErrorCode: {
		"en": "Database error",
		"cn": "数据库错误",
	},
	UserAlreadyExistsCode: {
		"en": "User already exists",
		"cn": "用户已存在",
	},
}

type ErrorCode struct {
	Code         int    `json:"code"`
	Message      string `json:"message,omitempty"`
	ExtraMessage string `json:"extra_message,omitempty"`
}

func NewByCode(code int, extraMsg string) *ErrorCode {
	message := DefaultMessages[code]
	if extraMsg != "" {
		return &ErrorCode{
			Code:         code,
			Message:      message[CurrentLang],
			ExtraMessage: extraMsg,
		}
	}
	return &ErrorCode{
		Code:         code,
		Message:      message[CurrentLang],
		ExtraMessage: "",
	}
}
