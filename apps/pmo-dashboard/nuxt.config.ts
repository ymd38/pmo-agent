import tailwindcss from '@tailwindcss/vite'

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-05-01',
  devtools: { enabled: true },

  modules: ['@nuxt/eslint', '@nuxt/fonts'],

  css: ['~/assets/css/main.css'],

  vite: {
    plugins: [tailwindcss()],
  },

  app: {
    head: {
      htmlAttrs: { lang: 'ja' },
      title: 'PMO Agent — プロジェクト統制基盤',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        {
          name: 'description',
          content:
            'プロジェクトコード発行から工数・コスト管理、経営レポートまで。組織のすべてのプロジェクトを、ひとつの統制基盤で。',
        },
      ],
    },
  },

  // 表示はダークモード専用（DESIGN.md）。
  fonts: {
    families: [
      { name: 'Inter', provider: 'google', weights: [400, 500, 600, 700] },
      { name: 'Geist', provider: 'google', weights: [500, 600, 700] },
    ],
  },

  runtimeConfig: {
    // SSR（サーバー）から API を叩く際のベースURL。NUXT_API_BASE_SERVER で上書き。
    // docker では compose ネットワーク経由の http://api:8080 を使う。
    apiBaseServer: 'http://localhost:8080',
    public: {
      // ブラウザ（クライアント）から API を叩く際のベースURL。NUXT_PUBLIC_API_BASE で上書き。
      // docker でもブラウザはホスト経由なので http://localhost:8080 を使う。
      apiBase: 'http://localhost:8080',
    },
  },
})
