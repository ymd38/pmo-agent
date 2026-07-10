package usecase

import (
	"context"
	"fmt"
	"time"

	"pmo-agent/api/internal/domain"
)

// MemberUsecase はプロジェクトへのメンバーアサイン CRUD を担う。
// 全操作でスコープ（担当PJのみ）を強制し、担当外PJへの操作は存在秘匿のため ErrNotFound を返す。
type MemberUsecase struct {
	members  ProjectMemberRepository
	projects ProjectRepository
	users    UserRepository
}

func NewMemberUsecase(members ProjectMemberRepository, projects ProjectRepository, users UserRepository) *MemberUsecase {
	return &MemberUsecase{members: members, projects: projects, users: users}
}

type AssignMemberInput struct {
	UserID            int
	AllocationPercent *float64
	StartDate         *time.Time
	EndDate           *time.Time
}

type UpdateMemberInput struct {
	AllocationPercent *float64
	StartDate         *time.Time
	EndDate           *time.Time
}

// List は指定プロジェクトのメンバー一覧を返す（スコープ外は ErrNotFound）。
func (uc *MemberUsecase) List(ctx context.Context, projectID int, scope domain.ProjectScope) ([]domain.ProjectMember, error) {
	if err := uc.ensureProject(ctx, projectID, scope); err != nil {
		return nil, err
	}
	return uc.members.ListByProject(ctx, projectID)
}

// Assign はメンバーをアサインする。対象ユーザーは有効（is_active=true）である必要がある。
func (uc *MemberUsecase) Assign(ctx context.Context, projectID int, in AssignMemberInput, scope domain.ProjectScope) (*domain.ProjectMember, error) {
	if err := uc.ensureProject(ctx, projectID, scope); err != nil {
		return nil, err
	}
	u, err := uc.users.FindByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if !u.IsActive {
		return nil, fmt.Errorf("%w: 無効化されたユーザーはアサインできません", domain.ErrValidation)
	}
	if err := validateAllocation(in.AllocationPercent); err != nil {
		return nil, err
	}
	m := &domain.ProjectMember{
		ProjectID:         projectID,
		UserID:            in.UserID,
		AllocationPercent: in.AllocationPercent,
		StartDate:         in.StartDate,
		EndDate:           in.EndDate,
	}
	if err := uc.members.Assign(ctx, m); err != nil {
		return nil, fmt.Errorf("usecase.Member.Assign: %w", err)
	}
	return m, nil
}

// Update はアサイン情報（割合・期間）を更新する。
func (uc *MemberUsecase) Update(ctx context.Context, projectID, userID int, in UpdateMemberInput, scope domain.ProjectScope) (*domain.ProjectMember, error) {
	if err := uc.ensureProject(ctx, projectID, scope); err != nil {
		return nil, err
	}
	if err := validateAllocation(in.AllocationPercent); err != nil {
		return nil, err
	}
	m := &domain.ProjectMember{
		ProjectID:         projectID,
		UserID:            userID,
		AllocationPercent: in.AllocationPercent,
		StartDate:         in.StartDate,
		EndDate:           in.EndDate,
	}
	if err := uc.members.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Unassign はアサインを解除する。
func (uc *MemberUsecase) Unassign(ctx context.Context, projectID, userID int, scope domain.ProjectScope) error {
	if err := uc.ensureProject(ctx, projectID, scope); err != nil {
		return err
	}
	return uc.members.Unassign(ctx, projectID, userID)
}

// ensureProject はプロジェクトの存在とスコープ内であることを確認する。
// スコープ外は存在秘匿のため（存在確認前に）ErrNotFound を返す。
func (uc *MemberUsecase) ensureProject(ctx context.Context, projectID int, scope domain.ProjectScope) error {
	if !scope.Allows(projectID) {
		return domain.ErrNotFound
	}
	if _, err := uc.projects.FindByID(ctx, projectID); err != nil {
		return err
	}
	return nil
}

func validateAllocation(pct *float64) error {
	if pct == nil {
		return nil
	}
	if *pct < 0 || *pct > 100 {
		return fmt.Errorf("%w: 工数割合は 0〜100 の範囲で指定してください", domain.ErrValidation)
	}
	return nil
}
