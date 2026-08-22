// Package service 业务逻辑层。
package service

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"shiguangji/internal/errs"
	"shiguangji/internal/model"
	"shiguangji/internal/pkg/jwtutil"
)

// AuthService 登录与改密业务。
type AuthService struct {
	db     *gorm.DB
	secret string
	jwtTTL time.Duration
}

// NewAuthService 构造认证服务。
func NewAuthService(db *gorm.DB, secret string, ttl time.Duration) *AuthService {
	return &AuthService{db: db, secret: secret, jwtTTL: ttl}
}

// Login 校验用户名密码并签发 JWT；错误信息统一，防用户名枚举。
func (s *AuthService) Login(ctx context.Context, username, password string) (string, *model.Admin, error) {
	var admin model.Admin
	err := s.db.WithContext(ctx).Where("username = ?", username).First(&admin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil, errs.New(40003, "用户名或密码错误")
	}
	if err != nil {
		return "", nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)) != nil {
		return "", nil, errs.New(40003, "用户名或密码错误")
	}
	tok, err := jwtutil.Issue(s.secret, admin.ID, s.jwtTTL)
	if err != nil {
		return "", nil, err
	}
	return tok.Token, &admin, nil
}

// ChangePassword 校验旧密码后更新；成功后清除强制改密标记。
func (s *AuthService) ChangePassword(ctx context.Context, adminID uint, oldPass, newPass string) error {
	var admin model.Admin
	if err := s.db.WithContext(ctx).First(&admin, adminID).Error; err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(oldPass)) != nil {
		return errs.New(40003, "旧密码错误")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&admin).
		Updates(map[string]interface{}{"password_hash": string(hash), "must_change_password": false}).Error
}

// AdminProfile 博主资料（不含密码哈希等敏感列）。
type AdminProfile struct {
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// GetProfile 读取博主资料（GET /api/admin/profile）。
func (s *AuthService) GetProfile(ctx context.Context, adminID uint) (*AdminProfile, error) {
	var admin model.Admin
	if err := s.db.WithContext(ctx).First(&admin, adminID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	return &AdminProfile{Username: admin.Username, Nickname: admin.Nickname, Avatar: admin.Avatar}, nil
}

// UpdateProfile 更新昵称/头像（PUT /api/admin/profile）。
func (s *AuthService) UpdateProfile(ctx context.Context, adminID uint, nickname, avatar string) error {
	if l := len([]rune(nickname)); l < 1 || l > 20 {
		return errs.New(40001, "昵称长度需在 1-20 字符")
	}
	return s.db.WithContext(ctx).Model(&model.Admin{}).Where("id = ?", adminID).
		Updates(map[string]interface{}{"nickname": nickname, "avatar": avatar}).Error
}
