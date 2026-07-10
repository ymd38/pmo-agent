package repository

import (
	"context"

	"pmo-agent/api/internal/domain"

	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

// user_roles 中間テーブルへの明示的な書き込み用。
type userRoleRow struct {
	UserID int `gorm:"column:user_id"`
	RoleID int `gorm:"column:role_id"`
}

func (userRoleRow) TableName() string { return "user_roles" }

func (r *UserRepo) FindByID(ctx context.Context, id int) (*domain.User, error) {
	var u domain.User
	if err := r.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return nil, wrapNotFound(err)
	}
	return &u, nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error; err != nil {
		return nil, wrapNotFound(err)
	}
	return &u, nil
}

func (r *UserRepo) List(ctx context.Context) ([]domain.User, error) {
	var users []domain.User
	if err := r.db.WithContext(ctx).Preload("Roles").Order("id").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User, roleIDs []int) error {
	return wrapConflict(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Roles").Create(u).Error; err != nil {
			return err
		}
		return insertUserRoles(tx, u.ID, roleIDs)
	}))
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User, roleIDs []int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// password_hash には触れない。name/grade/is_active のみ更新。
		if err := tx.Model(&domain.User{}).Where("id = ?", u.ID).
			Select("name", "grade", "is_active").
			Updates(map[string]any{"name": u.Name, "grade": u.Grade, "is_active": u.IsActive}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", u.ID).Delete(&userRoleRow{}).Error; err != nil {
			return err
		}
		return insertUserRoles(tx, u.ID, roleIDs)
	})
}

func (r *UserRepo) UpdatePasswordHash(ctx context.Context, userID int, hash string) error {
	return r.db.WithContext(ctx).Model(&domain.User{}).
		Where("id = ?", userID).Update("password_hash", hash).Error
}

func (r *UserRepo) Deactivate(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Model(&domain.User{}).
		Where("id = ?", id).Update("is_active", false).Error
}

// IsActive は users.is_active を単一カラムで引く（認証ミドルウェアの毎リクエスト確認用）。
// 該当ユーザーが存在しなければ false（＝アクセス不可）を返す。
func (r *UserRepo) IsActive(ctx context.Context, id int) (bool, error) {
	var active bool
	err := r.db.WithContext(ctx).Model(&domain.User{}).
		Where("id = ?", id).Select("is_active").Scan(&active).Error
	return active, err
}

func (r *UserRepo) FunctionsByUserID(ctx context.Context, userID int) ([]string, error) {
	var codes []string
	err := r.db.WithContext(ctx).
		Table("functions f").
		Joins("JOIN role_functions rf ON rf.function_id = f.id").
		Joins("JOIN user_roles ur ON ur.role_id = rf.role_id").
		Where("ur.user_id = ?", userID).
		Distinct().Pluck("f.code", &codes).Error
	return codes, err
}

func (r *UserRepo) RolesByUserID(ctx context.Context, userID int) ([]domain.Role, error) {
	var roles []domain.Role
	err := r.db.WithContext(ctx).
		Table("roles r").
		Joins("JOIN user_roles ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID).
		Order("r.id").Find(&roles).Error
	return roles, err
}

func insertUserRoles(tx *gorm.DB, userID int, roleIDs []int) error {
	if len(roleIDs) == 0 {
		return nil
	}
	rows := make([]userRoleRow, 0, len(roleIDs))
	for _, rid := range roleIDs {
		rows = append(rows, userRoleRow{UserID: userID, RoleID: rid})
	}
	return tx.Create(&rows).Error
}
