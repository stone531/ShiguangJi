package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"shiguangji/internal/errs"
	"shiguangji/internal/service"
)

// TaxonomyHandler 分类/标签公开接口。
type TaxonomyHandler struct{ svc *service.TaxonomyService }

// NewTaxonomyHandler 构造函数。
func NewTaxonomyHandler(svc *service.TaxonomyService) *TaxonomyHandler {
	return &TaxonomyHandler{svc: svc}
}

// ListCategories GET /api/categories。
func (h *TaxonomyHandler) ListCategories(c *gin.Context) {
	list, err := h.svc.ListCategories(c.Request.Context())
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list})
}

// ListTags GET /api/tags。
func (h *TaxonomyHandler) ListTags(c *gin.Context) {
	list, err := h.svc.ListTags(c.Request.Context())
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list})
}

// ---- 后台管理（M4）----

type categorySaveReq struct {
	ID   uint   `json:"id"`
	Name string `json:"name" binding:"required,max=50"`
	Sort int    `json:"sort"`
}

type tagSaveReq struct {
	ID   uint   `json:"id"`
	Name string `json:"name" binding:"required,max=50"`
}

func pathID(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return 0, errs.ErrParam
	}
	return uint(id), nil
}

// ListCategoriesAdmin GET /api/admin/categories：编辑器选项用全量列表。
func (h *TaxonomyHandler) ListCategoriesAdmin(c *gin.Context) {
	list, err := h.svc.ListCategoriesAdmin(c.Request.Context())
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list})
}

// ListTagsAdmin GET /api/admin/tags：编辑器选项用全量列表。
func (h *TaxonomyHandler) ListTagsAdmin(c *gin.Context) {
	list, err := h.svc.ListTagsAdmin(c.Request.Context())
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list})
}

// SaveCategory POST /api/admin/categories（id=0 新建）。
func (h *TaxonomyHandler) SaveCategory(c *gin.Context) {
	var req categorySaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	cat, err := h.svc.SaveCategory(c.Request.Context(), req.ID, req.Name, req.Sort)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, cat)
}

// DeleteCategory DELETE /api/admin/categories/:id。
func (h *TaxonomyHandler) DeleteCategory(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		Fail(c, err)
		return
	}
	if err := h.svc.DeleteCategory(c.Request.Context(), id); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

// SaveTag POST /api/admin/tags（id=0 新建）。
func (h *TaxonomyHandler) SaveTag(c *gin.Context) {
	var req tagSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	tag, err := h.svc.SaveTag(c.Request.Context(), req.ID, req.Name)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, tag)
}

// DeleteTag DELETE /api/admin/tags/:id。
func (h *TaxonomyHandler) DeleteTag(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		Fail(c, err)
		return
	}
	if err := h.svc.DeleteTag(c.Request.Context(), id); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

type mergeTagReq struct {
	TargetID uint `json:"target_id" binding:"required"`
}

// MergeTag POST /api/admin/tags/:id/merge：把 :id 合并到 target_id。
func (h *TaxonomyHandler) MergeTag(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		Fail(c, err)
		return
	}
	var req mergeTagReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	if err := h.svc.MergeTag(c.Request.Context(), id, req.TargetID); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}
