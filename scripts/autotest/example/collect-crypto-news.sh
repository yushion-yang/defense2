#!/bin/bash
# Crypto news collection script
# Collects latest crypto/blockchain news via Claude web search

export PATH="/usr/local/bin:/usr/bin:/bin:/Users/yushion/.local/bin:/Users/yushion/go/bin:$PATH"
export HOME="/Users/yushion"

LOG="/tmp/ai-crypto-news.log"
OUTPUT_DIR="/Users/yushion/GolandProjects/src/my_workspace/data/crypto-news"
TODAY=$(date +%Y-%m-%d)
OUTPUT="${OUTPUT_DIR}/${TODAY}.md"

mkdir -p "$OUTPUT_DIR"
echo "[$(date)] Starting crypto news collection..." >> "$LOG"

PROMPT="Search the web for the latest cryptocurrency and blockchain news from the past 24 hours. Write the top 10 most important news items to ${OUTPUT}. No title, no heading. Exact format per line: NUMBER. [SOURCE] headline - brief summary (tag) where tag is one of: BTC/ETH/DeFi/NFT/监管/Layer2/稳定币/交易所/其他. Write in Chinese. Example: 1. [CoinDesk] 比特币突破10万美元创历史新高 - 机构资金持续流入推动价格上涨 (BTC)"

echo "3" | codemax claude --print "$PROMPT" --dangerously-skip-permissions >> "$LOG" 2>&1

echo "[$(date)] Done. Output: ${OUTPUT}" >> "$LOG"
