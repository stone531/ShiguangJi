package model

import "time"

// Like 点赞（指纹去重）。
type Like struct {
	ID          uint      `gorm:"primaryKey" json:"-"`
	PostID      uint      `gorm:"uniqueIndex:uniq_post_fp" json:"post_id"`
	Fingerprint string    `gorm:"size:64;uniqueIndex:uniq_post_fp" json:"-"`
	CreatedAt   time.Time `json:"created_at"`
}
