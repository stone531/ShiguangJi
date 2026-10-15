package handler

import (
	"strconv"
	"time"

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
	Size    int    `form:"size" binding:"omitempty,min=1,max=50"` // 缺省走 handler 默认值
	Status  *int8  `form:"status"`
	Keyword string `form:"keyword" binding:"max=100"`
}

type publicListQuery struct {
	Page     int    `form:"page" binding:"min=0"`
	Size     int    `form:"size" binding:"omitempty,min=1,max=50"` // 缺省走 handler 默认值
	Type     *int8  `form:"type" binding:"omitempty,oneof=1 2 3"`
	Category string `form:"category" binding:"max=50"`
	Tag      string `form:"tag" binding:"max=50"`
	Keyword  string `form:"keyword" binding:"max=100"`
	Featured *bool  `form:"featured"`
}

// categoryBrief / tagBrief 列表卡片携带的维表摘要。
type categoryBrief struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type tagBrief struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// postCard 前台列表卡片 VO：不含正文（spec §5.1 列表瘦身）。
type postCard struct {
	ID           uint           `json:"id"`
	Title        string         `json:"title"`
	Slug         string         `json:"slug"`
	Summary      string         `json:"summary"`
	Cover        string         `json:"cover"`
	Type         int8           `json:"type"`
	IsTop        bool           `json:"is_top"`
	IsFeatured   bool           `json:"is_featured"`
	Views        int64          `json:"views"`
	LikeCount    int64          `json:"like_count"`
	CommentCount int64          `json:"comment_count"`
	PublishedAt  *time.Time     `json:"published_at"`
	Category     *categoryBrief `json:"category"`
	Tags         []tagBrief     `json:"tags"`
}

// toCards model 列表映射为轻量卡片（tags 空时给空切片，避免 JSON null）。
func toCards(list []model.Post) []postCard {
	cards := make([]postCard, 0, len(list))
	for i := range list {
		p := &list[i]
		c := postCard{
			ID: p.ID, Title: p.Title, Slug: p.Slug, Summary: p.Summary, Cover: p.Cover,
			Type: p.Type, IsTop: p.IsTop, IsFeatured: p.IsFeatured,
			Views: p.Views, LikeCount: p.LikeCount, CommentCount: p.CommentCount,
			PublishedAt: p.PublishedAt, Tags: []tagBrief{},
		}
		if p.Category != nil {
			c.Category = &categoryBrief{ID: p.Category.ID, Name: p.Category.Name}
		}
		for _, t := range p.Tags {
			c.Tags = append(c.Tags, tagBrief{ID: t.ID, Name: t.Name})
		}
		cards = append(cards, c)
	}
	return cards
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

// ListPublic 前台公开列表：仅已发布，支持 type/category/tag/keyword/featured 筛选。
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
	list, total, err := h.svc.ListPublic(c.Request.Context(), service.PublicListFilter{
		Page: q.Page, Size: q.Size, Type: q.Type,
		Category: q.Category, Tag: q.Tag, Keyword: q.Keyword, Featured: q.Featured,
	})
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": toCards(list), "total": total, "page": q.Page, "size": q.Size})
}

// GetBySlugPublic 前台公开详情：slug 定位，仅已发布，附上/下篇导航。
func (h *PostHandler) GetBySlugPublic(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		Fail(c, errs.ErrParam)
		return
	}
	p, prev, next, err := h.svc.GetBySlugPublic(c.Request.Context(), slug)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"post": p, "prev": prev, "next": next})
}

// Archive 前台归档数据（GET /api/archive）。
func (h *PostHandler) Archive(c *gin.Context) {
	groups, err := h.svc.Archive(c.Request.Context())
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": groups})
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
