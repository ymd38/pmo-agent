<script setup lang="ts">
definePageMeta({ middleware: 'auth' })
useHead({ title: 'ホーム — PMO Agent' })

const auth = useAuth()

const cards = computed(() =>
  [
    {
      to: '/programs',
      title: 'プログラム',
      body: 'プログラム作成とプロジェクトコードの発行。',
      show: auth.hasFunction('view_project_detail'),
    },
    {
      to: '/admin/users',
      title: 'メンバー管理',
      body: 'ユーザーの発行・ロール付与・招待リンクの再発行。',
      show: auth.hasFunction('manage_users'),
    },
    {
      to: '/admin/categories',
      title: '属性マスタ',
      body: 'プロジェクト分類カテゴリと値の管理。',
      show: auth.hasFunction('manage_categories'),
    },
  ].filter((c) => c.show),
)
</script>

<template>
  <AppShell>
    <p class="eyebrow text-ink-muted">Dashboard</p>
    <h1 class="display-lg mt-4 text-ink">こんにちは、{{ auth.user.value?.name }} さん</h1>
    <p class="body-lg mt-4 max-w-2xl text-ink-muted">
      事前開発フェーズ。まずは認証・メンバー・属性マスタから整えていきます。
    </p>

    <div class="mt-12 grid gap-5 md:grid-cols-2">
      <NuxtLink
        v-for="card in cards"
        :key="card.to"
        :to="card.to"
        class="rounded-xl bg-surface-1 p-8 ring-1 ring-hairline transition-colors hover:bg-surface-2"
      >
        <h2 class="display-md text-ink">{{ card.title }}</h2>
        <p class="body-lg mt-3 text-ink-muted">{{ card.body }}</p>
      </NuxtLink>
    </div>
  </AppShell>
</template>
