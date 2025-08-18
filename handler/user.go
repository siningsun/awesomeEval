package handler

import (
	"awesomeEval/errorsc"
	"awesomeEval/models"
	"awesomeEval/utils"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"strconv"
	"time"
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
		c.JSON(400, errorsc.NewByCode(errorsc.CommonInvalidParamCode, ""))
		return
	}
	if *user.Email == "" || *user.Password == "" {
		c.JSON(400, errorsc.NewByCode(errorsc.CommonInvalidParamCode, ""))
		return
	}
	// validate email format
	if !utils.IsValidEmail(*user.Email) {
		c.JSON(400, errorsc.NewByCode(errorsc.CommonInvalidParamCode, ""))
		return
	}
	// validate password strength
	if !utils.IsValidPassword(*user.Password) {
		c.JSON(400, errorsc.NewByCode(errorsc.CommonInvalidParamCode, ""))
		return
	}
	// validate email uniqueness
	var existingUser models.UserDB
	if err := h.DB.Where("email = ?", user.Email).First(&existingUser).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(500, errorsc.NewByCode(errorsc.CommonDbErrorCode, ""))
			return
		}
	} else {
		c.JSON(400, errorsc.NewByCode(errorsc.UserAlreadyExistsCode, ""))
		return
	}
	// 创建新用户
	hashedPassword, err := utils.HashPassword(*user.Password)
	if err != nil {
		c.JSON(500, errorsc.NewByCode(errorsc.CommonInternalErrorCode, ""))
		return
	}
	userDB := models.UserDB{
		Email:     user.Email,
		Password:  &hashedPassword,
		AvatarUrl: nil,
		NickName:  nil,
		Mobile:    nil,
	}
	if err := h.DB.Create(&userDB).Error; err != nil {
		c.JSON(500, errorsc.NewByCode(errorsc.CommonDbErrorCode, ""))
		return
	}
	tokenString, err := utils.GenerateJWT(userDB.ID)
	if err != nil {
		c.JSON(500, errorsc.NewByCode(errorsc.CommonInternalErrorCode, ""))
		return
	}
	c.JSON(200, models.UserRegisterResponse{
		UserInfo: &models.UserInfo{
			ID:        userDB.ID,
			Email:     *userDB.Email,
			AvatarUrl: *userDB.AvatarUrl,
			NickName:  *userDB.NickName,
		},
		Token:      &tokenString,                                // 这里可以生成 JWT 或其他类型的令牌
		ExpireTime: time.Now().Add(utils.ExpireDuration).Unix(), // 这里可以设置令牌的过期时间
	})
}

// LoginByMobile 处理用户登录手机号+验证码请求
func (h *UserHandler) LoginByMobile(c *gin.Context) {
	// 这里可以添加用户登录的逻辑
	// 例如解析请求体、验证用户凭据、生成 JWT 等
	// 示例响应
}

// GetUserInfo 获取用户信息
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, errorsc.NewByCode(errorsc.CommonInvalidParamCode, ""))
		return
	}
	userId, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(400, errorsc.NewByCode(errorsc.CommonInvalidParamCode, ""))
		return
	}
	userList, err := h.GetUserFromDB(int64(userId), "", "", "")
	if err != nil {
		c.JSON(500, errorsc.NewByCode(errorsc.CommonDbErrorCode, ""))
		return
	}
	if len(userList) == 0 {
		c.JSON(404, errorsc.NewByCode(errorsc.UserNotFoundCode, ""))
		return
	}
	user := userList[0]
	c.JSON(200, models.UserInfo{
		ID:        user.ID,
		Email:     *user.Email,
		AvatarUrl: *user.AvatarUrl,
		NickName:  *user.NickName,
	})
}

// UpdateUserInfo 更新用户信息
func (h *UserHandler) UpdateUserInfo(c *gin.Context) {
	// 这里可以添加更新用户信息的逻辑
	var updateRequest models.UpdateUserInfoRequest
	if err := c.ShouldBindBodyWithJSON(&updateRequest); err != nil {
		c.JSON(400, errorsc.NewByCode(errorsc.CommonInvalidParamCode, ""))
		return
	}
	if updateRequest.UserId <= 0 {
		c.JSON(400, errorsc.NewByCode(errorsc.CommonInvalidParamCode, ""))
		return
	}
	if updateRequest.NickName == nil && updateRequest.AvatarUrl == nil && updateRequest.Mobile == nil {
		c.JSON(400, errorsc.NewByCode(errorsc.CommonInvalidParamCode, ""))
		return
	}
	// 查询用户是否存在
	var userDB models.UserDB
	if err := h.DB.Where("id = ?", updateRequest.UserId).First(&userDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, errorsc.NewByCode(errorsc.UserNotFoundCode, ""))
			return
		}
		c.JSON(500, errorsc.NewByCode(errorsc.CommonDbErrorCode, ""))
		return
	}
	// 更新用户信息
	if updateRequest.NickName != nil {
		userDB.NickName = updateRequest.NickName
	}
	if updateRequest.AvatarUrl != nil {
		userDB.AvatarUrl = updateRequest.AvatarUrl
	}
	if updateRequest.Mobile != nil {
		userDB.Mobile = updateRequest.Mobile
	}
	if err := h.DB.Save(&userDB).Error; err != nil {
		c.JSON(500, errorsc.NewByCode(errorsc.CommonDbErrorCode, ""))
		return
	}
	c.JSON(200, models.UserInfo{
		ID:        userDB.ID,
		Email:     *userDB.Email,
		AvatarUrl: *userDB.AvatarUrl,
		NickName:  *userDB.NickName,
	})
}

// DeleteUser 删除用户
func (h *UserHandler) DeleteUser(c *gin.Context) {
	// 这里可以添加删除用户的逻辑
	var deleteRequest models.DeleteUserRequest
	if err := c.ShouldBindBodyWithJSON(&deleteRequest); err != nil {
		c.JSON(400, errorsc.NewByCode(errorsc.CommonInvalidParamCode, ""))
		return
	}
	if deleteRequest.UserId <= 0 {
		c.JSON(400, errorsc.NewByCode(errorsc.CommonInvalidParamCode, ""))
		return
	}
	// 例如从数据库中删除用户记录
	var userDB models.UserDB
	if err := h.DB.Where("id = ?", deleteRequest.UserId).First(&userDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, errorsc.NewByCode(errorsc.UserNotFoundCode, ""))
			return
		}
		c.JSON(500, errorsc.NewByCode(errorsc.CommonDbErrorCode, ""))
		return
	}
	if err := h.DB.Delete(&userDB).Error; err != nil {
		c.JSON(500, errorsc.NewByCode(errorsc.CommonDbErrorCode, ""))
		return
	}
	c.JSON(200, gin.H{"message": "User deleted successfully"}) // 返回成功消息
}

func (h *UserHandler) LoginByPassword(c *gin.Context) {
	var user models.UserLoginByPasswordRequest
	if err := c.ShouldBindBodyWithJSON(&user); err != nil {
		c.JSON(400, errorsc.NewByCode(errorsc.CommonInvalidParamCode, ""))
	}
	if *user.Email == "" || *user.Password == "" {
		c.JSON(400, errorsc.NewByCode(errorsc.CommonInvalidParamCode, ""))
		return
	}
	// validate email format
	if !utils.IsValidEmail(*user.Email) {
		c.JSON(400, errorsc.NewByCode(errorsc.CommonInvalidParamCode, ""))
		return
	}
	userDB := models.UserDB{}
	if err := h.DB.Where("email = ?", user.Email).First(&userDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(400, errorsc.NewByCode(errorsc.UserNotFoundCode, ""))
			return
		}
		c.JSON(500, errorsc.NewByCode(errorsc.CommonDbErrorCode, ""))
		return
	}
	err := utils.ComparePassword(*userDB.Password, *user.Password)
	if err != nil {
		c.JSON(500, errorsc.NewByCode(errorsc.PasswordMismatchCode, ""))
		return
	}
	// login successful, generate token
	tokenString, err := utils.GenerateJWT(userDB.ID)
	if err != nil {
		c.JSON(500, errorsc.NewByCode(errorsc.CommonInternalErrorCode, ""))
		return
	}
	c.JSON(200, models.UserLoginByPasswordResponse{
		UserInfo: &models.UserInfo{
			ID:        userDB.ID,
			Email:     *userDB.Email,
			AvatarUrl: *userDB.AvatarUrl,
			NickName:  *userDB.NickName,
		},
		Token:      &tokenString,                                // 这里可以生成 JWT 或其他类型的令牌
		ExpireTime: time.Now().Add(utils.ExpireDuration).Unix(), // 这里可以设置令牌的过期时间
	})
}

func (h *UserHandler) GetUserFromDB(id int64, nickname string, mobile string, email string) ([]*models.UserDB, error) {
	var users []*models.UserDB
	query := h.DB.Model(&models.UserDB{})

	if id != 0 {
		query = query.Where("id = ?", id)
	}
	if nickname != "" {
		query = query.Where("nick_name = ?", nickname)
	}
	if mobile != "" {
		query = query.Where("mobile = ?", mobile)
	}
	if email != "" {
		query = query.Where("email = ?", email)
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, errors.New("failed to query users: " + err.Error())
	}

	return users, nil
}
