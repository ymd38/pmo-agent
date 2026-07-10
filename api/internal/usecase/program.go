package usecase

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"pmo-agent/api/internal/domain"
)

// 種別プレフィックスは英大文字2〜5文字（例: INV / MNT / OPS）。
var programTypePattern = regexp.MustCompile(`^[A-Z]{2,5}$`)

// ProgramUsecase はプログラム（PMO集計単位・コードプレフィックス）の管理を担う。
type ProgramUsecase struct {
	programs ProgramRepository
	projects ProjectRepository
}

func NewProgramUsecase(programs ProgramRepository, projects ProjectRepository) *ProgramUsecase {
	return &ProgramUsecase{programs: programs, projects: projects}
}

// ProgramView はプログラム＋集計値（一覧用）。
type ProgramView struct {
	Program   domain.Program          `json:"program"`
	Aggregate domain.ProgramAggregate `json:"aggregate"`
}

// ProgramDetail はプログラム＋集計＋配下プロジェクト一覧。
type ProgramDetail struct {
	Program   domain.Program          `json:"program"`
	Aggregate domain.ProgramAggregate `json:"aggregate"`
	Projects  []domain.Project        `json:"projects"`
}

type CreateProgramInput struct {
	Type        string
	FiscalYear  int
	Name        string
	Description string
	CreatedBy   int
}

func (uc *ProgramUsecase) List(ctx context.Context) ([]ProgramView, error) {
	programs, err := uc.programs.List(ctx)
	if err != nil {
		return nil, err
	}
	aggs, err := uc.projects.AggregateByProgram(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]ProgramView, 0, len(programs))
	for _, p := range programs {
		views = append(views, ProgramView{Program: p, Aggregate: aggOrEmpty(aggs, p.ID)})
	}
	return views, nil
}

func (uc *ProgramUsecase) GetDetail(ctx context.Context, id int) (*ProgramDetail, error) {
	p, err := uc.programs.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	projects, err := uc.projects.ListByProgram(ctx, id)
	if err != nil {
		return nil, err
	}
	aggs, err := uc.projects.AggregateByProgram(ctx)
	if err != nil {
		return nil, err
	}
	return &ProgramDetail{Program: *p, Aggregate: aggOrEmpty(aggs, id), Projects: projects}, nil
}

// Create は種別＋会計年度からプログラムを作成し、連番(seq_no)を自動採番する。
// code は "種別-年度-連番(0埋め4桁)" で生成（例: INV-2026-0001）。
func (uc *ProgramUsecase) Create(ctx context.Context, in CreateProgramInput) (*domain.Program, error) {
	pType := strings.ToUpper(strings.TrimSpace(in.Type))
	if !programTypePattern.MatchString(pType) {
		return nil, fmt.Errorf("%w: 種別は英大文字2〜5文字で指定してください（例: INV）", domain.ErrValidation)
	}
	if in.FiscalYear < 2000 || in.FiscalYear > 2999 {
		return nil, fmt.Errorf("%w: 会計年度が不正です", domain.ErrValidation)
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("%w: プログラム名は必須です", domain.ErrValidation)
	}

	maxSeq, err := uc.programs.MaxSeqNo(ctx, pType, in.FiscalYear)
	if err != nil {
		return nil, fmt.Errorf("usecase.Program.Create seq: %w", err)
	}
	seq := maxSeq + 1

	p := &domain.Program{
		Code:        fmt.Sprintf("%s-%d-%04d", pType, in.FiscalYear, seq),
		Type:        pType,
		FiscalYear:  in.FiscalYear,
		SeqNo:       seq,
		Name:        strings.TrimSpace(in.Name),
		Description: in.Description,
		CreatedBy:   in.CreatedBy,
	}
	if err := uc.programs.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("usecase.Program.Create: %w", err)
	}
	return p, nil
}

func (uc *ProgramUsecase) Update(ctx context.Context, id int, name, description string) (*domain.Program, error) {
	p, err := uc.programs.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("%w: プログラム名は必須です", domain.ErrValidation)
	}
	p.Name = strings.TrimSpace(name)
	p.Description = description
	if err := uc.programs.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("usecase.Program.Update: %w", err)
	}
	return p, nil
}

// Delete は配下プロジェクトが存在する場合は拒否する（409）。
func (uc *ProgramUsecase) Delete(ctx context.Context, id int) error {
	if _, err := uc.programs.FindByID(ctx, id); err != nil {
		return err
	}
	n, err := uc.projects.CountByProgram(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("%w: 配下にプロジェクトが存在するため削除できません", domain.ErrConflict)
	}
	return uc.programs.Delete(ctx, id)
}

func aggOrEmpty(aggs map[int]domain.ProgramAggregate, id int) domain.ProgramAggregate {
	if a, ok := aggs[id]; ok {
		return a
	}
	return domain.ProgramAggregate{StatusCounts: map[string]int{}}
}
