package handler

import (
	"net/http"

	"pmo-agent/api/internal/usecase"

	"github.com/gin-gonic/gin"
)

type MemberHandler struct {
	uc *usecase.MemberUsecase
}

func NewMemberHandler(uc *usecase.MemberUsecase) *MemberHandler {
	return &MemberHandler{uc: uc}
}

// assignReq は POST /projects/:id/members の入力。
type assignReq struct {
	UserID            int      `json:"user_id" binding:"required"`
	AllocationPercent *float64 `json:"allocation_percent"`
	StartDate         string   `json:"start_date"`
	EndDate           string   `json:"end_date"`
}

// updateMemberReq は PUT /projects/:id/members/:userId の入力。
type updateMemberReq struct {
	AllocationPercent *float64 `json:"allocation_percent"`
	StartDate         string   `json:"start_date"`
	EndDate           string   `json:"end_date"`
}

func (h *MemberHandler) List(c *gin.Context) {
	projectID, ok := pathID(c)
	if !ok {
		return
	}
	scope, ok := requireScope(c)
	if !ok {
		return
	}
	members, err := h.uc.List(c.Request.Context(), projectID, scope)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"members": members})
}

func (h *MemberHandler) Assign(c *gin.Context) {
	projectID, ok := pathID(c)
	if !ok {
		return
	}
	scope, ok := requireScope(c)
	if !ok {
		return
	}
	var req assignReq
	if !bindJSON(c, &req, "アサインするユーザーを指定してください") {
		return
	}
	start, end, derr := parseDates(req.StartDate, req.EndDate)
	if derr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日付は YYYY-MM-DD 形式で入力してください"})
		return
	}
	m, err := h.uc.Assign(c.Request.Context(), projectID, usecase.AssignMemberInput{
		UserID:            req.UserID,
		AllocationPercent: req.AllocationPercent,
		StartDate:         start,
		EndDate:           end,
	}, scope)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"member": m})
}

func (h *MemberHandler) Update(c *gin.Context) {
	projectID, ok := pathID(c)
	if !ok {
		return
	}
	userID, ok := pathUserID(c)
	if !ok {
		return
	}
	scope, ok := requireScope(c)
	if !ok {
		return
	}
	var req updateMemberReq
	if !bindJSON(c, &req, "入力内容を確認してください") {
		return
	}
	start, end, derr := parseDates(req.StartDate, req.EndDate)
	if derr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日付は YYYY-MM-DD 形式で入力してください"})
		return
	}
	m, err := h.uc.Update(c.Request.Context(), projectID, userID, usecase.UpdateMemberInput{
		AllocationPercent: req.AllocationPercent,
		StartDate:         start,
		EndDate:           end,
	}, scope)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"member": m})
}

func (h *MemberHandler) Delete(c *gin.Context) {
	projectID, ok := pathID(c)
	if !ok {
		return
	}
	userID, ok := pathUserID(c)
	if !ok {
		return
	}
	scope, ok := requireScope(c)
	if !ok {
		return
	}
	if err := h.uc.Unassign(c.Request.Context(), projectID, userID, scope); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// pathUserID は :userId パスパラメータを int で取り出す。不正なら 400 を返して false。
func pathUserID(c *gin.Context) (int, bool) {
	return pathParam(c, "userId", "ユーザーIDが不正です")
}
