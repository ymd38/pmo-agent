package domain

import "time"

// Program は PMO の管理・集計単位（プロジェクトコードのプレフィックス名前空間）。
// 予算・期間は配下プロジェクトの集計値として算出し、DB には保持しない。
type Program struct {
	ID          int       `json:"id"          gorm:"primaryKey"`
	Code        string    `json:"code"`        // 種別-年度-連番（例: INV-2026-0001）。materialized・以後不変
	Type        string    `json:"type"`        // 種別プレフィックス（例: INV / MNT / OPS）
	FiscalYear  int       `json:"fiscal_year"` // 会計年度（例: 2026）
	SeqNo       int       `json:"seq_no"`      // (type, fiscal_year) 内の連番。自動採番
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   int       `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Program) TableName() string { return "programs" }

// ProgramAggregate は配下プロジェクトの集計値（表示用・DB非保持）。
type ProgramAggregate struct {
	ProjectCount int            `json:"project_count"`
	TotalBudget  int64          `json:"total_budget"`
	StartDate    *time.Time     `json:"start_date"`
	EndDate      *time.Time     `json:"end_date"`
	StatusCounts map[string]int `json:"status_counts"`
}
