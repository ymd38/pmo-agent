#!/usr/bin/env bash
# PostToolUse(Write|Edit) hook: 編集された単一ファイルを整形+lintする。
# モノレポのパスでツールをルーティング。ツール/プロジェクトが未整備なら静かに no-op。
# 整形は自動修正（gofmt -w / eslint --fix）、残った lint 指摘はモデルへ通知（非ブロッキング）。
set -uo pipefail

file="$(jq -r '.tool_input.file_path // .tool_response.filePath // empty' 2>/dev/null)"
[ -n "$file" ] && [ -f "$file" ] || exit 0

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
rel="${file#"$repo_root"/}"
issues=""

if [[ "$rel" == api/*.go ]]; then
  command -v gofmt >/dev/null 2>&1 && gofmt -w "$file"
  if command -v golangci-lint >/dev/null 2>&1 && [ -f "$repo_root/api/go.mod" ]; then
    issues="$(cd "$(dirname "$file")" && golangci-lint run 2>&1)" || true
  fi
elif [[ "$rel" =~ ^apps/(pmo-dashboard|worktrack)/.*\.(ts|vue|js|mts|cts)$ ]]; then
  app="$repo_root/apps/${BASH_REMATCH[1]}"
  eslint_bin="$app/node_modules/.bin/eslint"
  # ローカルにインストール済みの eslint のみ使用（npx のネットワーク取得は行わない）
  if [ -x "$eslint_bin" ]; then
    issues="$(cd "$app" && "$eslint_bin" --fix "$file" 2>&1)" || true
  fi
fi

if [ -n "${issues//[[:space:]]/}" ]; then
  jq -n --arg c "Lint結果 ($rel):
$issues" '{hookSpecificOutput:{hookEventName:"PostToolUse", additionalContext:$c}}'
fi
exit 0
