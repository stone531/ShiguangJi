package model

// Admin 管理员（单账号预置）。
type Admin struct {
	BaseModel
	Username     string `gorm:"size:50;uniqueIndex;not null" json:"username"`
	PasswordHash string `gorm:"size:100;not null" json:"-"`
	Nickname     string `gorm:"size:50;not null" json:"nickname"`
	Avatar       string `gorm:"size:255" json:"avatar"`
	// MustChangePassword 首次登录强制改密（M1 初始化账号为 true）
	MustChangePassword bool `gorm:"not null;default:false" json:"must_change_password"`
}
