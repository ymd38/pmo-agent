// バックエンド（/api）への $fetch クライアント。
// 認証は httpOnly Cookie。SSR時はブラウザの Cookie をサーバー→API へ転送する。
// SSR（コンテナ内）とブラウザでは API の到達経路が異なるため、ベースURLを出し分ける。
export function useApi() {
  const config = useRuntimeConfig()
  const base = import.meta.server ? config.apiBaseServer : config.public.apiBase
  const headers = import.meta.server ? useRequestHeaders(['cookie']) : undefined

  return $fetch.create({
    baseURL: `${base}/api`,
    credentials: 'include',
    headers,
  })
}
