#!/bin/bash
# 脚本1 — 环境检查 / 增量更新
# 检查整个项目的代码变更，发现新增/修改的系统后，让 AI 扩展自动对局测试能力

set -euo pipefail

export PATH="/usr/local/bin:/usr/bin:/bin:/Users/yushion/.local/bin:/Users/yushion/go/bin:$PATH"
export HOME="/Users/yushion"

PROJECT_DIR="/Users/yushion/Games/defense2"
DATA_DIR="${PROJECT_DIR}/docs/autotest"
LAST_COMMIT_FILE="${DATA_DIR}/last_commit.txt"
LOG="/tmp/autotest-env-check.log"

cd "$PROJECT_DIR"
echo "[$(date)] === 脚本1: 环境检查 ===" >> "$LOG"

# 确保目录存在
mkdir -p "${DATA_DIR}/M1" "${DATA_DIR}/M2" "${DATA_DIR}/M3" "${DATA_DIR}/M4"

# 读取上次 commit hash
OLD_HASH=""
if [ -s "$LAST_COMMIT_FILE" ]; then
    OLD_HASH=$(cat "$LAST_COMMIT_FILE" | tr -d '[:space:]')
fi
CURRENT_HASH=$(git rev-parse HEAD)

echo "[$(date)] Old hash: ${OLD_HASH:-<empty>}, Current hash: $CURRENT_HASH" >> "$LOG"

# hash 相同 -> 跳过
if [ "$OLD_HASH" = "$CURRENT_HASH" ]; then
    echo "[$(date)] No changes since last run, skipping." >> "$LOG"
    echo "SKIP"
    exit 0
fi

# 获取整个项目的变更摘要
DIFF_SUMMARY=""
if [ -z "$OLD_HASH" ]; then
    DIFF_SUMMARY="First run. Full project scan needed."
else
    # 整个项目的 diff stat (排除 docs/autotest 和 scripts/autotest 自身)
    DIFF_SUMMARY=$(git diff --stat "$OLD_HASH..HEAD" \
        -- ':!docs/autotest' ':!scripts/autotest' ':!docs/*.md' \
        2>/dev/null || echo "N/A")
fi

echo "[$(date)] Project diff since ${OLD_HASH:-<first run>}:" >> "$LOG"
echo "$DIFF_SUMMARY" >> "$LOG"

PROMPT="You are maintaining an autoplay testing system for a tower defense game.
The project code has changed since the last test run.

## Project Info
- Directory: ${PROJECT_DIR}
- Autoplay code: internal/autoplay/ , cmd/autoplay/main.go
- Game core: internal/core/ (tower, enemy, combat, skill, warden, buff, strength, physics, event, gamemode)
- Rendering: internal/render/
- Config: config/

## Changes Since Last Run
\`\`\`
${DIFF_SUMMARY}
\`\`\`

## Your Task
1. Read the diff to understand what changed in the project (new systems, modified mechanics, new towers/enemies/abilities, config changes, etc.)
2. Assess whether the current autoplay testing system can cover these changes:
   - Can existing strategies exercise the new/changed code paths?
   - Are there new enemy types, tower types, abilities, or game modes that need test coverage?
   - Are there new config fields that affect balance?
3. If gaps exist, update the autoplay testing infrastructure to cover them:
   - Add new test scenarios in internal/autoplay/ (strategies, test cases)
   - Update autoplay controller if new game state needs to be captured in reports
   - Add new anomaly detection rules if applicable
   - Update cmd/autoplay/main.go CLI flags if needed
4. Run \`go build ./cmd/autoplay/\` to verify your changes compile.

## Rules
- Only modify autoplay-related files (internal/autoplay/, cmd/autoplay/)
- Do NOT modify game logic code (internal/core/, internal/render/, config/)
- Keep changes minimal and focused on test coverage expansion
- If no updates needed, just confirm with a brief explanation of why existing coverage is sufficient"

echo "3" | codemax claude --print "$PROMPT" --dangerously-skip-permissions >> "$LOG" 2>&1

# 更新 commit hash
echo "$CURRENT_HASH" > "$LAST_COMMIT_FILE"
echo "[$(date)] Updated last_commit to $CURRENT_HASH" >> "$LOG"
echo "UPDATED"
