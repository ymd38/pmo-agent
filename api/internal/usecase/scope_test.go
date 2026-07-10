package usecase

import (
	"context"
	"testing"

	"pmo-agent/api/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- fakeProjectRepo のスコープ解決用メソッド（本体は program_project_test.go） ---

func (f *fakeProjectRepo) ListByIDs(_ context.Context, ids []int) ([]domain.Project, error) {
	want := map[int]struct{}{}
	for _, id := range ids {
		want[id] = struct{}{}
	}
	var out []domain.Project
	for id, p := range f.byID {
		if _, ok := want[id]; ok {
			cp := *p
			out = append(out, cp)
		}
	}
	return out, nil
}

func (f *fakeProjectRepo) IDsByPM(_ context.Context, userID int) ([]int, error) {
	var ids []int
	for id, p := range f.byID {
		if p.PMID != nil && *p.PMID == userID {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (f *fakeProjectRepo) IDsByCreator(_ context.Context, userID int) ([]int, error) {
	var ids []int
	for id, p := range f.byID {
		if p.CreatedBy == userID {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// --- fakeMemberRepo ---

type fakeMemberRepo struct {
	byKey map[[2]int]*domain.ProjectMember // key=(projectID,userID)
	seq   int
}

func newFakeMemberRepo() *fakeMemberRepo {
	return &fakeMemberRepo{byKey: map[[2]int]*domain.ProjectMember{}}
}

func (f *fakeMemberRepo) ListByProject(_ context.Context, projectID int) ([]domain.ProjectMember, error) {
	var out []domain.ProjectMember
	for k, m := range f.byKey {
		if k[0] == projectID {
			out = append(out, *m)
		}
	}
	return out, nil
}

func (f *fakeMemberRepo) Assign(_ context.Context, m *domain.ProjectMember) error {
	key := [2]int{m.ProjectID, m.UserID}
	if _, ok := f.byKey[key]; ok {
		return domain.ErrConflict
	}
	f.seq++
	m.ID = f.seq
	f.byKey[key] = m
	return nil
}

func (f *fakeMemberRepo) Update(_ context.Context, m *domain.ProjectMember) error {
	key := [2]int{m.ProjectID, m.UserID}
	if _, ok := f.byKey[key]; !ok {
		return domain.ErrNotFound
	}
	f.byKey[key] = m
	return nil
}

func (f *fakeMemberRepo) Unassign(_ context.Context, projectID, userID int) error {
	key := [2]int{projectID, userID}
	if _, ok := f.byKey[key]; !ok {
		return domain.ErrNotFound
	}
	delete(f.byKey, key)
	return nil
}

func (f *fakeMemberRepo) ProjectIDsByUser(_ context.Context, userID int) ([]int, error) {
	var ids []int
	for k := range f.byKey {
		if k[1] == userID {
			ids = append(ids, k[0])
		}
	}
	return ids, nil
}

// --- テスト ---

func intp(v int) *int { return &v }

func TestScopeUsecase_ResolveProjectScope(t *testing.T) {
	ctx := context.Background()

	// 共通のプロジェクト群を用意する。
	//   p1: pm_id=10        p2: created_by=20   p3: 他人のPJ   p4: pm_id=99（別PM）
	setup := func() (*fakeUserRepo, *fakeProjectRepo, *fakeMemberRepo) {
		users := newFakeUserRepo()
		projects := newFakeProjectRepo()
		members := newFakeMemberRepo()

		programs := newFakeProgramRepo()
		require.NoError(t, programs.Create(ctx, &domain.Program{Code: "INV-2026-0001", Name: "P"}))
		puc := NewProjectUsecase(projects, programs)
		mustPJ := func(in CreateProjectInput) *domain.Project {
			p, err := puc.Create(ctx, 1, in)
			require.NoError(t, err)
			return p
		}
		_ = mustPJ(CreateProjectInput{Name: "PJ1", PMID: intp(10)})               // id=1
		_ = mustPJ(CreateProjectInput{Name: "PJ2", CreatedBy: 20})                // id=2
		_ = mustPJ(CreateProjectInput{Name: "PJ3"})                               // id=3
		_ = mustPJ(CreateProjectInput{Name: "PJ4", PMID: intp(99), CreatedBy: 5}) // id=4
		return users, projects, members
	}

	t.Run("pmo_admin は全件（All=true）", func(t *testing.T) {
		users, projects, members := setup()
		users.add(&domain.User{ID: 1, Email: "a@x", IsActive: true})
		users.roles[1] = []domain.Role{{Code: domain.RoleCodePMOAdmin}}
		uc := NewScopeUsecase(users, projects, members)

		scope, err := uc.ResolveProjectScope(ctx, 1)
		require.NoError(t, err)
		assert.True(t, scope.All)
	})

	t.Run("executive も全件", func(t *testing.T) {
		users, projects, members := setup()
		users.roles[1] = []domain.Role{{Code: domain.RoleCodeExecutive}}
		uc := NewScopeUsecase(users, projects, members)

		scope, err := uc.ResolveProjectScope(ctx, 1)
		require.NoError(t, err)
		assert.True(t, scope.All)
	})

	t.Run("pm は担当PJ（members ∪ pm_id）", func(t *testing.T) {
		users, projects, members := setup()
		userID := 10
		users.roles[userID] = []domain.Role{{Code: domain.RoleCodePM}}
		// user10 は p1 の PM、かつ p3 にアサイン済み。
		require.NoError(t, members.Assign(ctx, &domain.ProjectMember{ProjectID: 3, UserID: userID}))
		uc := NewScopeUsecase(users, projects, members)

		scope, err := uc.ResolveProjectScope(ctx, userID)
		require.NoError(t, err)
		assert.False(t, scope.All)
		assert.ElementsMatch(t, []int{1, 3}, scope.ProjectIDs)
	})

	t.Run("member はアサインPJのみ（pm_idは無関係）", func(t *testing.T) {
		users, projects, members := setup()
		userID := 30
		users.roles[userID] = []domain.Role{{Code: domain.RoleCodeMember}}
		require.NoError(t, members.Assign(ctx, &domain.ProjectMember{ProjectID: 2, UserID: userID}))
		uc := NewScopeUsecase(users, projects, members)

		scope, err := uc.ResolveProjectScope(ctx, userID)
		require.NoError(t, err)
		assert.ElementsMatch(t, []int{2}, scope.ProjectIDs)
	})

	t.Run("planner は自起案PJ（created_by）", func(t *testing.T) {
		users, projects, members := setup()
		userID := 20
		users.roles[userID] = []domain.Role{{Code: domain.RoleCodePlanner}}
		uc := NewScopeUsecase(users, projects, members)

		scope, err := uc.ResolveProjectScope(ctx, userID)
		require.NoError(t, err)
		assert.ElementsMatch(t, []int{2}, scope.ProjectIDs)
	})

	t.Run("複数ロールは和集合", func(t *testing.T) {
		users, projects, members := setup()
		userID := 5
		// user5 は pm（p4 の PM）と planner（p4 の起案者）を兼務。重複排除される。
		users.roles[userID] = []domain.Role{{Code: domain.RoleCodePM}, {Code: domain.RoleCodePlanner}}
		require.NoError(t, members.Assign(ctx, &domain.ProjectMember{ProjectID: 1, UserID: userID}))
		uc := NewScopeUsecase(users, projects, members)

		scope, err := uc.ResolveProjectScope(ctx, userID)
		require.NoError(t, err)
		// pm: p4(pm_id=99 は該当せず) + member(p1) / planner: p4(created_by=5)
		assert.ElementsMatch(t, []int{1, 4}, scope.ProjectIDs)
	})

	t.Run("ロール無しは空スコープ（全遮断）", func(t *testing.T) {
		users, projects, members := setup()
		users.roles[7] = nil
		uc := NewScopeUsecase(users, projects, members)

		scope, err := uc.ResolveProjectScope(ctx, 7)
		require.NoError(t, err)
		assert.False(t, scope.All)
		assert.Empty(t, scope.ProjectIDs)
	})
}

func TestProjectUsecase_List_Get_Scope(t *testing.T) {
	ctx := context.Background()
	programs := newFakeProgramRepo()
	projects := newFakeProjectRepo()
	require.NoError(t, programs.Create(ctx, &domain.Program{Code: "INV-2026-0001", Name: "P"}))
	puc := NewProjectUsecase(projects, programs)
	for i := 1; i <= 3; i++ {
		_, err := puc.Create(ctx, 1, CreateProjectInput{Name: "PJ"})
		require.NoError(t, err)
	}

	t.Run("All スコープは全件", func(t *testing.T) {
		got, err := puc.List(ctx, domain.UnrestrictedScope())
		require.NoError(t, err)
		assert.Len(t, got, 3)
	})

	t.Run("限定スコープは許可IDのみ", func(t *testing.T) {
		got, err := puc.List(ctx, domain.ProjectScope{ProjectIDs: []int{2}})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, 2, got[0].ID)
	})

	t.Run("空スコープは0件", func(t *testing.T) {
		got, err := puc.List(ctx, domain.ProjectScope{})
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("担当外PJのGetは存在秘匿で ErrNotFound", func(t *testing.T) {
		_, err := puc.Get(ctx, 3, domain.ProjectScope{ProjectIDs: []int{1, 2}})
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("担当PJのGetは成功", func(t *testing.T) {
		p, err := puc.Get(ctx, 1, domain.ProjectScope{ProjectIDs: []int{1}})
		require.NoError(t, err)
		assert.Equal(t, 1, p.ID)
	})
}
