#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FIXTURE_ROOT="${DOTBOT_GO_TUI_CHECK_ROOT:-/tmp/dotbot-go-tui-check}"
CACHE_ROOT="${DOTBOT_GO_TUI_CHECK_CACHE_ROOT:-/tmp/dotbot-go-tui-cache}"
FIXTURE_REPO="$FIXTURE_ROOT/repo"
FIXTURE_HOME="$FIXTURE_ROOT/home"
MAIN_CONFIG="$FIXTURE_REPO/dotbot-go.toml"
RISKY_CONFIG="$FIXTURE_REPO/risky.toml"
GO_MOD_CACHE="$CACHE_ROOT/mod"
GO_BUILD_CACHE="$CACHE_ROOT/build"

usage() {
  cat <<EOF
用法:
  scripts/tui-check.sh prepare
  scripts/tui-check.sh dry-run
  scripts/tui-check.sh check
  scripts/tui-check.sh risky
  scripts/tui-check.sh fallback-dry-run
  scripts/tui-check.sh fallback-check
  scripts/tui-check.sh commands

说明:
  prepare            重建固定测试夹具
  dry-run            启动交互式 dry-run 审阅界面
  check              启动交互式 check 审阅界面
  risky              启动高风险确认界面
  fallback-dry-run   以非交互模式运行 dry-run 文本回退
  fallback-check     以非交互模式运行 check 文本回退
  commands           打印所有可直接执行的测试命令

环境变量:
  DOTBOT_GO_TUI_CHECK_ROOT   自定义夹具目录, 默认: $FIXTURE_ROOT
  DOTBOT_GO_TUI_CHECK_CACHE_ROOT   自定义 Go 缓存目录, 默认: $CACHE_ROOT
EOF
}

prepare_fixture() {
  chmod -R u+w "$FIXTURE_ROOT" 2>/dev/null || true
  rm -rf "$FIXTURE_ROOT"
  mkdir -p "$FIXTURE_REPO/git" "$FIXTURE_REPO/ghostty" "$FIXTURE_HOME"
  mkdir -p "$GO_MOD_CACHE" "$GO_BUILD_CACHE"

  cat >"$FIXTURE_REPO/git/gitconfig" <<'EOF'
[user]
  name = tester
EOF

  cat >"$FIXTURE_REPO/ghostty/config" <<'EOF'
font-size = 14
EOF

  cat >"$MAIN_CONFIG" <<'EOF'
[create]
paths = ["~/.cache/zsh"]

[[link]]
target = "~/.gitconfig"
source = "./git/gitconfig"

[[link]]
target = "~/.config/very/long/path/for/tui/review/example/config.toml"
source = "./ghostty/config"
create = true
EOF

  cat >"$RISKY_CONFIG" <<'EOF'
[[link]]
target = "~"
source = "./ghostty/config"
force = true
EOF
}

run_dotbot() {
  (
    cd "$ROOT_DIR"
    HOME="$FIXTURE_HOME" GOMODCACHE="$GO_MOD_CACHE" GOCACHE="$GO_BUILD_CACHE" go run ./cmd/dotbot-go "$@"
  )
}

run_dotbot_non_interactive() {
  (
    cd "$ROOT_DIR"
    HOME="$FIXTURE_HOME" GOMODCACHE="$GO_MOD_CACHE" GOCACHE="$GO_BUILD_CACHE" go run ./cmd/dotbot-go "$@" </dev/null
  )
}

print_commands() {
  cat <<EOF
夹具目录:
  repo: $FIXTURE_REPO
  home: $FIXTURE_HOME
  cache: $CACHE_ROOT

可执行命令:
  scripts/tui-check.sh dry-run
  scripts/tui-check.sh check
  scripts/tui-check.sh risky
  scripts/tui-check.sh fallback-dry-run
  scripts/tui-check.sh fallback-check

直接命令:
  cd $ROOT_DIR && HOME=$FIXTURE_HOME GOMODCACHE=$GO_MOD_CACHE GOCACHE=$GO_BUILD_CACHE go run ./cmd/dotbot-go --dry-run --config $MAIN_CONFIG
  cd $ROOT_DIR && HOME=$FIXTURE_HOME GOMODCACHE=$GO_MOD_CACHE GOCACHE=$GO_BUILD_CACHE go run ./cmd/dotbot-go --check --config $MAIN_CONFIG
  cd $ROOT_DIR && HOME=$FIXTURE_HOME GOMODCACHE=$GO_MOD_CACHE GOCACHE=$GO_BUILD_CACHE go run ./cmd/dotbot-go --config $RISKY_CONFIG
  cd $ROOT_DIR && HOME=$FIXTURE_HOME GOMODCACHE=$GO_MOD_CACHE GOCACHE=$GO_BUILD_CACHE go run ./cmd/dotbot-go --dry-run --config $MAIN_CONFIG </dev/null
  cd $ROOT_DIR && HOME=$FIXTURE_HOME GOMODCACHE=$GO_MOD_CACHE GOCACHE=$GO_BUILD_CACHE go run ./cmd/dotbot-go --check --config $MAIN_CONFIG </dev/null
EOF
}

main() {
  local command="${1:-commands}"

  case "$command" in
    prepare)
      prepare_fixture
      printf '已重建测试夹具: %s\n' "$FIXTURE_ROOT"
      ;;
    dry-run)
      prepare_fixture
      run_dotbot --dry-run --config "$MAIN_CONFIG"
      ;;
    check)
      prepare_fixture
      run_dotbot --check --config "$MAIN_CONFIG"
      ;;
    risky)
      prepare_fixture
      run_dotbot --config "$RISKY_CONFIG"
      ;;
    fallback-dry-run)
      prepare_fixture
      run_dotbot_non_interactive --dry-run --config "$MAIN_CONFIG"
      ;;
    fallback-check)
      prepare_fixture
      run_dotbot_non_interactive --check --config "$MAIN_CONFIG"
      ;;
    commands)
      prepare_fixture
      print_commands
      ;;
    help|-h|--help)
      usage
      ;;
    *)
      printf '未知命令: %s\n\n' "$command" >&2
      usage >&2
      exit 2
      ;;
  esac
}

main "$@"
