package handler

import (
	"net/http"

	"pmo-agent/api/internal/middleware"
	"pmo-agent/api/internal/usecase"

	"github.com/gin-gonic/gin"
)

type ProgramHandler struct {
	uc *usecase.ProgramUsecase
}

func NewProgramHandler(uc *usecase.ProgramUsecase) *ProgramHandler {
	return &ProgramHandler{uc: uc}
}

func (h *ProgramHandler) List(c *gin.Context) {
	views, err := h.uc.List(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"programs": views})
}

func (h *ProgramHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	detail, err := h.uc.GetDetail(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, detail)
}

// ListProjects は GET /programs/:id/projects。担当PJスコープでフィルタする。
func (h *ProgramHandler) ListProjects(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	scope, ok := requireScope(c)
	if !ok {
		return
	}
	projects, err := h.uc.ListProjectsScoped(c.Request.Context(), id, scope)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

func (h *ProgramHandler) Create(c *gin.Context) {
	var req struct {
		Type        string `json:"type" binding:"required"`
		FiscalYear  int    `json:"fiscal_year" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if !bindJSON(c, &req, "種別・会計年度・名称を入力してください") {
		return
	}
	p, err := h.uc.Create(c.Request.Context(), usecase.CreateProgramInput{
		Type:        req.Type,
		FiscalYear:  req.FiscalYear,
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   middleware.UserID(c),
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"program": p})
}

func (h *ProgramHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if !bindJSON(c, &req, "名称を入力してください") {
		return
	}
	p, err := h.uc.Update(c.Request.Context(), id, req.Name, req.Description)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"program": p})
}

func (h *ProgramHandler) Delete(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.uc.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
