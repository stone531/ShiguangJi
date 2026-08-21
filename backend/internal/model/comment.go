package model

import "time"

// 评论状态枚举。
const (
	CommentPending  int8 = 0
	CommentApproved int8 = 1
	CommentRejected int8 = 2
)

// Comment 评论（匿名 + 博主回复，两级结构）。
type Comment struct {
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`
	SoftDelete
	PostID    uint      `gorm:"index:idx_post_status_created,priority:1" json:"post_id"`
	Status    int8      `gorm:"index:idx_post_status_created,priority:2;not null;default:0" json:"-"`
	ParentID  uint      `gorm:"not null;default:0" json:"parent_id"`
	Nickname  string    `gorm:"size:50;not null" json:"nickname"`
	Email     string    `gorm:"size:100" json:"-"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	IsAdmin   bool      `gorm:"not null;default:false" json:"is_admin"`
	IP        string    `gorm:"size:45;index:idx_ip_created,priority:1" json:"-"`
	UserAgent string    `gorm:"size:255" json:"-"`
	CreatedAt time.Time `gorm:"index:idx_post_status_created,priority:3" json:"created_at"`
}

// Email/IP json:"-"：前台接口永不返回；admin 接口用独立 AdminCommentVO。
