# 06 波次与经济系统审核指导

## 审核目标

验证波次出怪公式、经济收支平衡、难度缩放系统。

## 必读文件

| 文件 | 读取内容 |
|------|---------|
| `config/settings.json` | 难度配置（easy/normal/hard/extreme） |
| `internal/core/enemy/spawner.go` | 波次逻辑（enemyCount/waveCompositions/startWave） |
| `internal/core/economy/economy.go` | 经济公式（KillGold/WaveCompleteGold/Interest/SellRefund） |
| `internal/core/gamemode/difficulty.go` | LoadDifficulty() |
| `internal/scene/stage.go` | 难度应用到 spawner + economy |

## 检查项

### A. 波次出怪

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| A1 | 每波数量 | 读 enemyCount() | `EnemiesPerWave(5) + wave` |
| A2 | Boss 规则 | 读 startWave | `BossEveryWave \|\| wave%5==0` 时追加 1 个 Boss |
| A3 | 波次组合权重 | 读 waveCompositions | 5 阶段递进：wave1-3 只 normal(70%)+runner(30%)，wave15+ 全部 13 种 |
| A4 | 出怪间隔 | 读 SpawnInterval | 默认 0.6s |
| A5 | 波间间隔 | 读 WaveInterval | 默认 10s |
| A6 | 首波间隔 | 读 FirstWaveInterval | 默认 20s |
| A7 | HP 基础公式 | 读 spawner.Update | `baseHP = (10 + wave*5) * HPScale` |
| A8 | Speed 基础公式 | 同上 | `baseSpeed = (50 + wave*3) * SpeedScale` |
| A9 | ManualWave | 读 Update | ManualWave=true 时倒计时到 0 不自动开波 |
| A10 | AllDone 标记 | 读 Update | wave >= maxWaves 时设 AllDone=true |

### B. 经济公式

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| B1 | 击杀奖金 | 读 KillGold() | 基础 15（受 rewardScale 缩放） |
| B2 | 波次奖金 | 读 WaveCompleteGold(wave) | `30 + wave*5` |
| B3 | 利息 | 读 InterestGold(gold) | `gold * 0.05`，上限 50 |
| B4 | 卖塔退款 | 读 SellRefund(cost) | `cost * 0.5` |
| B5 | 完美波次奖金 | 读 onWaveTransition | 0 泄漏时额外 `8 + wave*2` |

### C. 难度缩放

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| C1 | easy | 读 settings.json | HP×0.7, Speed×0.85, Reward×1.3, StartGold=180 |
| C2 | normal | 同上 | HP×1, Speed×1, Reward×1, StartGold=120 |
| C3 | hard | 同上 | HP×1.4, Speed×1.15, Reward×0.8, StartGold=100 |
| C4 | extreme | 同上 | HP×2, Speed×1.3, Reward×0.6, StartGold=80 |
| C5 | 应用到 spawner | 读 stage.go | `spawner.HPScale = diff.HPScale` |
| C6 | 应用到 economy | 读 stage.go | `econ.KillReward *= diff.RewardScale` |
| C7 | 初始金币 | 读 stage.go | `startGold = diff.StartGold` |

## 跨系统关联

- Spawner.Update ← stage.updatePlaying step 1（每帧驱动）
- 波次变化 → onWaveTransition → WaveCleared 事件 → 能力解锁
- KillGold → EventBus(EvtEnemyKilled) → stage.go `gold += p.GoldValue`
- 完美波次 → `lives == waveLivesSnapshot` → 额外奖金
