package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"shiguangji/internal/config"
	"shiguangji/internal/errs"
	"shiguangji/internal/middleware"
	"shiguangji/internal/service"
)

// AuthHandler 认证相关 HTTP 入口。
type AuthHandler struct {
	svc *service.AuthService
	cfg *config.Config
}

// NewAuthHandler 构造认证 handler。
func NewAuthHandler(svc *service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{svc: svc, cfg: cfg}
}

type loginReq struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// Login 登录：写 HttpOnly cookie，返回基本信息及是否需强制改密。
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	token, admin, err := h.svc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		Fail(c, err)
		return
	}
	sameSite := http.SameSiteLaxMode
	if h.cfg.IsProd { // 生产收紧为 Strict，跨站请求不带 cookie（spec §8）
		sameSite = http.SameSiteStrictMode
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     middleware.CookieName,
		Value:    token,
		Path:     "/api",
		Expires:  time.Now().Add(h.cfg.JWTTimeout),
		HttpOnly: true,
		Secure:   h.cfg.IsProd,
		SameSite: sameSite,
	})
	OK(c, gin.H{
		"nickname":             admin.Nickname,
		"avatar":               admin.Avatar,
		"must_change_password": admin.MustChangePassword,
	})
}

type changePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required,min=8,max=72"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

// ChangePassword 修改密码（需 JWT 中间件先行）。
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	adminID := c.GetUint("adminID")
	if err := h.svc.ChangePassword(c.Request.Context(), adminID, req.OldPassword, req.NewPassword); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

type profileReq struct {
	Nickname string `json:"nickname" binding:"required,max=20"`
	Avatar   string `json:"avatar" binding:"max=255"`
}

// GetProfile 读取博主资料（GET /api/admin/profile）。
func (h *AuthHandler) GetProfile(c *gin.Context) {
	vo, err := h.svc.GetProfile(c.Request.Context(), c.GetUint("adminID"))
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, vo)
}

// UpdateProfile 更新昵称/头像（PUT /api/admin/profile）。
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req profileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	if err := h.svc.UpdateProfile(c.Request.Context(), c.GetUint("adminID"), req.Nickname, req.Avatar); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}
