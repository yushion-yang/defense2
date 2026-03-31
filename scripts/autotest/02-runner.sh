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

# 用时间戳+轮次作为子目录名，保证唯一且不覆盖旧数据
BATCH_ID="$(date +%Y%m%d-%H%M%S)_round-$(printf '%03d' "$ROUND")"

echo "[$(date)] Round: $ROUND, Batch: $BATCH_ID" >> "$LOG"

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

# autoplay 直接分离输出: --json-dir → M1, --png-dir → M2
M1_DIR="${DATA_DIR}/M1/${BATCH_ID}"
M2_DIR="${DATA_DIR}/M2/${BATCH_ID}"

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

# 统计本次产出
JSON_COUNT=$(find "$M1_DIR" -name "*.json" 2>/dev/null | wc -l | tr -d ' ')
PNG_COUNT=$(find "$M2_DIR" -name "*.png" 2>/dev/null | wc -l | tr -d ' ')

# 统计 M1/M2 中全部未处理数据
TOTAL_JSON=$(find "${DATA_DIR}/M1" -name "*.json" 2>/dev/null | wc -l | tr -d ' ')
TOTAL_PNG=$(find "${DATA_DIR}/M2" -name "*.png" 2>/dev/null | wc -l | tr -d ' ')

echo "[$(date)] This batch: JSON=$JSON_COUNT, PNG=$PNG_COUNT" >> "$LOG"
echo "[$(date)] Total pending: JSON=$TOTAL_JSON, PNG=$TOTAL_PNG" >> "$LOG"

echo "DONE round=$ROUND batch=$BATCH_ID json=$JSON_COUNT png=$PNG_COUNT total_json=$TOTAL_JSON total_png=$TOTAL_PNG"
