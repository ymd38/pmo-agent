package domain

import "time"

// ProjectMember はプロジェクトへのメンバーアサイン。
// user_id は論理削除保護（ON DELETE RESTRICT）のため物理削除されたユーザーを指すことはない。
type ProjectMember struct {
	ID                int        `json:"id"                 gorm:"primaryKey"`
	ProjectID         int        `json:"project_id"`
	UserID            int        `json:"user_id"`
	AllocationPercent *float64   `json:"allocation_percent" gorm:"column:allocation_percent"`
	StartDate         *time.Time `json:"start_date"         gorm:"type:date"`
	EndDate           *time.Time `json:"end_date"           gorm:"type:date"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (ProjectMember) TableName() string { return "project_members" }
