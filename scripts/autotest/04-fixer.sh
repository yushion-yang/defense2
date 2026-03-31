#!/bin/bash
# 脚本4 — 自动修复 (Fixer)
# 读取 M4 中的问题，逐个修复，make build 验证，归档到 M3

set -euo pipefail

export PATH="/usr/local/bin:/usr/bin:/bin:/Users/yushion/.local/bin:/Users/yushion/go/bin:$PATH"
export HOME="/Users/yushion"

PROJECT_DIR="/Users/yushion/Games/defense2"
DATA_DIR="${PROJECT_DIR}/docs/autotest"
M3_DIR="${DATA_DIR}/M3"
M4_DIR="${DATA_DIR}/M4"
LOG="/tmp/autotest-fixer.log"

cd "$PROJECT_DIR"
echo "[$(date)] === 脚本4: 自动修复 ===" >> "$LOG"

# 检查 M4 是否有待修复问题
M4_FILES=$(find "$M4_DIR" -name "*.json" -type f 2>/dev/null | sort)
M4_COUNT=$(echo "$M4_FILES" | grep -c '.' || echo 0)

if [ "$M4_COUNT" -eq 0 ]; then
    echo "[$(date)] No issues in M4, nothing to fix." >> "$LOG"
    echo "SKIP"
    exit 0
fi

echo "[$(date)] Found $M4_COUNT issues to fix" >> "$LOG"

# 收集所有 issue 摘要
ISSUE_LIST=""
for f in $M4_FILES; do
    ISSUE_LIST="${ISSUE_LIST}
- $(basename "$f"): $(python3 -c "import json; d=json.load(open('$f')); print(f'{d.get(\"severity\",\"?\")} | {d.get(\"title\",\"?\")} | fix: {d.get(\"suggested_fix\",{}).get(\"description\",\"?\")}')" 2>/dev/null || echo "parse error")"
done

PROMPT="You are an automated fixer for a tower defense game. Fix the issues found by the analyst.

## Project
Directory: ${PROJECT_DIR}

## Issues to Fix (from ${M4_DIR}/)
${ISSUE_LIST}

Read each issue JSON file in ${M4_DIR}/ for full details.

## Fix Pipeline (for each issue, sorted by severity DESC)
1. Read the issue JSON and its suggested_fix
2. If type == \"config\": modify the JSON config file, validate syntax
3. If type == \"code\": modify Go source, run \`make lint\` on modified files
4. Run \`make build\` to verify compilation
   - If FAIL: keep fixing build errors until it passes
   - If still stuck after 3 attempts: revert YOUR changes to that file, skip this issue
5. If build passes, move the issue JSON from ${M4_DIR}/ to ${M3_DIR}/

## Safety Rules (CRITICAL)
- Do NOT use: git reset --hard, git checkout --, git clean
- Only modify files referenced in the issue's suggested_fix
- Max 5 files modified total, max 50 lines changed per file
- If make build fails 3 times for one issue, skip it and move on
- Do NOT touch game logic beyond what the issue specifies
- Do NOT run make test (only make build for speed)

## After All Issues
Report how many issues were fixed vs skipped."

echo "3" | codemax claude --print "$PROMPT" --dangerously-skip-permissions >> "$LOG" 2>&1

# 统计结果
FIXED=$(find "$M3_DIR" -name "*.json" -newer "$LOG" -type f 2>/dev/null | wc -l | tr -d ' ')
REMAINING=$(find "$M4_DIR" -name "*.json" -type f 2>/dev/null | wc -l | tr -d ' ')
echo "[$(date)] Fix complete. Fixed: $FIXED, Remaining in M4: $REMAINING" >> "$LOG"

echo "DONE fixed=$FIXED remaining=$REMAINING"
