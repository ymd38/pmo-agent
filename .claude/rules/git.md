# Git 運用規約

## ブランチ命名規則

```
<type>/<issue-number>-<kebab-case-summary>
```

### type 一覧

| type | 用途 |
|---|---|
| `feat` | 新機能追加 |
| `fix` | バグ修正 |
| `refactor` | 機能変更なしのリファクタリング |
| `style` | UI・スタイルのみの変更（ロジック変更なし）|
| `docs` | ドキュメント・コメントのみ |
| `test` | テストの追加・修正のみ |
| `chore` | ビルド・設定・依存関係の変更 |
| `db` | マイグレーションファイルの追加・修正 |

### 例

```
feat/12-project-code-issuance
fix/34-work-hours-scope-check
style/8-dashboard-spacing
db/21-add-grade-rates-table
```

## コミットメッセージ規約（Conventional Commits）

```
<type>(<scope>): <summary>

[body（任意）]
```

### ルール

- summary は命令形・現在形で書く（日本語可）。過去形・体言止め禁止
- summary は72文字以内
- scope はオプション。影響範囲が明確な場合に使う
- body は変更の「なぜ」を書く。「何を変えたか」はコードを見ればわかる

### scope 例

| scope | 対象 |
|---|---|
| `auth` | 認証・JWT・OTP |
| `project` | プロジェクト管理 |
| `worktrack` | 工数入力・集計 |
| `report` | 日次・週次・エグゼクティブレポート |
| `admin` | ユーザー・ロール・単価管理 |
| `acl` | 権限・スコープ制御 |
| `db` | マイグレーション |
| `infra` | Docker・CI・設定 |
| `n8n` | Backlog連携ワークフロー |

### 例

```
feat(project): プロジェクトコード発行エンドポイントを追加する
fix(acl): 担当プロジェクト以外の工数が閲覧できるバグを修正する
refactor(auth): DIコンテナのJWT依存解決順序を整理する
style(dashboard): コストサマリーカードの余白をトークンに統一する
db: grade_rates テーブルに年度カラムを追加するマイグレーションを作成する
chore: docker-compose に storybook サービスを追加する
```

### 禁止パターン

```
# NG：何を変えたか だけで なぜ が不明
fix: バグ修正

# NG：過去形
feat: 認証を追加した

# NG：コミットメッセージに issue 番号だけ
#42

# NG：WIPコミットをそのままPRに含める
WIP: 途中
```

## PR（Pull Request）規約

### タイトル

コミットメッセージと同じ形式で書く。

```
feat(project): プロジェクトコード発行フローを実装する
```

### 本文テンプレート

```markdown
## 概要
<!-- 何を・なぜ変更したか、1〜3行で -->

## 変更内容
<!-- 主な変更点を箇条書き -->

## 動作確認
<!-- 確認した手順・環境 -->

## 関連 Issue
closes #<issue-number>
```

### PR のルール

- 1PR = 1つの目的。複数の機能を混在させない
- フロント変更は必ず Storybook で確認済みであること
- バックエンド変更は `go test -race ./...` がパスしていること
- マイグレーションを含む場合は `make migrate-up` の実行確認を PR 本文に記載する
- レビュー前に自分でセルフレビュー（diff を一読）すること

## その他

- `main` ブランチへの直接 push 禁止
- マージ方式：Squash merge を基本とする（コミット履歴をきれいに保つ）
- ブランチは PR マージ後に削除する
