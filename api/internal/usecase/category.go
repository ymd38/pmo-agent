package usecase

import (
	"context"
	"fmt"
	"strings"

	"pmo-agent/api/internal/domain"
)

// CategoryUsecase はプロジェクト属性マスタ（カテゴリ／値）の管理を担う。
// 値・カテゴリの削除は is_active=false の論理削除のみ（過去アサインを保護）。
type CategoryUsecase struct {
	repo CategoryRepository
}

func NewCategoryUsecase(repo CategoryRepository) *CategoryUsecase {
	return &CategoryUsecase{repo: repo}
}

type CategoryInput struct {
	Code        string
	Name        string
	Description string
	IsRequired  bool
	SortOrder   int
}

type ValueInput struct {
	Code      string
	Label     string
	SortOrder int
}

func (uc *CategoryUsecase) ListCategories(ctx context.Context, includeInactive bool) ([]domain.Category, error) {
	return uc.repo.ListCategories(ctx, includeInactive)
}

func (uc *CategoryUsecase) CreateCategory(ctx context.Context, in CategoryInput) (*domain.Category, error) {
	if strings.TrimSpace(in.Code) == "" || strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("%w: コードと名称は必須です", domain.ErrValidation)
	}
	c := &domain.Category{
		Code:        strings.TrimSpace(in.Code),
		Name:        strings.TrimSpace(in.Name),
		Description: in.Description,
		IsRequired:  in.IsRequired,
		SortOrder:   in.SortOrder,
		IsActive:    true,
	}
	if err := uc.repo.CreateCategory(ctx, c); err != nil {
		return nil, fmt.Errorf("usecase.CreateCategory: %w", err)
	}
	return c, nil
}

func (uc *CategoryUsecase) UpdateCategory(ctx context.Context, id int, in CategoryInput) (*domain.Category, error) {
	cats, err := uc.repo.ListCategories(ctx, true)
	if err != nil {
		return nil, err
	}
	c := findCategory(cats, id)
	if c == nil {
		return nil, domain.ErrNotFound
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("%w: 名称は必須です", domain.ErrValidation)
	}
	// code は不変。name 以下を更新。
	c.Name = strings.TrimSpace(in.Name)
	c.Description = in.Description
	c.IsRequired = in.IsRequired
	c.SortOrder = in.SortOrder
	if err := uc.repo.UpdateCategory(ctx, c); err != nil {
		return nil, fmt.Errorf("usecase.UpdateCategory: %w", err)
	}
	return c, nil
}

func (uc *CategoryUsecase) DeactivateCategory(ctx context.Context, id int) error {
	return uc.repo.DeactivateCategory(ctx, id)
}

// ReactivateCategory は無効化したカテゴリを再有効化する（誤操作からの復帰用）。
func (uc *CategoryUsecase) ReactivateCategory(ctx context.Context, id int) error {
	return uc.repo.ReactivateCategory(ctx, id)
}

func (uc *CategoryUsecase) ListValues(ctx context.Context, categoryID int, includeInactive bool) ([]domain.CategoryValue, error) {
	return uc.repo.ListValues(ctx, categoryID, includeInactive)
}

func (uc *CategoryUsecase) CreateValue(ctx context.Context, categoryID int, in ValueInput) (*domain.CategoryValue, error) {
	if strings.TrimSpace(in.Code) == "" || strings.TrimSpace(in.Label) == "" {
		return nil, fmt.Errorf("%w: コードとラベルは必須です", domain.ErrValidation)
	}
	v := &domain.CategoryValue{
		CategoryID: categoryID,
		Code:       strings.TrimSpace(in.Code),
		Label:      strings.TrimSpace(in.Label),
		SortOrder:  in.SortOrder,
		IsActive:   true,
	}
	if err := uc.repo.CreateValue(ctx, v); err != nil {
		return nil, fmt.Errorf("usecase.CreateValue: %w", err)
	}
	return v, nil
}

func (uc *CategoryUsecase) UpdateValue(ctx context.Context, categoryID, valueID int, in ValueInput) (*domain.CategoryValue, error) {
	vals, err := uc.repo.ListValues(ctx, categoryID, true)
	if err != nil {
		return nil, err
	}
	v := findValue(vals, valueID)
	if v == nil {
		return nil, domain.ErrNotFound
	}
	if strings.TrimSpace(in.Label) == "" {
		return nil, fmt.Errorf("%w: ラベルは必須です", domain.ErrValidation)
	}
	v.Label = strings.TrimSpace(in.Label)
	v.SortOrder = in.SortOrder
	if err := uc.repo.UpdateValue(ctx, v); err != nil {
		return nil, fmt.Errorf("usecase.UpdateValue: %w", err)
	}
	return v, nil
}

func (uc *CategoryUsecase) DeactivateValue(ctx context.Context, valueID int) error {
	return uc.repo.DeactivateValue(ctx, valueID)
}

// ReactivateValue は無効化した値を再有効化する（誤操作からの復帰用）。
func (uc *CategoryUsecase) ReactivateValue(ctx context.Context, valueID int) error {
	return uc.repo.ReactivateValue(ctx, valueID)
}

func findCategory(cats []domain.Category, id int) *domain.Category {
	for i := range cats {
		if cats[i].ID == id {
			return &cats[i]
		}
	}
	return nil
}

func findValue(vals []domain.CategoryValue, id int) *domain.CategoryValue {
	for i := range vals {
		if vals[i].ID == id {
			return &vals[i]
		}
	}
	return nil
}
