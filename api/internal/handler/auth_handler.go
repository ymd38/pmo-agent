package handler

import (
	"net/http"

	"pmo-agent/api/internal/middleware"
	"pmo-agent/api/internal/usecase"

	"github.com/gin-gonic/gin"
)

// CookieConfig は認証 Cookie の設定。
type CookieConfig struct {
	AccessMaxAge  int // 秒
	RefreshMaxAge int // 秒
	Secure        bool
}

type AuthHandler struct {
	uc     *usecase.AuthUsecase
	cookie CookieConfig
}

func NewAuthHandler(uc *usecase.AuthUsecase, cookie CookieConfig) *AuthHandler {
	return &AuthHandler{uc: uc, cookie: cookie}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "メールアドレスとパスワードを入力してください"})
		return
	}
	user, toks, err := h.uc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		respondError(c, err)
		return
	}
	h.setAuthCookies(c, toks)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refresh, _ := c.Cookie(middleware.CookieRefresh)
	toks, err := h.uc.Refresh(c.Request.Context(), refresh)
	if err != nil {
		respondError(c, err)
		return
	}
	h.setAuthCookies(c, toks)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	refresh, _ := c.Cookie(middleware.CookieRefresh)
	_ = h.uc.Logout(c.Request.Context(), refresh)
	h.clearAuthCookies(c)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, fns, err := h.uc.Me(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user, "functions": fns})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "現在のパスワードと新しいパスワードを入力してください"})
		return
	}
	if err := h.uc.ChangePassword(c.Request.Context(), middleware.UserID(c), req.CurrentPassword, req.NewPassword); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHandler) VerifySetToken(c *gin.Context) {
	email, err := h.uc.VerifySetToken(c.Request.Context(), c.Param("token"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"email": email})
}

func (h *AuthHandler) SetPassword(c *gin.Context) {
	var req struct {
		Token    string `json:"token" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "トークンとパスワードを入力してください"})
		return
	}
	if err := h.uc.SetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHandler) setAuthCookies(c *gin.Context, toks usecase.Tokens) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.CookieAccess, toks.Access, h.cookie.AccessMaxAge, "/", "", h.cookie.Secure, true)
	c.SetCookie(middleware.CookieRefresh, toks.Refresh, h.cookie.RefreshMaxAge, "/", "", h.cookie.Secure, true)
}

func (h *AuthHandler) clearAuthCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.CookieAccess, "", -1, "/", "", h.cookie.Secure, true)
	c.SetCookie(middleware.CookieRefresh, "", -1, "/", "", h.cookie.Secure, true)
}
