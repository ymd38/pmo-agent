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

func (r *RefreshTokenRepo) Revoke(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Model(&domain.RefreshToken{}).
		Where("id = ?", id).Update("revoked_at", time.Now()).Error
}

func (r *RefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID int) error {
	return r.db.WithContext(ctx).Model(&domain.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}
