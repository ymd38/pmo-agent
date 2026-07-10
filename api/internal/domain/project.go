package domain

import "time"

// ProjectStatus はプロジェクトのステータス。
type ProjectStatus string

const (
	StatusPlanning  ProjectStatus = "planning"
	StatusActive    ProjectStatus = "active"
	StatusCompleted ProjectStatus = "completed"
	StatusCancelled ProjectStatus = "cancelled"
)

// Project は実作業単位。必ずプログラムに属する（ProgramID NOT NULL）。
// ProjectCode は承認時（active遷移）に枝番を採番して発行し、以後不変。発行前は nil。
type Project struct {
	ID               int           `json:"id"                gorm:"primaryKey"`
	ProgramID        int           `json:"program_id"`
	BranchNo         *int          `json:"branch_no"`
	ProjectCode      *string       `json:"project_code"`
	Name             string        `json:"name"`
	Description      string        `json:"description"`
	PMID             *int          `json:"pm_id"             gorm:"column:pm_id"`
	ApproverID       *int          `json:"approver_id"`
	Vendor           string        `json:"vendor"`
	Budget           *int64        `json:"budget"`
	StartDate        *time.Time    `json:"start_date"        gorm:"type:date"`
	EndDate          *time.Time    `json:"end_date"          gorm:"type:date"`
	Status           ProjectStatus `json:"status"`
	BacklogProjectID string        `json:"backlog_project_id"`
	AIReviewDocPath  string        `json:"ai_review_doc_path" gorm:"column:ai_review_doc_path"`
	CreatedBy        int           `json:"created_by"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

func (Project) TableName() string { return "projects" }

// IsValidStatus はステータス値が許可された4値のいずれかなら true。
func IsValidStatus(s ProjectStatus) bool {
	switch s {
	case StatusPlanning, StatusActive, StatusCompleted, StatusCancelled:
		return true
	}
	return false
}

// CanTransitionTo は PUT /projects/:id 経由で許可するステータス遷移を判定する。
//   - 同一ステータス（フィールドのみの編集）は常に許可
//   - planning → active は「コード発行（IssueCode）」専用経路のため、ここでは許可しない
//     （active なプロジェクトは必ず project_code を持つ、という不変条件を守る）
//   - planning からは中止のみ可
//   - active からは完了・中止のみ可
//   - completed / cancelled は終端。以後の遷移は不可
func CanTransitionTo(from, to ProjectStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case StatusPlanning:
		return to == StatusCancelled
	case StatusActive:
		return to == StatusCompleted || to == StatusCancelled
	default:
		return false
	}
}
