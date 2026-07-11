package handler

import (
	"net/http"

	"pmo-agent/api/internal/usecase"

	"github.com/gin-gonic/gin"
)

type AttributeHandler struct {
	uc *usecase.AttributeUsecase
}

func NewAttributeHandler(uc *usecase.AttributeUsecase) *AttributeHandler {
	return &AttributeHandler{uc: uc}
}

// List は GET /projects/:id/attributes。
func (h *AttributeHandler) List(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	attrs, err := h.uc.List(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"attributes": attrs})
}

type attributeReq struct {
	ValueID int `json:"value_id" binding:"required"`
}

// Assign は POST /projects/:id/attributes。value_id のみ受け取り、カテゴリは値から導出する。
func (h *AttributeHandler) Assign(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req attributeReq
	if !bindJSON(c, &req, "属性値を指定してください") {
		return
	}
	a, err := h.uc.Assign(c.Request.Context(), id, req.ValueID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"assignment": a})
}

// Delete は DELETE /projects/:id/attributes/:valueId。
func (h *AttributeHandler) Delete(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	valueID, ok := pathValueID(c)
	if !ok {
		return
	}
	if err := h.uc.Unassign(c.Request.Context(), id, valueID); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
