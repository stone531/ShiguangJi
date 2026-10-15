package service

import (
	"context"

	"gorm.io/gorm"

	"shiguangji/internal/model"
)

// StatsVO 仪表盘六指标（待审数用于顶部提醒条）。
type StatsVO struct {
	PostTotal      int64 `json:"post_total"`
	PostPublished  int64 `json:"post_published"`
	CommentTotal   int64 `json:"comment_total"`
	CommentPending int64 `json:"comment_pending"`
	ViewTotal      int64 `json:"view_total"`
	LikeTotal      int64 `json:"like_total"`
}

// StatsService 后台统计。
type StatsService struct{ db *gorm.DB }

// NewStatsService 构造函数。
func NewStatsService(db *gorm.DB) *StatsService { return &StatsService{db: db} }

// Get 六指标一次取齐（软删均不计入；status=1 即布尔求和，MySQL/sqlite 通用）。
func (s *StatsService) Get(ctx context.Context) (*StatsVO, error) {
	vo := &StatsVO{}
	err := s.db.WithContext(ctx).Model(&model.Post{}).
		Select("COUNT(*) AS post_total, COALESCE(SUM(status = 1), 0) AS post_published, " +
			"COALESCE(SUM(views), 0) AS view_total, COALESCE(SUM(like_count), 0) AS like_total").
		Scan(&vo).Error
	if err != nil {
		return nil, err
	}
	var c struct {
		CommentTotal   int64
		CommentPending int64
	}
	if err := s.db.WithContext(ctx).Model(&model.Comment{}).
		Select("COUNT(*) AS comment_total, COALESCE(SUM(status = 0), 0) AS comment_pending").
		Scan(&c).Error; err != nil {
		return nil, err
	}
	vo.CommentTotal, vo.CommentPending = c.CommentTotal, c.CommentPending
	return vo, nil
}
