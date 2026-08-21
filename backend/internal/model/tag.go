package model

// Tag 标签。
type Tag struct {
	BaseModel
	Name string `gorm:"size:50;uniqueIndex;not null" json:"name"`
}

// PostTag 文章标签关联（多对多）。
type PostTag struct {
	PostID uint `gorm:"primaryKey"`
	TagID  uint `gorm:"primaryKey;index"`
}
