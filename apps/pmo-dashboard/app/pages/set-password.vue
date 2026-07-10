<script setup lang="ts">
useHead({ title: 'パスワード設定 — PMO Agent' })

const route = useRoute()
const api = useApi()
const token = computed(() => (route.query.token as string) || '')

const email = ref('')
const password = ref('')
const confirm = ref('')
const error = ref('')
const done = ref(false)
const loading = ref(false)

const inputClass =
  'w-full rounded-md bg-surface-1 px-4 py-3 text-sm text-ink placeholder:text-ink-muted outline-none ring-1 ring-hairline transition-shadow focus-visible:ring-2 focus-visible:ring-accent-blue/40'

// トークンの有効性を確認し、対象メールを表示する。
const { error: verifyError } = await useAsyncData('verify-set-token', async () => {
  if (!token.value) throw new Error('no token')
  const res = await api<{ email: string }>(`/auth/set-password/${token.value}`)
  email.value = res.email
  return res
})

async function onSubmit() {
  error.value = ''
  if (password.value.length < 8) {
    error.value = 'パスワードは8文字以上にしてください'
    return
  }
  if (password.value !== confirm.value) {
    error.value = 'パスワードが一致しません'
    return
  }
  loading.value = true
  try {
    await api('/auth/set-password', {
      method: 'POST',
      body: { token: token.value, password: password.value },
    })
    done.value = true
  } catch {
    error.value = 'パスワードの設定に失敗しました。リンクが期限切れの可能性があります。'
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
        <!-- トークン無効 -->
        <template v-if="verifyError">
          <h1 class="display-md text-ink">リンクが無効です</h1>
          <p class="mt-3 text-sm text-ink-muted">
            このリンクは期限切れか、すでに使用されています。PMO管理者にリンクの再発行を依頼してください。
          </p>
          <NuxtLink to="/login" class="mt-8 inline-block text-sm text-accent-blue hover:opacity-80">
            ← ログインへ
          </NuxtLink>
        </template>

        <!-- 設定完了 -->
        <template v-else-if="done">
          <h1 class="display-md text-ink">設定が完了しました</h1>
          <p class="mt-3 text-sm text-ink-muted">新しいパスワードでログインできます。</p>
          <PillButton variant="primary" to="/login" class="mt-8">ログインへ</PillButton>
        </template>

        <!-- 設定フォーム -->
        <template v-else>
          <h1 class="display-md text-ink">パスワード設定</h1>
          <p class="mt-2 text-sm text-ink-muted">{{ email }}</p>

          <form class="mt-8 flex flex-col gap-4" @submit.prevent="onSubmit">
            <label class="flex flex-col gap-2">
              <span class="text-sm text-ink-muted">新しいパスワード（8文字以上）</span>
              <input v-model="password" type="password" autocomplete="new-password" placeholder="••••••••" :class="inputClass">
            </label>
            <label class="flex flex-col gap-2">
              <span class="text-sm text-ink-muted">パスワード（確認）</span>
              <input v-model="confirm" type="password" autocomplete="new-password" placeholder="••••••••" :class="inputClass">
            </label>

            <p v-if="error" class="text-sm text-grad-coral" role="alert">{{ error }}</p>

            <PillButton variant="primary" type="submit" class="mt-2 w-full">
              {{ loading ? '設定中…' : 'パスワードを設定' }}
            </PillButton>
          </form>
        </template>
      </div>
    </main>
  </div>
</template>
