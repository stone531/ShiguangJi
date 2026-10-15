package handler

import (
	"github.com/gin-gonic/gin"

	"shiguangji/internal/service"
)

// LikeHandler 点赞接口（公开）。
type LikeHandler struct{ svc *service.LikeService }

// NewLikeHandler 构造函数。
func NewLikeHandler(svc *service.LikeService) *LikeHandler { return &LikeHandler{svc: svc} }

// Status GET /api/posts/:slug/like → {liked, like_count}。
func (h *LikeHandler) Status(c *gin.Context) {
	liked, count, err := h.svc.Status(c.Request.Context(), c.Param("slug"), c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"liked": liked, "like_count": count})
}

// Toggle POST /api/posts/:slug/like → 切换后 {liked, like_count}。
func (h *LikeHandler) Toggle(c *gin.Context) {
	liked, count, err := h.svc.Toggle(c.Request.Context(), c.Param("slug"), c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"liked": liked, "like_count": count})
}
