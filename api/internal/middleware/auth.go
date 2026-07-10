package middleware

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"pmo-agent/api/internal/domain"

	"github.com/gin-gonic/gin"
)

// Cookie 名。ログイン/リフレッシュ/ログアウトで共有する。
const (
	CookieAccess  = "access_token"
	CookieRefresh = "refresh_token"
	ctxUserID     = "userID"
	ctxScope      = "projectScope"
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

// ScopeResolver はユーザーのロールから許可プロジェクト範囲を解決する（usecase.ScopeUsecase 実装）。
type ScopeResolver interface {
	ResolveProjectScope(ctx context.Context, userID int) (domain.ProjectScope, error)
}

type Middleware struct {
	verifier  TokenVerifier
	functions FunctionResolver
	active    ActiveResolver
	scope     ScopeResolver
}

func New(verifier TokenVerifier, functions FunctionResolver, active ActiveResolver, scope ScopeResolver) *Middleware {
	return &Middleware{verifier: verifier, functions: functions, active: active, scope: scope}
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
// スコープ制御（担当PJのみ等）は ResolveProjectScope で解決する。
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
		if !slices.Contains(fns, code) {
			abort(c, http.StatusForbidden, "この操作を行う権限がありません")
			return
		}
		c.Next()
	}
}

// ResolveProjectScope はログインユーザーのロールから許可プロジェクト範囲を解決し、
// コンテキストへ格納する。Authenticate の後段に置き、プロジェクト参照系エンドポイントに適用する。
func (m *Middleware) ResolveProjectScope() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := UserID(c)
		if userID == 0 {
			abort(c, http.StatusUnauthorized, "認証が必要です")
			return
		}
		scope, err := m.scope.ResolveProjectScope(c.Request.Context(), userID)
		if err != nil {
			abort(c, http.StatusInternalServerError, "権限の確認に失敗しました")
			return
		}
		c.Set(ctxScope, scope)
		c.Next()
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

// ProjectScope はコンテキストから解決済みスコープを取得する。
// ResolveProjectScope ミドルウェアを通っていない場合は ok=false。
func ProjectScope(c *gin.Context) (domain.ProjectScope, bool) {
	if v, ok := c.Get(ctxScope); ok {
		if s, ok := v.(domain.ProjectScope); ok {
			return s, true
		}
	}
	return domain.ProjectScope{}, false
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
