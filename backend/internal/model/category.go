package model

// Category 分类。
type Category struct {
	BaseModel
	Name string `gorm:"size:50;uniqueIndex;not null" json:"name"`
	Sort int    `gorm:"not null;default:0" json:"sort"`
}
