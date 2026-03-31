#!/bin/bash
# 脚本3 — AI 分析 (Analyst)
# 读取 M1(JSON) + M2(截图) 中所有待处理数据，分析后将新问题写入 M4
# 已分析的数据由 AI 标记/清理，未处理的保留给下次

set -euo pipefail

export PATH="/usr/local/bin:/usr/bin:/bin:/Users/yushion/.local/bin:/Users/yushion/go/bin:$PATH"
export HOME="/Users/yushion"

PROJECT_DIR="/Users/yushion/Games/defense2"
DATA_DIR="${PROJECT_DIR}/docs/autotest"
STATE_FILE="${DATA_DIR}/state.json"
LOG="/tmp/autotest-analyst.log"

cd "$PROJECT_DIR"
echo "[$(date)] === 脚本3: AI 分析 ===" >> "$LOG"

# 读取当前轮次
ROUND=1
if [ -f "$STATE_FILE" ]; then
    ROUND=$(python3 -c "import json; print(json.load(open('$STATE_FILE')).get('current_round', 1))" 2>/dev/null || echo 1)
fi

M1_DIR="${DATA_DIR}/M1"
M2_DIR="${DATA_DIR}/M2"
M3_DIR="${DATA_DIR}/M3"
M4_DIR="${DATA_DIR}/M4"

# 清空上轮 M4 残留 (上轮未修复的已在 fixer 中处理或跳过)
rm -f "${M4_DIR}"/*.json "${M4_DIR}"/rerun_scenarios.txt 2>/dev/null || true

# 收集 M1 中所有待处理 JSON (所有子目录)
M1_COUNT=$(find "$M1_DIR" -name "*.json" -type f 2>/dev/null | wc -l | tr -d ' ')

# 收集 M2 中所有待处理截图 (所有子目录)
M2_COUNT=$(find "$M2_DIR" -name "*.png" -type f 2>/dev/null | wc -l | tr -d ' ')

# 列出 M1 子目录 (batch 列表)
M1_BATCHES=$(find "$M1_DIR" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | sort | xargs -I{} basename {} | tr '\n' ', ')

# 收集 M3 已处理历史 (避免重复报告)
M3_HISTORY=""
if [ -d "$M3_DIR" ] && [ "$(ls -A "$M3_DIR" 2>/dev/null)" ]; then
    M3_HISTORY=$(find "$M3_DIR" -name "*.json" -type f -exec basename {} \; 2>/dev/null | sort | tr '\n' ', ')
fi

echo "[$(date)] Round $ROUND: M1=$M1_COUNT JSONs, M2=$M2_COUNT PNGs, batches=[${M1_BATCHES:-none}], M3 history: ${M3_HISTORY:-none}" >> "$LOG"

if [ "$M1_COUNT" -eq 0 ]; then
    echo "[$(date)] No M1 data found, skipping analysis." >> "$LOG"
    echo "SKIP"
    exit 0
fi

PROMPT="You are an AI analyst for a tower defense game's automated testing system.

## Your Task
Analyze ALL pending autoplay test results and identify bugs, balance issues, and anomalies.

## Input Data
- JSON report files in: ${M1_DIR}/ (${M1_COUNT} files across batches: ${M1_BATCHES:-none})
- Screenshots in: ${M2_DIR}/ (${M2_COUNT} files)
- Previously fixed issues (avoid duplicates): ${M3_HISTORY:-none}

Note: M1 and M2 contain data from multiple batches (subdirectories). Analyze ALL of them.

## Analysis Dimensions
1. **Balance**: Tower win rates (< 30% = underpowered, > 90% on hard = overpowered), gold efficiency, DPS curves
2. **Enemies**: Leak rate, survival time, special mechanic triggers
3. **Anomalies**: Group anomalies[] by type, deduplicate, assign severity (CRITICAL > HIGH > MEDIUM > LOW)
4. **Coverage**: Flag untested abilities, enemy types, pipeline steps
5. **Visual**: Check screenshots for UI glitches, overlapping text, stuck effects

## Output Requirements
For each new issue found, write a separate JSON file to ${M4_DIR}/ named {id}.json:
\`\`\`json
{
  \"id\": \"BAL-001\",
  \"category\": \"balance\",
  \"severity\": \"HIGH\",
  \"title\": \"Brief title\",
  \"detail\": \"Detailed description with evidence\",
  \"evidence\": {\"sessions\": [], \"screenshots\": [], \"metrics\": {}},
  \"suggested_fix\": {\"type\": \"config\", \"file\": \"path\", \"description\": \"what to change\"}
}
\`\`\`

Also write ${M4_DIR}/rerun_scenarios.txt with one strategy per line for the next Runner round to re-test.

If no new issues found, write nothing to M4 (leave it empty).

## After Analysis
Delete the M1 and M2 batch subdirectories that you have fully analyzed (rm -rf the subdirectory).
Keep any subdirectory you did NOT have time to fully analyze — it will be picked up next round.

Read the JSON files first, examine screenshots, then produce your analysis."

echo "3" | codemax claude --print "$PROMPT" --dangerously-skip-permissions >> "$LOG" 2>&1

# 检查 M4 是否有新问题
M4_COUNT=$(find "$M4_DIR" -name "*.json" -type f 2>/dev/null | wc -l | tr -d ' ')
# 检查 M1 是否还有未处理的数据
M1_REMAINING=$(find "$M1_DIR" -name "*.json" -type f 2>/dev/null | wc -l | tr -d ' ')

echo "[$(date)] Analysis complete. New issues in M4: $M4_COUNT, M1 remaining: $M1_REMAINING" >> "$LOG"

if [ "$M4_COUNT" -eq 0 ]; then
    echo "CONVERGED round=$ROUND remaining=$M1_REMAINING"
else
    echo "ISSUES round=$ROUND count=$M4_COUNT remaining=$M1_REMAINING"
fi
