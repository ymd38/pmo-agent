package domain

import "time"

// Grade は内部単価グレード。
type Grade string

const (
	GradeManager Grade = "manager"
	GradeStaff   Grade = "staff"
)

// User は認証と社員情報を統合したアカウント。
// PasswordHash が nil のときは「未アクティベート」（招待リンクで設定後に確定）。
type User struct {
	ID           int       `json:"id"           gorm:"primaryKey"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash *string   `json:"-"            gorm:"column:password_hash"`
	Grade        Grade     `json:"grade"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Roles        []Role    `json:"roles,omitempty" gorm:"many2many:user_roles;"`
}

func (User) TableName() string { return "users" }

// CanLogin はログイン可能条件（パスワード設定済み かつ 有効）を返す。
func (u User) CanLogin() bool {
	return u.PasswordHash != nil && *u.PasswordHash != "" && u.IsActive
}

// Role はロール定義。
type Role struct {
	ID          int       `json:"id"          gorm:"primaryKey"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Role) TableName() string { return "roles" }

// Function は機能権限定義。
type Function struct {
	ID          int       `json:"id"   gorm:"primaryKey"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Function) TableName() string { return "functions" }
