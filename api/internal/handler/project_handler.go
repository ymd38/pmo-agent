package handler

import (
	"net/http"
	"time"

	"pmo-agent/api/internal/domain"
	"pmo-agent/api/internal/middleware"
	"pmo-agent/api/internal/usecase"

	"github.com/gin-gonic/gin"
)

type ProjectHandler struct {
	uc *usecase.ProjectUsecase
}

func NewProjectHandler(uc *usecase.ProjectUsecase) *ProjectHandler {
	return &ProjectHandler{uc: uc}
}

// projectReq は作成・更新で共通の入力。日付は "YYYY-MM-DD" 文字列で受ける。
type projectReq struct {
	Name             string `json:"name" binding:"required"`
	Description      string `json:"description"`
	PMID             *int   `json:"pm_id"`
	ApproverID       *int   `json:"approver_id"`
	Vendor           string `json:"vendor"`
	Budget           *int64 `json:"budget"`
	StartDate        string `json:"start_date"`
	EndDate          string `json:"end_date"`
	Status           string `json:"status"`
	BacklogProjectID string `json:"backlog_project_id"`
}

func (h *ProjectHandler) List(c *gin.Context) {
	scope, ok := requireScope(c)
	if !ok {
		return
	}
	projects, err := h.uc.List(c.Request.Context(), scope)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

func (h *ProjectHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	scope, ok := requireScope(c)
	if !ok {
		return
	}
	p, err := h.uc.Get(c.Request.Context(), id, scope)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"project": p})
}

// CreateUnderProgram は POST /programs/:id/projects。
func (h *ProjectHandler) CreateUnderProgram(c *gin.Context) {
	programID, ok := pathID(c)
	if !ok {
		return
	}
	var req projectReq
	if !bindJSON(c, &req, "入力内容を確認してください") {
		return
	}
	start, end, derr := parseDates(req.StartDate, req.EndDate)
	if derr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日付は YYYY-MM-DD 形式で入力してください"})
		return
	}
	p, err := h.uc.Create(c.Request.Context(), programID, usecase.CreateProjectInput{
		Name:             req.Name,
		Description:      req.Description,
		PMID:             req.PMID,
		ApproverID:       req.ApproverID,
		Vendor:           req.Vendor,
		Budget:           req.Budget,
		StartDate:        start,
		EndDate:          end,
		BacklogProjectID: req.BacklogProjectID,
		CreatedBy:        middleware.UserID(c),
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"project": p})
}

func (h *ProjectHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req projectReq
	if !bindJSON(c, &req, "入力内容を確認してください") {
		return
	}
	start, end, derr := parseDates(req.StartDate, req.EndDate)
	if derr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日付は YYYY-MM-DD 形式で入力してください"})
		return
	}
	p, err := h.uc.Update(c.Request.Context(), id, usecase.UpdateProjectInput{
		Name:             req.Name,
		Description:      req.Description,
		PMID:             req.PMID,
		ApproverID:       req.ApproverID,
		Vendor:           req.Vendor,
		Budget:           req.Budget,
		StartDate:        start,
		EndDate:          end,
		Status:           domain.ProjectStatus(req.Status),
		BacklogProjectID: req.BacklogProjectID,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"project": p})
}

func (h *ProjectHandler) Delete(c *gin.Context) {
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

// IssueCode は POST /projects/:id/issue-code。枝番採番＋active遷移。
func (h *ProjectHandler) IssueCode(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	p, err := h.uc.IssueCode(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"project": p})
}

func parseDates(startStr, endStr string) (*time.Time, *time.Time, error) {
	start, err := parseDate(startStr)
	if err != nil {
		return nil, nil, err
	}
	end, err := parseDate(endStr)
	if err != nil {
		return nil, nil, err
	}
	return start, end, nil
}

func parseDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
