package models

import (
	"awesomeEval/internal/utils"
)

type UserDB struct {
	utils.BaseModel
	Password  *string `gorm:"column:password;" json:"password"`
	Email     *string `gorm:"column:email;" json:"email"`
	AvatarUrl *string `gorm:"column:avatar_url" json:"avatar_url"`
	NickName  *string `gorm:"column:nickname" json:"nickname"`
	Mobile    *string `gorm:"column:mobile" json:"mobile"`
	IsDeleted bool    `gorm:"column:is_deleted;" json:"is_deleted"`
}

type UserRegisterRequest struct {
	Email    *string `json:"email"`
	Password *string `json:"password"`
}
type UserRegisterResponse struct {
	UserInfo   *UserInfo `json:"user_info"`
	Token      *string   `json:"token"`
	ExpireTime int64     `json:"expire_time"`
}

type UserLoginByPasswordRequest struct {
	Email    *string `json:"email"`
	Password *string `json:"password"`
}
type UserLoginByPasswordResponse struct {
	UserInfo   *UserInfo `json:"user_info"`
	Token      *string   `json:"token"`
	ExpireTime int64     `json:"expire_time"`
}

type UserInfo struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	AvatarUrl string `json:"avatar_url"`
	NickName  string `json:"nickname"`
	Mobile    string `json:"mobile"`
}

type UpdateUserInfoRequest struct {
	UserId    int64   `json:"user_id"`
	NickName  *string `json:"nickname"`
	AvatarUrl *string `json:"avatar_url"`
	Mobile    *string `json:"mobile"`
}

type DeleteUserRequest struct {
	UserId int64 `json:"user_id"`
}

type LoginByMobileRequest struct {
	Mobile           *string `json:"mobile"`
	VerificationCode *string `json:"verification_code"`
}

type LoginByMobileResponse struct {
	UserInfo   *UserInfo `json:"user_info"`
	Token      *string   `json:"token"`
	ExpireTime int64     `json:"expire_time"`
}

type SendVerificationCodeRequest struct {
	Mobile string `json:"mobile"`
}

func (*UserDB) TableName() string {
	return "user"
}
