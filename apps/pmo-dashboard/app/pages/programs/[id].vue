<script setup lang="ts">
import type { ProgramDetail, Project, ProjectAttribute, ProjectStatus } from '~/types/api'

definePageMeta({ middleware: 'auth', requiredFunction: 'view_project_detail' })

const api = useApi()
const auth = useAuth()
const route = useRoute()
const programId = computed(() => Number(route.params.id))

const detail = ref<ProgramDetail | null>(null)
const attrsByProject = ref<Record<number, ProjectAttribute[]>>({})
const error = ref('')
const showForm = ref(false)
const editingProgram = ref(false)
const savingProgram = ref(false)

const form = reactive({ name: '', vendor: '', budget: '', startDate: '', endDate: '' })
const progForm = reactive({ name: '', description: '' })

const inputClass =
  'w-full rounded-md bg-surface-1 px-3 py-2 text-sm text-ink placeholder:text-ink-muted outline-none ring-1 ring-hairline focus-visible:ring-2 focus-visible:ring-accent-blue/40'

async function load() {
  try {
    detail.value = await api<ProgramDetail>(`/programs/${programId.value}`)
    await loadAttributes()
  } catch (e) {
    error.value = apiError(e, 'プログラムの取得に失敗しました')
  }
}

// 配下プロジェクトの紐付け属性を並列取得し、行内バッジで素早く確認できるようにする。
async function loadAttributes() {
  const projects = detail.value?.projects ?? []
  const entries = await Promise.all(
    projects.map(async (p) => {
      const res = await api<{ attributes: ProjectAttribute[] }>(`/projects/${p.id}/attributes`)
      return [p.id, res.attributes ?? []] as const
    }),
  )
  attrsByProject.value = Object.fromEntries(entries)
}
onMounted(load)
watchEffect(() => {
  if (detail.value) useHead({ title: `${detail.value.program.code} — PMO Agent` })
})

async function onCreateProject() {
  error.value = ''
  try {
    await api(`/programs/${programId.value}/projects`, {
      method: 'POST',
      body: {
        name: form.name,
        vendor: form.vendor,
        budget: form.budget ? Number(form.budget) : null,
        start_date: form.startDate,
        end_date: form.endDate,
      },
    })
    Object.assign(form, { name: '', vendor: '', budget: '', startDate: '', endDate: '' })
    showForm.value = false
    await load()
  } catch (e) {
    error.value = apiError(e, 'プロジェクトの作成に失敗しました')
  }
}

function startEditProgram() {
  const p = detail.value?.program
  if (!p) return
  progForm.name = p.name
  progForm.description = p.description
  error.value = ''
  showForm.value = false
  editingProgram.value = true
}

// プログラムは name / description のみ編集可（code は発行後不変）。
async function onSaveProgram() {
  error.value = ''
  savingProgram.value = true
  try {
    await api(`/programs/${programId.value}`, {
      method: 'PUT',
      body: { name: progForm.name, description: progForm.description },
    })
    editingProgram.value = false
    await load()
  } catch (e) {
    error.value = apiError(e, 'プログラムの更新に失敗しました')
  } finally {
    savingProgram.value = false
  }
}

async function onIssueCode(projectId: number) {
  error.value = ''
  try {
    await api(`/projects/${projectId}/issue-code`, { method: 'POST' })
    await load()
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
function projectCode(p: Project): string {
  return p.project_code ?? '未発行'
}
</script>

<template>
  <AppShell>
    <NuxtLink to="/programs" class="text-sm text-accent-blue hover:opacity-80">← プログラム一覧</NuxtLink>

    <template v-if="detail">
      <div class="mt-4 flex items-start justify-between gap-4">
        <div>
          <p class="font-medium text-accent-blue">{{ detail.program.code }}</p>
          <h1 class="display-md mt-1 text-ink">{{ detail.program.name }}</h1>
          <p v-if="detail.program.description" class="mt-2 max-w-2xl text-sm text-ink-muted">
            {{ detail.program.description }}
          </p>
        </div>
        <div class="flex shrink-0 flex-wrap justify-end gap-2">
          <PillButton
            v-if="auth.hasFunction('issue_project_code') && !editingProgram"
            variant="secondary"
            @click="startEditProgram"
          >
            編集
          </PillButton>
          <PillButton
            v-if="auth.hasFunction('manage_projects')"
            variant="primary"
            @click="showForm = !showForm"
          >
            {{ showForm ? '閉じる' : '新規プロジェクト' }}
          </PillButton>
        </div>
      </div>

      <!-- プログラム編集フォーム（name / description のみ。code は発行後不変） -->
      <form
        v-if="editingProgram"
        class="mt-6 grid gap-4 rounded-xl bg-surface-1 p-6"
        @submit.prevent="onSaveProgram"
      >
        <label class="flex flex-col gap-2">
          <span class="text-sm text-ink-muted">プログラム名</span>
          <input v-model="progForm.name" type="text" :class="inputClass">
        </label>
        <label class="flex flex-col gap-2">
          <span class="text-sm text-ink-muted">説明</span>
          <textarea v-model="progForm.description" rows="2" :class="inputClass" />
        </label>
        <div class="flex gap-3">
          <PillButton variant="primary" type="submit">{{ savingProgram ? '保存中…' : '保存する' }}</PillButton>
          <PillButton variant="secondary" type="button" @click="editingProgram = false">キャンセル</PillButton>
        </div>
      </form>

      <!-- 集計 -->
      <div class="mt-8 grid gap-4 sm:grid-cols-3">
        <div class="rounded-xl bg-surface-1 p-5">
          <p class="text-xs text-ink-muted">プロジェクト数</p>
          <p class="display-md mt-1 text-ink">{{ detail.aggregate.project_count }}</p>
        </div>
        <div class="rounded-xl bg-surface-1 p-5">
          <p class="text-xs text-ink-muted">集計予算</p>
          <p class="display-md mt-1 text-ink">{{ yen(detail.aggregate.total_budget) }}</p>
        </div>
        <div class="rounded-xl bg-surface-1 p-5">
          <p class="text-xs text-ink-muted">期間（最早〜最遅）</p>
          <p class="mt-2 text-sm text-ink">
            {{ detail.aggregate.start_date?.slice(0, 10) ?? '—' }} 〜
            {{ detail.aggregate.end_date?.slice(0, 10) ?? '—' }}
          </p>
        </div>
      </div>

      <p v-if="error" class="mt-4 text-sm text-grad-coral" role="alert">{{ error }}</p>

      <!-- 新規プロジェクトフォーム -->
      <form
        v-if="showForm"
        class="mt-6 grid gap-4 rounded-xl bg-surface-1 p-6 md:grid-cols-2"
        @submit.prevent="onCreateProject"
      >
        <label class="flex flex-col gap-2">
          <span class="text-sm text-ink-muted">プロジェクト名</span>
          <input v-model="form.name" type="text" placeholder="申込フェーズ" :class="inputClass">
        </label>
        <label class="flex flex-col gap-2">
          <span class="text-sm text-ink-muted">委託先</span>
          <input v-model="form.vendor" type="text" placeholder="ベンダー名" :class="inputClass">
        </label>
        <label class="flex flex-col gap-2">
          <span class="text-sm text-ink-muted">外注予算（円）</span>
          <input v-model="form.budget" type="number" placeholder="5000000" :class="inputClass">
        </label>
        <div class="grid grid-cols-2 gap-3">
          <label class="flex flex-col gap-2">
            <span class="text-sm text-ink-muted">開始日</span>
            <input v-model="form.startDate" type="date" :class="inputClass">
          </label>
          <label class="flex flex-col gap-2">
            <span class="text-sm text-ink-muted">終了日</span>
            <input v-model="form.endDate" type="date" :class="inputClass">
          </label>
        </div>
        <div class="md:col-span-2">
          <PillButton variant="primary" type="submit">起案する</PillButton>
        </div>
      </form>

      <!-- 配下プロジェクト -->
      <h2 class="mt-10 text-sm font-semibold text-ink">配下プロジェクト</h2>
      <div class="mt-3 overflow-hidden rounded-xl ring-1 ring-hairline">
        <table class="w-full text-left text-sm">
          <thead class="bg-surface-1 text-ink-muted">
            <tr>
              <th class="px-4 py-3 font-medium">コード</th>
              <th class="px-4 py-3 font-medium">名称</th>
              <th class="px-4 py-3 font-medium">委託先</th>
              <th class="px-4 py-3 font-medium">予算</th>
              <th class="px-4 py-3 font-medium">ステータス</th>
              <th class="px-4 py-3 font-medium">操作</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="p in detail.projects" :key="p.id">
              <tr class="border-t border-hairline-soft">
                <td class="px-4 py-3 font-medium" :class="p.project_code ? 'text-accent-blue' : 'text-ink-muted'">
                  {{ projectCode(p) }}
                </td>
                <td class="px-4 py-3">
                  <NuxtLink :to="`/projects/${p.id}`" class="text-ink hover:text-accent-blue">{{ p.name }}</NuxtLink>
                </td>
                <td class="px-4 py-3 text-ink-muted">{{ p.vendor || '—' }}</td>
                <td class="px-4 py-3 text-ink-muted">{{ yen(p.budget) }}</td>
                <td class="px-4 py-3"><span :class="statusClass(p.status)">{{ statusLabel[p.status] }}</span></td>
                <td class="px-4 py-3">
                  <button
                    v-if="p.status === 'planning' && auth.hasFunction('issue_project_code')"
                    class="text-accent-blue hover:opacity-80"
                    @click="onIssueCode(p.id)"
                  >
                    コード発行
                  </button>
                  <span v-else class="text-ink-muted">—</span>
                </td>
              </tr>
              <!-- 紐付け属性をプロジェクト行直下にインライン表示（素早い確認用） -->
              <tr v-if="(attrsByProject[p.id] ?? []).length">
                <td class="px-4 pb-3 pt-0" />
                <td colspan="5" class="px-4 pb-3 pt-0">
                  <AttributeBadges :attributes="attrsByProject[p.id] ?? []" />
                </td>
              </tr>
            </template>
            <tr v-if="detail.projects.length === 0">
              <td colspan="6" class="px-4 py-8 text-center text-ink-muted">プロジェクトがありません</td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </AppShell>
</template>
