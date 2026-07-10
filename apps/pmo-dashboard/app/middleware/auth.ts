// 認証ルートミドルウェア。未認証は /login へ、権限不足は /home へリダイレクト。
// 各ページは definePageMeta({ middleware: 'auth', requiredFunction: 'manage_users' }) で利用する。
export default defineNuxtRouteMiddleware(async (to) => {
  const auth = useAuth()

  const authed = await auth.ensureSession()
  if (!authed) {
    return navigateTo(`/login?redirect=${encodeURIComponent(to.fullPath)}`)
  }

  const required = to.meta.requiredFunction as string | undefined
  if (required && !auth.hasFunction(required)) {
    return navigateTo('/home')
  }
})
