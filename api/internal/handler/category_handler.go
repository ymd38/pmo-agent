package handler

import (
	"net/http"

	"pmo-agent/api/internal/usecase"

	"github.com/gin-gonic/gin"
)

type CategoryHandler struct {
	uc *usecase.CategoryUsecase
}

func NewCategoryHandler(uc *usecase.CategoryUsecase) *CategoryHandler {
	return &CategoryHandler{uc: uc}
}

func (h *CategoryHandler) List(c *gin.Context) {
	cats, err := h.uc.ListCategories(c.Request.Context(), includeInactive(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"categories": cats})
}

func (h *CategoryHandler) Create(c *gin.Context) {
	var req categoryReq
	if !bindJSON(c, &req, "入力内容を確認してください") {
		return
	}
	cat, err := h.uc.CreateCategory(c.Request.Context(), req.toInput())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"category": cat})
}

func (h *CategoryHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req categoryReq
	if !bindJSON(c, &req, "入力内容を確認してください") {
		return
	}
	cat, err := h.uc.UpdateCategory(c.Request.Context(), id, req.toInput())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"category": cat})
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.uc.DeactivateCategory(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *CategoryHandler) Reactivate(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.uc.ReactivateCategory(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *CategoryHandler) ListValues(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	vals, err := h.uc.ListValues(c.Request.Context(), id, includeInactive(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"values": vals})
}

func (h *CategoryHandler) CreateValue(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req valueReq
	if !bindJSON(c, &req, "入力内容を確認してください") {
		return
	}
	v, err := h.uc.CreateValue(c.Request.Context(), id, req.toInput())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"value": v})
}

func (h *CategoryHandler) UpdateValue(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	valueID, ok := pathValueID(c)
	if !ok {
		return
	}
	var req valueReq
	if !bindJSON(c, &req, "入力内容を確認してください") {
		return
	}
	v, err := h.uc.UpdateValue(c.Request.Context(), id, valueID, req.toInput())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"value": v})
}

func (h *CategoryHandler) DeleteValue(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	valueID, ok := pathValueID(c)
	if !ok {
		return
	}
	if err := h.uc.DeactivateValue(c.Request.Context(), id, valueID); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *CategoryHandler) ReactivateValue(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	valueID, ok := pathValueID(c)
	if !ok {
		return
	}
	if err := h.uc.ReactivateValue(c.Request.Context(), id, valueID); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// pathValueID は :valueId パスセグメントを解析する。不正なら 400 を返して false。
func pathValueID(c *gin.Context) (int, bool) {
	return pathParam(c, "valueId", "値IDが不正です")
}

type categoryReq struct {
	Code        string `json:"code"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	IsRequired  bool   `json:"is_required"`
	SortOrder   int    `json:"sort_order"`
}

func (r categoryReq) toInput() usecase.CategoryInput {
	return usecase.CategoryInput{
		Code:        r.Code,
		Name:        r.Name,
		Description: r.Description,
		IsRequired:  r.IsRequired,
		SortOrder:   r.SortOrder,
	}
}

type valueReq struct {
	Code      string `json:"code"`
	Label     string `json:"label" binding:"required"`
	SortOrder int    `json:"sort_order"`
}

func (r valueReq) toInput() usecase.ValueInput {
	return usecase.ValueInput{Code: r.Code, Label: r.Label, SortOrder: r.SortOrder}
}

func includeInactive(c *gin.Context) bool {
	return c.Query("include_inactive") == "true"
}
