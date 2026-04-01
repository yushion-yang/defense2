# AI 策划评审 Prompt

> 将 autoplay session 的 report.json 数据喂给 Claude/GPT，以"资深策划"视角输出体验报告。

## 使用方法

```bash
# 1. 跑一局 autoplay
go run cmd/autoplay/main.go --strategies greedy --runs 1 --json-dir /tmp/review

# 2. 把 report.json 内容复制，连同下面的 prompt 一起发给 Claude
```

## Prompt 模板

```
你是一名资深塔防游戏策划，请分析以下自动对局数据并给出体验评审报告。

## 对局数据

[粘贴 report.json 内容]

## 分析维度

### 1. 难度曲线
- 前 3 波是否有泄漏？（earlyLeakCount > 0 = 开局太难）
- 最后 3 波是否 0 泄漏？（lateZeroLeakWaves == 3 = 结局太简单）
- 难度 verdict 是什么？合理吗？

### 2. 经济节奏
- wave_log 中每波的 gold_earned vs gold_spent 比例如何？
- 是否有连续 3 波 gold_spent == 0（攒钱期太长 = 无聊）？
- economy_alerts 中有几次断档？出现在第几波？

### 3. Boss 压力
- boss_stats 中每个 Boss 存活多久？
- 存活 <5s = 太弱（没存在感），>60s = 可能无解
- Boss 波的泄漏数是否 > 普通波？

### 4. 节奏感
- pace_stats 的 idle_combat_ratio 是多少？
  - > 3.0 = 等待太久，玩家无聊
  - < 0.3 = 喘不过气，压力太大
  - 0.5~2.0 = 理想区间

### 5. 特殊怪影响力
- experience_stats.special_enemy_impact 中：
  - impact_ratio < 1.1 = 该原型和 normal 没区别（机制形同虚设）
  - impact_ratio > 2.0 = 该原型明显更难杀（机制有效）
- 哪些特殊怪需要加强/削弱？

### 6. DPS 曲线
- dps_snapshots 是否平滑递增？
- 是否有 DPS 突降（卖塔？塔被禁用？）或突增（解锁强力能力？）

## 输出格式

```
## 体验评审报告

### 总评
一句话总结：这局游戏体验如何？

### 问题清单（按严重度排序）
1. [严重] xxx
2. [中等] xxx
3. [轻微] xxx

### 数值调整建议
- xxx 建议从 A 调整到 B，原因：xxx

### 亮点
- xxx 做得好，因为：xxx
```
```

## 进阶：批量评审

```bash
# 跑 sweep 后批量生成评审
for f in docs/autotest/M1/*/report.json; do
  session=$(basename $(dirname $f))
  echo "=== $session ==="
  cat $f
  echo ""
done > /tmp/all_sessions.txt

# 将 all_sessions.txt + prompt 一起发给 Claude
# 让它对比多局数据，找出跨 session 的共性问题
```

## 进阶：自动化评审（脚本集成）

```python
# 可以用 Anthropic SDK 自动化：
import anthropic, json

client = anthropic.Anthropic()
report = json.load(open("report.json"))

response = client.messages.create(
    model="claude-sonnet-4-20250514",
    max_tokens=2000,
    messages=[{
        "role": "user",
        "content": f"你是资深塔防策划，分析这局数据：\n{json.dumps(report, indent=2, ensure_ascii=False)}\n\n按难度曲线/经济节奏/Boss压力/节奏感/特殊怪影响力 5 个维度给出评审报告。"
    }]
)
print(response.content[0].text)
```
