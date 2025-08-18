package models

type UserDB struct {
	ID        int64   `gorm:"primarykey;autoIncrement;column:id" json:"id"`
	Password  *string `gorm:"column:password;type:varchar(255);not null;comment:密码" json:"password"`
	Email     *string `gorm:"column:email;type:varchar(255);uniqueIndex;not null;comment:邮箱" json:"email"`
	AvatarUrl *string `gorm:"avatar_url;null" json:"avatar_url"`
	NickName  *string `gorm:"nickname;null" json:"nickname"`
	Mobile    *string `gorm:"mobile;null" json:"mobile"`
	IsDeleted bool    `gorm:"column:is_deleted;default:false;not null;comment:是否删除" json:"is_deleted"`
}

type UserRegisterRequest struct {
	Email    *string `json:"email"`
	Password *string `json:"password"`
}
type UserRegisterResponse struct {
	UserInfo   *UserDB `json:"user_info"`
	Token      *string `json:"token"`
	ExpireTime int64   `json:"expire_time"`
}

type UserLoginByPasswordRequest struct {
	Email    *string `json:"email"`
	Password *string `json:"password"`
}
type UserLoginByPasswordResponse struct {
	UserInfo   *UserDB `json:"user_info"`
	Token      *string `json:"token"`
	ExpireTime int64   `json:"expire_time"`
}
