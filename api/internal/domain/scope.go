package domain

import "slices"

// ロールコード。スコープ解決（担当PJ集合の算出）で参照する。
// 権限そのものは role_functions で管理するが、スコープの適用範囲はロール種別に依存するため
// ここで定数化する（ハードコード散在を防ぐ）。
const (
	RoleCodeExecutive = "executive"
	RoleCodePMOAdmin  = "pmo_admin"
	RoleCodePM        = "pm"
	RoleCodeMember    = "member"
	RoleCodePlanner   = "planner"
)

// ProjectScope はリクエストユーザーが閲覧・操作を許可されるプロジェクトの範囲を表す値オブジェクト。
//   - All=true  … 全プロジェクト（pmo_admin / executive）。ProjectIDs は無視する
//   - All=false … ProjectIDs に含まれるプロジェクトのみ許可（担当PJ / 自起案PJ）
//
// スコープミドルウェアがロールから解決し、usecase がフィルタに用いる。
type ProjectScope struct {
	All        bool  `json:"all"`
	ProjectIDs []int `json:"project_ids"`
}

// UnrestrictedScope は全プロジェクトを許可するスコープを返す。
func UnrestrictedScope() ProjectScope { return ProjectScope{All: true} }

// Allows は指定プロジェクトへのアクセスが許可されるかを返す。
func (s ProjectScope) Allows(projectID int) bool {
	if s.All {
		return true
	}
	return slices.Contains(s.ProjectIDs, projectID)
}
