<script setup lang="ts">
// トップナビ（DESIGN.md top-nav）。canvas 背景、高さ 56px、右にログイン pill。
const links = [
  { label: '機能', href: '#capabilities' },
  { label: '仕組み', href: '#how' },
  { label: 'ロードマップ', href: '#roadmap' },
]
</script>

<template>
  <header
    class="sticky top-0 z-50 border-b border-hairline-soft bg-canvas/80 backdrop-blur-md"
  >
    <nav
      class="mx-auto flex h-14 max-w-6xl items-center justify-between px-6"
      aria-label="グローバル"
    >
      <NuxtLink to="/" aria-label="PMO Agent ホーム">
        <BrandMark />
      </NuxtLink>

      <ul class="hidden items-center gap-8 md:flex">
        <li v-for="link in links" :key="link.href">
          <a
            :href="link.href"
            class="text-sm text-ink-muted transition-colors hover:text-ink"
          >
            {{ link.label }}
          </a>
        </li>
      </ul>

      <PillButton variant="secondary" to="/login">ログイン</PillButton>
    </nav>
  </header>
</template>
