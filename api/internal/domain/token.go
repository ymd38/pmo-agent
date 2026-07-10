package domain

import "time"

// PasswordSetToken は招待・リセット共通のワンタイムトークン（ハッシュ保存・単回利用）。
type PasswordSetToken struct {
	ID        int        `json:"id"         gorm:"primaryKey"`
	UserID    int        `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at"`
	CreatedAt time.Time  `json:"created_at"`
}

func (PasswordSetToken) TableName() string { return "password_set_tokens" }

// IsUsable は未使用かつ未失効なら true。
func (t PasswordSetToken) IsUsable(now time.Time) bool {
	return t.UsedAt == nil && now.Before(t.ExpiresAt)
}

// RefreshToken はリフレッシュトークン（ローテーション・失効管理）。
type RefreshToken struct {
	ID        int        `json:"id"         gorm:"primaryKey"`
	UserID    int        `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at"`
	CreatedAt time.Time  `json:"created_at"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

// IsUsable は未失効かつ未期限切れなら true。
func (t RefreshToken) IsUsable(now time.Time) bool {
	return t.RevokedAt == nil && now.Before(t.ExpiresAt)
}
