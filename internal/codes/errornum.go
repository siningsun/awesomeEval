package codes

const (
	CommonInvalidParamCode  = 1001
	CommonInternalErrorCode = 1002
	CommonDbErrorCode       = 1003
	UserAlreadyExistsCode   = 2001
	UserNotFoundCode        = 2002
	PasswordMismatchCode    = 2003
)

var DefaultMessages = map[int]map[string]string{
	CommonInvalidParamCode: {
		"en": "Invalid parameters",
		"cn": "无效的参数",
	},
	CommonInternalErrorCode: {
		"en": "Internal server codes",
		"cn": "服务器内部错误",
	},
	CommonDbErrorCode: {
		"en": "Database codes",
		"cn": "数据库错误",
	},
	UserAlreadyExistsCode: {
		"en": "User already exists",
		"cn": "用户已存在",
	},
	UserNotFoundCode: {
		"en": "User not found",
		"cn": "用户未找到",
	},
	PasswordMismatchCode: {
		"en": "Password mismatch",
		"cn": "密码不匹配",
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
