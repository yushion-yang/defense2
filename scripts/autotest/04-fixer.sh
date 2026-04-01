#!/bin/bash
# 脚本4 — 自动修复 (Fixer)
# 处理 M4 中所有 issue，修复后归档到 M3（含修复记录）

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

# 收集所有 issue
M4_FILES=$(find "$M4_DIR" -name "*.json" -type f 2>/dev/null | sort)
M4_COUNT=$(echo "$M4_FILES" | grep -c '.' 2>/dev/null || echo 0)

if [ "$M4_COUNT" -eq 0 ]; then
    echo "[$(date)] No issues in M4, nothing to fix." >> "$LOG"
    echo "SKIP"
    exit 0
fi

echo "[$(date)] Found $M4_COUNT issues to fix" >> "$LOG"

# 收集 issue 摘要
ISSUE_LIST=""
for f in $M4_FILES; do
    ISSUE_LIST="${ISSUE_LIST}
- $(basename "$f"): $(python3 -c "import json; d=json.load(open('$f')); print(f'{d.get(\"severity\",\"?\")} | {d.get(\"category\",\"?\")} | {d.get(\"title\",\"?\")}')" 2>/dev/null || echo "parse error")"
done

PROMPT="You are an automated fixer for a tower defense game. Fix ALL issues in M4.

## Project
Directory: ${PROJECT_DIR}

## Issues to Fix (from ${M4_DIR}/)
${ISSUE_LIST}

Read each issue JSON file in ${M4_DIR}/ for full details including suggested_fix.

## Fix Pipeline (for each issue, sorted by severity DESC)
1. Read the issue JSON and its suggested_fix
2. Understand the codebase context: read the referenced files first
3. If type == \"config\": modify the JSON config file, validate syntax
4. If type == \"code\": modify Go source file with minimal surgical changes
5. Run \`go build ./cmd/autoplay/\` to verify compilation (do NOT use make build)
   - If FAIL: keep fixing build errors until it passes
   - If still stuck after 3 attempts: revert YOUR changes to that file only, skip this issue
6. If build passes, write the archive document to ${M3_DIR}/{id}.json, then delete ${M4_DIR}/{id}.json
   The archive document MUST contain the original issue fields PLUS a \"fix_result\" object:
   \`\`\`json
   {
     \"id\": \"BUG-001\",
     \"category\": \"bug\",
     \"severity\": \"CRITICAL\",
     \"title\": \"original title\",
     \"detail\": \"original detail\",
     \"suggested_fix\": { ... },
     \"fix_result\": {
       \"status\": \"fixed\",
       \"files_modified\": [\"path/to/file.go\"],
       \"description\": \"Brief description of what was actually changed\",
       \"diff_summary\": \"e.g. Added prevKills tracking + delta-based OnKill() calls\"
     }
   }
   \`\`\`
   If skipped, write fix_result with status=\"skipped\" and reason, and leave the issue in M4.

## Safety Rules (CRITICAL)
- Do NOT use: git reset --hard, git checkout --, git clean
- Only modify files referenced in the issue's suggested_fix
- Max 5 files modified total per issue, max 50 lines changed per file
- If go build fails 3 times for one issue, skip it and move on
- Do NOT run make test (only go build for speed)

## After All Issues
Report how many issues were fixed vs skipped."

echo "3" | codemax claude --print "$PROMPT" --dangerously-skip-permissions >> "$LOG" 2>&1

# 统计结果
FIXED=$(find "$M3_DIR" -name "*.json" -type f 2>/dev/null | wc -l | tr -d ' ')
REMAINING=$(find "$M4_DIR" -name "*.json" -type f 2>/dev/null | wc -l | tr -d ' ')
echo "[$(date)] Fix complete. Fixed: $FIXED, Remaining in M4: $REMAINING" >> "$LOG"

echo "DONE fixed=$FIXED remaining=$REMAINING"
