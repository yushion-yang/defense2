#!/bin/bash
# 脚本2 — 自动对局 (Runner)
# 编译并运行自动对局，直接分离输出到 M1(纯JSON) + M2(纯PNG)
# 保留旧数据不删除，新数据按时间戳目录追加

set -euo pipefail

export PATH="/usr/local/bin:/usr/bin:/bin:/Users/yushion/.local/bin:/Users/yushion/go/bin:$PATH"
export HOME="/Users/yushion"

PROJECT_DIR="/Users/yushion/Games/defense2"
DATA_DIR="${PROJECT_DIR}/docs/autotest"
STATE_FILE="${DATA_DIR}/state.json"
LOG="/tmp/autotest-runner.log"

cd "$PROJECT_DIR"
echo "[$(date)] === 脚本2: 自动对局 ===" >> "$LOG"

# 读取当前轮次 (从 state.json 或默认 1)
ROUND=1
if [ -f "$STATE_FILE" ]; then
    ROUND=$(python3 -c "import json; print(json.load(open('$STATE_FILE')).get('current_round', 1))" 2>/dev/null || echo 1)
fi

echo "[$(date)] Round: $ROUND" >> "$LOG"

# 编译
echo "[$(date)] Building autoplay binary..." >> "$LOG"
go build -o /tmp/autoplay-bin ./cmd/autoplay/ >> "$LOG" 2>&1

# 决定策略
ARGS=""
if [ "$ROUND" -eq 1 ]; then
    ARGS="--sweep"
else
    if [ -f "${DATA_DIR}/M4/rerun_scenarios.txt" ]; then
        SCENARIOS=$(cat "${DATA_DIR}/M4/rerun_scenarios.txt" | tr '\n' ',' | sed 's/,$//')
        if [ -n "$SCENARIOS" ]; then
            ARGS="--strategies ${SCENARIOS}"
        else
            ARGS="--sweep"
        fi
    else
        ARGS="--sweep"
    fi
fi

# autoplay --json-dir/--png-dir 分离输出
# 分离模式下 autoplay 不再自动创建 run_timestamp 子目录
M1_DIR="${DATA_DIR}/M1"
M2_DIR="${DATA_DIR}/M2"

echo "[$(date)] Running autoplay: $ARGS --json-dir $M1_DIR --png-dir $M2_DIR" >> "$LOG"

# 运行自动对局 (超时 10 分钟)
timeout 600 /tmp/autoplay-bin $ARGS \
    --json-dir "$M1_DIR" \
    --png-dir "$M2_DIR" \
    >> "$LOG" 2>&1 || {
    EXIT_CODE=$?
    if [ $EXIT_CODE -eq 124 ]; then
        echo "[$(date)] WARNING: autoplay timed out after 600s" >> "$LOG"
    else
        echo "[$(date)] WARNING: autoplay exited with code $EXIT_CODE" >> "$LOG"
    fi
}

# 统计本次产出及全部待处理数据
TOTAL_JSON=$(find "$M1_DIR" -name "*.json" 2>/dev/null | wc -l | tr -d ' ')
TOTAL_PNG=$(find "$M2_DIR" -name "*.png" 2>/dev/null | wc -l | tr -d ' ')

echo "[$(date)] Total in M1: JSON=$TOTAL_JSON, M2: PNG=$TOTAL_PNG" >> "$LOG"

echo "DONE round=$ROUND total_json=$TOTAL_JSON total_png=$TOTAL_PNG"
