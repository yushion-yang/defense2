# 07 游戏模式与会话审核指导

## 审核目标

验证 6+1 种游戏模式的胜负判定、波次配置、特殊规则。

## 必读文件

| 文件 | 读取内容 |
|------|---------|
| `internal/core/gamemode/mode.go` | Mode 接口定义 |
| `internal/core/gamemode/session.go` | Session struct、CheckEndConditions()、Stats |
| `internal/core/gamemode/base.go` | 默认实现 |
| `internal/core/gamemode/campaign.go` | 战役模式 |
| `internal/core/gamemode/endless.go` | 无尽模式 |
| `internal/core/gamemode/timed.go` | 限时模式 |
| `internal/core/gamemode/bossrush.go` | Boss 竞速 |
| `internal/core/gamemode/challenge.go` | 挑战模式 |
| `internal/core/gamemode/testmode.go` | 测试模式 |
| `internal/core/gamemode/autoplay.go` | 自动对局模式 |
| `internal/core/gamemode/register.go` | 模式注册 |

## 检查项

### A. 模式注册

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| A1 | 注册表 | 读 register.go init() | 7 种模式全部注册 |
| A2 | GetOrDefault | 读 mode.go | 未知 ID 返回 campaign 默认 |

### B. 胜负判定（逐模式）

| # | 模式 | CheckVictory | CheckDefeat |
|---|------|-------------|-------------|
| B1 | campaign | `wave >= maxWaves && !spawning` | `lives <= 0` |
| B2 | endless | 永远 false | `lives <= 0` |
| B3 | timed | `remainingTime <= 0 && lives > 0` | `lives <= 0` |
| B4 | bossRush | `bossesKilled >= totalBosses` | `lives <= 0` |
| B5 | challenge | 同 campaign | `lives <= 0` |
| B6 | test | 有 maxWaves 时同 campaign，否则 false | 永远 false |
| B7 | autoplay | 同 campaign | 永远 false |

### C. 关键逻辑

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| C1 | CheckEndConditions 顺序 | 读 session.go | 先检 Defeat 再检 Victory（lives=0 优先判败） |
| C2 | Spawning 定义 | 读 stage.go buildModeCtx | `!spawner.IsClear(enemies)` = AllDone && pool.Count==0 |
| C3 | bossRush OnInit | 读 bossrush.go | 设 maxWaves = totalBosses(5) |
| C4 | bossRush OnEnemyKilled | 读 bossrush.go | boss=true 时 bossesKilled++ |
| C5 | timed OnTick | 读 timed.go | remainingTime -= dt |
| C6 | autoplay ShouldAutoStart | 读 autoplay.go | 返回 true（自动开波） |

### D. 波次奖金（逐模式）

| # | 模式 | OnWaveCleared 奖金公式 |
|---|------|----------------------|
| D1 | campaign | bonus = 12 + wave*4, perfect = 8 + wave*2 |
| D2 | endless | bonus = 15 + wave*6（递增更快） |
| D3 | timed | bonus = 8 + wave*3 |
| D4 | bossRush | bonus = 20 + wave*10 |
| D5 | autoplay | bonus = 12 + wave*4（同 campaign） |

## 跨系统关联

- CheckEndConditions ← stage.updatePlaying step 10+
- Context.Spawning ← spawner.IsClear(pool)
- Session.Stats ← EventBus 订阅（kills/leaked/towersBuilt/wavesCleared）
- stage.go 根据 modeID 设 bossRush 的 `spawner.BossEveryWave=true`
