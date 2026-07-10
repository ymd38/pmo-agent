#!/usr/bin/env bash
# Stop hook: 作業ツリーに変更のあるモノレポ・プロジェクトのテストを実行する。
# 失敗時は decision:block で結果を返し、修正を促す。テストが未整備なら静かに no-op。
set -uo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root" || exit 0

changed="$({ git diff --name-only; git diff --name-only --cached; git ls-files --others --exclude-standard; } 2>/dev/null | sort -u)"
[ -z "$changed" ] && exit 0

ran=0
fails=""

# --- api (Go) ---
if echo "$changed" | grep -q '^api/.*\.go$' && [ -f api/go.mod ] && command -v go >/dev/null 2>&1; then
  ran=1
  if ! out="$(cd api && go test -race ./... 2>&1)"; then
    fails="${fails}

[api]
$(echo "$out" | tail -n 30)"
  fi
fi

# --- フロント (Nuxt/Vitest) ---
for app in pmo-dashboard worktrack; do
  if echo "$changed" | grep -q "^apps/$app/" \
     && [ -f "apps/$app/package.json" ] \
     && jq -e '.scripts.test' "apps/$app/package.json" >/dev/null 2>&1 \
     && command -v npm >/dev/null 2>&1; then
    ran=1
    if ! out="$(cd "apps/$app" && CI=true npm test --silent 2>&1)"; then
      fails="${fails}

[apps/$app]
$(echo "$out" | tail -n 30)"
    fi
  fi
done

if [ -n "$fails" ]; then
  jq -n --arg r "変更プロジェクトのテストが失敗しています。完了前に修正してください:${fails}" \
    '{decision:"block", reason:$r}'
elif [ "$ran" -eq 1 ]; then
  echo '{"systemMessage":"✓ 変更プロジェクトのテストが全てパスしました"}'
fi
exit 0
