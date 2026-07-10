package usecase

import (
	"context"
	"testing"

	"pmo-agent/api/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMemberUC(t *testing.T) (*MemberUsecase, *fakeUserRepo, *fakeMemberRepo) {
	t.Helper()
	ctx := context.Background()
	users := newFakeUserRepo()
	projects := newFakeProjectRepo()
	members := newFakeMemberRepo()
	programs := newFakeProgramRepo()
	require.NoError(t, programs.Create(ctx, &domain.Program{Code: "INV-2026-0001", Name: "P"}))
	puc := NewProjectUsecase(projects, programs)
	// 担当PJ(id=1)と担当外PJ(id=2)を用意。
	_, err := puc.Create(ctx, 1, CreateProjectInput{Name: "担当PJ"})
	require.NoError(t, err)
	_, err = puc.Create(ctx, 1, CreateProjectInput{Name: "担当外PJ"})
	require.NoError(t, err)
	return NewMemberUsecase(members, projects, users), users, members
}

// 担当PJ(1)だけを許可するスコープ。
var scopePJ1 = domain.ProjectScope{ProjectIDs: []int{1}}

func TestMemberUsecase_Assign(t *testing.T) {
	ctx := context.Background()

	t.Run("担当外PJへのアサインは存在秘匿で ErrNotFound", func(t *testing.T) {
		uc, users, _ := setupMemberUC(t)
		users.add(&domain.User{ID: 50, Email: "u@x", IsActive: true})
		_, err := uc.Assign(ctx, 2, AssignMemberInput{UserID: 50}, scopePJ1)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("担当PJへは有効ユーザーをアサインできる", func(t *testing.T) {
		uc, users, members := setupMemberUC(t)
		users.add(&domain.User{ID: 50, Email: "u@x", IsActive: true})
		m, err := uc.Assign(ctx, 1, AssignMemberInput{UserID: 50}, scopePJ1)
		require.NoError(t, err)
		assert.Equal(t, 1, m.ProjectID)
		got, _ := members.ProjectIDsByUser(ctx, 50)
		assert.ElementsMatch(t, []int{1}, got)
	})

	t.Run("無効化ユーザーのアサインは ErrValidation", func(t *testing.T) {
		uc, users, _ := setupMemberUC(t)
		users.add(&domain.User{ID: 51, Email: "d@x", IsActive: false})
		_, err := uc.Assign(ctx, 1, AssignMemberInput{UserID: 51}, scopePJ1)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("工数割合が範囲外は ErrValidation", func(t *testing.T) {
		uc, users, _ := setupMemberUC(t)
		users.add(&domain.User{ID: 52, Email: "p@x", IsActive: true})
		over := 150.0
		_, err := uc.Assign(ctx, 1, AssignMemberInput{UserID: 52, AllocationPercent: &over}, scopePJ1)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("全件スコープ(admin)はどのPJにもアサイン可", func(t *testing.T) {
		uc, users, _ := setupMemberUC(t)
		users.add(&domain.User{ID: 53, Email: "a@x", IsActive: true})
		_, err := uc.Assign(ctx, 2, AssignMemberInput{UserID: 53}, domain.UnrestrictedScope())
		require.NoError(t, err)
	})
}

func TestMemberUsecase_List_Unassign_Scope(t *testing.T) {
	ctx := context.Background()

	t.Run("担当外PJの一覧は ErrNotFound", func(t *testing.T) {
		uc, _, _ := setupMemberUC(t)
		_, err := uc.List(ctx, 2, scopePJ1)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("担当外PJの解除は ErrNotFound", func(t *testing.T) {
		uc, _, _ := setupMemberUC(t)
		err := uc.Unassign(ctx, 2, 50, scopePJ1)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("担当PJのメンバー解除は成功", func(t *testing.T) {
		uc, users, members := setupMemberUC(t)
		users.add(&domain.User{ID: 60, Email: "m@x", IsActive: true})
		require.NoError(t, members.Assign(ctx, &domain.ProjectMember{ProjectID: 1, UserID: 60}))
		require.NoError(t, uc.Unassign(ctx, 1, 60, scopePJ1))
		got, _ := members.ProjectIDsByUser(ctx, 60)
		assert.Empty(t, got)
	})
}
