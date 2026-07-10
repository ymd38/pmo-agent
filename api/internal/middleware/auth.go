package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Cookie 名。ログイン/リフレッシュ/ログアウトで共有する。
const (
	CookieAccess  = "access_token"
	CookieRefresh = "refresh_token"
	ctxUserID     = "userID"
)

// TokenVerifier はアクセストークンを検証して userID を返す（infra.JWTService 実装）。
type TokenVerifier interface {
	Verify(token string) (int, error)
}

// FunctionResolver はユーザーの保有機能権限コードを返す（repository.UserRepo 実装）。
type FunctionResolver interface {
	FunctionsByUserID(ctx context.Context, userID int) ([]string, error)
}

// ActiveResolver はユーザーが有効（is_active=true）かを返す（repository.UserRepo 実装）。
type ActiveResolver interface {
	IsActive(ctx context.Context, userID int) (bool, error)
}

type Middleware struct {
	verifier  TokenVerifier
	functions FunctionResolver
	active    ActiveResolver
}

func New(verifier TokenVerifier, functions FunctionResolver, active ActiveResolver) *Middleware {
	return &Middleware{verifier: verifier, functions: functions, active: active}
}

// Authenticate はアクセストークン（Cookie or Authorization ヘッダ）を検証し、
// リクエストごとにユーザーの有効性（is_active）を DB で再確認する。
// トークン有効期限内でも無効化済みユーザーのアクセスは即座に遮断する。
func (m *Middleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			abort(c, http.StatusUnauthorized, "認証が必要です")
			return
		}
		userID, err := m.verifier.Verify(token)
		if err != nil {
			abort(c, http.StatusUnauthorized, "認証に失敗しました")
			return
		}
		active, err := m.active.IsActive(c.Request.Context(), userID)
		if err != nil {
			abort(c, http.StatusInternalServerError, "認証に失敗しました")
			return
		}
		if !active {
			abort(c, http.StatusUnauthorized, "アカウントが無効です")
			return
		}
		c.Set(ctxUserID, userID)
		c.Next()
	}
}

// RequireFunction は指定した機能権限コードを保有していなければ 403 を返す。
// スコープ制御（担当PJのみ等）は別のスコープミドルウェアで実装する。
func (m *Middleware) RequireFunction(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := UserID(c)
		if userID == 0 {
			abort(c, http.StatusUnauthorized, "認証が必要です")
			return
		}
		fns, err := m.functions.FunctionsByUserID(c.Request.Context(), userID)
		if err != nil {
			abort(c, http.StatusInternalServerError, "権限の確認に失敗しました")
			return
		}
		for _, f := range fns {
			if f == code {
				c.Next()
				return
			}
		}
		abort(c, http.StatusForbidden, "この操作を行う権限がありません")
	}
}

// UserID はコンテキストからログイン中ユーザーIDを取得する（未認証なら 0）。
func UserID(c *gin.Context) int {
	if v, ok := c.Get(ctxUserID); ok {
		if id, ok := v.(int); ok {
			return id
		}
	}
	return 0
}

func extractToken(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	if cookie, err := c.Cookie(CookieAccess); err == nil {
		return cookie
	}
	return ""
}

func abort(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}
