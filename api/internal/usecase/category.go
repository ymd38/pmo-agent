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
	c, err := uc.repo.FindCategoryByID(ctx, id)
	if err != nil {
		return nil, err
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
	v, err := uc.findValueInCategory(ctx, categoryID, valueID)
	if err != nil {
		return nil, err
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

func (uc *CategoryUsecase) DeactivateValue(ctx context.Context, categoryID, valueID int) error {
	if _, err := uc.findValueInCategory(ctx, categoryID, valueID); err != nil {
		return err
	}
	return uc.repo.DeactivateValue(ctx, valueID)
}

// ReactivateValue は無効化した値を再有効化する（誤操作からの復帰用）。
func (uc *CategoryUsecase) ReactivateValue(ctx context.Context, categoryID, valueID int) error {
	if _, err := uc.findValueInCategory(ctx, categoryID, valueID); err != nil {
		return err
	}
	return uc.repo.ReactivateValue(ctx, valueID)
}

// findValueInCategory は valueID の値を取得し、指定カテゴリに所属することを検証する。
// 既存値を対象とする全ミューテーション（Update/Deactivate/Reactivate）が通る単一経路。
// 値が存在しない、または別カテゴリ所属なら ErrNotFound を返す
// （別カテゴリの値の存在を呼び出し側に漏らさない＝存在秘匿）。
func (uc *CategoryUsecase) findValueInCategory(ctx context.Context, categoryID, valueID int) (*domain.CategoryValue, error) {
	v, err := uc.repo.FindValueByID(ctx, valueID)
	if err != nil {
		return nil, err
	}
	if v.CategoryID != categoryID {
		return nil, domain.ErrNotFound
	}
	return v, nil
}
