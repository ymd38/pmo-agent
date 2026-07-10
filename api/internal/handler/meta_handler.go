package handler

import (
	"net/http"

	"pmo-agent/api/internal/usecase"

	"github.com/gin-gonic/gin"
)

// MetaHandler はロール一覧などの参照系を提供する（メンバー管理UIのロール選択等）。
type MetaHandler struct {
	roles usecase.RoleRepository
}

func NewMetaHandler(roles usecase.RoleRepository) *MetaHandler {
	return &MetaHandler{roles: roles}
}

func (h *MetaHandler) Roles(c *gin.Context) {
	roles, err := h.roles.List(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"roles": roles})
}
