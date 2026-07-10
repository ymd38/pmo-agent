<script setup lang="ts">
// 認証済みエリアの共通シェル（トップバー＋本文スロット）。
const auth = useAuth()

const nav = computed(() =>
  [
    { to: '/home', label: 'ホーム', show: true },
    { to: '/programs', label: 'プログラム', show: auth.hasFunction('view_project_detail') },
    { to: '/admin/users', label: 'メンバー管理', show: auth.hasFunction('manage_users') },
    { to: '/admin/categories', label: '属性マスタ', show: auth.hasFunction('manage_categories') },
  ].filter((n) => n.show),
)

async function onLogout() {
  await auth.logout()
  await navigateTo('/login')
}
</script>

<template>
  <div class="min-h-screen bg-canvas text-ink">
    <header class="sticky top-0 z-50 border-b border-hairline-soft bg-canvas/80 backdrop-blur-md">
      <div class="mx-auto flex h-14 max-w-6xl items-center justify-between px-6">
        <div class="flex items-center gap-8">
          <NuxtLink to="/home" aria-label="ホーム"><BrandMark /></NuxtLink>
          <nav class="hidden items-center gap-6 md:flex">
            <NuxtLink
              v-for="n in nav"
              :key="n.to"
              :to="n.to"
              class="text-sm text-ink-muted transition-colors hover:text-ink"
              active-class="text-ink"
            >
              {{ n.label }}
            </NuxtLink>
          </nav>
        </div>
        <div class="flex items-center gap-4">
          <span class="hidden text-sm text-ink-muted sm:inline">{{ auth.user.value?.name }}</span>
          <PillButton variant="secondary" @click="onLogout">ログアウト</PillButton>
        </div>
      </div>
    </header>

    <main class="mx-auto max-w-6xl px-6 py-10">
      <slot />
    </main>
  </div>
</template>
