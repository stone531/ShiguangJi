package model

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 所有表通用；软删除表额外嵌 SoftDelete。
type BaseModel struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SoftDelete GORM 软删除（索引自动创建）。
type SoftDelete struct {
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
