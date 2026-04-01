#!/bin/bash
# 编排器 — 自动化闭环测试主循环
# 按顺序调用脚本1-4，循环直到收敛或达到轮次上限

set -euo pipefail

export PATH="/usr/local/bin:/usr/bin:/bin:/Users/yushion/.local/bin:/Users/yushion/go/bin:$PATH"
export HOME="/Users/yushion"

PROJECT_DIR="/Users/yushion/Games/defense2"
DATA_DIR="${PROJECT_DIR}/docs/autotest"
SCRIPT_DIR="${PROJECT_DIR}/scripts/autotest"
STATE_FILE="${DATA_DIR}/state.json"
LOG="/tmp/autotest-loop.log"

MAX_ROUNDS=${1:-5}  # 最大轮次，默认 5

cd "$PROJECT_DIR"
echo "" >> "$LOG"
echo "============================================" >> "$LOG"
echo "[$(date)] AutoTest loop started. Max rounds: $MAX_ROUNDS" >> "$LOG"
echo "============================================" >> "$LOG"

# 初始化 state.json
init_state() {
    cat > "$STATE_FILE" << EOF
{
  "current_round": 1,
  "max_rounds": $MAX_ROUNDS,
  "phase": "ENV_CHECK",
  "started_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "rounds": []
}
EOF
}

# 更新 state.json
update_state() {
    local round=$1 phase=$2
    python3 -c "
import json
with open('$STATE_FILE') as f:
    state = json.load(f)
state['current_round'] = $round
state['phase'] = '$phase'
with open('$STATE_FILE', 'w') as f:
    json.dump(state, f, indent=2)
" 2>/dev/null || true
}

# 追加轮次记录
append_round_record() {
    local round=$1 issues_found=$2 issues_fixed=$3
    python3 -c "
import json
with open('$STATE_FILE') as f:
    state = json.load(f)
state['rounds'].append({
    'round': $round,
    'issues_found': $issues_found,
    'issues_fixed': $issues_fixed
})
with open('$STATE_FILE', 'w') as f:
    json.dump(state, f, indent=2)
" 2>/dev/null || true
}

# macOS grep 没有 -P, 用 sed 提取
extract_val() {
    local key=$1 text=$2
    echo "$text" | sed -n "s/.*${key}=\([0-9]*\).*/\1/p" | head -1
}

# --- 主循环 ---

init_state

# 脚本1: 环境检查 (只跑一次)
echo "[$(date)] Phase: ENV_CHECK" >> "$LOG"
update_state 1 "ENV_CHECK"
bash "$SCRIPT_DIR/01-env-check.sh" >> "$LOG" 2>&1 || true

for ROUND in $(seq 1 "$MAX_ROUNDS"); do
    echo "" >> "$LOG"
    echo "[$(date)] ===== Round $ROUND / $MAX_ROUNDS =====" >> "$LOG"

    # 脚本2: 自动对局
    echo "[$(date)] Phase: RUN (Round $ROUND)" >> "$LOG"
    update_state "$ROUND" "RUN"
    RUNNER_RESULT=$(bash "$SCRIPT_DIR/02-runner.sh" 2>> "$LOG") || RUNNER_RESULT="ERROR"
    echo "[$(date)] Runner result: $RUNNER_RESULT" >> "$LOG"

    # 脚本3: AI 分析
    echo "[$(date)] Phase: ANALYZE (Round $ROUND)" >> "$LOG"
    update_state "$ROUND" "ANALYZE"
    ANALYST_RESULT=$(bash "$SCRIPT_DIR/03-analyst.sh" 2>> "$LOG") || ANALYST_RESULT="ERROR"
    echo "[$(date)] Analyst result: $ANALYST_RESULT" >> "$LOG"

    # 检查收敛
    if echo "$ANALYST_RESULT" | grep -q "CONVERGED"; then
        echo "[$(date)] CONVERGED at round $ROUND! No new issues found." >> "$LOG"
        append_round_record "$ROUND" 0 0
        update_state "$ROUND" "DONE"
        break
    fi

    # 提取 issue 数量
    ISSUES_FOUND=$(extract_val "count" "$ANALYST_RESULT")
    ISSUES_FOUND=${ISSUES_FOUND:-0}

    # 脚本4: 自动修复
    echo "[$(date)] Phase: FIX (Round $ROUND)" >> "$LOG"
    update_state "$ROUND" "FIX"
    FIXER_RESULT=$(bash "$SCRIPT_DIR/04-fixer.sh" 2>> "$LOG") || FIXER_RESULT="ERROR"
    echo "[$(date)] Fixer result: $FIXER_RESULT" >> "$LOG"

    ISSUES_FIXED=$(extract_val "fixed" "$FIXER_RESULT")
    ISSUES_FIXED=${ISSUES_FIXED:-0}

    append_round_record "$ROUND" "$ISSUES_FOUND" "$ISSUES_FIXED"

    # 最后一轮不再循环
    if [ "$ROUND" -eq "$MAX_ROUNDS" ]; then
        echo "[$(date)] Reached max rounds ($MAX_ROUNDS). Stopping." >> "$LOG"
        update_state "$ROUND" "DONE"
    fi
done

# 生成总结报告
echo "[$(date)] Generating summary report..." >> "$LOG"
update_state "$ROUND" "SUMMARY"

PROMPT="You are generating a summary report for the autoplay automated testing loop.

Read the state file at ${STATE_FILE} to get round-by-round results.
Read any remaining issues in ${DATA_DIR}/M4/ (unfixed).
Read fixed issues in ${DATA_DIR}/M3/ (archived).
Check if ${DATA_DIR}/M1/ still has unprocessed data.

Write a summary report in Chinese to ${DATA_DIR}/summary.md with:
1. Run overview (rounds completed, convergence status, total time)
2. Issues found & fixed table (ID, category, severity, status, fix description)
3. Coverage stats (if available from M1 data)
4. Balance changes applied
5. Unprocessed data remaining (if any)
6. Recommendations for issues not auto-fixed

Use the format specified in docs/autoplay-agent-collaboration.md."

echo "3" | codemax claude --print "$PROMPT" --dangerously-skip-permissions >> "$LOG" 2>&1

echo "[$(date)] AutoTest loop completed." >> "$LOG"
echo "DONE"
