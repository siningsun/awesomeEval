package handler

import (
	"awesomeEval/error"
	"awesomeEval/models"
	"awesomeEval/utils"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type UserHandler struct {
	DB    *gorm.DB
	Redis *redis.Client
}

// NewUserHandler 创建一个新的 UserHandler 实例
func NewUserHandler(globalDB *gorm.DB, globalRedis *redis.Client) *UserHandler {
	return &UserHandler{
		DB:    globalDB,
		Redis: globalRedis,
	}
}

// Signup 处理用户注册请求
func (h *UserHandler) Signup(c *gin.Context) {
	var user models.UserRegisterRequest
	if err := c.ShouldBindBodyWithJSON(&user); err != nil {
		c.JSON(400, error.NewByCode(error.CommonInvalidParamCode, ""))
		return
	}
	if *user.Email == "" || *user.Password == "" {
		c.JSON(400, error.NewByCode(error.CommonInvalidParamCode, ""))
		return
	}
	// validate email format
	if !utils.IsValidEmail(*user.Email) {
		c.JSON(400, error.NewByCode(error.CommonInvalidParamCode, ""))
		return
	}
	// validate password strength
	if !utils.IsValidPassword(*user.Password) {
		c.JSON(400, error.NewByCode(error.CommonInvalidParamCode, ""))
		return
	}
	// validate email uniqueness
	var existingUser models.UserDB
	if err := h.DB.Where("email = ?", user.Email).First(&existingUser).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(500, error.NewByCode(error.CommonDbErrorCode, ""))
			return
		}
	} else {
		c.JSON(400, error.NewByCode(error.UserAlreadyExistsCode, ""))
		return
	}
	// 创建新用户
	userDB := models.UserDB{
		Email:     user.Email,
		Password:  user.Password,
		AvatarUrl: nil,
		NickName:  nil,
		Mobile:    nil,
	}
	if err := h.DB.Create(&userDB).Error; err != nil {
		c.JSON(500, error.NewByCode(error.CommonDbErrorCode, ""))
		return
	}
	c.JSON(200, models.UserRegisterResponse{
		UserInfo:   &userDB,
		Token:      nil, // 这里可以生成 JWT 或其他类型的令牌
		ExpireTime: nil, // 这里可以设置令牌的过期时间
	})
}

// Login 处理用户登录请求
func (h *UserHandler) Login(c *gin.Context) {
	// 这里可以添加用户登录的逻辑
	// 例如解析请求体、验证用户凭据、生成 JWT 等

	// 示例响应
	c.JSON(200, gin.H{
		"message": "用户登录成功",
	})
}

// GetUserInfo 获取用户信息
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	// 这里可以添加获取用户信息的逻辑
	// 例如从数据库中查询用户信息

	// 示例响应
	c.JSON(200, gin.H{})
}

// UpdateUserInfo 更新用户信息
func (h *UserHandler) UpdateUserInfo(c *gin.Context) {
	// 这里可以添加更新用户信息的逻辑
	// 例如解析请求体、验证数据、更新到数据库等

	// 示例响应
	c.JSON(200, gin.H{
		"message": "用户信息更新成功",
	})
}

// DeleteUser 删除用户
func (h *UserHandler) DeleteUser(c *gin.Context) {
	// 这里可以添加删除用户的逻辑
	// 例如从数据库中删除用户记录

	// 示例响应
	c.JSON(200, gin.H{
		"message": "用户删除成功",
	})
}
