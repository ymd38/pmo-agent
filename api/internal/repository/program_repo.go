package repository

import (
	"context"
	"fmt"

	"pmo-agent/api/internal/domain"

	"gorm.io/gorm"
)

type ProgramRepo struct {
	db *gorm.DB
}

func NewProgramRepo(db *gorm.DB) *ProgramRepo { return &ProgramRepo{db: db} }

func (r *ProgramRepo) Create(ctx context.Context, p *domain.Program) error {
	return wrapConflict(r.db.WithContext(ctx).Create(p).Error)
}

func (r *ProgramRepo) FindByID(ctx context.Context, id int) (*domain.Program, error) {
	var p domain.Program
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, wrapNotFound(err)
	}
	return &p, nil
}

func (r *ProgramRepo) FindByCode(ctx context.Context, code string) (*domain.Program, error) {
	var p domain.Program
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&p).Error; err != nil {
		return nil, wrapNotFound(err)
	}
	return &p, nil
}

func (r *ProgramRepo) List(ctx context.Context) ([]domain.Program, error) {
	var ps []domain.Program
	if err := r.db.WithContext(ctx).Order("code").Find(&ps).Error; err != nil {
		return nil, err
	}
	return ps, nil
}

// Update は name / description のみ更新する（code は不変）。
func (r *ProgramRepo) Update(ctx context.Context, p *domain.Program) error {
	return r.db.WithContext(ctx).Model(&domain.Program{}).
		Where("id = ?", p.ID).
		Select("name", "description").
		Updates(map[string]any{"name": p.Name, "description": p.Description}).Error
}

// Delete は配下にプロジェクトが存在する場合（FK RESTRICT）は 409、
// 対象が存在しない場合は 404 を返す。
// usecase 側でも配下チェックしているが、count 後の同時 INSERT（TOCTOU）に対する多層防御。
func (r *ProgramRepo) Delete(ctx context.Context, id int) error {
	res := r.db.WithContext(ctx).Delete(&domain.Program{}, id)
	if res.Error != nil {
		if isForeignKeyViolation(res.Error) {
			return fmt.Errorf("%w: 配下にプロジェクトが存在するため削除できません", domain.ErrConflict)
		}
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ProgramRepo) MaxSeqNo(ctx context.Context, programType string, fiscalYear int) (int, error) {
	var max *int
	err := r.db.WithContext(ctx).Model(&domain.Program{}).
		Where("type = ? AND fiscal_year = ?", programType, fiscalYear).
		Select("MAX(seq_no)").Scan(&max).Error
	if err != nil || max == nil {
		return 0, err
	}
	return *max, nil
}
