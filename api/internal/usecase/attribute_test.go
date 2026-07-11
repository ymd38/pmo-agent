package usecase

import (
	"context"
	"testing"

	"pmo-agent/api/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- フェイク ---

// fakeCategoryRepo は属性アサインのテストに必要な値参照のみ実装する。
type fakeCategoryRepo struct {
	values map[int]*domain.CategoryValue
}

func newFakeCategoryRepo() *fakeCategoryRepo {
	return &fakeCategoryRepo{values: map[int]*domain.CategoryValue{}}
}

func (f *fakeCategoryRepo) FindValueByID(_ context.Context, id int) (*domain.CategoryValue, error) {
	if v, ok := f.values[id]; ok {
		cp := *v
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

// 以下はインターフェース充足のためのスタブ（本テストでは未使用）。
func (f *fakeCategoryRepo) ListCategories(_ context.Context, _ bool) ([]domain.Category, error) {
	return nil, nil
}
func (f *fakeCategoryRepo) FindCategoryByID(_ context.Context, _ int) (*domain.Category, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeCategoryRepo) CreateCategory(_ context.Context, _ *domain.Category) error   { return nil }
func (f *fakeCategoryRepo) UpdateCategory(_ context.Context, _ *domain.Category) error   { return nil }
func (f *fakeCategoryRepo) DeactivateCategory(_ context.Context, _ int) error            { return nil }
func (f *fakeCategoryRepo) ReactivateCategory(_ context.Context, _ int) error            { return nil }
func (f *fakeCategoryRepo) CreateValue(_ context.Context, _ *domain.CategoryValue) error { return nil }
func (f *fakeCategoryRepo) UpdateValue(_ context.Context, _ *domain.CategoryValue) error { return nil }
func (f *fakeCategoryRepo) DeactivateValue(_ context.Context, _ int) error               { return nil }
func (f *fakeCategoryRepo) ReactivateValue(_ context.Context, _ int) error               { return nil }
func (f *fakeCategoryRepo) ListValues(_ context.Context, _ int, _ bool) ([]domain.CategoryValue, error) {
	return nil, nil
}

type fakeAttributeRepo struct {
	// key: project_id*100000 + value_id
	assigned map[int]*domain.AttributeAssignment
	seq      int
}

func newFakeAttributeRepo() *fakeAttributeRepo {
	return &fakeAttributeRepo{assigned: map[int]*domain.AttributeAssignment{}}
}

func key(projectID, valueID int) int { return projectID*100000 + valueID }

func (f *fakeAttributeRepo) ListByProject(_ context.Context, projectID int) ([]domain.ProjectAttribute, error) {
	var out []domain.ProjectAttribute
	for _, a := range f.assigned {
		if a.ProjectID == projectID {
			out = append(out, domain.ProjectAttribute{ID: a.ID, CategoryID: a.CategoryID, ValueID: a.ValueID})
		}
	}
	return out, nil
}

func (f *fakeAttributeRepo) Exists(_ context.Context, projectID, valueID int) (bool, error) {
	_, ok := f.assigned[key(projectID, valueID)]
	return ok, nil
}

func (f *fakeAttributeRepo) Assign(_ context.Context, a *domain.AttributeAssignment) error {
	f.seq++
	a.ID = f.seq
	f.assigned[key(a.ProjectID, a.ValueID)] = a
	return nil
}

func (f *fakeAttributeRepo) Unassign(_ context.Context, projectID, valueID int) error {
	k := key(projectID, valueID)
	if _, ok := f.assigned[k]; !ok {
		return domain.ErrNotFound
	}
	delete(f.assigned, k)
	return nil
}

// --- テスト ---

func setupAttributeUsecase(t *testing.T) (*AttributeUsecase, *fakeAttributeRepo, int) {
	t.Helper()
	ctx := context.Background()
	programs := newFakeProgramRepo()
	projects := newFakeProjectRepo()
	cats := newFakeCategoryRepo()
	attrs := newFakeAttributeRepo()

	prog := &domain.Program{Code: "INV-2026-0001", Name: "P"}
	require.NoError(t, programs.Create(ctx, prog))
	puc := NewProjectUsecase(projects, programs)
	proj, err := puc.Create(ctx, prog.ID, CreateProjectInput{Name: "PJ"})
	require.NoError(t, err)

	// カテゴリ1の有効な値10、無効な値11。
	cats.values[10] = &domain.CategoryValue{ID: 10, CategoryID: 1, Label: "申込", IsActive: true}
	cats.values[11] = &domain.CategoryValue{ID: 11, CategoryID: 1, Label: "旧区分", IsActive: false}

	return NewAttributeUsecase(attrs, projects, cats), attrs, proj.ID
}

func TestAttributeUsecase_Assign(t *testing.T) {
	ctx := context.Background()

	t.Run("値を紐付けると category_id が値から導出される", func(t *testing.T) {
		uc, _, pid := setupAttributeUsecase(t)
		a, err := uc.Assign(ctx, pid, 10)
		require.NoError(t, err)
		assert.Equal(t, 1, a.CategoryID)
		assert.Equal(t, 10, a.ValueID)
	})

	t.Run("無効化された値はアサイン不可（ErrValidation）", func(t *testing.T) {
		uc, _, pid := setupAttributeUsecase(t)
		_, err := uc.Assign(ctx, pid, 11)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("重複アサインは ErrConflict", func(t *testing.T) {
		uc, _, pid := setupAttributeUsecase(t)
		_, err := uc.Assign(ctx, pid, 10)
		require.NoError(t, err)
		_, err = uc.Assign(ctx, pid, 10)
		assert.ErrorIs(t, err, domain.ErrConflict)
	})

	t.Run("存在しないプロジェクトは ErrNotFound", func(t *testing.T) {
		uc, _, _ := setupAttributeUsecase(t)
		_, err := uc.Assign(ctx, 999, 10)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("存在しない値は ErrNotFound", func(t *testing.T) {
		uc, _, pid := setupAttributeUsecase(t)
		_, err := uc.Assign(ctx, pid, 999)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestAttributeUsecase_Unassign(t *testing.T) {
	ctx := context.Background()

	t.Run("紐付け済みなら解除できる", func(t *testing.T) {
		uc, attrs, pid := setupAttributeUsecase(t)
		_, err := uc.Assign(ctx, pid, 10)
		require.NoError(t, err)
		require.NoError(t, uc.Unassign(ctx, pid, 10))
		ok, _ := attrs.Exists(ctx, pid, 10)
		assert.False(t, ok)
	})

	t.Run("未紐付けの解除は ErrNotFound", func(t *testing.T) {
		uc, _, pid := setupAttributeUsecase(t)
		assert.ErrorIs(t, uc.Unassign(ctx, pid, 10), domain.ErrNotFound)
	})
}

func TestAttributeUsecase_List(t *testing.T) {
	ctx := context.Background()
	uc, _, pid := setupAttributeUsecase(t)
	_, err := uc.Assign(ctx, pid, 10)
	require.NoError(t, err)

	got, err := uc.List(ctx, pid)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, 10, got[0].ValueID)

	t.Run("存在しないプロジェクトは ErrNotFound", func(t *testing.T) {
		_, err := uc.List(ctx, 999)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}
