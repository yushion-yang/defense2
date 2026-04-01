# 09 配置一致性审核指导

## 审核目标

验证 JSON 配置与 Go 代码的字段映射、常量一致性、注册表完整性。
**历史教训**：json tag `"name"` 写成 `"type"` 导致能力系统整体失效。

## 必读文件

| 文件 | 对照 |
|------|------|
| `config/towers/towers.json` ↔ `internal/config/tower_config.go` | TowerJSON struct 的 json tag |
| `config/abilities/abilities.json` ↔ `internal/config/ability_config.go` | AbilityDef struct 的 json tag |
| `config/enemies/enemies-core.json` ↔ `internal/config/enemy_config.go` | EnemyArchetype struct 的 json tag |
| `config/wardens/wardens.json` ↔ `internal/config/warden_config.go` | WardenConfig struct 的 json tag |
| `config/settings.json` ↔ `internal/config/settings_config.go` | DifficultyMode struct 的 json tag |
| `config/level-list.json` ↔ `internal/config/loader.go` | LevelEntry struct 的 json tag |

## 检查项

### A. JSON↔Go 字段映射

对每个配置文件执行：
1. 列出 JSON 中所有顶层和嵌套字段名
2. 列出 Go struct 中所有 `json:"xxx"` tag
3. 逐一对照，标记不匹配的

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| A1 | towers.json 字段 | 对照 TowerJSON | 所有 JSON 字段有对应 tag |
| A2 | abilities.json 字段 | 对照 AbilityDef | type/label/icon/category/scaleDim/base/potential/param/paramDim/display 全匹配 |
| A3 | enemies-core.json 字段 | 对照 EnemyArchetype | hpScale/speedScale/radius/rewardScale/boss 等全匹配 |
| A4 | wardens.json 字段 | 对照 WardenConfig | 全匹配 |
| A5 | settings.json 字段 | 对照 DifficultyMode | hpScale/speedScale/rewardScale/startGold 全匹配 |

### B. 常量一致性

| # | 常量 | 定义位置 A | 定义位置 B | 验证 |
|---|------|-----------|-----------|------|
| B1 | MinSpeedRatio=0.2 | `combat/crowd_control.go` | `enemy/enemy.go` | 两处值相同 |
| B2 | DotTickInterval=0.5 | `enemy/enemy.go` | — | burn/bleed/zone 都用此值 |
| B3 | MaxAbilitySlots=6 | `tower/upgrade.go` | `config/ability_config.go` AbilityCatCount=6 | 一致 |
| B4 | StrengthBuyCost=10 | `tower/tower.go` | `autoplay/controller.go` 硬编码 10 | 一致 |
| B5 | MaxTowers=64 | `tower/pool.go` | — | 足够大 |
| B6 | MaxEnemies=256 | `enemy/pool.go` | — | 足够大 |
| B7 | ScreenWidth=1200 | `game/constants.go` | 渲染/HUD 中引用 | 一致 |
| B8 | ScreenHeight=540 | `game/constants.go` | 同上 | 一致 |

### C. 注册表完整性

| # | 注册表 | 定义位置 | 检查方法 |
|---|--------|---------|---------|
| C1 | 攻击 handler | `combat/attack.go` init() | abilities.json 中 category=attack 的每种能力，其对应 attackStyle 是否注册 |
| C2 | 能力注册 | `tower/abilities/` init() | abilities.json 中所有 type 是否在 tower.Registry 中 |
| C3 | 游戏模式 | `gamemode/register.go` init() | campaign/endless/timed/bossRush/challenge/test/autoplay 7 种 |
| C4 | 战灵类型 | `warden/types/` init() | prince/core/chain/skystrike/envoy 5 种 |
| C5 | 波次组合 | `spawner.go` waveCompositions | enemies-core.json 中每种原型至少在某阶段的权重中出现 |

### D. 覆盖率列表一致性

| # | 列表 | 定义位置 | 检查 |
|---|------|---------|------|
| D1 | TowerKeys | `autoplay/coverage.go` | 与 towers.json 中的 key 一致 |
| D2 | EnemyArchetypes | `autoplay/coverage.go` | 与 enemies-core.json 中的 key 一致 |
| D3 | Wardens | `autoplay/coverage.go` | 与 wardens.json 中的 key 一致 |
| D4 | GameModes | `autoplay/coverage.go` | 与 register.go 中注册的一致 |

### E. 地图配置

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| E1 | level-list ↔ 实际地图 | 读 level-list.json vs ls config/levels/ | 每个 id 有对应 JSON 文件 |
| E2 | Grid 有效 | 读 map_*.json | Grid 中只有 0/1/2/4/5（合法 cell type） |
| E3 | 路径连通 | 读 pathOrder/entries | 入口(4) → 路径(1) → 基地(5) 连通 |
| E4 | 建造位(2) | 读 Grid | 至少有 1 个 CellBuildable(2) |

## 跨系统关联

配置一致性问题会导致：
- JSON 字段不匹配 → 数据加载为零值 → 系统静默失效（最危险）
- 常量不一致 → 逻辑分叉 → 难以复现的 bug
- 注册表缺项 → 运行时 nil → panic 或静默跳过
