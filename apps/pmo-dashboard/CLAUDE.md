# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

PMO管理画面。経営層・PMO管理者・PMが使用するNuxt 4アプリケーション。
工数管理UI（`../worktrack`）と同一APIを使用し、ポート3000で起動する。

## Stack

Nuxt 4 / TypeScript strict / Nuxt UI v4 / Tailwind CSS v4 / Vitest / ESLint

## Commands

```bash
# 開発サーバー起動（http://localhost:3000）
npm run dev

# ビルド
npm run build

# 型チェック
npm run typecheck

# Lint / 自動修正
npm run lint
npm run lint:fix

# テスト
npm run test
npm run test:watch

# Storybook（http://localhost:6006）
npm run storybook
```

## Routes & Middleware

全認証ページは Nuxt middleware で `function.code` を検査し、未認証は `/login` へリダイレクトする。

| パス | 画面 | 必要権限 |
|---|---|---|
| `/` | ダッシュボード | `view_dashboard` |
| `/programs` | プログラム一覧（配下プロジェクトをツリー表示） | `view_project_detail` |
| `/programs/new` | プログラム新規登録 | `issue_project_code` |
| `/programs/:id` | プログラム詳細（配下プロジェクト一覧・集計コスト） | `view_project_detail` |
| `/programs/:id/edit` | プログラム編集 | `issue_project_code` |
| `/programs/:id/projects/new` | プロジェクト新規作成 | `manage_projects` |
| `/projects/:id` | プロジェクト詳細（属性・メンバー・進捗・コスト） | `view_project_detail` |
| `/projects/:id/edit` | プロジェクト編集 | `manage_projects` |
| `/reports/executive` | エグゼクティブレポート | `view_executive_report` |
| `/reports/:id` | プロジェクト別レポート詳細 | `view_project_report` |
| `/admin/users` | ユーザー管理 | `manage_users` |
| `/admin/roles` | ロール・権限管理 | `manage_roles` |
| `/admin/categories` | 属性カテゴリ管理 | `manage_categories` |
| `/admin/grade-rates` | 単価管理（pmo_adminのみ） | `manage_grade_rates` |

## Architecture

- **API通信**: `useFetch` / `$fetch` でバックエンド（`/api/*`）を呼ぶ。認証はhttpOnly Cookie（自動送信）
- **認証状態**: `useAuth()` composable でログインユーザー・権限を管理。`GET /api/auth/me` をSSR時に呼ぶ
- **権限制御**: Nuxt middleware で `function.code` を検査。APIレスポンスのフィールド制御はバックエンド側が行うので、フロントはフィールドが返ってこないケースを想定して実装する
- **エラーハンドリング**: `composables/useApiError.ts` に集約し、全API呼び出しで共通利用
- **状態管理**: `useState` / Nuxt composables 優先。グローバル状態が必要な場合のみ Pinia を使用

## Mandatory Conventions

- `<script setup lang="ts">` + Composition API 必須（Options API 禁止）
- Auto-import を最大限活用（`ref`, `computed`, `useFetch` 等の明示的 import 不要）
- 禁止: `any` 型、`console.log`（本番コード）、直接DOM操作、未使用 import

## Component Design

Atomic Design で整理する:

```
components/
  atoms/       ボタン・バッジ・アイコン等の最小単位
  molecules/   フォーム行・カードヘッダー等の組み合わせ
  organisms/   プロジェクトテーブル・コストサマリー等の機能単位
```

- Nuxt UI v4 コンポーネントを優先活用。独自実装は最小限に
- Props は TypeScript で型安全に定義。1コンポーネント1責務

## Design System

**[`../../DESIGN.md`](../../DESIGN.md) を必ず参照すること。**

デザイントークンは `app/assets/css/main.css` の `@theme` ブロックで定義し、Tailwind クラス経由でのみ使用する:

```css
/* app/assets/css/main.css */
@theme {
  --color-canvas: #090909;
  --color-surface-1: #141414;
  --color-surface-2: #1c1c1c;
  --color-ink: #ffffff;
  --color-ink-muted: #999999;
  --color-accent-blue: #0099ff;
  --color-hairline: #262626;
}
```

デザイン変更時は `app/assets/css/main.css` と `../../DESIGN.md` を**同時に**更新すること。後回し禁止。

## Style Rules（厳守）

- `<style>` ブロック（scoped / global 問わず）への CSS 直接記述は禁止
- `style="..."` インラインスタイル属性は禁止
- 上記2点の例外：Tailwind では表現不可能な場合のみ `<style>` 使用可。コメントで理由を必ず明記
- Tailwind 任意値（`text-[#FF0000]`・`mt-[13px]` 等）は禁止。トークン未定義の値はまず `@theme` に追加する
- セマンティック命名を使う（`bg-canvas`・`text-ink-muted`・`border-hairline`）。`blue-500` 等の直接参照禁止

コンポーネント実装時のデザインルール（詳細は DESIGN.md）:
- ダークモード専用 — `dark:` プレフィックスは使わない（常にダーク）
- CTAは全て pill: `rounded-full`
- Inter Variable + `font-feature-settings: 'cv11' 'ss03'` 等をボディテキストに適用
- `accent-blue` はリンク・フォーカス・選択状態のみ。背景や CTA フィルに使わない

## Responsive

- **Mobile-first** で実装（モバイルをベースに `md:` `lg:` で上書き）
- ブレークポイント: `sm` 640px / `md` 768px / `lg` 1024px
- 固定px幅（`w-[375px]` 等）は禁止。`w-full` / `max-w-*` / `grid-cols-*` で可変レイアウト

## Storybook

- Storybookはフロントエンドと独立したコンテナで起動（ポート6006）
- コンポーネント実装と `ComponentName.stories.ts` は**同時に作成**する（後回し禁止）
- Story ファイルはコンポーネントと同階層に配置
- `Default` Story は必須。バリアント（サイズ・状態・エラー）ごとに Story を分ける
- インタラクション（クリック・フォーム送信）がある場合は `play` 関数でテストを記述
- atoms / molecules 層は Story 必須、organisms 層は主要バリアント必須

## Testing

- Vitest + `@nuxt/test-utils`
- テーブル駆動テストを基本とする
- 新機能時は期待出力・テストケースを先に明記（TDD推奨）

## File Structure

```
app/
  assets/css/main.css    デザイントークン（@theme）
  components/
    atoms/
    molecules/
    organisms/
  composables/
    useAuth.ts           ログインユーザー・権限管理
    useApiError.ts       APIエラーハンドリング共通
  middleware/
    auth.ts              JWT検証 + function.code チェック
  pages/
  types/                 APIレスポンス型（api/internal/domain と対応）
```
