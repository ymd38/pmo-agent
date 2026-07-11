package usecase

import (
	"context"

	"pmo-agent/api/internal/domain"
)

// usecase が依存するリポジトリ／サービスのインターフェース。
// 具体実装は repository / infra に置き、di でバインドする（依存性逆転）。

type UserRepository interface {
	FindByID(ctx context.Context, id int) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, u *domain.User, roleIDs []int) error
	Update(ctx context.Context, u *domain.User, roleIDs []int) error
	UpdatePasswordHash(ctx context.Context, userID int, hash string) error
	Deactivate(ctx context.Context, id int) error
	List(ctx context.Context) ([]domain.User, error)
	FunctionsByUserID(ctx context.Context, userID int) ([]string, error)
	RolesByUserID(ctx context.Context, userID int) ([]domain.Role, error)
}

type PasswordSetTokenRepository interface {
	Create(ctx context.Context, t *domain.PasswordSetToken) error
	FindByHash(ctx context.Context, hash string) (*domain.PasswordSetToken, error)
	MarkUsed(ctx context.Context, id int) error
	InvalidateForUser(ctx context.Context, userID int) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, t *domain.RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error)
	Revoke(ctx context.Context, id int) error
	// Rotate は旧トークンの失効（CASガード）と新トークンの発行を1トランザクションで原子的に行う。
	// 旧トークンが既に失効済みなら並行リプレイとみなし domain.ErrTokenReuse を返す。
	Rotate(ctx context.Context, oldID int, newTok *domain.RefreshToken) error
	RevokeAllForUser(ctx context.Context, userID int) error
}

type RoleRepository interface {
	List(ctx context.Context) ([]domain.Role, error)
	AllIDsExist(ctx context.Context, ids []int) (bool, error)
}

type CategoryRepository interface {
	ListCategories(ctx context.Context, includeInactive bool) ([]domain.Category, error)
	CreateCategory(ctx context.Context, c *domain.Category) error
	UpdateCategory(ctx context.Context, c *domain.Category) error
	DeactivateCategory(ctx context.Context, id int) error
	ReactivateCategory(ctx context.Context, id int) error
	ListValues(ctx context.Context, categoryID int, includeInactive bool) ([]domain.CategoryValue, error)
	FindValueByID(ctx context.Context, id int) (*domain.CategoryValue, error)
	CreateValue(ctx context.Context, v *domain.CategoryValue) error
	UpdateValue(ctx context.Context, v *domain.CategoryValue) error
	DeactivateValue(ctx context.Context, id int) error
	ReactivateValue(ctx context.Context, id int) error
}

type AttributeRepository interface {
	ListByProject(ctx context.Context, projectID int) ([]domain.ProjectAttribute, error)
	Exists(ctx context.Context, projectID, valueID int) (bool, error)
	Assign(ctx context.Context, a *domain.AttributeAssignment) error
	// Unassign は紐付けを物理削除する（アサインは履歴ではなく現在の分類なので削除可）。
	// 対象が無ければ ErrNotFound。
	Unassign(ctx context.Context, projectID, valueID int) error
}

type ProgramRepository interface {
	Create(ctx context.Context, p *domain.Program) error
	FindByID(ctx context.Context, id int) (*domain.Program, error)
	FindByCode(ctx context.Context, code string) (*domain.Program, error)
	List(ctx context.Context) ([]domain.Program, error)
	Update(ctx context.Context, p *domain.Program) error
	Delete(ctx context.Context, id int) error
	// MaxSeqNo は (type, fiscalYear) 内の最大連番を返す（0=未使用）。自動採番に使う。
	MaxSeqNo(ctx context.Context, programType string, fiscalYear int) (int, error)
}

type ProjectRepository interface {
	Create(ctx context.Context, p *domain.Project) error
	FindByID(ctx context.Context, id int) (*domain.Project, error)
	List(ctx context.Context) ([]domain.Project, error)
	ListByIDs(ctx context.Context, ids []int) ([]domain.Project, error)
	ListByProgram(ctx context.Context, programID int) ([]domain.Project, error)
	Update(ctx context.Context, p *domain.Project) error
	Delete(ctx context.Context, id int) error
	CountByProgram(ctx context.Context, programID int) (int64, error)
	MaxBranchNo(ctx context.Context, programID int) (int, error)
	IssueCode(ctx context.Context, projectID, branchNo int, code string) error
	AggregateByProgram(ctx context.Context) (map[int]domain.ProgramAggregate, error)
	// IDsByPM / IDsByCreator はスコープ解決（担当PJ / 自起案PJ）で使う逆引き。
	IDsByPM(ctx context.Context, userID int) ([]int, error)
	IDsByCreator(ctx context.Context, userID int) ([]int, error)
}

// ProjectMemberRepository はプロジェクトへのメンバーアサインを永続化する。
type ProjectMemberRepository interface {
	ListByProject(ctx context.Context, projectID int) ([]domain.ProjectMember, error)
	Assign(ctx context.Context, m *domain.ProjectMember) error
	Update(ctx context.Context, m *domain.ProjectMember) error
	// Unassign はアサインを物理削除する（対象が無ければ ErrNotFound）。
	Unassign(ctx context.Context, projectID, userID int) error
	// ProjectIDsByUser はスコープ解決で使う「担当PJ集合」の逆引き。
	ProjectIDsByUser(ctx context.Context, userID int) ([]int, error)
}

// Hasher は bcrypt 実装（infra.PasswordHasher）。
type Hasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) error
}

// AccessTokenIssuer はアクセストークン発行（infra.JWTService）。
type AccessTokenIssuer interface {
	Generate(userID int) (string, error)
}

// TokenManager は不透明トークンの生成とハッシュ（infra.TokenManager）。
type TokenManager interface {
	Generate() (string, error)
	Hash(plain string) string
}
