<script setup lang="ts">
import type { Category, CategoryValue, Project, ProjectAttribute, ProjectStatus } from '~/types/api'

definePageMeta({ middleware: 'auth', requiredFunction: 'view_project_detail' })

const api = useApi()
const auth = useAuth()
const route = useRoute()
const projectId = computed(() => Number(route.params.id))
const canManage = computed(() => auth.hasFunction('manage_projects'))

const project = ref<Project | null>(null)
const attributes = ref<ProjectAttribute[]>([])
const categories = ref<Category[]>([])
const valuesByCat = ref<Record<number, CategoryValue[]>>({})
const error = ref('')
const busyValueId = ref<number | null>(null)

const editing = ref(false)
const saving = ref(false)
const form = reactive({ name: '', description: '', vendor: '', budget: '', startDate: '', endDate: '', backlogProjectId: '' })

const inputClass =
  'w-full rounded-md bg-surface-1 px-3 py-2 text-sm text-ink placeholder:text-ink-muted outline-none ring-1 ring-hairline focus-visible:ring-2 focus-visible:ring-accent-blue/40'

async function loadProject() {
  const res = await api<{ project: Project }>(`/projects/${projectId.value}`)
  project.value = res.project
}

async function loadAttributes() {
  const res = await api<{ attributes: ProjectAttribute[] }>(`/projects/${projectId.value}/attributes`)
  attributes.value = res.attributes ?? []
}

// マスタ（有効なカテゴリと値）を読み込む。チップのトグル候補として使う。
async function loadMaster() {
  const res = await api<{ categories: Category[] }>('/categories')
  categories.value = res.categories ?? []
  const entries = await Promise.all(
    categories.value.map(async (c) => {
      const v = await api<{ values: CategoryValue[] }>(`/categories/${c.id}/values`)
      return [c.id, v.values ?? []] as const
    }),
  )
  valuesByCat.value = Object.fromEntries(entries)
}

onMounted(async () => {
  try {
    await Promise.all([loadProject(), loadAttributes(), loadMaster()])
  } catch (e) {
    error.value = apiError(e, 'プロジェクトの取得に失敗しました')
  }
})

watchEffect(() => {
  if (project.value) {
    const code = project.value.project_code ?? project.value.name
    useHead({ title: `${code} — PMO Agent` })
  }
})

const assignedValueIds = computed(() => new Set(attributes.value.map(a => a.value_id)))

// 論理削除済み（無効）の値が過去アサインとして残っているもの。カテゴリ別に表示し、解除のみ可能。
const inactiveByCategory = computed(() => {
  const map = new Map<number, ProjectAttribute[]>()
  for (const a of attributes.value) {
    if (a.value_is_active) continue
    const list = map.get(a.category_id) ?? []
    list.push(a)
    map.set(a.category_id, list)
  }
  return map
})

async function toggleValue(value: CategoryValue) {
  if (!canManage.value || busyValueId.value !== null) return
  error.value = ''
  busyValueId.value = value.id
  try {
    if (assignedValueIds.value.has(value.id)) {
      await api(`/projects/${projectId.value}/attributes/${value.id}`, { method: 'DELETE' })
    } else {
      await api(`/projects/${projectId.value}/attributes`, {
        method: 'POST',
        body: { value_id: value.id },
      })
    }
    await loadAttributes()
  } catch (e) {
    error.value = apiError(e, '属性の更新に失敗しました')
  } finally {
    busyValueId.value = null
  }
}

async function removeValue(valueId: number) {
  if (!canManage.value || busyValueId.value !== null) return
  error.value = ''
  busyValueId.value = valueId
  try {
    await api(`/projects/${projectId.value}/attributes/${valueId}`, { method: 'DELETE' })
    await loadAttributes()
  } catch (e) {
    error.value = apiError(e, '属性の解除に失敗しました')
  } finally {
    busyValueId.value = null
  }
}

function startEdit() {
  const p = project.value
  if (!p) return
  Object.assign(form, {
    name: p.name,
    description: p.description,
    vendor: p.vendor,
    budget: p.budget != null ? String(p.budget) : '',
    startDate: p.start_date?.slice(0, 10) ?? '',
    endDate: p.end_date?.slice(0, 10) ?? '',
    backlogProjectId: p.backlog_project_id,
  })
  error.value = ''
  editing.value = true
}

async function onSave() {
  const p = project.value
  if (!p) return
  error.value = ''
  saving.value = true
  try {
    // pm_id / approver_id / status は本フォームでは変更せず現在値を維持する
    // （担当者割当は別画面、ステータスは遷移アクションで扱う）。
    await api(`/projects/${projectId.value}`, {
      method: 'PUT',
      body: {
        name: form.name,
        description: form.description,
        pm_id: p.pm_id,
        approver_id: p.approver_id,
        vendor: form.vendor,
        budget: form.budget ? Number(form.budget) : null,
        start_date: form.startDate,
        end_date: form.endDate,
        status: p.status,
        backlog_project_id: form.backlogProjectId,
      },
    })
    editing.value = false
    await loadProject()
  } catch (e) {
    error.value = apiError(e, 'プロジェクトの更新に失敗しました')
  } finally {
    saving.value = false
  }
}

// ステータス遷移は現在の全フィールドを維持したまま status のみ差し替えて PUT する。
async function changeStatus(next: ProjectStatus, confirmMessage: string) {
  const p = project.value
  if (!p || !window.confirm(confirmMessage)) return
  error.value = ''
  try {
    await api(`/projects/${projectId.value}`, {
      method: 'PUT',
      body: {
        name: p.name,
        description: p.description,
        pm_id: p.pm_id,
        approver_id: p.approver_id,
        vendor: p.vendor,
        budget: p.budget,
        start_date: p.start_date?.slice(0, 10) ?? '',
        end_date: p.end_date?.slice(0, 10) ?? '',
        status: next,
        backlog_project_id: p.backlog_project_id,
      },
    })
    await loadProject()
  } catch (e) {
    error.value = apiError(e, 'ステータスの変更に失敗しました')
  }
}

// planning → active はコード発行専用経路（枝番採番＋active遷移）。
async function onIssueCode() {
  if (!window.confirm('プロジェクトコードを発行し、進行中に移行しますか？（コードは発行後変更できません）')) return
  error.value = ''
  try {
    await api(`/projects/${projectId.value}/issue-code`, { method: 'POST' })
    await loadProject()
  } catch (e) {
    error.value = apiError(e, 'コード発行に失敗しました')
  }
}

function yen(n: number | null): string {
  return n === null ? '—' : `¥${n.toLocaleString('ja-JP')}`
}

const statusLabel: Record<ProjectStatus, string> = {
  planning: '起案中',
  active: '進行中',
  completed: '完了',
  cancelled: '中止',
}
function statusClass(s: ProjectStatus): string {
  return {
    planning: 'text-ink-muted',
    active: 'text-success',
    completed: 'text-accent-blue',
    cancelled: 'text-grad-coral',
  }[s]
}
</script>

<template>
  <AppShell>
    <NuxtLink
      v-if="project"
      :to="`/programs/${project.program_id}`"
      class="text-sm text-accent-blue hover:opacity-80"
    >
      ← プログラム詳細
    </NuxtLink>

    <template v-if="project">
      <!-- ヘッダー -->
      <div class="mt-4 flex items-start justify-between gap-4">
        <div>
          <p class="font-medium" :class="project.project_code ? 'text-accent-blue' : 'text-ink-muted'">
            {{ project.project_code ?? 'コード未発行' }}
          </p>
          <h1 class="display-md mt-1 text-ink">{{ project.name }}</h1>
          <p v-if="project.description" class="mt-2 max-w-2xl text-sm text-ink-muted">
            {{ project.description }}
          </p>
          <!-- 紐付け済み属性のサマリ（素早い確認用・読み取り専用。編集は下部のチップで行う） -->
          <AttributeBadges :attributes="attributes" :show-empty="true" class="mt-3" />
        </div>

        <!-- 編集・ステータス遷移アクション -->
        <div
          v-if="canManage || auth.hasFunction('issue_project_code')"
          class="flex shrink-0 flex-col items-end gap-2"
        >
          <PillButton v-if="canManage && !editing" variant="secondary" @click="startEdit">編集</PillButton>
          <div class="flex flex-wrap justify-end gap-2">
            <PillButton
              v-if="project.status === 'planning' && auth.hasFunction('issue_project_code')"
              variant="primary"
              @click="onIssueCode"
            >
              コード発行
            </PillButton>
            <PillButton
              v-if="canManage && project.status === 'planning'"
              variant="secondary"
              @click="changeStatus('cancelled', 'このプロジェクトを中止しますか？')"
            >
              中止する
            </PillButton>
            <PillButton
              v-if="canManage && project.status === 'active'"
              variant="primary"
              @click="changeStatus('completed', 'このプロジェクトを完了にしますか？')"
            >
              完了にする
            </PillButton>
            <PillButton
              v-if="canManage && project.status === 'active'"
              variant="secondary"
              @click="changeStatus('cancelled', 'このプロジェクトを中止しますか？')"
            >
              中止する
            </PillButton>
          </div>
        </div>
      </div>

      <!-- 概要 -->
      <div v-if="!editing" class="mt-8 grid gap-4 sm:grid-cols-3">
        <div class="rounded-xl bg-surface-1 p-5">
          <p class="text-xs text-ink-muted">ステータス</p>
          <p class="mt-1 text-lg" :class="statusClass(project.status)">{{ statusLabel[project.status] }}</p>
        </div>
        <div class="rounded-xl bg-surface-1 p-5">
          <p class="text-xs text-ink-muted">委託先</p>
          <p class="mt-1 text-lg text-ink">{{ project.vendor || '—' }}</p>
        </div>
        <div class="rounded-xl bg-surface-1 p-5">
          <p class="text-xs text-ink-muted">外注予算</p>
          <p class="mt-1 text-lg text-ink">{{ yen(project.budget) }}</p>
        </div>
      </div>

      <p v-if="error" class="mt-4 text-sm text-grad-coral" role="alert">{{ error }}</p>

      <!-- 編集フォーム（name / description / vendor / budget / 期間 / Backlog ID） -->
      <form
        v-if="editing"
        class="mt-6 grid gap-4 rounded-xl bg-surface-1 p-6 md:grid-cols-2"
        @submit.prevent="onSave"
      >
        <label class="flex flex-col gap-2 md:col-span-2">
          <span class="text-sm text-ink-muted">プロジェクト名</span>
          <input v-model="form.name" type="text" :class="inputClass">
        </label>
        <label class="flex flex-col gap-2 md:col-span-2">
          <span class="text-sm text-ink-muted">説明</span>
          <textarea v-model="form.description" rows="2" :class="inputClass" />
        </label>
        <label class="flex flex-col gap-2">
          <span class="text-sm text-ink-muted">委託先</span>
          <input v-model="form.vendor" type="text" :class="inputClass">
        </label>
        <label class="flex flex-col gap-2">
          <span class="text-sm text-ink-muted">外注予算（円）</span>
          <input v-model="form.budget" type="number" :class="inputClass">
        </label>
        <label class="flex flex-col gap-2">
          <span class="text-sm text-ink-muted">開始日</span>
          <input v-model="form.startDate" type="date" :class="inputClass">
        </label>
        <label class="flex flex-col gap-2">
          <span class="text-sm text-ink-muted">終了日</span>
          <input v-model="form.endDate" type="date" :class="inputClass">
        </label>
        <label class="flex flex-col gap-2 md:col-span-2">
          <span class="text-sm text-ink-muted">Backlog プロジェクトID</span>
          <input v-model="form.backlogProjectId" type="text" :class="inputClass">
        </label>
        <div class="flex gap-3 md:col-span-2">
          <PillButton variant="primary" type="submit">{{ saving ? '保存中…' : '保存する' }}</PillButton>
          <PillButton variant="secondary" type="button" @click="editing = false">キャンセル</PillButton>
        </div>
      </form>

      <!-- 属性 -->
      <div class="mt-10 flex items-baseline justify-between">
        <h2 class="text-sm font-semibold text-ink">プロジェクト属性</h2>
        <p v-if="canManage" class="text-xs text-ink-muted">値をクリックして紐付け／解除</p>
      </div>

      <div class="mt-3 space-y-4">
        <div v-for="cat in categories" :key="cat.id" class="rounded-xl bg-surface-1 p-5">
          <div class="flex items-center gap-2">
            <p class="text-sm font-medium text-ink">{{ cat.name }}</p>
            <span v-if="cat.is_required" class="text-xs text-grad-coral">必須</span>
          </div>

          <div class="mt-3 flex flex-wrap gap-2">
            <button
              v-for="val in (valuesByCat[cat.id] ?? [])"
              :key="val.id"
              type="button"
              :disabled="!canManage || busyValueId !== null"
              class="rounded-full px-3 py-1 text-sm ring-1 transition disabled:opacity-50"
              :class="assignedValueIds.has(val.id)
                ? 'bg-surface-2 text-accent-blue ring-accent-blue/50'
                : 'text-ink-muted ring-hairline hover:text-ink enabled:hover:ring-ink-muted'"
              @click="toggleValue(val)"
            >
              {{ val.label }}
            </button>

            <!-- 無効化済みだが過去に紐付けられた値（解除のみ可） -->
            <button
              v-for="a in (inactiveByCategory.get(cat.id) ?? [])"
              :key="`x-${a.value_id}`"
              type="button"
              :disabled="!canManage || busyValueId !== null"
              class="rounded-full px-3 py-1 text-sm text-ink-muted line-through ring-1 ring-hairline disabled:opacity-50"
              :title="canManage ? '無効化された値です。クリックで解除' : '無効化された値です'"
              @click="removeValue(a.value_id)"
            >
              {{ a.value_label }}（無効）
            </button>

            <p
              v-if="(valuesByCat[cat.id] ?? []).length === 0 && !(inactiveByCategory.get(cat.id) ?? []).length"
              class="text-sm text-ink-muted"
            >
              値が登録されていません
            </p>
          </div>
        </div>

        <p v-if="categories.length === 0" class="rounded-xl bg-surface-1 p-5 text-sm text-ink-muted">
          属性カテゴリが登録されていません。
        </p>
      </div>
    </template>
  </AppShell>
</template>
