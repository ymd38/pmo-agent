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

// AggregateProjects は与えられたプロジェクト群から集計値を算出する（DB非保持の表示用）。
// リポジトリの AggregateByProgram（全プログラムを GROUP BY 集計）と同じ結果を、
// 既に取得済みの単一プログラム配下 projects からメモリ上で求めるためのヘルパ。
// budget が nil の行は 0 として合算し、start/end は非 nil の最小/最大を採る。
func AggregateProjects(projects []Project) ProgramAggregate {
	agg := ProgramAggregate{StatusCounts: map[string]int{}}
	agg.ProjectCount = len(projects)
	for i := range projects {
		p := projects[i]
		if p.Budget != nil {
			agg.TotalBudget += *p.Budget
		}
		if p.StartDate != nil && (agg.StartDate == nil || p.StartDate.Before(*agg.StartDate)) {
			agg.StartDate = p.StartDate
		}
		if p.EndDate != nil && (agg.EndDate == nil || p.EndDate.After(*agg.EndDate)) {
			agg.EndDate = p.EndDate
		}
		agg.StatusCounts[string(p.Status)]++
	}
	return agg
}
