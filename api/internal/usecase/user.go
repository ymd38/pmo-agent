package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"pmo-agent/api/internal/domain"
)

// UserUsecase はメンバー管理（ユーザーCRUD＋招待/リセットリンク発行）を担う。
type UserUsecase struct {
	users   UserRepository
	roles   RoleRepository
	setToks PasswordSetTokenRepository
	refToks RefreshTokenRepository
	tokens  TokenManager
	setTTL  time.Duration
	baseURL string
	now     func() time.Time
}

func NewUserUsecase(
	users UserRepository,
	roles RoleRepository,
	setToks PasswordSetTokenRepository,
	refToks RefreshTokenRepository,
	tokens TokenManager,
	setTTL time.Duration,
	baseURL string,
) *UserUsecase {
	return &UserUsecase{
		users: users, roles: roles, setToks: setToks, refToks: refToks, tokens: tokens,
		setTTL: setTTL, baseURL: baseURL, now: time.Now,
	}
}

// CreateInput はユーザー作成の入力。
type CreateInput struct {
	Email   string
	Name    string
	Grade   domain.Grade
	RoleIDs []int
}

// UpdateInput はユーザー更新の入力。
// IsActive は nil のとき現在値を維持する（未指定での意図しない無効化を防ぐ）。
type UpdateInput struct {
	Name     string
	Grade    domain.Grade
	IsActive *bool
	RoleIDs  []int
}

func (uc *UserUsecase) List(ctx context.Context) ([]domain.User, error) {
	return uc.users.List(ctx)
}

func (uc *UserUsecase) Get(ctx context.Context, id int) (*domain.User, error) {
	user, err := uc.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	roles, err := uc.users.RolesByUserID(ctx, id)
	if err != nil {
		return nil, err
	}
	user.Roles = roles
	return user, nil
}

// Create はユーザーを未アクティベート状態で作成し、設定用リンクを返す。
func (uc *UserUsecase) Create(ctx context.Context, in CreateInput) (*domain.User, string, error) {
	if err := uc.validate(ctx, in.Email, in.Name, in.Grade, in.RoleIDs); err != nil {
		return nil, "", err
	}
	if existing, _ := uc.users.FindByEmail(ctx, in.Email); existing != nil {
		return nil, "", fmt.Errorf("%w: このメールアドレスは既に登録されています", domain.ErrConflict)
	}
	user := &domain.User{
		Email:    strings.TrimSpace(in.Email),
		Name:     strings.TrimSpace(in.Name),
		Grade:    in.Grade,
		IsActive: true,
		// PasswordHash は nil（未アクティベート）。
	}
	if err := uc.users.Create(ctx, user, in.RoleIDs); err != nil {
		return nil, "", fmt.Errorf("usecase.Create: %w", err)
	}
	link, err := uc.issueSetLink(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}
	return user, link, nil
}

// Update はユーザー情報とロールを更新する。
func (uc *UserUsecase) Update(ctx context.Context, id int, in UpdateInput) (*domain.User, error) {
	user, err := uc.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := uc.validate(ctx, user.Email, in.Name, in.Grade, in.RoleIDs); err != nil {
		return nil, err
	}
	user.Name = strings.TrimSpace(in.Name)
	user.Grade = in.Grade
	if in.IsActive != nil {
		user.IsActive = *in.IsActive
	}
	if err := uc.users.Update(ctx, user, in.RoleIDs); err != nil {
		return nil, fmt.Errorf("usecase.Update: %w", err)
	}
	// 明示的な無効化時は既存セッション（リフレッシュトークン）も失効させる。
	if in.IsActive != nil && !*in.IsActive {
		if err := uc.refToks.RevokeAllForUser(ctx, id); err != nil {
			return nil, fmt.Errorf("usecase.Update revoke: %w", err)
		}
	}
	return uc.Get(ctx, id)
}

// Deactivate はユーザーを論理削除する（is_active=false）。物理削除はしない。
// 無効化にあわせてリフレッシュトークンを失効させ、既存セッションを断つ。
func (uc *UserUsecase) Deactivate(ctx context.Context, id int) error {
	if _, err := uc.users.FindByID(ctx, id); err != nil {
		return err
	}
	if err := uc.users.Deactivate(ctx, id); err != nil {
		return err
	}
	if err := uc.refToks.RevokeAllForUser(ctx, id); err != nil {
		return fmt.Errorf("usecase.Deactivate revoke: %w", err)
	}
	return nil
}

// ReissueLink は招待/リセット用リンクを再発行する（自己リセットにも使う）。
func (uc *UserUsecase) ReissueLink(ctx context.Context, id int) (string, error) {
	if _, err := uc.users.FindByID(ctx, id); err != nil {
		return "", err
	}
	return uc.issueSetLink(ctx, id)
}

// issueSetLink は既存トークンを失効させ、新規トークンを発行して設定用URLを返す。
func (uc *UserUsecase) issueSetLink(ctx context.Context, userID int) (string, error) {
	if err := uc.setToks.InvalidateForUser(ctx, userID); err != nil {
		return "", fmt.Errorf("usecase.issueSetLink invalidate: %w", err)
	}
	plain, err := uc.tokens.Generate()
	if err != nil {
		return "", fmt.Errorf("usecase.issueSetLink gen: %w", err)
	}
	tok := &domain.PasswordSetToken{
		UserID:    userID,
		TokenHash: uc.tokens.Hash(plain),
		ExpiresAt: uc.now().Add(uc.setTTL),
	}
	if err := uc.setToks.Create(ctx, tok); err != nil {
		return "", fmt.Errorf("usecase.issueSetLink store: %w", err)
	}
	return fmt.Sprintf("%s/set-password?token=%s", strings.TrimRight(uc.baseURL, "/"), plain), nil
}

func (uc *UserUsecase) validate(ctx context.Context, email, name string, grade domain.Grade, roleIDs []int) error {
	if strings.TrimSpace(email) == "" || !strings.Contains(email, "@") {
		return fmt.Errorf("%w: 有効なメールアドレスを入力してください", domain.ErrValidation)
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: 氏名は必須です", domain.ErrValidation)
	}
	if grade != domain.GradeManager && grade != domain.GradeStaff {
		return fmt.Errorf("%w: グレードが不正です", domain.ErrValidation)
	}
	if len(roleIDs) == 0 {
		return fmt.Errorf("%w: ロールを1つ以上指定してください", domain.ErrValidation)
	}
	ok, err := uc.roles.AllIDsExist(ctx, roleIDs)
	if err != nil {
		return fmt.Errorf("usecase.validate roles: %w", err)
	}
	if !ok {
		return fmt.Errorf("%w: 指定されたロールが存在しません", domain.ErrValidation)
	}
	return nil
}
