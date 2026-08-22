package service

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"shiguangji/internal/errs"
	"shiguangji/internal/model"
)

// CategoryWithCount 分类 + 已发布文章数（前台导航/首页 Tab）。
type CategoryWithCount struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Sort      int    `json:"sort"`
	PostCount int64  `json:"post_count"`
}

// TagWithCount 标签 + 已发布文章数（标签云）。
type TagWithCount struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	PostCount int64  `json:"post_count"`
}

// TaxonomyService 分类/标签公开查询。
type TaxonomyService struct{ db *gorm.DB }

// NewTaxonomyService 构造函数。
func NewTaxonomyService(db *gorm.DB) *TaxonomyService { return &TaxonomyService{db: db} }

// ListCategories 全部分类（含空分类），携带已发布文章数，sort 升序。
func (s *TaxonomyService) ListCategories(ctx context.Context) ([]CategoryWithCount, error) {
	var list []CategoryWithCount
	err := s.db.WithContext(ctx).Model(&model.Category{}).
		Select("categories.id, categories.name, categories.sort, COUNT(posts.id) AS post_count").
		Joins("LEFT JOIN posts ON posts.category_id = categories.id AND posts.status = ? AND posts.deleted_at IS NULL", model.StatusPublished).
		Group("categories.id").Order("categories.sort, categories.id").
		Scan(&list).Error
	return list, err
}

// ListTags 仅返回有已发布文章的标签，按文章数降序（标签云热度排序）。
func (s *TaxonomyService) ListTags(ctx context.Context) ([]TagWithCount, error) {
	var list []TagWithCount
	err := s.db.WithContext(ctx).Model(&model.Tag{}).
		Select("tags.id, tags.name, COUNT(posts.id) AS post_count").
		Joins("JOIN post_tags ON post_tags.tag_id = tags.id").
		Joins("JOIN posts ON posts.id = post_tags.post_id AND posts.status = ? AND posts.deleted_at IS NULL", model.StatusPublished).
		Group("tags.id").Order("post_count DESC, tags.id").
		Scan(&list).Error
	return list, err
}

// ---- 后台管理（M4）----

// ListCategoriesAdmin 后台全量分类（含空分类；编辑器选项用，不筛发布状态）。
func (s *TaxonomyService) ListCategoriesAdmin(ctx context.Context) ([]model.Category, error) {
	var list []model.Category
	err := s.db.WithContext(ctx).Order("sort, id").Find(&list).Error
	return list, err
}

// ListTagsAdmin 后台全量标签（公开 /tags 只含有已发布文章的标签，编辑草稿会丢选项）。
func (s *TaxonomyService) ListTagsAdmin(ctx context.Context) ([]model.Tag, error) {
	var list []model.Tag
	err := s.db.WithContext(ctx).Order("id").Find(&list).Error
	return list, err
}

// SaveCategory 新建（id=0）或更新分类；重名由唯一索引兜底转 40001。
func (s *TaxonomyService) SaveCategory(ctx context.Context, id uint, name string, sort int) (*model.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 50 {
		return nil, errs.New(40001, "分类名需在 1-50 字符")
	}
	cat := model.Category{Name: name, Sort: sort}
	if id == 0 {
		if err := s.db.WithContext(ctx).Create(&cat).Error; err != nil {
			if isDuplicateErr(err) {
				return nil, errs.New(40001, "分类已存在")
			}
			return nil, err
		}
		return &cat, nil
	}
	res := s.db.WithContext(ctx).Model(&model.Category{}).Where("id = ?", id).
		Updates(map[string]interface{}{"name": name, "sort": sort})
	if res.Error != nil {
		if isDuplicateErr(res.Error) {
			return nil, errs.New(40001, "分类已存在")
		}
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, errs.ErrNotFound
	}
	if err := s.db.WithContext(ctx).First(&cat, id).Error; err != nil {
		return nil, err
	}
	return &cat, nil
}

// DeleteCategory 其下仍有文章（含草稿/回收站，仅排除软删）→ 拒绝。
func (s *TaxonomyService) DeleteCategory(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cat model.Category
		if err := tx.First(&cat, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		var n int64
		if err := tx.Model(&model.Post{}).Where("category_id = ?", id).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return errs.New(40001, "分类下仍有文章，请先移出或删除文章")
		}
		return tx.Delete(&cat).Error
	})
}

// SaveTag 新建（id=0）或更新标签。
func (s *TaxonomyService) SaveTag(ctx context.Context, id uint, name string) (*model.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 50 {
		return nil, errs.New(40001, "标签名需在 1-50 字符")
	}
	tag := model.Tag{Name: name}
	if id == 0 {
		if err := s.db.WithContext(ctx).Create(&tag).Error; err != nil {
			if isDuplicateErr(err) {
				return nil, errs.New(40001, "标签已存在")
			}
			return nil, err
		}
		return &tag, nil
	}
	res := s.db.WithContext(ctx).Model(&model.Tag{}).Where("id = ?", id).Update("name", name)
	if res.Error != nil {
		if isDuplicateErr(res.Error) {
			return nil, errs.New(40001, "标签已存在")
		}
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, errs.ErrNotFound
	}
	if err := s.db.WithContext(ctx).First(&tag, id).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

// DeleteTag 标签被文章引用（含草稿）→ 拒绝；无引用直接删（post_tags 级联清理）。
func (s *TaxonomyService) DeleteTag(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tag model.Tag
		if err := tx.First(&tag, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		var n int64
		if err := tx.Model(&model.PostTag{}).Where("tag_id = ?", id).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return errs.New(40001, "标签仍被文章使用，请先在文章中移除或合并")
		}
		return tx.Delete(&tag).Error
	})
}

// MergeTag 合并标签：把 source 下的 post_tags 移到 target（去重），随后删除 source。
func (s *TaxonomyService) MergeTag(ctx context.Context, sourceID, targetID uint) error {
	if sourceID == targetID {
		return errs.New(40001, "不能合并到自身")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var source, target model.Tag
		if err := tx.First(&source, sourceID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if err := tx.First(&target, targetID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		// 已挂 target 的文章先解除 source 关联，避免复合主键冲突
		// 子查询外包一层派生表：MySQL 不允许 DELETE 与 IN 子查询直接同表（ERROR 1093）
		if err := tx.Exec(
			"DELETE FROM post_tags WHERE tag_id = ? AND post_id IN (SELECT post_id FROM (SELECT post_id FROM post_tags WHERE tag_id = ?) AS tmp)",
			sourceID, targetID,
		).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.PostTag{}).Where("tag_id = ?", sourceID).
			Update("tag_id", targetID).Error; err != nil {
			return err
		}
		return tx.Delete(&source).Error
	})
}

// isDuplicateErr 兼容 MySQL(1062) 与 sqlite(UNIQUE constraint failed) 的重名判定
// （未开 TranslateError 时 gorm.ErrDuplicatedKey 不一定出现，字符串兜底）。
func isDuplicateErr(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "Error 1062") || strings.Contains(msg, "Duplicate entry")
}
