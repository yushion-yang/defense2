#!/usr/bin/env bash
# llm_eval.sh — Evaluate LLM strategy vs greedy baseline
#
# Usage:
#   ./scripts/llm_eval.sh --model config/llm/models/small_trained.bin
#   ./scripts/llm_eval.sh --model config/llm/models/small_trained.bin --map map_02 --runs 20
#
# Requires: autoplay binary (go build ./cmd/autoplay/)

set -euo pipefail

# Defaults
MODEL_PATH=""
VOCAB_PATH="config/llm/vocab.json"
MAP_ID="map_01"
DIFFICULTY="normal"
RUNS=10
OUTPUT_DIR="./eval-results"
SEED=42

# Parse args
while [[ $# -gt 0 ]]; do
    case $1 in
        --model)      MODEL_PATH="$2"; shift 2 ;;
        --vocab)      VOCAB_PATH="$2"; shift 2 ;;
        --map)        MAP_ID="$2"; shift 2 ;;
        --difficulty) DIFFICULTY="$2"; shift 2 ;;
        --runs)       RUNS="$2"; shift 2 ;;
        --output)     OUTPUT_DIR="$2"; shift 2 ;;
        --seed)       SEED="$2"; shift 2 ;;
        -h|--help)
            echo "Usage: $0 --model <path.bin> [--map map_01] [--runs 10] [--seed 42]"
            exit 0
            ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

if [[ -z "$MODEL_PATH" ]]; then
    echo "ERROR: --model is required"
    echo "Usage: $0 --model <path.bin>"
    exit 1
fi

if [[ ! -f "$MODEL_PATH" ]]; then
    echo "ERROR: Model file not found: $MODEL_PATH"
    exit 1
fi

echo "========================================="
echo "  DTLM Evaluation: LLM vs Greedy"
echo "========================================="
echo "Model:      $MODEL_PATH"
echo "Map:        $MAP_ID"
echo "Difficulty: $DIFFICULTY"
echo "Runs:       $RUNS per strategy"
echo "Seed:       $SEED"
echo "Output:     $OUTPUT_DIR"
echo ""

# Build autoplay
echo "Building autoplay..."
go build -o /tmp/autoplay-eval ./cmd/autoplay/
echo "Build OK"
echo ""

# Create output dirs
GREEDY_DIR="$OUTPUT_DIR/greedy"
LLM_DIR="$OUTPUT_DIR/llm"
mkdir -p "$GREEDY_DIR" "$LLM_DIR"

# Run greedy baseline
echo "--- Running greedy strategy ($RUNS runs) ---"
/tmp/autoplay-eval \
    --strategies greedy \
    --runs "$RUNS" \
    --map "$MAP_ID" \
    --difficulty "$DIFFICULTY" \
    --json-dir "$GREEDY_DIR" \
    --png-dir "$GREEDY_DIR/png" \
    --seed "$SEED" \
    2>&1 | tail -5

echo ""

# Run LLM strategy
echo "--- Running LLM strategy ($RUNS runs) ---"
/tmp/autoplay-eval \
    --strategies llm \
    --runs "$RUNS" \
    --map "$MAP_ID" \
    --difficulty "$DIFFICULTY" \
    --model-path "$MODEL_PATH" \
    --vocab-path "$VOCAB_PATH" \
    --json-dir "$LLM_DIR" \
    --png-dir "$LLM_DIR/png" \
    --seed "$SEED" \
    2>&1 | tail -5

echo ""

# Compare results
echo "========================================="
echo "  Results Comparison"
echo "========================================="

analyze_dir() {
    local dir=$1
    local name=$2
    local wins=0
    local total=0
    local total_waves=0
    local total_kills=0

    for f in "$dir"/*.json; do
        [[ -f "$f" ]] || continue
        total=$((total + 1))

        # Extract fields from JSON (basic parsing)
        victory=$(python3 -c "import json; d=json.load(open('$f')); print(d.get('victory', False))" 2>/dev/null || echo "False")
        waves=$(python3 -c "import json; d=json.load(open('$f')); print(d.get('waves_completed', 0))" 2>/dev/null || echo "0")
        kills=$(python3 -c "import json; d=json.load(open('$f')); print(d.get('total_kills', 0))" 2>/dev/null || echo "0")

        if [[ "$victory" == "True" ]]; then
            wins=$((wins + 1))
        fi
        total_waves=$((total_waves + waves))
        total_kills=$((total_kills + kills))
    done

    if [[ $total -eq 0 ]]; then
        echo "  $name: No results found in $dir"
        return
    fi

    local win_rate=$((wins * 100 / total))
    local avg_waves=$((total_waves / total))
    local avg_kills=$((total_kills / total))

    printf "  %-10s | Win Rate: %3d%% (%d/%d) | Avg Waves: %3d | Avg Kills: %4d\n" \
        "$name" "$win_rate" "$wins" "$total" "$avg_waves" "$avg_kills"
}

analyze_dir "$GREEDY_DIR" "Greedy"
analyze_dir "$LLM_DIR" "LLM"

echo ""
echo "Raw results: $OUTPUT_DIR"
echo "Done."
