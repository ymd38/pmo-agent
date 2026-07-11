package repository

import (
	"context"
	"time"

	"pmo-agent/api/internal/domain"

	"gorm.io/gorm"
)

// --- パスワード設定トークン ---

type PasswordSetTokenRepo struct {
	db *gorm.DB
}

func NewPasswordSetTokenRepo(db *gorm.DB) *PasswordSetTokenRepo {
	return &PasswordSetTokenRepo{db: db}
}

func (r *PasswordSetTokenRepo) Create(ctx context.Context, t *domain.PasswordSetToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *PasswordSetTokenRepo) FindByHash(ctx context.Context, hash string) (*domain.PasswordSetToken, error) {
	var t domain.PasswordSetToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&t).Error; err != nil {
		return nil, wrapNotFound(err)
	}
	return &t, nil
}

func (r *PasswordSetTokenRepo) MarkUsed(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Model(&domain.PasswordSetToken{}).
		Where("id = ?", id).Update("used_at", time.Now()).Error
}

// InvalidateForUser は当該ユーザーの未使用トークンをすべて失効させる（最新1件のみ有効に保つ）。
func (r *PasswordSetTokenRepo) InvalidateForUser(ctx context.Context, userID int) error {
	return r.db.WithContext(ctx).Model(&domain.PasswordSetToken{}).
		Where("user_id = ? AND used_at IS NULL", userID).
		Update("used_at", time.Now()).Error
}

// --- リフレッシュトークン ---

type RefreshTokenRepo struct {
	db *gorm.DB
}

func NewRefreshTokenRepo(db *gorm.DB) *RefreshTokenRepo {
	return &RefreshTokenRepo{db: db}
}

func (r *RefreshTokenRepo) Create(ctx context.Context, t *domain.RefreshToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *RefreshTokenRepo) FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	var t domain.RefreshToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&t).Error; err != nil {
		return nil, wrapNotFound(err)
	}
	return &t, nil
}

// Revoke は未失効のトークンだけを失効させる（CASガード）。
// 既に失効済み（該当0件）なら domain.ErrTokenReuse を返し、二重失効・リプレイを検知可能にする。
func (r *RefreshTokenRepo) Revoke(ctx context.Context, id int) error {
	return revokeGuarded(r.db.WithContext(ctx), id)
}

// Rotate は旧トークンの失効と新トークンの発行を1トランザクションで原子的に行う（ローテーション）。
// 旧トークンが既に失効済み（CAS該当0件）なら並行リプレイとみなし、ロールバックして
// domain.ErrTokenReuse を返す。
func (r *RefreshTokenRepo) Rotate(ctx context.Context, oldID int, newTok *domain.RefreshToken) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := revokeGuarded(tx, oldID); err != nil {
			return err
		}
		return tx.Create(newTok).Error
	})
}

// revokeGuarded は `revoked_at IS NULL` ガード付きで失効させ、該当0件なら ErrTokenReuse を返す。
// 単一の UPDATE 文なので MySQL の行ロックにより並行実行時も1件しか成功しない（アトミックCAS）。
func revokeGuarded(db *gorm.DB, id int) error {
	res := db.Model(&domain.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", time.Now())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrTokenReuse
	}
	return nil
}

func (r *RefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID int) error {
	return r.db.WithContext(ctx).Model(&domain.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}
