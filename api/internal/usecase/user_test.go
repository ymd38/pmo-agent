package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"pmo-agent/api/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUserUC(users *fakeUserRepo, roles *fakeRoleRepo, set *fakeSetTokenRepo) *UserUsecase {
	return NewUserUsecase(users, roles, set, newFakeRefreshRepo(), &fakeTokens{}, 72*time.Hour, "http://localhost:3000")
}

// newUserUCWithRef はリフレッシュトークン失効の検証用にフェイクを外から差し込む。
func newUserUCWithRef(users *fakeUserRepo, roles *fakeRoleRepo, set *fakeSetTokenRepo, ref *fakeRefreshRepo) *UserUsecase {
	return NewUserUsecase(users, roles, set, ref, &fakeTokens{}, 72*time.Hour, "http://localhost:3000")
}

func TestUserUsecase_Create(t *testing.T) {
	t.Run("作成成功で未アクティベート＋招待リンク発行", func(t *testing.T) {
		users := newFakeUserRepo()
		set := newFakeSetTokenRepo()
		uc := newUserUC(users, &fakeRoleRepo{existing: map[int]bool{1: true, 2: true}}, set)

		user, link, err := uc.Create(context.Background(), CreateInput{
			Email: "new@example.com", Name: "新人", Grade: domain.GradeStaff, RoleIDs: []int{1, 2},
		})
		require.NoError(t, err)
		assert.Nil(t, user.PasswordHash, "未アクティベート(password_hash=nil)")
		assert.True(t, user.IsActive)
		assert.True(t, strings.HasPrefix(link, "http://localhost:3000/set-password?token="), "招待リンクが返る: %s", link)
		assert.Len(t, set.byHash, 1, "設定トークンが1件発行される")
	})

	tests := []struct {
		name  string
		in    CreateInput
		roles map[int]bool
		dup   bool
		want  error
	}{
		{"メール不正", CreateInput{Email: "bad", Name: "N", Grade: domain.GradeStaff, RoleIDs: []int{1}}, map[int]bool{1: true}, false, domain.ErrValidation},
		{"氏名空", CreateInput{Email: "a@x.jp", Name: " ", Grade: domain.GradeStaff, RoleIDs: []int{1}}, map[int]bool{1: true}, false, domain.ErrValidation},
		{"グレード不正", CreateInput{Email: "a@x.jp", Name: "N", Grade: "boss", RoleIDs: []int{1}}, map[int]bool{1: true}, false, domain.ErrValidation},
		{"ロール未指定", CreateInput{Email: "a@x.jp", Name: "N", Grade: domain.GradeStaff, RoleIDs: []int{}}, map[int]bool{1: true}, false, domain.ErrValidation},
		{"存在しないロール", CreateInput{Email: "a@x.jp", Name: "N", Grade: domain.GradeStaff, RoleIDs: []int{99}}, map[int]bool{1: true}, false, domain.ErrValidation},
		{"メール重複", CreateInput{Email: "dup@x.jp", Name: "N", Grade: domain.GradeStaff, RoleIDs: []int{1}}, map[int]bool{1: true}, true, domain.ErrConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := newFakeUserRepo()
			if tt.dup {
				users.add(&domain.User{ID: 1, Email: tt.in.Email})
			}
			uc := newUserUC(users, &fakeRoleRepo{existing: tt.roles}, newFakeSetTokenRepo())
			_, _, err := uc.Create(context.Background(), tt.in)
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

func boolPtr(b bool) *bool { return &b }

func TestUserUsecase_Update_IsActive(t *testing.T) {
	newUC := func() (*UserUsecase, *fakeUserRepo, *fakeRefreshRepo) {
		users := newFakeUserRepo()
		users.add(&domain.User{ID: 5, Email: "u@x.jp", Name: "旧名", Grade: domain.GradeStaff, IsActive: true})
		ref := newFakeRefreshRepo()
		uc := newUserUCWithRef(users, &fakeRoleRepo{existing: map[int]bool{1: true}}, newFakeSetTokenRepo(), ref)
		return uc, users, ref
	}

	t.Run("is_active 未指定なら現在値を維持し失効もしない", func(t *testing.T) {
		uc, users, ref := newUC()
		got, err := uc.Update(context.Background(), 5, UpdateInput{
			Name: "新名", Grade: domain.GradeStaff, IsActive: nil, RoleIDs: []int{1},
		})
		require.NoError(t, err)
		assert.True(t, got.IsActive, "省略時に無効化されない")
		assert.Equal(t, "新名", users.byID[5].Name)
		assert.False(t, ref.revokedUser[5], "有効なままなので失効しない")
	})

	t.Run("is_active=false を明示するとリフレッシュトークンも失効させる", func(t *testing.T) {
		uc, _, ref := newUC()
		got, err := uc.Update(context.Background(), 5, UpdateInput{
			Name: "新名", Grade: domain.GradeStaff, IsActive: boolPtr(false), RoleIDs: []int{1},
		})
		require.NoError(t, err)
		assert.False(t, got.IsActive)
		assert.True(t, ref.revokedUser[5], "無効化時にセッションを断つ")
	})
}

func TestUserUsecase_Deactivate_RevokesTokens(t *testing.T) {
	users := newFakeUserRepo()
	users.add(&domain.User{ID: 9, Email: "d@x.jp", IsActive: true})
	ref := newFakeRefreshRepo()
	uc := newUserUCWithRef(users, &fakeRoleRepo{}, newFakeSetTokenRepo(), ref)

	require.NoError(t, uc.Deactivate(context.Background(), 9))
	assert.False(t, users.byID[9].IsActive, "論理削除される")
	assert.True(t, ref.revokedUser[9], "無効化にあわせてリフレッシュトークンを失効させる")

	assert.ErrorIs(t, uc.Deactivate(context.Background(), 999), domain.ErrNotFound)
}

func TestUserUsecase_ReissueLink(t *testing.T) {
	users := newFakeUserRepo()
	users.add(&domain.User{ID: 7, Email: "u@x.jp", IsActive: true})
	set := newFakeSetTokenRepo()
	uc := newUserUC(users, &fakeRoleRepo{}, set)

	link, err := uc.ReissueLink(context.Background(), 7)
	require.NoError(t, err)
	assert.Contains(t, link, "/set-password?token=")
	assert.True(t, set.invalidate[7], "再発行時に既存トークンを失効させる")

	_, err = uc.ReissueLink(context.Background(), 999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
