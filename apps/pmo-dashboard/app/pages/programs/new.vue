<script setup lang="ts">
import type { Program } from '~/types/api'

definePageMeta({ middleware: 'auth', requiredFunction: 'issue_project_code' })
useHead({ title: 'プログラム作成 — PMO Agent' })

const api = useApi()

// 会計年度の既定値（4月始まり: 1〜3月は前年度）
const now = new Date()
const defaultFiscalYear = now.getMonth() + 1 >= 4 ? now.getFullYear() : now.getFullYear() - 1

const form = reactive({ type: 'INV', fiscalYear: defaultFiscalYear, name: '', description: '' })
const error = ref('')
const loading = ref(false)

// 種別の候補（選択 or 自由記載）
const knownTypes = [
  { code: 'INV', label: 'INV（投資）' },
  { code: 'MNT', label: 'MNT（保守）' },
  { code: 'OPS', label: 'OPS（運用）' },
]

const inputClass =
  'w-full rounded-md bg-surface-1 px-4 py-2.5 text-sm text-ink placeholder:text-ink-muted outline-none ring-1 ring-hairline focus-visible:ring-2 focus-visible:ring-accent-blue/40'

// 入力中のコードプレビュー（連番はサーバーが自動採番）
const codePreview = computed(() => {
  const t = form.type.trim().toUpperCase() || '????'
  return `${t}-${form.fiscalYear}-NNNN`
})

async function onSubmit() {
  error.value = ''
  loading.value = true
  try {
    const res = await api<{ program: Program }>('/programs', {
      method: 'POST',
      body: {
        type: form.type.trim().toUpperCase(),
        fiscal_year: form.fiscalYear,
        name: form.name,
        description: form.description,
      },
    })
    await navigateTo(`/programs/${res.program.id}`)
  } catch (e) {
    const data = (e as { data?: { error?: string } })?.data
    error.value = data?.error ?? 'プログラムの作成に失敗しました'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AppShell>
    <NuxtLink to="/programs" class="text-sm text-accent-blue hover:opacity-80">← プログラム一覧</NuxtLink>
    <h1 class="display-md mt-4 text-ink">プログラム作成</h1>
    <p class="body-lg mt-3 max-w-xl text-ink-muted">
      種別と会計年度を指定すると、コードの連番は自動採番されます。発行後は変更できません。
    </p>

    <form class="mt-8 flex max-w-xl flex-col gap-4" @submit.prevent="onSubmit">
      <div class="grid grid-cols-2 gap-4">
        <label class="flex flex-col gap-2">
          <span class="text-sm text-ink-muted">種別</span>
          <input
            v-model="form.type"
            type="text"
            list="program-types"
            placeholder="INV"
            :class="inputClass"
          >
          <datalist id="program-types">
            <option v-for="t in knownTypes" :key="t.code" :value="t.code">{{ t.label }}</option>
          </datalist>
        </label>
        <label class="flex flex-col gap-2">
          <span class="text-sm text-ink-muted">会計年度</span>
          <input v-model.number="form.fiscalYear" type="number" min="2000" max="2999" :class="inputClass">
        </label>
      </div>

      <p class="text-xs text-ink-muted">
        コード（プレビュー）: <code class="text-accent-blue">{{ codePreview }}</code>
        <span class="ml-1">— 連番(NNNN)は自動採番されます</span>
      </p>

      <label class="flex flex-col gap-2">
        <span class="text-sm text-ink-muted">名称</span>
        <input v-model="form.name" type="text" placeholder="会員システム刷新" :class="inputClass">
      </label>
      <label class="flex flex-col gap-2">
        <span class="text-sm text-ink-muted">説明（任意）</span>
        <textarea v-model="form.description" rows="3" :class="inputClass" />
      </label>

      <p v-if="error" class="text-sm text-grad-coral" role="alert">{{ error }}</p>

      <div>
        <PillButton variant="primary" type="submit">{{ loading ? '作成中…' : '作成する' }}</PillButton>
      </div>
    </form>
  </AppShell>
</template>
