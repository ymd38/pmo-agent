package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	now     func() time.Time

	// dummyHash は起動時に生成する使い捨ての bcrypt ハッシュ。ユーザーが存在しない／
	// 未アクティベートのときも、これに対して Compare を1回実行して bcrypt 相当の
	// 処理コストを払い、応答時間差によるユーザー列挙（アカウント存在の推測）を防ぐ。
	dummyHash string
}

// fallbackDummyHash は起動時のダミーハッシュ生成に失敗した場合の代替。
// 実在するパスワードとの一致は問題にならない（Compare の結果は捨て、常に
// ErrInvalidCredentials を返す）。目的は bcrypt の処理時間を消費することだけ。
// 値は bcrypt.DefaultCost(=10) で生成した有効なハッシュ。
const fallbackDummyHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

func NewAuthUsecase(
	users UserRepository,
	setToks PasswordSetTokenRepository,
	refToks RefreshTokenRepository,
	hasher Hasher,
	jwt AccessTokenIssuer,
	tokens TokenManager,
	refTTL time.Duration,
) *AuthUsecase {
	return &AuthUsecase{
		users: users, setToks: setToks, refToks: refToks,
		hasher: hasher, jwt: jwt, tokens: tokens,
		refTTL:    refTTL,
		now:       time.Now,
		dummyHash: newDummyHash(hasher),
	}
}

// newDummyHash はランダムな平文から使い捨ての bcrypt ハッシュを生成する。
// 実 Compare と同じコスト（同じ hasher 実装）で処理時間を揃えるのが目的。
// 生成に失敗した場合は固定のフォールバックハッシュを使う。
func newDummyHash(h Hasher) string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fallbackDummyHash
	}
	hash, err := h.Hash(hex.EncodeToString(buf))
	if err != nil {
		return fallbackDummyHash
	}
	return hash
}

// Tokens はログイン/リフレッシュの結果（平文トークン）。Cookie へ載せるのは handler の責務。
type Tokens struct {
	Access  string
	Refresh string
}

const minPasswordLen = 8

// expiredTokenRetention は期限切れトークンを削除せず残す猶予期間。
// 期限切れ直後のリフレッシュトークンも一定期間は再利用検知の対象として残すため、
// リフレッシュトークン TTL（7日）を十分に上回る値にする。
const expiredTokenRetention = 30 * 24 * time.Hour

// Login はメール+パスワードを検証し、アクセス/リフレッシュトークンを発行する。
// 認証失敗・未アクティベート・無効化はすべて ErrInvalidCredentials に集約（情報を漏らさない）。
//
// タイミング攻撃対策: ユーザーが存在しない／未アクティベートの経路でも、実在ユーザーの
// パスワード照合と同等の bcrypt コストを払う（ダミー Compare）。これにより応答時間差から
// アカウントの存在有無を推測されるのを防ぐ。
func (uc *AuthUsecase) Login(ctx context.Context, email, password string) (*domain.User, Tokens, error) {
	user, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			uc.dummyCompare(password)
			return nil, Tokens{}, domain.ErrInvalidCredentials
		}
		return nil, Tokens{}, fmt.Errorf("usecase.Login: %w", err)
	}
	if !user.CanLogin() {
		uc.dummyCompare(password)
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

// dummyCompare はダミーハッシュに対して Compare を1回実行し、bcrypt 相当のコストを払う。
// 結果は常に捨てる（呼び出し側は経路によらず ErrInvalidCredentials を返す）。
func (uc *AuthUsecase) dummyCompare(password string) {
	_ = uc.hasher.Compare(uc.dummyHash, password)
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
		if err := uc.refToks.RevokeAllForUser(ctx, rt.UserID); err != nil {
			return Tokens{}, fmt.Errorf("usecase.Refresh revokeAll: %w", err)
		}
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
			if chainErr := uc.refToks.RevokeAllForUser(ctx, rt.UserID); chainErr != nil {
				return Tokens{}, fmt.Errorf("usecase.Refresh reuse revokeAll: %w", chainErr)
			}
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

// CleanupExpiredTokens は期限切れの refresh_tokens / password_set_tokens を削除する
// （テーブルの無限増加を防ぐ低頻度メンテナンス。起動時に1回呼ぶ想定）。
//
// 削除対象は「expires_at が now - 猶予期間 より前」の行に限定する。ローテーション時や
// 失効時に行を消すと再利用検知（失効済みトークンが FindByHash で見つかることに依存）が
// 壊れるため、削除は期限切れ＋猶予経過の行だけに限る。削除件数（refresh, set）を返す。
func (uc *AuthUsecase) CleanupExpiredTokens(ctx context.Context) (refreshDeleted, setDeleted int64, err error) {
	cutoff := uc.now().Add(-expiredTokenRetention)
	refreshDeleted, err = uc.refToks.DeleteExpiredBefore(ctx, cutoff)
	if err != nil {
		return 0, 0, fmt.Errorf("usecase.CleanupExpiredTokens refresh: %w", err)
	}
	setDeleted, err = uc.setToks.DeleteExpiredBefore(ctx, cutoff)
	if err != nil {
		return refreshDeleted, 0, fmt.Errorf("usecase.CleanupExpiredTokens set: %w", err)
	}
	return refreshDeleted, setDeleted, nil
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
	// セキュリティ: 既存セッションを失効させる。失敗はセキュリティ制御の silent failure
	// になるためエラーとして伝播する。
	if err := uc.refToks.RevokeAllForUser(ctx, tok.UserID); err != nil {
		return fmt.Errorf("usecase.SetPassword revokeAll: %w", err)
	}
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
	if err := uc.users.UpdatePasswordHash(ctx, userID, hash); err != nil {
		return fmt.Errorf("usecase.ChangePassword update: %w", err)
	}
	// セキュリティ: パスワード変更後は既存セッション（盗用された可能性のある
	// リフレッシュトークンを含む）を全失効させる。失敗はセキュリティ制御の
	// silent failure になるためエラーとして伝播する。
	if err := uc.refToks.RevokeAllForUser(ctx, userID); err != nil {
		return fmt.Errorf("usecase.ChangePassword revokeAll: %w", err)
	}
	return nil
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
