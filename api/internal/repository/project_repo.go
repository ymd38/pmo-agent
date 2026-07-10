package repository

import (
	"context"
	"fmt"
	"time"

	"pmo-agent/api/internal/domain"

	"gorm.io/gorm"
)

type ProjectRepo struct {
	db *gorm.DB
}

func NewProjectRepo(db *gorm.DB) *ProjectRepo { return &ProjectRepo{db: db} }

func (r *ProjectRepo) Create(ctx context.Context, p *domain.Project) error {
	return wrapConflict(r.db.WithContext(ctx).Create(p).Error)
}

func (r *ProjectRepo) FindByID(ctx context.Context, id int) (*domain.Project, error) {
	var p domain.Project
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, wrapNotFound(err)
	}
	return &p, nil
}

func (r *ProjectRepo) List(ctx context.Context) ([]domain.Project, error) {
	var ps []domain.Project
	if err := r.db.WithContext(ctx).Order("id").Find(&ps).Error; err != nil {
		return nil, err
	}
	return ps, nil
}

func (r *ProjectRepo) ListByProgram(ctx context.Context, programID int) ([]domain.Project, error) {
	var ps []domain.Project
	if err := r.db.WithContext(ctx).Where("program_id = ?", programID).Order("branch_no, id").Find(&ps).Error; err != nil {
		return nil, err
	}
	return ps, nil
}

// Update は可変フィールドを更新する（program_id / branch_no / project_code は不変）。
func (r *ProjectRepo) Update(ctx context.Context, p *domain.Project) error {
	return r.db.WithContext(ctx).Model(&domain.Project{}).
		Where("id = ?", p.ID).
		Select("name", "description", "pm_id", "approver_id", "vendor", "budget", "start_date", "end_date", "status", "backlog_project_id").
		Updates(map[string]any{
			"name":               p.Name,
			"description":        p.Description,
			"pm_id":              p.PMID,
			"approver_id":        p.ApproverID,
			"vendor":             p.Vendor,
			"budget":             p.Budget,
			"start_date":         p.StartDate,
			"end_date":           p.EndDate,
			"status":             p.Status,
			"backlog_project_id": p.BacklogProjectID,
		}).Error
}

func (r *ProjectRepo) Delete(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&domain.Project{}, id).Error
}

func (r *ProjectRepo) CountByProgram(ctx context.Context, programID int) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Project{}).Where("program_id = ?", programID).Count(&n).Error
	return n, err
}

func (r *ProjectRepo) MaxBranchNo(ctx context.Context, programID int) (int, error) {
	var max *int
	err := r.db.WithContext(ctx).Model(&domain.Project{}).
		Where("program_id = ?", programID).
		Select("MAX(branch_no)").Scan(&max).Error
	if err != nil || max == nil {
		return 0, err
	}
	return *max, nil
}

// IssueCode は枝番・コードを採番し active へ遷移させる（発行は一度きり）。
// 発行済み（project_code IS NOT NULL）または planning 以外への UPDATE は WHERE 句で弾き、
// 更新行が無ければ ErrConflict を返す。usecase の事前チェックをすり抜けた並行リクエストでも
// project_code の不変性（Critical Business Rule）を DB レベルで保証する。
func (r *ProjectRepo) IssueCode(ctx context.Context, projectID, branchNo int, code string) error {
	res := r.db.WithContext(ctx).Model(&domain.Project{}).
		Where("id = ? AND project_code IS NULL AND status = ?", projectID, domain.StatusPlanning).
		Select("branch_no", "project_code", "status").
		Updates(map[string]any{
			"branch_no":    branchNo,
			"project_code": code,
			"status":       domain.StatusActive,
		})
	if err := wrapConflict(res.Error); err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("%w: このプロジェクトは既にコードが発行されています", domain.ErrConflict)
	}
	return nil
}

func (r *ProjectRepo) AggregateByProgram(ctx context.Context) (map[int]domain.ProgramAggregate, error) {
	type aggRow struct {
		ProgramID int
		Cnt       int
		Total     int64
		MinStart  *time.Time
		MaxEnd    *time.Time
	}
	var rows []aggRow
	if err := r.db.WithContext(ctx).Model(&domain.Project{}).
		Select("program_id, COUNT(*) AS cnt, COALESCE(SUM(budget),0) AS total, MIN(start_date) AS min_start, MAX(end_date) AS max_end").
		Group("program_id").Scan(&rows).Error; err != nil {
		return nil, err
	}

	type statusRow struct {
		ProgramID int
		Status    string
		C         int
	}
	var sRows []statusRow
	if err := r.db.WithContext(ctx).Model(&domain.Project{}).
		Select("program_id, status, COUNT(*) AS c").
		Group("program_id, status").Scan(&sRows).Error; err != nil {
		return nil, err
	}

	out := make(map[int]domain.ProgramAggregate, len(rows))
	for _, r := range rows {
		out[r.ProgramID] = domain.ProgramAggregate{
			ProjectCount: r.Cnt,
			TotalBudget:  r.Total,
			StartDate:    r.MinStart,
			EndDate:      r.MaxEnd,
			StatusCounts: map[string]int{},
		}
	}
	for _, s := range sRows {
		if agg, ok := out[s.ProgramID]; ok {
			agg.StatusCounts[s.Status] = s.C
			out[s.ProgramID] = agg
		}
	}
	return out, nil
}
