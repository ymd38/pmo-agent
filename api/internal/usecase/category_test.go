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
	values        map[int]*domain.CategoryValue
	deactivatedID int
	reactivatedID int
	updatedID     int
}

func newStubCategoryRepo() *stubCategoryRepo {
	return &stubCategoryRepo{values: map[int]*domain.CategoryValue{}}
}

func (s *stubCategoryRepo) add(v *domain.CategoryValue) { s.values[v.ID] = v }

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
func (s *stubCategoryRepo) UpdateCategory(_ context.Context, _ *domain.Category) error { return nil }
func (s *stubCategoryRepo) DeactivateCategory(_ context.Context, _ int) error          { return nil }
func (s *stubCategoryRepo) ReactivateCategory(_ context.Context, _ int) error          { return nil }
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
