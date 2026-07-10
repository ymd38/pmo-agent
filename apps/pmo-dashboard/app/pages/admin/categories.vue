<script setup lang="ts">
import type { Category, CategoryValue } from '~/types/api'

definePageMeta({ middleware: 'auth', requiredFunction: 'manage_categories' })
useHead({ title: '属性マスタ — PMO Agent' })

const api = useApi()

const categories = ref<Category[]>([])
const values = ref<CategoryValue[]>([])
const selectedId = ref<number | null>(null)
const errorMsg = ref('')

const newCat = reactive({ code: '', name: '' })
const newVal = reactive({ code: '', label: '' })

const inputClass =
  'rounded-md bg-surface-1 px-3 py-2 text-sm text-ink placeholder:text-ink-muted outline-none ring-1 ring-hairline focus-visible:ring-2 focus-visible:ring-accent-blue/40'

const selected = computed(() => categories.value.find((c) => c.id === selectedId.value) ?? null)

async function loadCategories() {
  try {
    const res = await api<{ categories: Category[] }>('/categories', { query: { include_inactive: 'true' } })
    categories.value = res.categories
    if (selectedId.value === null && categories.value.length > 0) {
      await select(categories.value[0]!.id)
    }
  } catch (e) {
    errorMsg.value = apiError(e, 'カテゴリの取得に失敗しました')
  }
}

async function loadValues(categoryId: number) {
  const res = await api<{ values: CategoryValue[] }>(`/categories/${categoryId}/values`, {
    query: { include_inactive: 'true' },
  })
  values.value = res.values
}

async function select(id: number) {
  selectedId.value = id
  await loadValues(id)
}

onMounted(loadCategories)

async function addCategory() {
  errorMsg.value = ''
  try {
    await api('/categories', { method: 'POST', body: { code: newCat.code, name: newCat.name } })
    newCat.code = ''
    newCat.name = ''
    await loadCategories()
  } catch (e) {
    errorMsg.value = apiError(e, 'カテゴリ作成に失敗しました')
  }
}

async function deactivateCategory(id: number) {
  errorMsg.value = ''
  try {
    await api(`/categories/${id}`, { method: 'DELETE' })
    await loadCategories()
  } catch (e) {
    errorMsg.value = apiError(e, '無効化に失敗しました')
  }
}

async function reactivateCategory(id: number) {
  errorMsg.value = ''
  try {
    await api(`/categories/${id}/reactivate`, { method: 'POST' })
    await loadCategories()
  } catch (e) {
    errorMsg.value = apiError(e, '再有効化に失敗しました')
  }
}

async function addValue() {
  if (selectedId.value === null) return
  errorMsg.value = ''
  try {
    await api(`/categories/${selectedId.value}/values`, {
      method: 'POST',
      body: { code: newVal.code, label: newVal.label },
    })
    newVal.code = ''
    newVal.label = ''
    await loadValues(selectedId.value)
  } catch (e) {
    errorMsg.value = apiError(e, '値の追加に失敗しました')
  }
}

async function deactivateValue(valueId: number) {
  if (selectedId.value === null) return
  errorMsg.value = ''
  try {
    await api(`/categories/${selectedId.value}/values/${valueId}`, { method: 'DELETE' })
    await loadValues(selectedId.value)
  } catch (e) {
    errorMsg.value = apiError(e, '無効化に失敗しました')
  }
}

async function reactivateValue(valueId: number) {
  if (selectedId.value === null) return
  errorMsg.value = ''
  try {
    await api(`/categories/${selectedId.value}/values/${valueId}/reactivate`, { method: 'POST' })
    await loadValues(selectedId.value)
  } catch (e) {
    errorMsg.value = apiError(e, '再有効化に失敗しました')
  }
}

</script>

<template>
  <AppShell>
    <p class="eyebrow text-ink-muted">Masters</p>
    <h1 class="display-md mt-3 text-ink">属性マスタ</h1>
    <p class="body-lg mt-3 max-w-2xl text-ink-muted">
      プロジェクトの分類カテゴリと値。削除は論理削除（無効化）のみで、過去のアサインを保護します。
    </p>

    <p v-if="errorMsg" class="mt-4 text-sm text-grad-coral" role="alert">{{ errorMsg }}</p>

    <div class="mt-8 grid gap-6 md:grid-cols-3">
      <!-- カテゴリ一覧 -->
      <section class="rounded-xl bg-surface-1 p-5 md:col-span-1">
        <h2 class="text-sm font-semibold text-ink">カテゴリ</h2>
        <ul class="mt-3 flex flex-col gap-1">
          <li v-for="c in categories" :key="c.id">
            <button
              class="flex w-full items-center justify-between rounded-md px-3 py-2 text-left text-sm transition-colors"
              :class="c.id === selectedId ? 'bg-surface-2 text-ink' : 'text-ink-muted hover:text-ink'"
              @click="select(c.id)"
            >
              <span>{{ c.name }}<span v-if="!c.is_active" class="ml-2 text-xs">（無効）</span></span>
              <span v-if="c.is_required" class="text-xs text-accent-blue">必須</span>
            </button>
          </li>
        </ul>

        <form class="mt-4 flex flex-col gap-2 border-t border-hairline-soft pt-4" @submit.prevent="addCategory">
          <input v-model="newCat.code" type="text" placeholder="code（例: risk_level）" :class="inputClass">
          <input v-model="newCat.name" type="text" placeholder="名称" :class="inputClass">
          <PillButton variant="secondary" type="submit">カテゴリ追加</PillButton>
        </form>
      </section>

      <!-- 値一覧 -->
      <section class="rounded-xl bg-surface-1 p-5 md:col-span-2">
        <div class="flex items-center justify-between">
          <h2 class="text-sm font-semibold text-ink">
            {{ selected ? `${selected.name} の値` : '値' }}
          </h2>
          <button
            v-if="selected && selected.is_active"
            class="text-xs text-ink-muted hover:text-ink"
            @click="deactivateCategory(selected.id)"
          >
            このカテゴリを無効化
          </button>
          <button
            v-else-if="selected"
            class="text-xs text-accent-blue hover:opacity-80"
            @click="reactivateCategory(selected.id)"
          >
            このカテゴリを再有効化
          </button>
        </div>

        <ul class="mt-3 flex flex-col gap-1">
          <li
            v-for="v in values"
            :key="v.id"
            class="flex items-center justify-between rounded-md px-3 py-2 text-sm"
            :class="v.is_active ? 'text-ink' : 'text-ink-muted'"
          >
            <span>{{ v.label }} <span class="text-ink-muted">/ {{ v.code }}</span><span v-if="!v.is_active"> （無効）</span></span>
            <button v-if="v.is_active" class="text-xs text-ink-muted hover:text-ink" @click="deactivateValue(v.id)">無効化</button>
            <button v-else class="text-xs text-accent-blue hover:opacity-80" @click="reactivateValue(v.id)">再有効化</button>
          </li>
          <li v-if="selected && values.length === 0" class="px-3 py-2 text-sm text-ink-muted">値がありません</li>
        </ul>

        <form v-if="selected" class="mt-4 flex flex-wrap items-center gap-2 border-t border-hairline-soft pt-4" @submit.prevent="addValue">
          <input v-model="newVal.code" type="text" placeholder="code" :class="inputClass">
          <input v-model="newVal.label" type="text" placeholder="ラベル" :class="inputClass">
          <PillButton variant="secondary" type="submit">値を追加</PillButton>
        </form>
      </section>
    </div>
  </AppShell>
</template>
