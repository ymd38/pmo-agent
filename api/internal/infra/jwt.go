package infra

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTService はアクセストークン（HS256）の発行・検証を行う。
// 権限（functions）は埋め込まず、ACL ミドルウェアが毎回 DB から解決する
// （ロール変更を即時反映するため）。クレームは sub=userID のみ。
type JWTService struct {
	secret []byte
	ttl    time.Duration
}

func NewJWTService(secret string, ttl time.Duration) *JWTService {
	return &JWTService{secret: []byte(secret), ttl: ttl}
}

// Generate は userID を主体とするアクセストークンを発行する。
func (s *JWTService) Generate(userID int) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", userID),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// Verify はトークンを検証し、userID を返す。
func (s *JWTService) Verify(tokenString string) (int, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("予期しない署名方式: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return 0, err
	}
	var userID int
	if _, err := fmt.Sscanf(claims.Subject, "%d", &userID); err != nil {
		return 0, fmt.Errorf("不正な subject: %w", err)
	}
	return userID, nil
}
