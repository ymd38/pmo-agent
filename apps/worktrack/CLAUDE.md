# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

工数入力UI。全メンバーが日次工数を入力するシンプルなNuxt 4アプリケーション。
PMO Dashboard（`../pmo-dashboard`）と同一APIを使用し、ポート3001で起動する。

## Stack

Nuxt 4 / TypeScript strict / Nuxt UI v4 / Tailwind CSS v4 / Vitest / ESLint

## Commands

```bash
# 開発サーバー起動（http://localhost:3001）
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

# Storybook（http://localhost:6007）
npm run storybook
```

## Routes & Middleware

全ページは Nuxt middleware で `input_work_hours` 権限を検証し、未認証は `/login` へリダイレクトする。
このアプリは `input_work_hours` 権限を持つユーザー専用。他権限のルートは存在しない。

| パス | 画面 |
|---|---|
| `/` | 自分の工数カレンダー（週表示） |
| `/projects` | 担当プロジェクト一覧 |
| `/input` | 工数入力フォーム |

## Architecture

- **API通信**: `useFetch` / `$fetch` でバックエンド（`/api/*`）を呼ぶ。認証はhttpOnly Cookie（自動送信）
- **認証状態**: `useAuth()` composable — PMO Dashboardと同じ実装パターン
- **エラーハンドリング**: `composables/useApiError.ts` に集約（PMO Dashboardと共通パターン）
- **自分の工数のみ編集可**: フォームにはログインユーザーの工数のみ表示・編集できる（APIもサーバー側で強制）

工数入力フォームの実装で考慮すべき点:
- 工数は0.5時間単位（`hours` フィールドは `DECIMAL(4,1)`）
- 同一日・同一プロジェクトへの重複入力はAPIがエラーを返す（`useApiError` で処理）
- 担当プロジェクト一覧は `GET /api/projects` のレスポンスを使う

## Mandatory Conventions

PMO Dashboardと同じ規約に従う:
- `<script setup lang="ts">` + Composition API 必須（Options API 禁止）
- Auto-import を最大限活用
- 禁止: `any` 型、`console.log`（本番コード）、直接DOM操作、未使用 import

## Component Design

Atomic Design で整理する（`atoms/` `molecules/` `organisms/`）。
Nuxt UI v4 コンポーネントを優先活用。独自実装は最小限に。

## Design System

**[`../../DESIGN.md`](../../DESIGN.md) を必ず参照すること。**

デザイントークンは `app/assets/css/main.css` の `@theme` ブロックで定義（PMO Dashboardと同じトークンセットを使用）。
デザイン変更時は `app/assets/css/main.css` と `../../DESIGN.md` を**同時に**更新すること。

## Style Rules（厳守）

- `<style>` ブロックへの CSS 直接記述は禁止
- `style="..."` インラインスタイル属性は禁止
- Tailwind 任意値（`text-[#FF0000]`・`mt-[13px]` 等）は禁止
- セマンティック命名を使う（`bg-canvas`・`text-ink-muted` 等。`blue-500` 等の直接参照禁止）
- ダークモード専用 — `dark:` プレフィックスは使わない

## Responsive

- **Mobile-first** で実装（モバイルをベースに `md:` `lg:` で上書き）
- ブレークポイント: `sm` 640px / `md` 768px / `lg` 1024px
- 工数カレンダーはモバイルで1日1列・PCで7日横並びになるよう `grid-cols-*` で実装

## Storybook

- Storybookはフロントエンドと独立したコンテナで起動（ポート6007）
- コンポーネント実装と `ComponentName.stories.ts` は**同時に作成**する（後回し禁止）
- `Default` Story は必須。バリアント（状態・エラー）ごとに Story を分ける

## Testing

- Vitest + `@nuxt/test-utils`
- テーブル駆動テストを基本とする
