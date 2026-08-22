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

// PostBrief 相邻文章摘要（详情页上/下篇导航，spec §6.2）。
type PostBrief struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

// GetBySlugPublic 前台：仅已发布。views 原子 +1，并返回时间线上/下篇。
func (s *PostService) GetBySlugPublic(ctx context.Context, slug string) (*model.Post, *PostBrief, *PostBrief, error) {
	var p model.Post
	err := s.db.WithContext(ctx).Preload("Category").Preload("Tags").
		Where("slug = ? AND status = ?", slug, model.StatusPublished).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, nil, nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", p.ID).
		UpdateColumn("views", gorm.Expr("views + 1")).Error; err != nil {
		slog.Warn("update views failed", "post_id", p.ID, "err", err)
	} else {
		p.Views++ // 返回值与库内保持一致（本次浏览计入）
	}
	prev, next := s.neighbors(ctx, p.ID, p.PublishedAt)
	return &p, prev, next, nil
}

// neighbors prev=发布更早的相邻一篇，next=更新的相邻一篇（时间线语义）。
func (s *PostService) neighbors(ctx context.Context, id uint, at *time.Time) (*PostBrief, *PostBrief) {
	if at == nil {
		return nil, nil
	}
	prev := s.neighborBrief(ctx, "published_at < ?", at, id, "published_at DESC")
	next := s.neighborBrief(ctx, "published_at > ?", at, id, "published_at ASC")
	return prev, next
}

func (s *PostService) neighborBrief(ctx context.Context, cond string, at *time.Time, id uint, order string) *PostBrief {
	var b PostBrief
	err := s.db.WithContext(ctx).Model(&model.Post{}).
		Where("status = ? AND "+cond+" AND id <> ?", model.StatusPublished, at, id).
		Order(order).Select("slug, title").First(&b).Error
	if err != nil {
		return nil
	}
	return &b
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

// PublicListFilter 前台列表筛选（spec §5.1：type/category/tag/keyword + 推荐位）。
type PublicListFilter struct {
	Page     int
	Size     int
	Type     *int8
	Category string // 分类名
	Tag      string // 标签名
	Keyword  string // 标题/摘要/标签名模糊匹配
	Featured *bool
}

// ListPublic 前台列表：仅已发布，置顶优先，分页，支持多条件筛选。
func (s *PostService) ListPublic(ctx context.Context, f PublicListFilter) ([]model.Post, int64, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Size < 1 || f.Size > 50 {
		f.Size = 10
	}
	q := s.db.WithContext(ctx).Model(&model.Post{}).
		Where("posts.status = ?", model.StatusPublished)
	if f.Type != nil {
		q = q.Where("posts.type = ?", *f.Type)
	}
	if f.Category != "" {
		q = q.Joins("JOIN categories ON categories.id = posts.category_id AND categories.name = ?", f.Category)
	}
	if f.Tag != "" {
		q = q.Joins("JOIN post_tags ON post_tags.post_id = posts.id").
			Joins("JOIN tags ON tags.id = post_tags.tag_id AND tags.name = ?", f.Tag)
	}
	if f.Featured != nil && *f.Featured {
		q = q.Where("posts.is_featured = ?", true)
	}
	if f.Keyword != "" {
		kw := "%" + likeEscaper.Replace(f.Keyword) + "%"
		q = q.Where(`posts.title LIKE ? ESCAPE '\' OR posts.summary LIKE ? ESCAPE '\'
			OR posts.id IN (SELECT pt.post_id FROM post_tags pt JOIN tags t ON t.id = pt.tag_id WHERE t.name LIKE ? ESCAPE '\')`, kw, kw, kw)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Post
	err := q.Preload("Category").Preload("Tags").
		Order("posts.is_top DESC, posts.published_at DESC").
		Offset((f.Page - 1) * f.Size).Limit(f.Size).Find(&list).Error
	return list, total, err
}

// ArchivePost 归档条目。
type ArchivePost struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	PublishedAt time.Time `json:"published_at"`
}

// ArchiveGroup 年月分组（spec §5.1 /archive）。
type ArchiveGroup struct {
	Year  int           `json:"year"`
	Month int           `json:"month"`
	Posts []ArchivePost `json:"posts"`
}

// Archive 已发布文章按年月分组倒序（时间线归档页数据源）。
func (s *PostService) Archive(ctx context.Context) ([]ArchiveGroup, error) {
	var rows []ArchivePost
	err := s.db.WithContext(ctx).Model(&model.Post{}).
		Where("status = ? AND published_at IS NOT NULL", model.StatusPublished).
		Order("published_at DESC").
		Select("id, title, slug, published_at").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	groups := make([]ArchiveGroup, 0, len(rows))
	for _, r := range rows {
		last := len(groups) - 1
		if last < 0 || groups[last].Year != r.PublishedAt.Year() || groups[last].Month != int(r.PublishedAt.Month()) {
			groups = append(groups, ArchiveGroup{Year: r.PublishedAt.Year(), Month: int(r.PublishedAt.Month())})
			last++
		}
		groups[last].Posts = append(groups[last].Posts, r)
	}
	return groups, nil
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
