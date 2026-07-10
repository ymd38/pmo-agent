package repository

import (
	"context"

	"pmo-agent/api/internal/domain"

	"gorm.io/gorm"
)

type CategoryRepo struct {
	db *gorm.DB
}

func NewCategoryRepo(db *gorm.DB) *CategoryRepo { return &CategoryRepo{db: db} }

func (r *CategoryRepo) ListCategories(ctx context.Context, includeInactive bool) ([]domain.Category, error) {
	q := r.db.WithContext(ctx).Order("sort_order, id")
	if !includeInactive {
		q = q.Where("is_active = ?", true)
	}
	var cats []domain.Category
	if err := q.Find(&cats).Error; err != nil {
		return nil, err
	}
	return cats, nil
}

func (r *CategoryRepo) CreateCategory(ctx context.Context, c *domain.Category) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *CategoryRepo) UpdateCategory(ctx context.Context, c *domain.Category) error {
	return r.db.WithContext(ctx).Model(&domain.Category{}).
		Where("id = ?", c.ID).
		Select("name", "description", "is_required", "sort_order").
		Updates(map[string]any{
			"name":        c.Name,
			"description": c.Description,
			"is_required": c.IsRequired,
			"sort_order":  c.SortOrder,
		}).Error
}

func (r *CategoryRepo) DeactivateCategory(ctx context.Context, id int) error {
	return r.setCategoryActive(ctx, id, false)
}

func (r *CategoryRepo) ReactivateCategory(ctx context.Context, id int) error {
	return r.setCategoryActive(ctx, id, true)
}

func (r *CategoryRepo) setCategoryActive(ctx context.Context, id int, active bool) error {
	res := r.db.WithContext(ctx).Model(&domain.Category{}).
		Where("id = ?", id).Update("is_active", active)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *CategoryRepo) ListValues(ctx context.Context, categoryID int, includeInactive bool) ([]domain.CategoryValue, error) {
	q := r.db.WithContext(ctx).Where("category_id = ?", categoryID).Order("sort_order, id")
	if !includeInactive {
		q = q.Where("is_active = ?", true)
	}
	var vals []domain.CategoryValue
	if err := q.Find(&vals).Error; err != nil {
		return nil, err
	}
	return vals, nil
}

func (r *CategoryRepo) FindValueByID(ctx context.Context, id int) (*domain.CategoryValue, error) {
	var v domain.CategoryValue
	if err := r.db.WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, wrapNotFound(err)
	}
	return &v, nil
}

func (r *CategoryRepo) CreateValue(ctx context.Context, v *domain.CategoryValue) error {
	return r.db.WithContext(ctx).Create(v).Error
}

func (r *CategoryRepo) UpdateValue(ctx context.Context, v *domain.CategoryValue) error {
	return r.db.WithContext(ctx).Model(&domain.CategoryValue{}).
		Where("id = ?", v.ID).
		Select("label", "sort_order").
		Updates(map[string]any{"label": v.Label, "sort_order": v.SortOrder}).Error
}

func (r *CategoryRepo) DeactivateValue(ctx context.Context, id int) error {
	return r.setValueActive(ctx, id, false)
}

func (r *CategoryRepo) ReactivateValue(ctx context.Context, id int) error {
	return r.setValueActive(ctx, id, true)
}

func (r *CategoryRepo) setValueActive(ctx context.Context, id int, active bool) error {
	res := r.db.WithContext(ctx).Model(&domain.CategoryValue{}).
		Where("id = ?", id).Update("is_active", active)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
