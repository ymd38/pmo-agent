package domain

import "time"

// Category はプロジェクト属性カテゴリ定義（EAVパターン）。
type Category struct {
	ID          int       `json:"id"          gorm:"primaryKey"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsRequired  bool      `json:"is_required"`
	SortOrder   int       `json:"sort_order"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Category) TableName() string { return "project_categories" }

// CategoryValue はカテゴリに属する値。削除は is_active=false の論理削除のみ。
type CategoryValue struct {
	ID         int       `json:"id"          gorm:"primaryKey"`
	CategoryID int       `json:"category_id"`
	Code       string    `json:"code"`
	Label      string    `json:"label"`
	SortOrder  int       `json:"sort_order"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (CategoryValue) TableName() string { return "project_category_values" }
