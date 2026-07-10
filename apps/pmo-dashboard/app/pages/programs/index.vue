<script setup lang="ts">
import type { ProgramView } from '~/types/api'

definePageMeta({ middleware: 'auth', requiredFunction: 'view_project_detail' })
useHead({ title: 'プログラム — PMO Agent' })

const api = useApi()
const auth = useAuth()
const programs = ref<ProgramView[]>([])
const errorMsg = ref('')

async function load() {
  try {
    const res = await api<{ programs: ProgramView[] }>('/programs')
    programs.value = res.programs
  } catch (e) {
    errorMsg.value = apiError(e, 'プログラム一覧の取得に失敗しました')
  }
}
onMounted(load)

function yen(n: number): string {
  return `¥${n.toLocaleString('ja-JP')}`
}
function period(start: string | null, end: string | null): string {
  if (!start && !end) return '—'
  return `${start?.slice(0, 10) ?? '?'} 〜 ${end?.slice(0, 10) ?? '?'}`
}
</script>

<template>
  <AppShell>
    <div class="flex items-center justify-between">
      <div>
        <p class="eyebrow text-ink-muted">Programs</p>
        <h1 class="display-md mt-3 text-ink">プログラム</h1>
      </div>
      <PillButton v-if="auth.hasFunction('issue_project_code')" variant="primary" to="/programs/new">
        新規プログラム
      </PillButton>
    </div>

    <p v-if="errorMsg" class="mt-4 text-sm text-grad-coral" role="alert">{{ errorMsg }}</p>

    <div class="mt-8 overflow-hidden rounded-xl ring-1 ring-hairline">
      <table class="w-full text-left text-sm">
        <thead class="bg-surface-1 text-ink-muted">
          <tr>
            <th class="px-4 py-3 font-medium">コード</th>
            <th class="px-4 py-3 font-medium">名称</th>
            <th class="px-4 py-3 font-medium">プロジェクト</th>
            <th class="px-4 py-3 font-medium">集計予算</th>
            <th class="px-4 py-3 font-medium">期間</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="pv in programs"
            :key="pv.program.id"
            class="cursor-pointer border-t border-hairline-soft hover:bg-surface-1"
            @click="navigateTo(`/programs/${pv.program.id}`)"
          >
            <td class="px-4 py-3 font-medium text-accent-blue">{{ pv.program.code }}</td>
            <td class="px-4 py-3 text-ink">{{ pv.program.name }}</td>
            <td class="px-4 py-3 text-ink-muted">{{ pv.aggregate.project_count }} 件</td>
            <td class="px-4 py-3 text-ink-muted">{{ yen(pv.aggregate.total_budget) }}</td>
            <td class="px-4 py-3 text-ink-muted">{{ period(pv.aggregate.start_date, pv.aggregate.end_date) }}</td>
          </tr>
          <tr v-if="programs.length === 0">
            <td colspan="5" class="px-4 py-8 text-center text-ink-muted">プログラムがありません</td>
          </tr>
        </tbody>
      </table>
    </div>
  </AppShell>
</template>
