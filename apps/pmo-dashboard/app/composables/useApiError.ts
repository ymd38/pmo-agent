// APIエラーからユーザー向けメッセージを取り出す共通ヘルパー。
// バックエンドは {"error": "..."} 形式で返す（api/internal/handler/response.go）。
export function apiError(e: unknown, fallback: string): string {
  return (e as { data?: { error?: string } })?.data?.error ?? fallback
}
