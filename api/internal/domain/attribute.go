package domain

import "time"

// AttributeAssignment はプロジェクトと属性値の紐付け（EAVパターン）。
// 同一カテゴリに複数値を紐付け可。UNIQUE(project_id, value_id)。
type AttributeAssignment struct {
	ID         int       `json:"id"          gorm:"primaryKey"`
	ProjectID  int       `json:"project_id"`
	CategoryID int       `json:"category_id"`
	ValueID    int       `json:"value_id"`
	CreatedAt  time.Time `json:"created_at"`
}

func (AttributeAssignment) TableName() string { return "project_attribute_assignments" }

// ProjectAttribute は属性アサインにカテゴリ／値のラベルを結合した表示用ビュー。
// UI がカテゴリ単位でグルーピングして描画できるよう、コード・名称を同梱する。
// ValueIsActive は論理削除済みの値が過去アサインとして残っているケースを示す。
type ProjectAttribute struct {
	ID            int    `json:"id"`
	CategoryID    int    `json:"category_id"`
	CategoryCode  string `json:"category_code"`
	CategoryName  string `json:"category_name"`
	ValueID       int    `json:"value_id"`
	ValueCode     string `json:"value_code"`
	ValueLabel    string `json:"value_label"`
	ValueIsActive bool   `json:"value_is_active"`
}
