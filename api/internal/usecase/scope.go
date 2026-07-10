package usecase

import (
	"context"
	"fmt"
	"sort"

	"pmo-agent/api/internal/domain"
)

// ScopeUsecase はリクエストユーザーのロールから、閲覧・操作を許可する
// プロジェクト範囲（domain.ProjectScope）を解決する。
//
// スコープ規則（SPEC.md「初期ロール別権限マッピング」）:
//   - pmo_admin / executive … 全プロジェクト
//   - pm / member           … 担当PJ（project_members ∪ projects.pm_id = user）
//   - planner               … 自起案PJ（projects.created_by = user）
//
// 複数ロールを持つ場合は各ロールの許可集合の和集合を返す。
type ScopeUsecase struct {
	users    UserRepository
	projects ProjectRepository
	members  ProjectMemberRepository
}

func NewScopeUsecase(users UserRepository, projects ProjectRepository, members ProjectMemberRepository) *ScopeUsecase {
	return &ScopeUsecase{users: users, projects: projects, members: members}
}

// ResolveProjectScope はユーザーのロールを解決し、許可プロジェクト範囲を返す。
func (uc *ScopeUsecase) ResolveProjectScope(ctx context.Context, userID int) (domain.ProjectScope, error) {
	roles, err := uc.users.RolesByUserID(ctx, userID)
	if err != nil {
		return domain.ProjectScope{}, fmt.Errorf("usecase.Scope.Resolve roles: %w", err)
	}

	var assignedRole, plannerRole bool
	for _, r := range roles {
		switch r.Code {
		case domain.RoleCodePMOAdmin, domain.RoleCodeExecutive:
			// 全件許可。以降の集合計算は不要。
			return domain.UnrestrictedScope(), nil
		case domain.RoleCodePM, domain.RoleCodeMember:
			assignedRole = true
		case domain.RoleCodePlanner:
			plannerRole = true
		}
	}

	idset := map[int]struct{}{}
	if assignedRole {
		memberIDs, err := uc.members.ProjectIDsByUser(ctx, userID)
		if err != nil {
			return domain.ProjectScope{}, fmt.Errorf("usecase.Scope.Resolve members: %w", err)
		}
		pmIDs, err := uc.projects.IDsByPM(ctx, userID)
		if err != nil {
			return domain.ProjectScope{}, fmt.Errorf("usecase.Scope.Resolve pm: %w", err)
		}
		addAll(idset, memberIDs)
		addAll(idset, pmIDs)
	}
	if plannerRole {
		creatorIDs, err := uc.projects.IDsByCreator(ctx, userID)
		if err != nil {
			return domain.ProjectScope{}, fmt.Errorf("usecase.Scope.Resolve creator: %w", err)
		}
		addAll(idset, creatorIDs)
	}

	ids := make([]int, 0, len(idset))
	for id := range idset {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return domain.ProjectScope{All: false, ProjectIDs: ids}, nil
}

func addAll(set map[int]struct{}, ids []int) {
	for _, id := range ids {
		set[id] = struct{}{}
	}
}
