<script setup lang="ts">
// CTAは全て pill 形状（DESIGN.md）。primary=白ピル / secondary=チャコールピル。
// 枠線のみのゴーストボタンは使わない。
interface Props {
  variant?: 'primary' | 'secondary'
  to?: string
  href?: string
  type?: 'button' | 'submit'
}

const { variant = 'primary', to = undefined, href = undefined, type = 'button' } = defineProps<Props>()

const base =
  'inline-flex min-h-11 items-center justify-center gap-2 rounded-pill px-5 py-3 text-sm font-medium leading-none transition-transform duration-150 will-change-transform active:scale-95 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-blue/40'

const variants: Record<NonNullable<Props['variant']>, string> = {
  primary: 'bg-ink text-canvas hover:bg-ink/90',
  secondary: 'bg-surface-1 text-ink hover:bg-surface-2',
}
</script>

<template>
  <NuxtLink v-if="to" :to="to" :class="[base, variants[variant]]">
    <slot />
  </NuxtLink>
  <a v-else-if="href" :href="href" :class="[base, variants[variant]]">
    <slot />
  </a>
  <button v-else :type="type" :class="[base, variants[variant]]">
    <slot />
  </button>
</template>
