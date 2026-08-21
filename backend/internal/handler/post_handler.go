package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"shiguangji/internal/errs"
	"shiguangji/internal/model"
	"shiguangji/internal/service"
)

// PostHandler 文章接口。
type PostHandler struct{ svc *service.PostService }

// NewPostHandler 构造函数。
func NewPostHandler(svc *service.PostService) *PostHandler { return &PostHandler{svc: svc} }

type postReq struct {
	Title      string `json:"title" binding:"required,max=200"`
	Slug       string `json:"slug" binding:"required,max=200"`
	Summary    string `json:"summary" binding:"max=500"`
	Cover      string `json:"cover" binding:"max=255"`
	Content    string `json:"content" binding:"required"`
	Type       int8   `json:"type" binding:"oneof=1 2 3"`
	VideoURL   string `json:"video_url" binding:"max=500"`
	CategoryID *uint  `json:"category_id"`
	Status     int8   `json:"status" binding:"oneof=0 1 2"`
	IsTop      bool   `json:"is_top"`
	IsFeatured bool   `json:"is_featured"`
	TagIDs     []uint `json:"tag_ids"`
}

type listQuery struct {
	Page    int    `form:"page" binding:"min=0"`
	Size    int    `form:"size" binding:"min=1,max=50"`
	Status  *int8  `form:"status"`
	Keyword string `form:"keyword" binding:"max=100"`
}

type publicListQuery struct {
	Page       int   `form:"page" binding:"min=0"`
	Size       int   `form:"size" binding:"min=1,max=50"`
	CategoryID *uint `form:"category_id"`
}

func (h *PostHandler) Create(c *gin.Context) {
	var req postReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	p := postFromReq(&req)
	created, err := h.svc.Create(c.Request.Context(), p, req.TagIDs)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, created)
}

// Update 更新文章（PUT /api/admin/posts/:id）。
func (h *PostHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	var req postReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	if err := h.svc.Update(c.Request.Context(), uint(id), postFromReq(&req), req.TagIDs); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

func postFromReq(req *postReq) *model.Post {
	return &model.Post{
		Title: req.Title, Slug: req.Slug, Summary: req.Summary, Cover: req.Cover,
		Content: req.Content, Type: req.Type, VideoURL: req.VideoURL,
		CategoryID: req.CategoryID, Status: req.Status,
		IsTop: req.IsTop, IsFeatured: req.IsFeatured,
	}
}

func (h *PostHandler) ListAdmin(c *gin.Context) {
	var q listQuery
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
	list, total, err := h.svc.ListAdmin(c.Request.Context(), q.Page, q.Size, q.Status, q.Keyword)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list, "total": total, "page": q.Page, "size": q.Size})
}

// ListPublic 前台公开列表：仅已发布文章。
func (h *PostHandler) ListPublic(c *gin.Context) {
	var q publicListQuery
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
	list, total, err := h.svc.ListPublic(c.Request.Context(), q.Page, q.Size, q.CategoryID)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list, "total": total, "page": q.Page, "size": q.Size})
}

// GetBySlugPublic 前台公开详情：slug 定位，仅已发布。
func (h *PostHandler) GetBySlugPublic(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		Fail(c, errs.ErrParam)
		return
	}
	p, err := h.svc.GetBySlugPublic(c.Request.Context(), slug)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, p)
}

func (h *PostHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	p, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, p)
}

type statusReq struct {
	Status int8 `json:"status" binding:"oneof=0 1 2"`
}

func (h *PostHandler) SetStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	var req statusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	if err := h.svc.SetStatus(c.Request.Context(), uint(id), req.Status); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

func (h *PostHandler) Delete(c *gin.Context) {
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
