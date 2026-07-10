<script setup lang="ts">
import type { Role, User } from '~/types/api'

definePageMeta({ middleware: 'auth', requiredFunction: 'manage_users' })
useHead({ title: 'メンバー管理 — PMO Agent' })

const api = useApi()

const users = ref<User[]>([])
const roles = ref<Role[]>([])
const inviteLink = ref('')
const errorMsg = ref('')
const showForm = ref(false)

const form = reactive({
  email: '',
  name: '',
  grade: 'staff' as 'manager' | 'staff',
  roleIds: [] as number[],
})

const inputClass =
  'w-full rounded-md bg-surface-1 px-4 py-2.5 text-sm text-ink placeholder:text-ink-muted outline-none ring-1 ring-hairline focus-visible:ring-2 focus-visible:ring-accent-blue/40'

async function load() {
  try {
    const [u, r] = await Promise.all([
      api<{ users: User[] }>('/users'),
      api<{ roles: Role[] }>('/roles'),
    ])
    users.value = u.users
    roles.value = r.roles
  } catch (e) {
    errorMsg.value = apiError(e, 'メンバー一覧の取得に失敗しました')
  }
}

onMounted(load)

function resetForm() {
  form.email = ''
  form.name = ''
  form.grade = 'staff'
  form.roleIds = []
}

async function onCreate() {
  errorMsg.value = ''
  inviteLink.value = ''
  try {
    const res = await api<{ set_password_url: string }>('/users', {
      method: 'POST',
      body: {
        email: form.email,
        name: form.name,
        grade: form.grade,
        role_ids: form.roleIds,
      },
    })
    inviteLink.value = res.set_password_url
    showForm.value = false
    resetForm()
    await load()
  } catch (e) {
    errorMsg.value = apiError(e, 'メンバーの作成に失敗しました')
  }
}

async function onReissue(id: number) {
  errorMsg.value = ''
  try {
    const res = await api<{ set_password_url: string }>(`/users/${id}/reissue-link`, { method: 'POST' })
    inviteLink.value = res.set_password_url
  } catch (e) {
    errorMsg.value = apiError(e, 'リンク再発行に失敗しました')
  }
}

async function onDeactivate(id: number) {
  errorMsg.value = ''
  try {
    await api(`/users/${id}`, { method: 'DELETE' })
    await load()
  } catch (e) {
    errorMsg.value = apiError(e, '無効化に失敗しました')
  }
}


function roleNames(u: User): string {
  return (u.roles ?? []).map((r) => r.name).join('、') || '—'
}
</script>

<template>
  <AppShell>
    <div class="flex items-center justify-between">
      <div>
        <p class="eyebrow text-ink-muted">Members</p>
        <h1 class="display-md mt-3 text-ink">メンバー管理</h1>
      </div>
      <PillButton variant="primary" @click="showForm = !showForm">
        {{ showForm ? '閉じる' : '新規メンバー' }}
      </PillButton>
    </div>

    <p v-if="errorMsg" class="mt-4 text-sm text-grad-coral" role="alert">{{ errorMsg }}</p>

    <!-- 招待リンク表示 -->
    <div v-if="inviteLink" class="mt-6 rounded-lg bg-surface-1 p-5 ring-1 ring-accent-blue/40">
      <p class="text-sm text-ink">招待/設定リンク（本人へ共有してください）</p>
      <code class="mt-2 block break-all text-sm text-accent-blue">{{ inviteLink }}</code>
    </div>

    <!-- 作成フォーム -->
    <form
      v-if="showForm"
      class="mt-6 grid gap-4 rounded-xl bg-surface-1 p-6 md:grid-cols-2"
      @submit.prevent="onCreate"
    >
      <label class="flex flex-col gap-2">
        <span class="text-sm text-ink-muted">メールアドレス</span>
        <input v-model="form.email" type="email" placeholder="you@example.com" :class="inputClass">
      </label>
      <label class="flex flex-col gap-2">
        <span class="text-sm text-ink-muted">氏名</span>
        <input v-model="form.name" type="text" placeholder="山田 太郎" :class="inputClass">
      </label>
      <label class="flex flex-col gap-2">
        <span class="text-sm text-ink-muted">グレード</span>
        <select v-model="form.grade" :class="inputClass">
          <option value="staff">staff</option>
          <option value="manager">manager</option>
        </select>
      </label>
      <fieldset class="flex flex-col gap-2">
        <span class="text-sm text-ink-muted">ロール（1つ以上）</span>
        <div class="flex flex-wrap gap-3">
          <label v-for="r in roles" :key="r.id" class="flex items-center gap-2 text-sm text-ink">
            <input v-model="form.roleIds" type="checkbox" :value="r.id" class="accent-accent-blue">
            {{ r.name }}
          </label>
        </div>
      </fieldset>
      <div class="md:col-span-2">
        <PillButton variant="primary" type="submit">発行する</PillButton>
      </div>
    </form>

    <!-- 一覧 -->
    <div class="mt-8 overflow-hidden rounded-xl ring-1 ring-hairline">
      <table class="w-full text-left text-sm">
        <thead class="bg-surface-1 text-ink-muted">
          <tr>
            <th class="px-4 py-3 font-medium">氏名</th>
            <th class="px-4 py-3 font-medium">メール</th>
            <th class="px-4 py-3 font-medium">グレード</th>
            <th class="px-4 py-3 font-medium">ロール</th>
            <th class="px-4 py-3 font-medium">状態</th>
            <th class="px-4 py-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.id" class="border-t border-hairline-soft">
            <td class="px-4 py-3 text-ink">{{ u.name }}</td>
            <td class="px-4 py-3 text-ink-muted">{{ u.email }}</td>
            <td class="px-4 py-3 text-ink-muted">{{ u.grade }}</td>
            <td class="px-4 py-3 text-ink-muted">{{ roleNames(u) }}</td>
            <td class="px-4 py-3">
              <span :class="u.is_active ? 'text-success' : 'text-ink-muted'">
                {{ u.is_active ? '有効' : '無効' }}
              </span>
            </td>
            <td class="px-4 py-3">
              <div class="flex gap-3">
                <button class="text-accent-blue hover:opacity-80" @click="onReissue(u.id)">リンク再発行</button>
                <button v-if="u.is_active" class="text-ink-muted hover:text-ink" @click="onDeactivate(u.id)">無効化</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </AppShell>
</template>
