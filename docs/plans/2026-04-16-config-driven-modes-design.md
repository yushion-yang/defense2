# 配置驱动游戏模式重构设计

> 2026-04-16 | 状态: 设计中

## 目标

将 10 个独立模式 Go 文件 + 3 个 Ruleset 文件合并为 `config/gamemodes.json` + 一个 `UniversalMode` + 2 个 Hook 插件。新增模式 = 加一段 JSON，零代码改动。

## 架构

```
config/gamemodes.json
  ↓ (加载)
UniversalMode (实现 Mode 接口全部 18 方法)
  ├─ Tier 1: 直接返回配置值 (autoStart/intermission/victoryTarget/...)
  ├─ Tier 2: preset 分发 (victory="allWaves"→公式, defeat="livesZero"→公式)
  └─ Tier 3: 委托 Hook 插件 (countdown/bossCounter)
```

## gamemodes.json 配置 schema

```json
{
  "<modeID>": {
    "autoStart": bool,           // ShouldAutoStart
    "intermission": float,       // IntermissionSecs (0=spawner默认)
    "initMaxWaves": int,         // OnInit 设置 MaxWaves (0=地图默认)
    "victory": string,           // "allWaves" | "never" | "hook:countdown" | "hook:bossCounter"
    "defeat": string,            // "livesZero" | "never"
    "enableEvents": bool,
    "econID": string,            // economy.json modes 键名
    "perfectBonus": bool,        // OnWaveCleared 是否发完美奖励
    "coopEnabled": bool,         // 合作模式标记
    "ruleset": {
      "presetTowers": bool,
      "includePresets": bool,
      "classicWaves": bool,
      "wardenEnabled": bool,
      "itemDrop": string,        // "probability" | "everyKill" | "byWave" | "none"
      "blueprints": bool,
      "budgetCap": int           // -1=无限
    },
    "score": {
      "wave": int, "kill": int, "leaked": int, "lives": int,
      "timeBase": int, "timeDecay": float
    },
    "hooks": [
      {"type": "countdown", "targetSeconds": 300},
      {"type": "bossCounter", "totalBosses": 5}
    ],
    "endFields": ["waves","maxWaves","kills","leaked"],
    "ui": {
      "nameKey": string,         // i18n key
      "descKey": string,
      "icon": string,
      "flow": string,            // "campaignSelect" | "testSelect" | "direct"
      "comingSoon": bool,
      "devOnly": bool
    }
  }
}
```

## Hook 接口

```go
type Hook interface {
    Init(ctx *Context, params map[string]any)
    Tick(dt float64, ctx *Context)
    OnEnemyKilled(boss bool, ctx *Context)
    CheckVictory(ctx *Context) (result bool, handled bool)
    HUDExtra() map[string]any
    EndExtra() map[string]any
}
```

2 个实现：CountdownHook (remainingTime)、BossCounterHook (bossesKilled)

## 删除文件清单

- `internal/core/gamemode/campaign.go`
- `internal/core/gamemode/classic.go`
- `internal/core/gamemode/coop.go`
- `internal/core/gamemode/endless.go`
- `internal/core/gamemode/timed.go`
- `internal/core/gamemode/bossrush.go`
- `internal/core/gamemode/challenge.go`
- `internal/core/gamemode/testmode.go`
- `internal/core/gamemode/autoplay.go`
- `internal/core/gamemode/simulation.go`
- `internal/core/gamemode/campaign_ruleset.go`
- `internal/core/gamemode/classic_ruleset.go`
- `internal/core/gamemode/test_ruleset.go`
- `internal/core/gamemode/tower_ruleset.go` (接口保留,实现改为配置驱动)

## 新增文件

- `config/gamemodes.json` — 全部模式定义
- `internal/core/gamemode/universal.go` — UniversalMode
- `internal/core/gamemode/hook.go` — Hook 接口 + CountdownHook + BossCounterHook
- `internal/core/gamemode/config_ruleset.go` — ConfigRuleset (从JSON读取)
- `internal/core/gamemode/loader.go` — 加载 gamemodes.json

## 验证

- `make test` 全量通过
- 10 个模式行为不变
- 新增模式只需加 JSON
