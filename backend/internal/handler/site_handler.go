package handler

import (
	"github.com/gin-gonic/gin"

	"shiguangji/internal/errs"
	"shiguangji/internal/service"
)

// SiteHandler 站点信息接口。
type SiteHandler struct{ svc *service.SiteService }

// NewSiteHandler 构造函数。
func NewSiteHandler(svc *service.SiteService) *SiteHandler { return &SiteHandler{svc: svc} }

// GetPublic 前台站点信息（GET /api/site）。
func (h *SiteHandler) GetPublic(c *gin.Context) {
	vo, err := h.svc.GetPublic(c.Request.Context())
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, vo)
}

type siteSaveReq struct {
	SiteName       string               `json:"site_name" binding:"required,max=100"`
	SiteDesc       string               `json:"site_desc" binding:"max=200"`
	Avatar         string               `json:"avatar" binding:"max=255"`
	WechatQR       string               `json:"wechat_qr" binding:"max=255"`
	AlipayQR       string               `json:"alipay_qr" binding:"max=255"`
	DonateText     string               `json:"donate_text" binding:"max=200"`
	SocialLinks    []service.SocialLink `json:"social_links"`
	PageSize       int                  `json:"page_size" binding:"min=1,max=50"`
	CommentEnabled bool                 `json:"comment_enabled"`
	SensitiveWords string               `json:"sensitive_words" binding:"max=5000"`
}

// AdminGet 后台站点设置（GET /api/admin/site），含敏感词表。
func (h *SiteHandler) AdminGet(c *gin.Context) {
	vo, err := h.svc.AdminGet(c.Request.Context())
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, vo)
}

// AdminSave 保存站点设置（PUT /api/admin/site）。
func (h *SiteHandler) AdminSave(c *gin.Context) {
	var req siteSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	err := h.svc.Save(c.Request.Context(), &service.SiteSaveInput{
		SiteName: req.SiteName, SiteDesc: req.SiteDesc,
		Avatar: req.Avatar, WechatQR: req.WechatQR, AlipayQR: req.AlipayQR,
		DonateText: req.DonateText, SocialLinks: req.SocialLinks,
		PageSize: req.PageSize, CommentEnabled: req.CommentEnabled,
		SensitiveWords: req.SensitiveWords,
	})
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// StatsHandler 后台统计（GET /api/admin/stats）。
type StatsHandler struct{ svc *service.StatsService }

// NewStatsHandler 构造函数。
func NewStatsHandler(svc *service.StatsService) *StatsHandler { return &StatsHandler{svc: svc} }

// Get 六指标。
func (h *StatsHandler) Get(c *gin.Context) {
	vo, err := h.svc.Get(c.Request.Context())
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, vo)
}
