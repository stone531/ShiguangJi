// Package service 业务逻辑层。
package service

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"shiguangji/internal/errs"
	"shiguangji/internal/model"
)

var slugRe = regexp.MustCompile(`^[a-z0-9-]+$`)

var videoDomainWhitelist = map[string]bool{
	"player.bilibili.com":      true,
	"www.youtube.com":          true,
	"www.youtube-nocookie.com": true,
	"player.youku.com":         true,
}

// likeEscaper 转义 LIKE 通配符，防止 keyword 中的 %/_ 改变匹配语义。
var likeEscaper = strings.NewReplacer("%", "\\%", "_", "\\_")

// PostService 文章业务逻辑。
type PostService struct{ db *gorm.DB }

// NewPostService 构造函数。
func NewPostService(db *gorm.DB) *PostService { return &PostService{db: db} }

// validate 校验不变量：slug 格式、视频域名白名单、type 约束。
func (s *PostService) validate(p *model.Post) error {
	if !slugRe.MatchString(p.Slug) {
		return errs.New(40001, "slug 只允许小写字母、数字和连字符")
	}
	if p.Type == model.TypeVideo {
		u, err := url.Parse(p.VideoURL)
		if err != nil || !videoDomainWhitelist[u.Host] {
			return errs.New(40001, "视频地址域名不在白名单")
		}
	}
	return nil
}

// deriveSummary 摘要为空时截取正文前 120 rune（剔除 Markdown 标记的粗略实现）。
func deriveSummary(content string) string {
	cleaned := stripMarkdown(content)
	runes := []rune(cleaned)
	if len(runes) > 120 {
		return string(runes[:120])
	}
	return cleaned
}

var mdSymbolRe = regexp.MustCompile("[#*>`\\[\\]()!]")

func stripMarkdown(s string) string {
	// 粗略剔除常见语法符号；列表页摘要够用
	return mdSymbolRe.ReplaceAllString(s, "")
}

// translateDBError 将底层数据库唯一冲突翻译为业务错误。
func translateDBError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "UNIQUE constraint failed") || strings.Contains(msg, "Error 1062") {
		return errs.New(40001, "slug 已存在")
	}
	return err
}

// Create 创建文章；事务内查重 slug 并写入标签关联，摘要缺省自动截取。
func (s *PostService) Create(ctx context.Context, p *model.Post, tagIDs []uint) (*model.Post, error) {
	if err := s.validate(p); err != nil {
		return nil, err
	}
	if p.Summary == "" {
		p.Summary = deriveSummary(p.Content)
	}
	if p.Status == model.StatusPublished && p.PublishedAt == nil {
		now := time.Now()
		p.PublishedAt = &now
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("slug = ?", p.Slug).First(&model.Post{}).Error; err == nil {
			return errs.New(40001, "slug 已存在")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Create(p).Error; err != nil {
			return err
		}
		for _, tid := range tagIDs {
			if err := tx.Create(&model.PostTag{PostID: p.ID, TagID: tid}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, translateDBError(err)
	}
	return p, nil
}

// Update 更新文章；slug 查重排除自身，事务内全量替换标签关联。
func (s *PostService) Update(ctx context.Context, id uint, req *model.Post, tagIDs []uint) error {
	if err := s.validate(req); err != nil {
		return err
	}
	return translateDBError(s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p model.Post
		if err := tx.First(&p, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		var cnt int64
		if err := tx.Model(&model.Post{}).Where("slug = ? AND id <> ?", req.Slug, id).Count(&cnt).Error; err != nil {
			return err
		}
		if cnt > 0 {
			return errs.New(40001, "slug 已存在")
		}
		summary := req.Summary
		if summary == "" {
			summary = deriveSummary(req.Content)
		}
		updates := map[string]interface{}{
			"title": req.Title, "slug": req.Slug, "summary": summary, "cover": req.Cover,
			"content": req.Content, "type": req.Type, "video_url": req.VideoURL,
			"category_id": req.CategoryID, "is_top": req.IsTop, "is_featured": req.IsFeatured,
		}
		if err := tx.Model(&p).Updates(updates).Error; err != nil {
			return err
		}
		// 全量替换标签关联
		if err := tx.Exec("DELETE FROM post_tags WHERE post_id = ?", id).Error; err != nil {
			return err
		}
		for _, tid := range tagIDs {
			if err := tx.Create(&model.PostTag{PostID: id, TagID: tid}).Error; err != nil {
				return err
			}
		}
		return nil
	}))
}

// GetByID 后台详情。
func (s *PostService) GetByID(ctx context.Context, id uint) (*model.Post, error) {
	var p model.Post
	err := s.db.WithContext(ctx).Preload("Category").Preload("Tags").First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.ErrNotFound
	}
	return &p, err
}

// GetBySlugPublic 前台：仅已发布。views 原子 +1。
func (s *PostService) GetBySlugPublic(ctx context.Context, slug string) (*model.Post, error) {
	var p model.Post
	err := s.db.WithContext(ctx).Preload("Category").Preload("Tags").
		Where("slug = ? AND status = ?", slug, model.StatusPublished).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", p.ID).
		UpdateColumn("views", gorm.Expr("views + 1")).Error; err != nil {
		slog.Warn("update views failed", "post_id", p.ID, "err", err)
	}
	return &p, nil
}

// SetStatus 状态切换；首次发布写入 published_at。
func (s *PostService) SetStatus(ctx context.Context, id uint, status int8) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p model.Post
		if err := tx.First(&p, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		updates := map[string]interface{}{"status": status}
		if status == model.StatusPublished && p.PublishedAt == nil {
			updates["published_at"] = time.Now()
		}
		return tx.Model(&p).Updates(updates).Error
	})
}

// ListPublic 前台列表：仅已发布，置顶优先，分页，可按分类筛选。
func (s *PostService) ListPublic(ctx context.Context, page, size int, categoryID *uint) ([]model.Post, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 10
	}
	q := s.db.WithContext(ctx).Model(&model.Post{}).
		Where("status = ?", model.StatusPublished)
	if categoryID != nil {
		q = q.Where("category_id = ?", *categoryID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Post
	err := q.Preload("Category").Preload("Tags").
		Order("is_top DESC, published_at DESC").
		Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

// ListAdmin 后台列表：全部状态 + 关键词筛选（LIKE 通配符已转义）。
func (s *PostService) ListAdmin(ctx context.Context, page, size int, status *int8, keyword string) ([]model.Post, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 10
	}
	q := s.db.WithContext(ctx).Model(&model.Post{})
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	if keyword != "" {
		q = q.Where("title LIKE ? ESCAPE '\\'", "%"+likeEscaper.Replace(keyword)+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Post
	err := q.Order("updated_at DESC").
		Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

// Delete 真删除（后台"彻底删除"）；软删除语义由 status=2 承担。事务内清理 post_tags。
func (s *PostService) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Delete(&model.Post{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errs.ErrNotFound
		}
		return tx.Exec("DELETE FROM post_tags WHERE post_id = ?", id).Error
	})
}
