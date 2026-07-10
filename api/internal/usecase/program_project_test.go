package usecase

import (
	"context"
	"fmt"
	"testing"

	"pmo-agent/api/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- フェイク ---

type fakeProgramRepo struct {
	byID   map[int]*domain.Program
	byCode map[string]*domain.Program
	seq    int
}

func newFakeProgramRepo() *fakeProgramRepo {
	return &fakeProgramRepo{byID: map[int]*domain.Program{}, byCode: map[string]*domain.Program{}}
}

func (f *fakeProgramRepo) Create(_ context.Context, p *domain.Program) error {
	f.seq++
	p.ID = f.seq
	f.byID[p.ID] = p
	f.byCode[p.Code] = p
	return nil
}
func (f *fakeProgramRepo) FindByID(_ context.Context, id int) (*domain.Program, error) {
	if p, ok := f.byID[id]; ok {
		return p, nil
	}
	return nil, domain.ErrNotFound
}
func (f *fakeProgramRepo) FindByCode(_ context.Context, code string) (*domain.Program, error) {
	if p, ok := f.byCode[code]; ok {
		return p, nil
	}
	return nil, domain.ErrNotFound
}
func (f *fakeProgramRepo) List(_ context.Context) ([]domain.Program, error)  { return nil, nil }
func (f *fakeProgramRepo) Update(_ context.Context, _ *domain.Program) error { return nil }
func (f *fakeProgramRepo) Delete(_ context.Context, id int) error {
	delete(f.byID, id)
	return nil
}
func (f *fakeProgramRepo) MaxSeqNo(_ context.Context, programType string, fiscalYear int) (int, error) {
	max := 0
	for _, p := range f.byID {
		if p.Type == programType && p.FiscalYear == fiscalYear && p.SeqNo > max {
			max = p.SeqNo
		}
	}
	return max, nil
}

type fakeProjectRepo struct {
	byID map[int]*domain.Project
	seq  int
}

func newFakeProjectRepo() *fakeProjectRepo {
	return &fakeProjectRepo{byID: map[int]*domain.Project{}}
}

func (f *fakeProjectRepo) Create(_ context.Context, p *domain.Project) error {
	f.seq++
	p.ID = f.seq
	f.byID[p.ID] = p
	return nil
}
func (f *fakeProjectRepo) FindByID(_ context.Context, id int) (*domain.Project, error) {
	if p, ok := f.byID[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}
func (f *fakeProjectRepo) List(_ context.Context) ([]domain.Project, error) { return nil, nil }
func (f *fakeProjectRepo) ListByProgram(_ context.Context, _ int) ([]domain.Project, error) {
	return nil, nil
}
func (f *fakeProjectRepo) Update(_ context.Context, p *domain.Project) error {
	f.byID[p.ID] = p
	return nil
}
func (f *fakeProjectRepo) Delete(_ context.Context, id int) error { delete(f.byID, id); return nil }
func (f *fakeProjectRepo) CountByProgram(_ context.Context, programID int) (int64, error) {
	var n int64
	for _, p := range f.byID {
		if p.ProgramID == programID {
			n++
		}
	}
	return n, nil
}
func (f *fakeProjectRepo) MaxBranchNo(_ context.Context, programID int) (int, error) {
	max := 0
	for _, p := range f.byID {
		if p.ProgramID == programID && p.BranchNo != nil && *p.BranchNo > max {
			max = *p.BranchNo
		}
	}
	return max, nil
}
func (f *fakeProjectRepo) IssueCode(_ context.Context, projectID, branchNo int, code string) error {
	p := f.byID[projectID]
	if p == nil {
		return domain.ErrNotFound
	}
	// 実 DB の WHERE 句（project_code IS NULL AND status = planning）と同じガードを模す。
	if p.ProjectCode != nil || p.Status != domain.StatusPlanning {
		return fmt.Errorf("%w: このプロジェクトは既にコードが発行されています", domain.ErrConflict)
	}
	p.BranchNo = &branchNo
	p.ProjectCode = &code
	p.Status = domain.StatusActive
	return nil
}
func (f *fakeProjectRepo) AggregateByProgram(_ context.Context) (map[int]domain.ProgramAggregate, error) {
	return map[int]domain.ProgramAggregate{}, nil
}

// --- テスト ---

func TestProjectUsecase_IssueCode(t *testing.T) {
	ctx := context.Background()
	programs := newFakeProgramRepo()
	projects := newFakeProjectRepo()
	puc := NewProjectUsecase(projects, programs)

	prog := &domain.Program{Code: "INV-2026-0001", Name: "投資プログラム"}
	require.NoError(t, programs.Create(ctx, prog))

	p1, err := puc.Create(ctx, prog.ID, CreateProjectInput{Name: "PJ1"})
	require.NoError(t, err)
	p2, err := puc.Create(ctx, prog.ID, CreateProjectInput{Name: "PJ2"})
	require.NoError(t, err)

	t.Run("枝番は連番採番され active になる", func(t *testing.T) {
		issued1, err := puc.IssueCode(ctx, p1.ID)
		require.NoError(t, err)
		require.NotNil(t, issued1.ProjectCode)
		assert.Equal(t, "INV-2026-0001-001", *issued1.ProjectCode)
		assert.Equal(t, domain.StatusActive, issued1.Status)

		issued2, err := puc.IssueCode(ctx, p2.ID)
		require.NoError(t, err)
		assert.Equal(t, "INV-2026-0001-002", *issued2.ProjectCode)
	})

	t.Run("発行済みプロジェクトの再発行は ErrConflict（不変）", func(t *testing.T) {
		_, err := puc.IssueCode(ctx, p1.ID)
		assert.ErrorIs(t, err, domain.ErrConflict)
	})

	t.Run("planning 以外は発行不可", func(t *testing.T) {
		p3, _ := puc.Create(ctx, prog.ID, CreateProjectInput{Name: "PJ3"})
		stored := projects.byID[p3.ID]
		stored.Status = domain.StatusCancelled
		_, err := puc.IssueCode(ctx, p3.ID)
		assert.ErrorIs(t, err, domain.ErrConflict)
	})

	// usecase の事前チェックをすり抜けた 2 度目の発行（並行リクエスト相当）が
	// 永続化層で拒否されることを保証する。project_code の不変性は DB のガードが最終防衛線。
	t.Run("発行済みへの2度目の永続化は ErrConflict（TOCTOU 防御）", func(t *testing.T) {
		p4, err := puc.Create(ctx, prog.ID, CreateProjectInput{Name: "PJ4"})
		require.NoError(t, err)
		require.NoError(t, projects.IssueCode(ctx, p4.ID, 1, "INV-2026-0001-099"))

		err = projects.IssueCode(ctx, p4.ID, 2, "INV-2026-0001-100")
		assert.ErrorIs(t, err, domain.ErrConflict)
		assert.Equal(t, "INV-2026-0001-099", *projects.byID[p4.ID].ProjectCode, "既存コードは書き換わらない")
	})
}

func TestProjectUsecase_Update_StatusTransition(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		from    domain.ProjectStatus
		to      domain.ProjectStatus
		wantErr error // nil なら遷移可
	}{
		{"同一ステータス（フィールド編集）は可", domain.StatusActive, domain.StatusActive, nil},
		{"planning→cancelled（中止）は可", domain.StatusPlanning, domain.StatusCancelled, nil},
		{"planning→active はコード発行専用のため不可", domain.StatusPlanning, domain.StatusActive, domain.ErrConflict},
		{"planning→completed は不可", domain.StatusPlanning, domain.StatusCompleted, domain.ErrConflict},
		{"active→completed（完了）は可", domain.StatusActive, domain.StatusCompleted, nil},
		{"active→cancelled（中止）は可", domain.StatusActive, domain.StatusCancelled, nil},
		{"active→planning（差し戻し）は不可", domain.StatusActive, domain.StatusPlanning, domain.ErrConflict},
		{"completed→active（終端からの復活）は不可", domain.StatusCompleted, domain.StatusActive, domain.ErrConflict},
		{"cancelled→active（終端からの復活）は不可", domain.StatusCancelled, domain.StatusActive, domain.ErrConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			programs := newFakeProgramRepo()
			projects := newFakeProjectRepo()
			prog := &domain.Program{Code: "INV-2026-0001", Name: "P"}
			require.NoError(t, programs.Create(ctx, prog))
			puc := NewProjectUsecase(projects, programs)
			p, err := puc.Create(ctx, prog.ID, CreateProjectInput{Name: "PJ"})
			require.NoError(t, err)
			projects.byID[p.ID].Status = tc.from // 起点ステータスを強制設定

			got, err := puc.Update(ctx, p.ID, UpdateProjectInput{Name: "PJ", Status: tc.to})
			if tc.wantErr == nil {
				require.NoError(t, err)
				assert.Equal(t, tc.to, got.Status)
			} else {
				assert.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}

func TestProjectUsecase_Create_RequiresProgram(t *testing.T) {
	puc := NewProjectUsecase(newFakeProjectRepo(), newFakeProgramRepo())
	_, err := puc.Create(context.Background(), 999, CreateProjectInput{Name: "孤児PJ"})
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestProgramUsecase_Delete_BlockedWithChildren(t *testing.T) {
	ctx := context.Background()
	programs := newFakeProgramRepo()
	projects := newFakeProjectRepo()
	guc := NewProgramUsecase(programs, projects)
	puc := NewProjectUsecase(projects, programs)

	prog := &domain.Program{Code: "MNT-2026-0001", Name: "保守"}
	require.NoError(t, programs.Create(ctx, prog))

	t.Run("配下なしなら削除可", func(t *testing.T) {
		require.NoError(t, guc.Delete(ctx, prog.ID))
	})

	t.Run("配下ありなら ErrConflict", func(t *testing.T) {
		require.NoError(t, programs.Create(ctx, prog))
		_, err := puc.Create(ctx, prog.ID, CreateProjectInput{Name: "子PJ"})
		require.NoError(t, err)
		err = guc.Delete(ctx, prog.ID)
		assert.ErrorIs(t, err, domain.ErrConflict)
	})
}

func TestProgramUsecase_Create_AutoNumbering(t *testing.T) {
	ctx := context.Background()
	programs := newFakeProgramRepo()
	guc := NewProgramUsecase(programs, newFakeProjectRepo())

	mustCode := func(in CreateProgramInput) string {
		p, err := guc.Create(ctx, in)
		require.NoError(t, err)
		return p.Code
	}

	// (type, fiscal_year) ごとに連番が自動採番される。
	assert.Equal(t, "INV-2026-0001", mustCode(CreateProgramInput{Type: "INV", FiscalYear: 2026, Name: "A"}))
	assert.Equal(t, "INV-2026-0002", mustCode(CreateProgramInput{Type: "inv", FiscalYear: 2026, Name: "B"})) // 小文字は大文字化
	assert.Equal(t, "MNT-2026-0001", mustCode(CreateProgramInput{Type: "MNT", FiscalYear: 2026, Name: "C"})) // 種別が違えば別連番
	assert.Equal(t, "INV-2027-0001", mustCode(CreateProgramInput{Type: "INV", FiscalYear: 2027, Name: "D"})) // 年度が違えば別連番

	t.Run("不正な種別は ErrValidation", func(t *testing.T) {
		_, err := guc.Create(ctx, CreateProgramInput{Type: "inv-1", FiscalYear: 2026, Name: "X"})
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("会計年度が範囲外は ErrValidation", func(t *testing.T) {
		_, err := guc.Create(ctx, CreateProgramInput{Type: "INV", FiscalYear: 1800, Name: "X"})
		assert.ErrorIs(t, err, domain.ErrValidation)
	})
}
