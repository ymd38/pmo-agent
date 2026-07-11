package usecase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"pmo-agent/api/internal/domain"
)

// --- テスト用フェイク ---

type fakeUserRepo struct {
	byID    map[int]*domain.User
	byEmail map[string]*domain.User
	funcs   map[int][]string
	roles   map[int][]domain.Role
	created []*domain.User
	hashSet map[int]string
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byID: map[int]*domain.User{}, byEmail: map[string]*domain.User{},
		funcs: map[int][]string{}, roles: map[int][]domain.Role{},
		hashSet: map[int]string{},
	}
}

func (f *fakeUserRepo) add(u *domain.User) {
	f.byID[u.ID] = u
	f.byEmail[u.Email] = u
}

func (f *fakeUserRepo) FindByID(_ context.Context, id int) (*domain.User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	if u, ok := f.byEmail[email]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUserRepo) Create(_ context.Context, u *domain.User, _ []int) error {
	u.ID = len(f.byID) + 1
	f.add(u)
	f.created = append(f.created, u)
	return nil
}

func (f *fakeUserRepo) Update(_ context.Context, _ *domain.User, _ []int) error { return nil }

func (f *fakeUserRepo) UpdatePasswordHash(_ context.Context, userID int, hash string) error {
	f.hashSet[userID] = hash
	if u, ok := f.byID[userID]; ok {
		u.PasswordHash = &hash
	}
	return nil
}

func (f *fakeUserRepo) Deactivate(_ context.Context, id int) error {
	if u, ok := f.byID[id]; ok {
		u.IsActive = false
	}
	return nil
}

func (f *fakeUserRepo) List(_ context.Context) ([]domain.User, error) { return nil, nil }

func (f *fakeUserRepo) FunctionsByUserID(_ context.Context, id int) ([]string, error) {
	return f.funcs[id], nil
}

func (f *fakeUserRepo) RolesByUserID(_ context.Context, id int) ([]domain.Role, error) {
	return f.roles[id], nil
}

type fakeSetTokenRepo struct {
	byHash     map[string]*domain.PasswordSetToken
	used       map[int]bool
	invalidate map[int]bool
	seq        int
}

func newFakeSetTokenRepo() *fakeSetTokenRepo {
	return &fakeSetTokenRepo{byHash: map[string]*domain.PasswordSetToken{}, used: map[int]bool{}, invalidate: map[int]bool{}}
}

func (f *fakeSetTokenRepo) Create(_ context.Context, t *domain.PasswordSetToken) error {
	f.seq++
	t.ID = f.seq
	f.byHash[t.TokenHash] = t
	return nil
}

func (f *fakeSetTokenRepo) FindByHash(_ context.Context, hash string) (*domain.PasswordSetToken, error) {
	if t, ok := f.byHash[hash]; ok {
		return t, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeSetTokenRepo) MarkUsed(_ context.Context, id int) error { f.used[id] = true; return nil }

func (f *fakeSetTokenRepo) InvalidateForUser(_ context.Context, userID int) error {
	f.invalidate[userID] = true
	return nil
}

// fakeRefreshRepo は DB のアトミックCASを mutex で模擬する。並行リプレイテスト用に
// Rotate / Revoke は revoked 状態を検査してから更新し、1度しか成功しない。
type fakeRefreshRepo struct {
	mu           sync.Mutex
	byHash       map[string]*domain.RefreshToken
	byID         map[int]*domain.RefreshToken
	revoked      map[int]bool
	revokedUser  map[int]bool
	seq          int
	revokeAllErr error // RevokeAllForUser の失敗注入用
}

func newFakeRefreshRepo() *fakeRefreshRepo {
	return &fakeRefreshRepo{
		byHash:      map[string]*domain.RefreshToken{},
		byID:        map[int]*domain.RefreshToken{},
		revoked:     map[int]bool{},
		revokedUser: map[int]bool{},
	}
}

// seed は永続化済みトークンを直接投入する（テストの前提条件セットアップ用）。
func (f *fakeRefreshRepo) seed(t *domain.RefreshToken) {
	f.byHash[t.TokenHash] = t
	f.byID[t.ID] = t
	if t.ID > f.seq {
		f.seq = t.ID
	}
}

func (f *fakeRefreshRepo) Create(_ context.Context, t *domain.RefreshToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	t.ID = f.seq
	f.byHash[t.TokenHash] = t
	f.byID[t.ID] = t
	return nil
}

func (f *fakeRefreshRepo) FindByHash(_ context.Context, hash string) (*domain.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t, ok := f.byHash[hash]; ok {
		snap := *t // DB 読み取りを模して呼び出し側にコピーを渡す。
		return &snap, nil
	}
	return nil, domain.ErrNotFound
}

// Revoke は未失効時のみ成功する（CAS）。既に失効済みなら ErrTokenReuse。
func (f *fakeRefreshRepo) Revoke(_ context.Context, id int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.revokeLocked(id)
}

func (f *fakeRefreshRepo) revokeLocked(id int) error {
	if f.revoked[id] {
		return domain.ErrTokenReuse
	}
	f.revoked[id] = true
	if t, ok := f.byID[id]; ok {
		now := time.Now()
		t.RevokedAt = &now
	}
	return nil
}

// Rotate は旧失効(CAS)＋新規発行をアトミックに行う。CAS 敗北時は ErrTokenReuse。
func (f *fakeRefreshRepo) Rotate(_ context.Context, oldID int, newTok *domain.RefreshToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.revokeLocked(oldID); err != nil {
		return err
	}
	f.seq++
	newTok.ID = f.seq
	f.byHash[newTok.TokenHash] = newTok
	f.byID[newTok.ID] = newTok
	return nil
}

func (f *fakeRefreshRepo) RevokeAllForUser(_ context.Context, userID int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.revokeAllErr != nil {
		return f.revokeAllErr
	}
	f.revokedUser[userID] = true
	for id, t := range f.byID {
		if t.UserID == userID {
			f.revoked[id] = true
		}
	}
	return nil
}

type fakeRoleRepo struct {
	existing map[int]bool
}

func (f *fakeRoleRepo) List(_ context.Context) ([]domain.Role, error) { return nil, nil }

func (f *fakeRoleRepo) AllIDsExist(_ context.Context, ids []int) (bool, error) {
	for _, id := range ids {
		if !f.existing[id] {
			return false, nil
		}
	}
	return true, nil
}

// fakeJWT は固定のアクセストークンを返す。
type fakeJWT struct{}

func (fakeJWT) Generate(userID int) (string, error) { return fmt.Sprintf("access-%d", userID), nil }

// fakeTokens は決定的な不透明トークンを生成する。Hash は "h:"+plain。
// 並行 Refresh テストでの競合を避けるため mutex で採番を保護する。
type fakeTokens struct {
	mu  sync.Mutex
	seq int
}

func (f *fakeTokens) Generate() (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	return fmt.Sprintf("plain-%d", f.seq), nil
}

func (f *fakeTokens) Hash(plain string) string { return "h:" + plain }

// countingHasher は Compare / Hash の呼び出し回数を数えるフェイク。
// タイミング平準化（未知メールでも Compare が呼ばれること）の検証に使う。
// Hash は "h:"+plain を返し、Compare は hash == "h:"+plain のときのみ一致とする。
type countingHasher struct {
	mu        sync.Mutex
	compareN  int
	comparedH []string // Compare に渡されたハッシュ（ダミー使用の確認用）
}

func (h *countingHasher) Hash(plain string) (string, error) { return "h:" + plain, nil }

func (h *countingHasher) Compare(hash, plain string) error {
	h.mu.Lock()
	h.compareN++
	h.comparedH = append(h.comparedH, hash)
	h.mu.Unlock()
	if hash == "h:"+plain {
		return nil
	}
	return errCompareMismatch
}

var errCompareMismatch = fmt.Errorf("hash mismatch")
