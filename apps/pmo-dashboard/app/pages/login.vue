<script setup lang="ts">
useHead({ title: 'ログイン — PMO Agent' })

const auth = useAuth()
const route = useRoute()

const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

const inputClass =
  'w-full rounded-md bg-surface-1 px-4 py-3 text-sm text-ink placeholder:text-ink-muted outline-none ring-1 ring-hairline transition-shadow focus-visible:ring-2 focus-visible:ring-accent-blue/40'

async function onSubmit() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(email.value, password.value)
    // オープンリダイレクト対策: サイト内パス（先頭が "/" かつ "//" でない）のみ許可する
    const raw = route.query.redirect
    const redirect = typeof raw === 'string' && raw.startsWith('/') && !raw.startsWith('//') ? raw : '/home'
    await navigateTo(redirect)
  } catch {
    error.value = 'メールアドレスまたはパスワードが正しくありません'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="relative flex min-h-screen flex-col bg-canvas text-ink">
    <div class="hero-aura pointer-events-none absolute inset-0" aria-hidden="true" />

    <header class="relative mx-auto flex h-14 w-full max-w-6xl items-center px-6">
      <NuxtLink to="/" aria-label="PMO Agent ホーム">
        <BrandMark />
      </NuxtLink>
    </header>

    <main class="relative flex flex-1 items-center justify-center px-6 py-16">
      <div class="w-full max-w-sm">
        <h1 class="display-md text-ink">ログイン</h1>
        <p class="mt-2 text-sm text-ink-muted">プロジェクト統制基盤</p>

        <form class="mt-8 flex flex-col gap-4" @submit.prevent="onSubmit">
          <label class="flex flex-col gap-2">
            <span class="text-sm text-ink-muted">メールアドレス</span>
            <input
              v-model="email"
              type="email"
              autocomplete="email"
              placeholder="you@example.com"
              :class="inputClass"
            >
          </label>

          <label class="flex flex-col gap-2">
            <span class="text-sm text-ink-muted">パスワード</span>
            <input
              v-model="password"
              type="password"
              autocomplete="current-password"
              placeholder="••••••••"
              :class="inputClass"
            >
          </label>

          <p v-if="error" class="text-sm text-grad-coral" role="alert">{{ error }}</p>

          <PillButton variant="primary" type="submit" class="mt-2 w-full">
            {{ loading ? 'ログイン中…' : 'ログイン' }}
          </PillButton>
        </form>

        <p class="mt-6 text-xs text-ink-muted">
          アカウントはPMO管理者が発行します。ログインできない場合は管理者にお問い合わせください。
        </p>

        <NuxtLink
          to="/"
          class="mt-8 inline-block text-sm text-accent-blue transition-opacity hover:opacity-80"
        >
          ← トップへ戻る
        </NuxtLink>
      </div>
    </main>
  </div>
</template>
