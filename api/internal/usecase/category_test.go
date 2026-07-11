package usecase

import (
	"context"
	"errors"
	"testing"

	"pmo-agent/api/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubCategoryRepo は値の所属検証テスト用フェイク。
// FindValueByID をサポートし、各ミューテーションが実際に呼ばれたか記録する。
type stubCategoryRepo struct {
	values          map[int]*domain.CategoryValue
	categories      map[int]*domain.Category
	deactivatedID   int
	reactivatedID   int
	updatedID       int
	updatedCategory *domain.Category
}

func newStubCategoryRepo() *stubCategoryRepo {
	return &stubCategoryRepo{values: map[int]*domain.CategoryValue{}, categories: map[int]*domain.Category{}}
}

func (s *stubCategoryRepo) add(v *domain.CategoryValue)    { s.values[v.ID] = v }
func (s *stubCategoryRepo) addCategory(c *domain.Category) { s.categories[c.ID] = c }

func (s *stubCategoryRepo) FindCategoryByID(_ context.Context, id int) (*domain.Category, error) {
	if c, ok := s.categories[id]; ok {
		cp := *c
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (s *stubCategoryRepo) FindValueByID(_ context.Context, id int) (*domain.CategoryValue, error) {
	if v, ok := s.values[id]; ok {
		cp := *v
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (s *stubCategoryRepo) UpdateValue(_ context.Context, v *domain.CategoryValue) error {
	s.updatedID = v.ID
	return nil
}

func (s *stubCategoryRepo) DeactivateValue(_ context.Context, id int) error {
	s.deactivatedID = id
	return nil
}

func (s *stubCategoryRepo) ReactivateValue(_ context.Context, id int) error {
	s.reactivatedID = id
	return nil
}

// 以下はインターフェース充足のためのスタブ（本テストでは未使用）。
func (s *stubCategoryRepo) ListCategories(_ context.Context, _ bool) ([]domain.Category, error) {
	return nil, nil
}
func (s *stubCategoryRepo) CreateCategory(_ context.Context, _ *domain.Category) error { return nil }
func (s *stubCategoryRepo) UpdateCategory(_ context.Context, c *domain.Category) error {
	s.updatedCategory = c
	return nil
}
func (s *stubCategoryRepo) DeactivateCategory(_ context.Context, _ int) error { return nil }
func (s *stubCategoryRepo) ReactivateCategory(_ context.Context, _ int) error { return nil }
func (s *stubCategoryRepo) CreateValue(_ context.Context, _ *domain.CategoryValue) error {
	return nil
}
func (s *stubCategoryRepo) ListValues(_ context.Context, _ int, _ bool) ([]domain.CategoryValue, error) {
	return nil, nil
}

// 既存値を対象とする各ミューテーションが、URL の categoryID と値の所属を検証することを確認する。
// 所属一致→成功 / 所属不一致→404 / 存在しない valueId→404 を全ミューテーションに適用。
func TestCategoryUsecase_ValueMembership(t *testing.T) {
	const (
		catA    = 10 // 値が所属するカテゴリ
		catB    = 999
		valID   = 7
		missing = 42
	)

	tests := []struct {
		name       string
		categoryID int
		valueID    int
		wantErr    bool
	}{
		{name: "所属一致で成功", categoryID: catA, valueID: valID, wantErr: false},
		{name: "別カテゴリ所属は404", categoryID: catB, valueID: valID, wantErr: true},
		{name: "存在しない値は404", categoryID: catA, valueID: missing, wantErr: true},
	}

	newValue := func() *domain.CategoryValue {
		return &domain.CategoryValue{ID: valID, CategoryID: catA, Code: "x", Label: "元ラベル", IsActive: true}
	}

	t.Run("UpdateValue", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := newStubCategoryRepo()
				repo.add(newValue())
				uc := NewCategoryUsecase(repo)

				v, err := uc.UpdateValue(context.Background(), tt.categoryID, tt.valueID, ValueInput{Label: "新ラベル"})
				if tt.wantErr {
					require.Error(t, err)
					assert.True(t, errors.Is(err, domain.ErrNotFound))
					assert.Zero(t, repo.updatedID, "所属不一致では更新してはならない")
					return
				}
				require.NoError(t, err)
				require.NotNil(t, v)
				assert.Equal(t, "新ラベル", v.Label)
				assert.Equal(t, valID, repo.updatedID)
			})
		}
	})

	t.Run("DeactivateValue", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := newStubCategoryRepo()
				repo.add(newValue())
				uc := NewCategoryUsecase(repo)

				err := uc.DeactivateValue(context.Background(), tt.categoryID, tt.valueID)
				if tt.wantErr {
					require.Error(t, err)
					assert.True(t, errors.Is(err, domain.ErrNotFound))
					assert.Zero(t, repo.deactivatedID, "所属不一致では無効化してはならない")
					return
				}
				require.NoError(t, err)
				assert.Equal(t, valID, repo.deactivatedID)
			})
		}
	})

	t.Run("ReactivateValue", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := newStubCategoryRepo()
				repo.add(newValue())
				uc := NewCategoryUsecase(repo)

				err := uc.ReactivateValue(context.Background(), tt.categoryID, tt.valueID)
				if tt.wantErr {
					require.Error(t, err)
					assert.True(t, errors.Is(err, domain.ErrNotFound))
					assert.Zero(t, repo.reactivatedID, "所属不一致では再有効化してはならない")
					return
				}
				require.NoError(t, err)
				assert.Equal(t, valID, repo.reactivatedID)
			})
		}
	})
}

// TestCategoryUsecase_UpdateCategory は UpdateCategory が FindCategoryByID 経由で
// 対象を解決し（全件ロード+線形探索をやめた）、存在すれば name 以下を更新、
// 存在しなければ ErrNotFound を返すことを確認する。code は不変。
func TestCategoryUsecase_UpdateCategory(t *testing.T) {
	t.Run("存在すれば更新しcodeは不変", func(t *testing.T) {
		repo := newStubCategoryRepo()
		repo.addCategory(&domain.Category{ID: 5, Code: "region", Name: "旧名称", SortOrder: 1})
		uc := NewCategoryUsecase(repo)

		got, err := uc.UpdateCategory(context.Background(), 5, CategoryInput{
			Code: "IGNORED", Name: " 新名称 ", Description: "説明", IsRequired: true, SortOrder: 9,
		})

		require.NoError(t, err)
		assert.Equal(t, "新名称", got.Name)
		assert.Equal(t, "region", got.Code, "code は不変")
		assert.True(t, got.IsRequired)
		assert.Equal(t, 9, got.SortOrder)
		require.NotNil(t, repo.updatedCategory)
		assert.Equal(t, 5, repo.updatedCategory.ID)
	})

	t.Run("存在しなければ404", func(t *testing.T) {
		repo := newStubCategoryRepo()
		uc := NewCategoryUsecase(repo)

		_, err := uc.UpdateCategory(context.Background(), 404, CategoryInput{Name: "x"})

		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("名称が空はバリデーションエラー", func(t *testing.T) {
		repo := newStubCategoryRepo()
		repo.addCategory(&domain.Category{ID: 5, Code: "region", Name: "旧名称"})
		uc := NewCategoryUsecase(repo)

		_, err := uc.UpdateCategory(context.Background(), 5, CategoryInput{Name: "  "})

		assert.ErrorIs(t, err, domain.ErrValidation)
	})
}
