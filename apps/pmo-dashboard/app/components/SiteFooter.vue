<script setup lang="ts">
const year = new Date().getFullYear()
</script>

<template>
  <footer class="border-t border-hairline-soft bg-canvas">
    <div
      class="mx-auto flex max-w-6xl flex-col gap-6 px-6 py-12 md:flex-row md:items-center md:justify-between"
    >
      <div class="flex flex-col gap-3">
        <BrandMark />
        <p class="text-xs text-ink-muted">
          プロジェクト統制基盤 — PMO Agent
        </p>
      </div>

      <div class="flex items-center gap-6 text-xs text-ink-muted">
        <a href="#capabilities" class="transition-colors hover:text-ink">機能</a>
        <a href="#how" class="transition-colors hover:text-ink">仕組み</a>
        <NuxtLink to="/login" class="transition-colors hover:text-ink">ログイン</NuxtLink>
      </div>
    </div>

    <div class="border-t border-hairline-soft">
      <p class="mx-auto max-w-6xl px-6 py-5 text-xs text-ink-muted">
        © {{ year }} PMO Agent
      </p>
    </div>
  </footer>
</template>
