package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pmo-agent/api/internal/domain"
)

// AuthUsecase は認証フロー（ログイン・リフレッシュ・パスワード設定/変更）を担う。
type AuthUsecase struct {
	users   UserRepository
	setToks PasswordSetTokenRepository
	refToks RefreshTokenRepository
	hasher  Hasher
	jwt     AccessTokenIssuer
	tokens  TokenManager
	refTTL  time.Duration
	setTTL  time.Duration
	baseURL string
	now     func() time.Time
}

func NewAuthUsecase(
	users UserRepository,
	setToks PasswordSetTokenRepository,
	refToks RefreshTokenRepository,
	hasher Hasher,
	jwt AccessTokenIssuer,
	tokens TokenManager,
	refTTL, setTTL time.Duration,
	baseURL string,
) *AuthUsecase {
	return &AuthUsecase{
		users: users, setToks: setToks, refToks: refToks,
		hasher: hasher, jwt: jwt, tokens: tokens,
		refTTL: refTTL, setTTL: setTTL, baseURL: baseURL,
		now: time.Now,
	}
}

// Tokens はログイン/リフレッシュの結果（平文トークン）。Cookie へ載せるのは handler の責務。
type Tokens struct {
	Access  string
	Refresh string
}

const minPasswordLen = 8

// Login はメール+パスワードを検証し、アクセス/リフレッシュトークンを発行する。
// 認証失敗・未アクティベート・無効化はすべて ErrInvalidCredentials に集約（情報を漏らさない）。
func (uc *AuthUsecase) Login(ctx context.Context, email, password string) (*domain.User, Tokens, error) {
	user, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, Tokens{}, domain.ErrInvalidCredentials
		}
		return nil, Tokens{}, fmt.Errorf("usecase.Login: %w", err)
	}
	if !user.CanLogin() {
		return nil, Tokens{}, domain.ErrInvalidCredentials
	}
	if err := uc.hasher.Compare(*user.PasswordHash, password); err != nil {
		return nil, Tokens{}, domain.ErrInvalidCredentials
	}
	toks, err := uc.issueTokens(ctx, user.ID)
	if err != nil {
		return nil, Tokens{}, err
	}
	return user, toks, nil
}

// Refresh はリフレッシュトークンを検証し、新しいアクセストークンを発行する。
// 旧リフレッシュトークンは失効させ、新しいものを原子的に発行する（ローテーション）。
// 失効済みトークンの再提示や並行リプレイを検知した場合は、当該ユーザーのトークン
// チェーンを全失効させる（盗用時の被害を最小化するセキュリティ対応）。
func (uc *AuthUsecase) Refresh(ctx context.Context, refreshPlain string) (Tokens, error) {
	rt, err := uc.refToks.FindByHash(ctx, uc.tokens.Hash(refreshPlain))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return Tokens{}, domain.ErrTokenInvalid
		}
		return Tokens{}, fmt.Errorf("usecase.Refresh: %w", err)
	}
	// 再利用検知(1): 失効済みトークンの再提示はチェーン侵害の兆候。全セッションを失効させる。
	if rt.RevokedAt != nil {
		_ = uc.refToks.RevokeAllForUser(ctx, rt.UserID)
		return Tokens{}, domain.ErrTokenReuse
	}
	if !rt.IsUsable(uc.now()) { // ここに到達するのは期限切れのみ。
		return Tokens{}, domain.ErrTokenInvalid
	}
	user, err := uc.users.FindByID(ctx, rt.UserID)
	if err != nil || !user.CanLogin() {
		return Tokens{}, domain.ErrTokenInvalid
	}
	// 新トークンを生成し、旧失効(CAS)＋新規発行を1トランザクションで原子的に行う。
	access, newRefreshPlain, newRT, err := uc.prepareTokens(user.ID)
	if err != nil {
		return Tokens{}, err
	}
	if err := uc.refToks.Rotate(ctx, rt.ID, newRT); err != nil {
		// 再利用検知(2): CAS 敗北 = 並行リプレイ。チェーンを全失効させる。
		if errors.Is(err, domain.ErrTokenReuse) {
			_ = uc.refToks.RevokeAllForUser(ctx, rt.UserID)
			return Tokens{}, domain.ErrTokenReuse
		}
		return Tokens{}, fmt.Errorf("usecase.Refresh rotate: %w", err)
	}
	return Tokens{Access: access, Refresh: newRefreshPlain}, nil
}

// Logout はリフレッシュトークンを失効させる。未知のトークンは黙って成功扱い。
func (uc *AuthUsecase) Logout(ctx context.Context, refreshPlain string) error {
	if refreshPlain == "" {
		return nil
	}
	rt, err := uc.refToks.FindByHash(ctx, uc.tokens.Hash(refreshPlain))
	if err != nil {
		return nil
	}
	// 既に失効済み（ErrTokenReuse）でもログアウトは冪等に成功扱いとする。
	if err := uc.refToks.Revoke(ctx, rt.ID); err != nil && !errors.Is(err, domain.ErrTokenReuse) {
		return err
	}
	return nil
}

// Me はログイン中ユーザーと、その保有機能権限コードを返す。
func (uc *AuthUsecase) Me(ctx context.Context, userID int) (*domain.User, []string, error) {
	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	roles, err := uc.users.RolesByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	user.Roles = roles
	fns, err := uc.users.FunctionsByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	return user, fns, nil
}

// VerifySetToken はパスワード設定トークンの有効性を確認し、対象メールを返す。
func (uc *AuthUsecase) VerifySetToken(ctx context.Context, tokenPlain string) (string, error) {
	tok, err := uc.setToks.FindByHash(ctx, uc.tokens.Hash(tokenPlain))
	if err != nil || !tok.IsUsable(uc.now()) {
		return "", domain.ErrTokenInvalid
	}
	user, err := uc.users.FindByID(ctx, tok.UserID)
	if err != nil {
		return "", domain.ErrTokenInvalid
	}
	return user.Email, nil
}

// SetPassword はトークンを消費してパスワードを設定する（招待・リセット共通）。
func (uc *AuthUsecase) SetPassword(ctx context.Context, tokenPlain, newPassword string) error {
	if len(newPassword) < minPasswordLen {
		return fmt.Errorf("%w: パスワードは%d文字以上にしてください", domain.ErrValidation, minPasswordLen)
	}
	tok, err := uc.setToks.FindByHash(ctx, uc.tokens.Hash(tokenPlain))
	if err != nil || !tok.IsUsable(uc.now()) {
		return domain.ErrTokenInvalid
	}
	hash, err := uc.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("usecase.SetPassword hash: %w", err)
	}
	if err := uc.users.UpdatePasswordHash(ctx, tok.UserID, hash); err != nil {
		return fmt.Errorf("usecase.SetPassword update: %w", err)
	}
	if err := uc.setToks.MarkUsed(ctx, tok.ID); err != nil {
		return fmt.Errorf("usecase.SetPassword markUsed: %w", err)
	}
	// セキュリティ: 既存セッションを失効させる。
	_ = uc.refToks.RevokeAllForUser(ctx, tok.UserID)
	return nil
}

// ChangePassword はログイン中ユーザーが現パスワード確認のうえ変更する。
func (uc *AuthUsecase) ChangePassword(ctx context.Context, userID int, current, next string) error {
	if len(next) < minPasswordLen {
		return fmt.Errorf("%w: パスワードは%d文字以上にしてください", domain.ErrValidation, minPasswordLen)
	}
	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.PasswordHash == nil {
		return fmt.Errorf("%w: パスワードが未設定です", domain.ErrValidation)
	}
	if err := uc.hasher.Compare(*user.PasswordHash, current); err != nil {
		return domain.ErrInvalidCredentials
	}
	hash, err := uc.hasher.Hash(next)
	if err != nil {
		return fmt.Errorf("usecase.ChangePassword hash: %w", err)
	}
	return uc.users.UpdatePasswordHash(ctx, userID, hash)
}

// prepareTokens はアクセストークンと新しいリフレッシュトークン（平文＋永続化前レコード）を
// 生成する。DB には触れないため、呼び出し側が Create / Rotate で永続化する。
func (uc *AuthUsecase) prepareTokens(userID int) (access, refreshPlain string, rt *domain.RefreshToken, err error) {
	access, err = uc.jwt.Generate(userID)
	if err != nil {
		return "", "", nil, fmt.Errorf("usecase.prepareTokens jwt: %w", err)
	}
	refreshPlain, err = uc.tokens.Generate()
	if err != nil {
		return "", "", nil, fmt.Errorf("usecase.prepareTokens gen: %w", err)
	}
	rt = &domain.RefreshToken{
		UserID:    userID,
		TokenHash: uc.tokens.Hash(refreshPlain),
		ExpiresAt: uc.now().Add(uc.refTTL),
	}
	return access, refreshPlain, rt, nil
}

// issueTokens は新規トークンを生成して保存する（ログイン時のローテーションを伴わない発行）。
func (uc *AuthUsecase) issueTokens(ctx context.Context, userID int) (Tokens, error) {
	access, refreshPlain, rt, err := uc.prepareTokens(userID)
	if err != nil {
		return Tokens{}, err
	}
	if err := uc.refToks.Create(ctx, rt); err != nil {
		return Tokens{}, fmt.Errorf("usecase.issueTokens store: %w", err)
	}
	return Tokens{Access: access, Refresh: refreshPlain}, nil
}
