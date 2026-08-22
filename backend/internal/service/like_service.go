package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"gorm.io/gorm"

	"shiguangji/internal/errs"
	"shiguangji/internal/model"
)

// LikeService 点赞：指纹 = SHA256(IP+UA+salt)，(post_id, fingerprint) 唯一约束幂等。
type LikeService struct {
	db   *gorm.DB
	salt string
}

// NewLikeService 构造函数。
func NewLikeService(db *gorm.DB, salt string) *LikeService {
	return &LikeService{db: db, salt: salt}
}

// fingerprint 匿名访客标识：IP+UA 加盐哈希，不可反推。
func (s *LikeService) fingerprint(ip, ua string) string {
	sum := sha256.Sum256([]byte(ip + "\x00" + ua + "\x00" + s.salt))
	return hex.EncodeToString(sum[:])
}

// Status 查询（GET /api/posts/:slug/like）。
func (s *LikeService) Status(ctx context.Context, slug, ip, ua string) (liked bool, count int64, err error) {
	post, err := s.findPublished(ctx, slug)
	if err != nil {
		return false, 0, err
	}
	fp := s.fingerprint(ip, ua)
	var hits int64
	if err := s.db.WithContext(ctx).Model(&model.Like{}).
		Where("post_id = ? AND fingerprint = ?", post.ID, fp).
		Count(&hits).Error; err != nil {
		return false, 0, err
	}
	return hits > 0, post.LikeCount, nil
}

// Toggle 切换点赞（POST /api/posts/:slug/like）：事务内 INSERT/DELETE + like_count 同步。
// 并发同指纹双击下唯一索引兜底：插入冲突视为已赞幂等返回，删除零行跳过减计数，杜绝 500 与计数漂移。
func (s *LikeService) Toggle(ctx context.Context, slug, ip, ua string) (liked bool, count int64, err error) {
	post, err := s.findPublished(ctx, slug)
	if err != nil {
		return false, 0, err
	}
	fp := s.fingerprint(ip, ua)
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.Like
		err := tx.Where("post_id = ? AND fingerprint = ?", post.ID, fp).First(&existing).Error
		switch {
		case err == nil: // 已赞 → 取消
			res := tx.Delete(&existing)
			if res.Error != nil {
				return res.Error
			}
			liked = false
			if res.RowsAffected == 0 {
				return nil // 并发赢家已删：不重复减计数
			}
			return tx.Model(&model.Post{}).Where("id = ? AND like_count > 0", post.ID).
				UpdateColumn("like_count", gorm.Expr("like_count - 1")).Error
		case errors.Is(err, gorm.ErrRecordNotFound): // 未赞 → 点赞
			if err := tx.Create(&model.Like{PostID: post.ID, Fingerprint: fp}).Error; err != nil {
				if isDuplicateErr(err) {
					liked = true // 并发赢家已插入：视为已赞，不重复加计数
					return nil
				}
				return err
			}
			liked = true
			return tx.Model(&model.Post{}).Where("id = ?", post.ID).
				UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
		default:
			return err
		}
	})
	if err != nil {
		return false, 0, err
	}
	// 事务外重读，拿到终态计数
	var p model.Post
	if err := s.db.WithContext(ctx).Select("like_count").First(&p, post.ID).Error; err != nil {
		return false, 0, err
	}
	return liked, p.LikeCount, nil
}

func (s *LikeService) findPublished(ctx context.Context, slug string) (*model.Post, error) {
	var post model.Post
	if err := s.db.WithContext(ctx).
		Where("slug = ? AND status = ?", slug, model.StatusPublished).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	return &post, nil
}
