package repository

import (
	"context"

	"pmo-agent/api/internal/domain"

	"gorm.io/gorm"
)

type MemberRepo struct {
	db *gorm.DB
}

func NewMemberRepo(db *gorm.DB) *MemberRepo { return &MemberRepo{db: db} }

func (r *MemberRepo) ListByProject(ctx context.Context, projectID int) ([]domain.ProjectMember, error) {
	var ms []domain.ProjectMember
	if err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).Order("id").Find(&ms).Error; err != nil {
		return nil, err
	}
	return ms, nil
}

// Assign はアサインを作成する。同一 (project_id, user_id) の重複は UNIQUE 制約により
// ErrConflict へ写像される。
func (r *MemberRepo) Assign(ctx context.Context, m *domain.ProjectMember) error {
	return wrapConflict(r.db.WithContext(ctx).Create(m).Error)
}

// Update は可変フィールド（割合・期間）を更新する。対象が無ければ ErrNotFound。
func (r *MemberRepo) Update(ctx context.Context, m *domain.ProjectMember) error {
	res := r.db.WithContext(ctx).Model(&domain.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", m.ProjectID, m.UserID).
		Select("allocation_percent", "start_date", "end_date").
		Updates(map[string]any{
			"allocation_percent": m.AllocationPercent,
			"start_date":         m.StartDate,
			"end_date":           m.EndDate,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *MemberRepo) Unassign(ctx context.Context, projectID, userID int) error {
	res := r.db.WithContext(ctx).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Delete(&domain.ProjectMember{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ProjectIDsByUser はユーザーがアサインされているプロジェクト ID を返す（スコープ解決用）。
func (r *MemberRepo) ProjectIDsByUser(ctx context.Context, userID int) ([]int, error) {
	var ids []int
	err := r.db.WithContext(ctx).Model(&domain.ProjectMember{}).
		Where("user_id = ?", userID).Pluck("project_id", &ids).Error
	return ids, err
}
