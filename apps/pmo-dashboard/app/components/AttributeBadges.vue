<script setup lang="ts">
import type { ProjectAttribute } from '~/types/api'

// 紐付け済み属性を「カテゴリ名: 値」の pill で一覧表示する読み取り専用バッジ。
// プログラム詳細・プロジェクト詳細で共通利用し、素早い確認に使う。
// 無効化済み（value_is_active=false）の値は取り消し線で履歴として示す。
const props = withDefaults(
  defineProps<{
    attributes: ProjectAttribute[]
    // 空のときに「属性なし」を出すか（一覧の行内では非表示にしたいので false が既定）。
    showEmpty?: boolean
  }>(),
  { showEmpty: false },
)

const hasAny = computed(() => props.attributes.length > 0)
</script>

<template>
  <div v-if="hasAny" class="flex flex-wrap gap-1.5">
    <span
      v-for="a in attributes"
      :key="a.id"
      class="inline-flex items-center gap-1 rounded-full bg-surface-2 px-2.5 py-0.5 text-xs ring-1 ring-hairline"
      :class="{ 'opacity-60': !a.value_is_active }"
    >
      <span class="text-ink-muted">{{ a.category_name }}</span>
      <span class="text-ink" :class="{ 'line-through': !a.value_is_active }">{{ a.value_label }}</span>
    </span>
  </div>
  <p v-else-if="showEmpty" class="text-xs text-ink-muted">属性なし</p>
</template>
