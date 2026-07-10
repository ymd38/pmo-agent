<script setup lang="ts">
// グラデーション spotlight カード（DESIGN.md シグネチャ）。
// 黒キャンバスに対する彩度で elevation を表現。1ビューに2〜3点まで。
interface Props {
  variant?: 'violet' | 'magenta' | 'orange' | 'coral'
  eyebrow?: string
  title: string
}

const { variant = 'violet', eyebrow = undefined } = defineProps<Props>()

const grounds: Record<NonNullable<Props['variant']>, string> = {
  violet: 'spotlight-violet',
  magenta: 'spotlight-magenta',
  orange: 'spotlight-orange',
  coral: 'spotlight-coral',
}
</script>

<template>
  <article
    :class="[
      'relative flex min-h-72 flex-col justify-start overflow-hidden rounded-xxl p-8 ring-1 ring-white/10 shadow-2xl shadow-black/40',
      grounds[variant],
    ]"
  >
    <!-- 上端の光のエッジ（level-2 light-edge） -->
    <span
      class="pointer-events-none absolute inset-x-0 top-0 h-px bg-white/25"
      aria-hidden="true"
    />
    <p v-if="eyebrow" class="eyebrow mb-3 text-white/70">{{ eyebrow }}</p>
    <h3 class="cap-title display-md max-w-sm text-white">{{ title }}</h3>
    <p class="body-lg mt-4 max-w-md text-white/85">
      <slot />
    </p>
  </article>
</template>

<style scoped>
/* タイトルは1〜2行で行数が揺れるため、本文の開始位置を複数カードで揃える目的で
   タイトル領域に2行分の高さを確保する。行数ベースの最小高さは Tailwind の
   デザイントークンでは表現できないため、ここでのみ素の CSS を用いる。 */
.cap-title {
  min-height: 2lh;
}
</style>
