import type { MeResponse, User } from '~/types/api'

// ログインユーザー・権限を管理する composable。状態は useState で SSR/CSR 共有。
export function useAuth() {
  const user = useState<User | null>('auth_user', () => null)
  const functions = useState<string[]>('auth_functions', () => [])
  const api = useApi()

  function set(me: MeResponse | null) {
    user.value = me?.user ?? null
    functions.value = me?.functions ?? []
  }

  async function fetchMe(): Promise<boolean> {
    try {
      set(await api<MeResponse>('/auth/me'))
      return true
    } catch {
      set(null)
      return false
    }
  }

  // ensureSession: /auth/me が 401 ならリフレッシュを一度試してから再取得する。
  async function ensureSession(): Promise<boolean> {
    if (user.value) return true
    if (await fetchMe()) return true
    try {
      await api('/auth/refresh', { method: 'POST' })
    } catch {
      return false
    }
    return fetchMe()
  }

  async function login(email: string, password: string): Promise<void> {
    await api('/auth/login', { method: 'POST', body: { email, password } })
    await fetchMe()
  }

  async function logout(): Promise<void> {
    try {
      await api('/auth/logout', { method: 'POST' })
    } finally {
      set(null)
    }
  }

  const hasFunction = (code: string) => functions.value.includes(code)

  return { user, functions, login, logout, fetchMe, ensureSession, hasFunction }
}
