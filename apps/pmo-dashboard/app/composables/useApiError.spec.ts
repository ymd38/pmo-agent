import { describe, expect, it } from 'vitest'
import { apiError } from './useApiError'

describe('apiError', () => {
  const tests: { name: string, input: unknown, fallback: string, want: string }[] = [
    {
      name: 'APIのエラーレスポンス（data.error）からメッセージを取り出す',
      input: { data: { error: 'コードは発行済みです' } },
      fallback: '失敗しました',
      want: 'コードは発行済みです',
    },
    {
      name: 'data.error が無ければフォールバックを返す',
      input: { data: {} },
      fallback: '失敗しました',
      want: '失敗しました',
    },
    {
      name: 'ネットワークエラー等の非オブジェクトでもフォールバックを返す',
      input: new Error('fetch failed'),
      fallback: '通信に失敗しました',
      want: '通信に失敗しました',
    },
    {
      name: 'null/undefined でもフォールバックを返す',
      input: null,
      fallback: '失敗しました',
      want: '失敗しました',
    },
  ]

  it.each(tests)('$name', ({ input, fallback, want }) => {
    expect(apiError(input, fallback)).toBe(want)
  })
})
