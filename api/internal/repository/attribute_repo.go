package repository

import (
	"context"

	"pmo-agent/api/internal/domain"

	"gorm.io/gorm"
)

type AttributeRepo struct {
	db *gorm.DB
}

func NewAttributeRepo(db *gorm.DB) *AttributeRepo { return &AttributeRepo{db: db} }

// ListByProject は属性アサインにカテゴリ／値のラベルを結合して返す。
// 論理削除済み（is_active=false）の値も過去アサインとして残るため除外しない。
// 並びはカテゴリ→値の sort_order に揃え、UI のグルーピング表示に乗せやすくする。
func (r *AttributeRepo) ListByProject(ctx context.Context, projectID int) ([]domain.ProjectAttribute, error) {
	var out []domain.ProjectAttribute
	err := r.db.WithContext(ctx).
		Table("project_attribute_assignments AS a").
		Select(`a.id AS id,
			a.category_id AS category_id,
			c.code AS category_code,
			c.name AS category_name,
			a.value_id AS value_id,
			v.code AS value_code,
			v.label AS value_label,
			v.is_active AS value_is_active`).
		Joins("JOIN project_categories c ON c.id = a.category_id").
		Joins("JOIN project_category_values v ON v.id = a.value_id").
		Where("a.project_id = ?", projectID).
		Order("c.sort_order, c.id, v.sort_order, v.id").
		Scan(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *AttributeRepo) Exists(ctx context.Context, projectID, valueID int) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.AttributeAssignment{}).
		Where("project_id = ? AND value_id = ?", projectID, valueID).
		Count(&n).Error
	return n > 0, err
}

func (r *AttributeRepo) Assign(ctx context.Context, a *domain.AttributeAssignment) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *AttributeRepo) Unassign(ctx context.Context, projectID, valueID int) error {
	res := r.db.WithContext(ctx).
		Where("project_id = ? AND value_id = ?", projectID, valueID).
		Delete(&domain.AttributeAssignment{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
