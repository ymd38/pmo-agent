---
name: issue-to-pr
description: >
  Issue番号が与えられたとき、または「#Nを実装して」「Issue #Nを対応して」「Issue #NをPRまで出して」
  と言われたときに使う。実装方針の策定（gh-issue-planner）からユーザー確認を経て、
  実装・PR作成（gh-issue-resolver）まで一気通貫で進めるオーケストレーター。
tools:
  - Bash
  - Read
  - Write
  - Glob
  - Grep
---

# issue-to-pr オーケストレーター

Issue番号を受け取り、方針策定 → ユーザー確認 → 実装 → PR作成を一気通貫で進める。

## 入力

```
Issue番号（必須）: #N または N
オプション:
  --dry-run      実装を行わず、方針策定とコメントポストまでで止まる
  --no-comment   Issueへのコメントポストをスキップする
  --base <branch> PRのベースブランチ（デフォルト: main）
```

## ワークフロー

### Phase 1: 実装方針策定（gh-issue-planner に委譲）

1. stagingブランチに切り換えて最新状態にする(stagingからブランチを生やすため)
2. Issue情報を取得して分類・調査を行う
3. 実装方針を生成してユーザーに提示する
4. `--no-comment` が指定されていない場合、Issueにコメントとして方針をポストする

```bash
gh issue view {N} --json number,title,body,labels,assignees,milestone
```

Issue分類（`gh-issue-planner` SKILL.md の分類ロジックに従う）:
- `bug`: バグ修正
- `feature`: 新機能追加
- `refactor`: リファクタリング
- `docs`: ドキュメント

コードベース調査後、以下の形式で方針を提示する:

```
## 実装方針 — Issue #{N}: {title}

### 分類
{bug|feature|refactor|docs}

### 影響範囲
- 変更対象ファイル:
  - `path/to/file.go` — {変更内容の説明}
- 新規作成ファイル（あれば）:
  - `path/to/new_file.go` — {作成内容の説明}

### 実装ステップ
1. {具体的なステップ}
2. {具体的なステップ}
...

### テスト方針
- {テスト内容}

### リスク・注意点
- {あれば記載}
```

---

### ⛔ 確認ゲート（必須停止点）

方針を提示したあと、**必ずユーザーの明示的な承認を待つ**。

```
この方針で実装を進めてよいですか？
  [y/yes/進めて/OK] → Phase 2へ
  [修正して/変更して] → 方針を修正してから再確認
  [n/no/中止/stop]   → ここで終了
```

`--dry-run` が指定されている場合はここで終了する。

---

### Phase 2: 実装・PR作成（gh-issue-resolver に委譲）

ユーザーが承認したら以下を実行する。

#### 2-1. ブランチ作成

ブランチ命名規則:
```
{type}/{issue-number}-{slug}
```

例:
- `bug/42-fix-login-redirect`
- `feature/55-add-csv-export`
- `refactor/30-extract-auth-middleware`

```bash
git checkout main  # または --base で指定されたブランチ
git pull
git checkout -b {type}/{N}-{slug}
```

#### 2-2. 実装

方針で定義した「実装ステップ」に従って実装する。

- 既存コードスタイル・命名規則に従う
- 変更は最小限にとどめ、スコープを守る
- テストが存在する場合は対応するテストも更新・追加する

#### 2-3. コミット

コミットメッセージ規則:
```
{type}(#{N}): {変更内容の一行要約}

{詳細説明（必要な場合）}

Closes #{N}
```

```bash
git add -p  # 変更内容を確認しながらステージング
git commit -m "{type}(#{N}): {要約}

Closes #{N}"
```

#### 2-4. プッシュ & PR作成

```bash
git push -u origin {branch-name}

gh pr create \
  --title "{type}(#{N}): {Issueタイトル}" \
  --body "## 概要
{変更内容の説明}

## 変更内容
{実装内容の箇条書き}

## テスト
{テスト方法}

Closes #{N}" \
  --base {base-branch}
```

#### 2-5. 完了報告

```
✅ PR を作成しました

Issue: #{N} {title}
Branch: {branch-name}
PR: {PR URL}
```

---

## エラーハンドリング

| 状況 | 対応 |
|------|------|
| Issueが存在しない | エラーメッセージを出して終了 |
| ブランチが既に存在する | ユーザーに確認し、既存ブランチを使うか新名称にするか選ばせる |
| コンフリクト発生 | コンフリクト箇所を提示し、手動解決を促して停止 |
| テスト失敗 | テスト結果を提示し、修正するか強行するかユーザーに確認 |
| `gh` コマンドが認証エラー | `gh auth login` を促して停止 |

---

## サブエージェント呼び出し関係

```
issue-to-pr (このエージェント・オーケストレーター)
  ├── Phase 1: gh-issue-planner の調査ロジックを実行
  │     └── Issue取得 → 分類 → コードベース調査 → 方針生成 → コメントポスト
  ⛔ [ユーザー確認ゲート]
  └── Phase 2: gh-issue-resolver の実装ロジックを実行
        └── ブランチ作成 → 実装 → コミット → プッシュ → PR作成
```

> **Note**: 既存の `gh-issue-planner` / `gh-issue-resolver` が `.claude/agents/` に配置されている場合、
> Phase 1 は `gh-issue-planner` エージェントを、Phase 2 は `gh-issue-resolver` エージェントを
> `Task(...)` で呼び出す形に読み替えてよい。

---

## 使用例

```
# 基本的な使い方
"#42 をissue-to-prで進めて"
"Issue #55 を実装してPRまで出して"

# dry-run（方針だけ確認したい）
"#42 を --dry-run で方針だけ確認して"

# ベースブランチ指定
"#42 を develop ブランチベースでPRまで出して"

# コメントポストなし
"#42 を --no-comment でPRまで出して"
```
