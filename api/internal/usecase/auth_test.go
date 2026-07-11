package usecase

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"pmo-agent/api/internal/domain"
	"pmo-agent/api/internal/infra"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPassword = "password123"

func newAuthUC(users *fakeUserRepo, set *fakeSetTokenRepo, ref *fakeRefreshRepo) *AuthUsecase {
	return NewAuthUsecase(
		users, set, ref,
		infra.NewPasswordHasher(), fakeJWT{}, &fakeTokens{},
		time.Hour, 72*time.Hour, "http://localhost:3000",
	)
}

func mustHash(t *testing.T, plain string) *string {
	t.Helper()
	h, err := infra.NewPasswordHasher().Hash(plain)
	require.NoError(t, err)
	return &h
}

func TestAuthUsecase_Login(t *testing.T) {
	emptyHash := ""
	tests := []struct {
		name    string
		user    *domain.User
		email   string
		pass    string
		wantErr error
	}{
		{
			name:  "正しい資格情報でログイン成功",
			user:  &domain.User{ID: 1, Email: "u@x.jp", PasswordHash: mustHash(t, testPassword), IsActive: true},
			email: "u@x.jp", pass: testPassword, wantErr: nil,
		},
		{
			name:  "存在しないユーザーは ErrInvalidCredentials",
			user:  nil,
			email: "none@x.jp", pass: testPassword, wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:  "未アクティベート(password_hash=nil)は拒否",
			user:  &domain.User{ID: 1, Email: "u@x.jp", PasswordHash: nil, IsActive: true},
			email: "u@x.jp", pass: testPassword, wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:  "空パスワードハッシュは拒否",
			user:  &domain.User{ID: 1, Email: "u@x.jp", PasswordHash: &emptyHash, IsActive: true},
			email: "u@x.jp", pass: testPassword, wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:  "無効化ユーザー(is_active=false)は拒否",
			user:  &domain.User{ID: 1, Email: "u@x.jp", PasswordHash: mustHash(t, testPassword), IsActive: false},
			email: "u@x.jp", pass: testPassword, wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:  "パスワード不一致は拒否",
			user:  &domain.User{ID: 1, Email: "u@x.jp", PasswordHash: mustHash(t, testPassword), IsActive: true},
			email: "u@x.jp", pass: "wrong-password", wantErr: domain.ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := newFakeUserRepo()
			if tt.user != nil {
				users.add(tt.user)
			}
			ref := newFakeRefreshRepo()
			uc := newAuthUC(users, newFakeSetTokenRepo(), ref)

			_, toks, err := uc.Login(context.Background(), tt.email, tt.pass)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, toks.Access)
			assert.NotEmpty(t, toks.Refresh)
			assert.Len(t, ref.byHash, 1, "リフレッシュトークンが保存される")
		})
	}
}

func TestAuthUsecase_SetPassword(t *testing.T) {
	setup := func() (*AuthUsecase, *fakeUserRepo, *fakeSetTokenRepo, *fakeRefreshRepo) {
		users := newFakeUserRepo()
		users.add(&domain.User{ID: 1, Email: "u@x.jp", IsActive: true})
		set := newFakeSetTokenRepo()
		ref := newFakeRefreshRepo()
		return newAuthUC(users, set, ref), users, set, ref
	}
	validToken := func(set *fakeSetTokenRepo) {
		set.byHash["h:tok"] = &domain.PasswordSetToken{ID: 9, UserID: 1, TokenHash: "h:tok", ExpiresAt: time.Now().Add(time.Hour)}
	}

	t.Run("有効トークンでパスワード設定成功", func(t *testing.T) {
		uc, users, set, ref := setup()
		validToken(set)
		err := uc.SetPassword(context.Background(), "tok", "newpassword")
		require.NoError(t, err)
		assert.NotEmpty(t, users.hashSet[1], "パスワードハッシュが設定される")
		assert.True(t, set.used[9], "トークンが使用済みになる")
		assert.True(t, ref.revokedUser[1], "既存セッションが失効する")
	})

	t.Run("8文字未満は ErrValidation", func(t *testing.T) {
		uc, _, set, _ := setup()
		validToken(set)
		err := uc.SetPassword(context.Background(), "tok", "short")
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("不正トークンは ErrTokenInvalid", func(t *testing.T) {
		uc, _, _, _ := setup()
		err := uc.SetPassword(context.Background(), "nope", "newpassword")
		assert.ErrorIs(t, err, domain.ErrTokenInvalid)
	})

	t.Run("期限切れトークンは ErrTokenInvalid", func(t *testing.T) {
		uc, _, set, _ := setup()
		set.byHash["h:expired"] = &domain.PasswordSetToken{ID: 1, UserID: 1, TokenHash: "h:expired", ExpiresAt: time.Now().Add(-time.Hour)}
		err := uc.SetPassword(context.Background(), "expired", "newpassword")
		assert.ErrorIs(t, err, domain.ErrTokenInvalid)
	})
}

func TestAuthUsecase_ChangePassword(t *testing.T) {
	setup := func() (*AuthUsecase, *fakeUserRepo) {
		users := newFakeUserRepo()
		users.add(&domain.User{ID: 1, Email: "u@x.jp", PasswordHash: mustHash(t, "current-pass"), IsActive: true})
		return newAuthUC(users, newFakeSetTokenRepo(), newFakeRefreshRepo()), users
	}

	t.Run("現パスワード一致で変更成功", func(t *testing.T) {
		uc, users := setup()
		err := uc.ChangePassword(context.Background(), 1, "current-pass", "newpassword")
		require.NoError(t, err)
		assert.NotEmpty(t, users.hashSet[1])
	})

	t.Run("現パスワード不一致は ErrInvalidCredentials", func(t *testing.T) {
		uc, _ := setup()
		err := uc.ChangePassword(context.Background(), 1, "wrong", "newpassword")
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("新パスワードが短いと ErrValidation", func(t *testing.T) {
		uc, _ := setup()
		err := uc.ChangePassword(context.Background(), 1, "current-pass", "short")
		assert.ErrorIs(t, err, domain.ErrValidation)
	})
}

func TestAuthUsecase_Refresh_Rotation(t *testing.T) {
	users := newFakeUserRepo()
	users.add(&domain.User{ID: 1, Email: "u@x.jp", PasswordHash: mustHash(t, testPassword), IsActive: true})
	ref := newFakeRefreshRepo()
	ref.seed(&domain.RefreshToken{ID: 5, UserID: 1, TokenHash: "h:rt1", ExpiresAt: time.Now().Add(time.Hour)})
	uc := newAuthUC(users, newFakeSetTokenRepo(), ref)

	toks, err := uc.Refresh(context.Background(), "rt1")
	require.NoError(t, err)
	assert.NotEmpty(t, toks.Access)
	assert.NotEmpty(t, toks.Refresh, "新しいリフレッシュトークンが発行される")
	assert.True(t, ref.revoked[5], "旧リフレッシュトークンが失効する（ローテーション）")
	assert.Len(t, ref.byHash, 2, "旧＋新の2トークンが存在する")

	t.Run("不正なリフレッシュトークンは ErrTokenInvalid", func(t *testing.T) {
		_, err := uc.Refresh(context.Background(), "unknown")
		assert.True(t, errors.Is(err, domain.ErrTokenInvalid))
	})
}

// TestAuthUsecase_Refresh_Validation はトークン状態ごとの検証結果を表駆動で確認する。
func TestAuthUsecase_Refresh_Validation(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name          string
		token         *domain.RefreshToken // nil ならトークンを投入しない
		plain         string
		wantErr       error
		wantChainKill bool // RevokeAllForUser が呼ばれるべきか
	}{
		{
			name:    "未知トークンは ErrTokenInvalid",
			token:   nil,
			plain:   "unknown",
			wantErr: domain.ErrTokenInvalid,
		},
		{
			name:    "期限切れトークンは ErrTokenInvalid（チェーン失効はしない）",
			token:   &domain.RefreshToken{ID: 1, UserID: 1, TokenHash: "h:expired", ExpiresAt: now.Add(-time.Hour)},
			plain:   "expired",
			wantErr: domain.ErrTokenInvalid,
		},
		{
			name:          "失効済みトークンの再提示は ErrTokenReuse＋チェーン全失効",
			token:         &domain.RefreshToken{ID: 1, UserID: 1, TokenHash: "h:revoked", ExpiresAt: now.Add(time.Hour), RevokedAt: &now},
			plain:         "revoked",
			wantErr:       domain.ErrTokenReuse,
			wantChainKill: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := newFakeUserRepo()
			users.add(&domain.User{ID: 1, Email: "u@x.jp", PasswordHash: mustHash(t, testPassword), IsActive: true})
			ref := newFakeRefreshRepo()
			if tt.token != nil {
				ref.seed(tt.token)
			}
			uc := newAuthUC(users, newFakeSetTokenRepo(), ref)

			_, err := uc.Refresh(context.Background(), tt.plain)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantChainKill, ref.revokedUser[1], "チェーン全失効の有無")
		})
	}
}

// TestAuthUsecase_Refresh_ConcurrentReplay は同一トークンの並行リプレイで
// 有効セッションが二重発行されないこと（成功ちょうど1件）を -race 下で検証する。
func TestAuthUsecase_Refresh_ConcurrentReplay(t *testing.T) {
	const goroutines = 16

	users := newFakeUserRepo()
	users.add(&domain.User{ID: 1, Email: "u@x.jp", PasswordHash: mustHash(t, testPassword), IsActive: true})
	ref := newFakeRefreshRepo()
	ref.seed(&domain.RefreshToken{ID: 42, UserID: 1, TokenHash: "h:stolen", ExpiresAt: time.Now().Add(time.Hour)})
	uc := newAuthUC(users, newFakeSetTokenRepo(), ref)

	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		okN   int
		reuse int
	)
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // 一斉スタートで競合を最大化する。
			_, err := uc.Refresh(context.Background(), "stolen")
			mu.Lock()
			switch {
			case err == nil:
				okN++
			case errors.Is(err, domain.ErrTokenReuse):
				reuse++
			}
			mu.Unlock()
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, 1, okN, "ローテーションに成功するのはちょうど1件（二重セッション防止）")
	assert.Equal(t, goroutines-1, reuse, "残りは全て再利用として弾かれる")
	assert.True(t, ref.revokedUser[1], "リプレイ検知でチェーンが全失効する")
}
