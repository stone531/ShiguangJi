package model

import "time"

// 内容类型枚举。
const (
	TypeArticle int8 = 1 // 文章
	TypeGallery int8 = 2 // 图片集
	TypeVideo   int8 = 3 // 视频
)

// 内容状态枚举。
const (
	StatusDraft     int8 = 0 // 草稿
	StatusPublished int8 = 1 // 已发布
	StatusTrashed   int8 = 2 // 回收站
)

// Post 内容表（文章/图片集/视频统一）。
type Post struct {
	BaseModel
	SoftDelete
	Title        string     `gorm:"size:200;not null" json:"title"`
	Slug         string     `gorm:"size:191;uniqueIndex;not null" json:"slug"` // 191：utf8mb4 索引不超 MySQL 5.6 的 767 字节前缀上限
	Summary      string     `gorm:"size:500" json:"summary"`
	Cover        string     `gorm:"size:255" json:"cover"`
	Content      string     `gorm:"type:longtext;not null" json:"content"`
	ContentHTML  string     `gorm:"type:longtext" json:"-"`
	Type         int8       `gorm:"not null;default:1" json:"type"`
	VideoURL     string     `gorm:"size:500" json:"video_url"`
	CategoryID   *uint      `gorm:"index:idx_category_status,priority:1" json:"category_id"` // SET NULL 需要指针
	Status       int8       `gorm:"index:idx_category_status,priority:2;not null;default:0" json:"status"`
	IsTop        bool       `gorm:"not null;default:false" json:"is_top"`
	IsFeatured   bool       `gorm:"index" json:"is_featured"`
	Views        int64      `gorm:"not null;default:0" json:"views"`
	LikeCount    int64      `gorm:"not null;default:0" json:"like_count"`
	CommentCount int64      `gorm:"not null;default:0" json:"comment_count"`
	PublishedAt  *time.Time `json:"published_at"`
	Category     *Category  `json:"category,omitempty"`
	Tags         []Tag      `gorm:"many2many:post_tags;" json:"tags,omitempty"`
}

// TableName 表名。
func (Post) TableName() string { return "posts" }
