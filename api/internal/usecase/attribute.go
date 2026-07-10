package usecase

import (
	"context"
	"fmt"

	"pmo-agent/api/internal/domain"
)

// AttributeUsecase はプロジェクトへの属性値アサインを担う。
// マスタ（カテゴリ／値）は CategoryUsecase が管理し、ここでは「どのプロジェクトに
// どの値を紐付けるか」のみを扱う。category_id は値から導出し、入力の不整合を防ぐ。
type AttributeUsecase struct {
	attrs    AttributeRepository
	projects ProjectRepository
	cats     CategoryRepository
}

func NewAttributeUsecase(attrs AttributeRepository, projects ProjectRepository, cats CategoryRepository) *AttributeUsecase {
	return &AttributeUsecase{attrs: attrs, projects: projects, cats: cats}
}

func (uc *AttributeUsecase) List(ctx context.Context, projectID int) ([]domain.ProjectAttribute, error) {
	if _, err := uc.projects.FindByID(ctx, projectID); err != nil {
		return nil, err // 未存在は ErrNotFound
	}
	return uc.attrs.ListByProject(ctx, projectID)
}

// Assign はプロジェクトに属性値を紐付ける。
// - プロジェクト・値が存在すること
// - 無効化（is_active=false）された値はアサイン不可（過去アサインの保護とは別。新規付与は禁止）
// - category_id は値の所属カテゴリから導出する（クライアント入力に依存しない）
// - 既に同じ値が紐付いていれば ErrConflict
func (uc *AttributeUsecase) Assign(ctx context.Context, projectID, valueID int) (*domain.AttributeAssignment, error) {
	if _, err := uc.projects.FindByID(ctx, projectID); err != nil {
		return nil, err
	}
	value, err := uc.cats.FindValueByID(ctx, valueID)
	if err != nil {
		return nil, err
	}
	if !value.IsActive {
		return nil, fmt.Errorf("%w: 無効化された属性値は紐付けできません", domain.ErrValidation)
	}
	exists, err := uc.attrs.Exists(ctx, projectID, valueID)
	if err != nil {
		return nil, fmt.Errorf("usecase.Attribute.Assign exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("%w: この属性値は既に紐付けられています", domain.ErrConflict)
	}
	a := &domain.AttributeAssignment{
		ProjectID:  projectID,
		CategoryID: value.CategoryID,
		ValueID:    valueID,
	}
	if err := uc.attrs.Assign(ctx, a); err != nil {
		return nil, fmt.Errorf("usecase.Attribute.Assign: %w", err)
	}
	return a, nil
}

// Unassign はプロジェクトから属性値の紐付けを解除する。対象が無ければ ErrNotFound。
func (uc *AttributeUsecase) Unassign(ctx context.Context, projectID, valueID int) error {
	if _, err := uc.projects.FindByID(ctx, projectID); err != nil {
		return err
	}
	return uc.attrs.Unassign(ctx, projectID, valueID)
}
