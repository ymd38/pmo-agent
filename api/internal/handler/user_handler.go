package handler

import (
	"net/http"

	"pmo-agent/api/internal/domain"
	"pmo-agent/api/internal/usecase"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	uc *usecase.UserUsecase
}

func NewUserHandler(uc *usecase.UserUsecase) *UserHandler { return &UserHandler{uc: uc} }

func (h *UserHandler) List(c *gin.Context) {
	users, err := h.uc.List(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *UserHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	user, err := h.uc.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *UserHandler) Create(c *gin.Context) {
	var req struct {
		Email   string `json:"email" binding:"required,email"`
		Name    string `json:"name" binding:"required"`
		Grade   string `json:"grade" binding:"required"`
		RoleIDs []int  `json:"role_ids" binding:"required"`
	}
	if !bindJSON(c, &req, "入力内容を確認してください") {
		return
	}
	user, link, err := h.uc.Create(c.Request.Context(), usecase.CreateInput{
		Email:   req.Email,
		Name:    req.Name,
		Grade:   domain.Grade(req.Grade),
		RoleIDs: req.RoleIDs,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": user, "set_password_url": link})
}

func (h *UserHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req struct {
		Name     string `json:"name" binding:"required"`
		Grade    string `json:"grade" binding:"required"`
		IsActive *bool  `json:"is_active"`
		RoleIDs  []int  `json:"role_ids" binding:"required"`
	}
	if !bindJSON(c, &req, "入力内容を確認してください") {
		return
	}
	user, err := h.uc.Update(c.Request.Context(), id, usecase.UpdateInput{
		Name:     req.Name,
		Grade:    domain.Grade(req.Grade),
		IsActive: req.IsActive,
		RoleIDs:  req.RoleIDs,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.uc.Deactivate(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *UserHandler) ReissueLink(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	link, err := h.uc.ReissueLink(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"set_password_url": link})
}

// pathID は :id パスパラメータを int で取り出す。不正なら 400 を返して false。
func pathID(c *gin.Context) (int, bool) {
	return pathParam(c, "id", "IDが不正です")
}
