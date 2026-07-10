// API レスポンス型（api/internal/domain と対応）

export type Grade = 'manager' | 'staff'

export interface Role {
  id: number
  code: string
  name: string
  description: string
}

export interface User {
  id: number
  email: string
  name: string
  grade: Grade
  is_active: boolean
  created_at: string
  updated_at: string
  roles?: Role[]
}

export interface Category {
  id: number
  code: string
  name: string
  description: string
  is_required: boolean
  sort_order: number
  is_active: boolean
}

export interface CategoryValue {
  id: number
  category_id: number
  code: string
  label: string
  sort_order: number
  is_active: boolean
}

export interface MeResponse {
  user: User
  functions: string[]
}

export type ProjectStatus = 'planning' | 'active' | 'completed' | 'cancelled'

export interface Program {
  id: number
  code: string
  name: string
  description: string
  created_by: number
  created_at: string
  updated_at: string
}

export interface ProgramAggregate {
  project_count: number
  total_budget: number
  start_date: string | null
  end_date: string | null
  status_counts: Record<string, number>
}

export interface Project {
  id: number
  program_id: number
  branch_no: number | null
  project_code: string | null
  name: string
  description: string
  pm_id: number | null
  approver_id: number | null
  vendor: string
  budget: number | null
  start_date: string | null
  end_date: string | null
  status: ProjectStatus
  backlog_project_id: string
  created_at: string
  updated_at: string
}

// ProjectAttribute は GET /projects/:id/attributes の要素。
// カテゴリ／値のラベルを結合済みで、value_is_active=false は論理削除済みの値が
// 過去アサインとして残っているケース（履歴保護）。
export interface ProjectAttribute {
  id: number
  category_id: number
  category_code: string
  category_name: string
  value_id: number
  value_code: string
  value_label: string
  value_is_active: boolean
}

export interface ProgramView {
  program: Program
  aggregate: ProgramAggregate
}

export interface ProgramDetail {
  program: Program
  aggregate: ProgramAggregate
  projects: Project[]
}
