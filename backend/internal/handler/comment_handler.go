package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"shiguangji/internal/errs"
	"shiguangji/internal/service"
)

// CommentHandler 评论接口（公开 + 后台管理）。
type CommentHandler struct{ svc *service.CommentService }

// NewCommentHandler 构造函数。
func NewCommentHandler(svc *service.CommentService) *CommentHandler { return &CommentHandler{svc: svc} }

// commentCreateReq website 为蜜罐字段：正常用户永远为空。
type commentCreateReq struct {
	Nickname string `json:"nickname" binding:"required,max=50"`
	Email    string `json:"email" binding:"max=100"`
	Content  string `json:"content" binding:"required,max=2000"`
	Website  string `json:"website" form:"website"`
}

type commentListQuery struct {
	Page int `form:"page" binding:"min=0"`
	Size int `form:"size" binding:"omitempty,min=1,max=50"` // 缺省走 handler 默认值
}

type adminCommentListQuery struct {
	Page   int   `form:"page" binding:"min=0"`
	Size   int   `form:"size" binding:"omitempty,min=1,max=50"` // 缺省走 handler 默认值
	Status *int8 `form:"status" binding:"omitempty,oneof=0 1 2"`
}

type reviewReq struct {
	Status int8 `json:"status" binding:"oneof=1 2"`
}

type replyReq struct {
	Content string `json:"content" binding:"required,max=2000"`
}

// Create 访客发表评论（POST /api/posts/:slug/comments）：蜜罐非空 → 假成功，不落库。
func (h *CommentHandler) Create(c *gin.Context) {
	var req commentCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	if req.Website != "" {
		OK(c, nil) // 蜜罐命中：对机器人伪装成功，静默丢弃
		return
	}
	cm, err := h.svc.Create(c.Request.Context(), c.Param("slug"), req.Nickname, req.Email,
		req.Content, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"id": cm.ID, "message": "评论已提交，审核通过后展示"})
}

// ListPublic 前台评论树（GET /api/posts/:slug/comments），仅已通过。
func (h *CommentHandler) ListPublic(c *gin.Context) {
	var q commentListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Size == 0 {
		q.Size = 20
	}
	list, total, err := h.svc.ListPublic(c.Request.Context(), c.Param("slug"), q.Page, q.Size)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list, "total": total, "page": q.Page, "size": q.Size})
}

// ListAdmin 后台评论列表（GET /api/admin/comments），可按状态筛选。
func (h *CommentHandler) ListAdmin(c *gin.Context) {
	var q adminCommentListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Size == 0 {
		q.Size = 10
	}
	list, total, err := h.svc.ListAdmin(c.Request.Context(), q.Page, q.Size, q.Status)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list, "total": total, "page": q.Page, "size": q.Size})
}

// Review 审核（PUT /api/admin/comments/:id/review）。
func (h *CommentHandler) Review(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	var req reviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	if err := h.svc.Review(c.Request.Context(), uint(id), req.Status); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// Reply 博主回复（POST /api/admin/comments/:id/reply），is_admin 自动通过。
func (h *CommentHandler) Reply(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	var req replyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	adminID := c.GetUint("adminID")
	cm, err := h.svc.Reply(c.Request.Context(), uint(id), adminID, req.Content)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, cm)
}

// Delete 软删评论（DELETE /api/admin/comments/:id）。
func (h *CommentHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), uint(id)); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}
