# 配置 → 代码映射

> 自动生成，勿手动编辑。运行 `make index` 更新。

扫描 69 个 JSON 配置文件，找到 643 条映射。

## config/ai/weights.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `episodes` | `internal/core/aiplayer/learning/weights.go` | 197 | Episodes int                    `json:"episodes"`        ... |
| `networks` | `internal/core/aiplayer/learning/weights.go` | 195 | Networks map[string]*Network    `json:"networks,omitempty... |
| `networks.upgrade` | `internal/core/aiplayer/learning/weights.go` | 185 | Upgrade []float64 `json:"upgrade"` // 升级评分权重 |
| `weights` | `internal/core/aiplayer/learning/weights.go` | 28 | Weights [][]float64 `json:"weights"` // [outputSize][inpu... |
| `weights.upgrade` | `internal/core/aiplayer/learning/weights.go` | 185 | Upgrade []float64 `json:"upgrade"` // 升级评分权重 |

## config/audio/bgm.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `tracks` | `internal/scene/audio_preview.go` | 110 | Tracks []rawBGMTrack `json:"tracks"` |

## config/audio/sfx.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `categories` | `internal/scene/audio_preview.go` | 94 | Categories []rawSFXCategory `json:"categories"` |

## config/audio/system.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `volumeCategories.categories` | `internal/scene/audio_preview.go` | 94 | Categories []rawSFXCategory `json:"categories"` |

## config/enemies/abilities.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `armorPlating.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `armorPlating.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `armorPlating.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `armorPlating.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `armorPlating.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `berserk.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `berserk.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `berserk.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `berserk.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `berserk.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `ccImmune.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `ccImmune.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `ccImmune.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `ccImmune.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `ccImmune.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `damageCap.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `damageCap.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `damageCap.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `damageCap.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `damageCap.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `damageCapPercent.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `damageCapPercent.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `damageCapPercent.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `damageCapPercent.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `damageCapPercent.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `damageReduce.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `damageReduce.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `damageReduce.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `damageReduce.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `damageReduce.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `dashOnHit.param2` | `internal/config/enemy_config.go` | 66 | Param2      float64 `json:"param2"` |
| `dashOnHit.param2Dim` | `internal/config/enemy_config.go` | 67 | Param2Dim   string  `json:"param2Dim"` |
| `dashOnHit.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `dashOnHit.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `dashOnHit.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `dashOnHit.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `dashOnHit.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `deathSpawn.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `deathSpawn.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `deathSpawn.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `deathSpawn.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `deathSpawn.spawnArch` | `internal/config/enemy_config.go` | 68 | SpawnArch   string  `json:"spawnArch"` // deathSpawn 子�... |
| `deathSpawn.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `deathSplit.param2` | `internal/config/enemy_config.go` | 66 | Param2      float64 `json:"param2"` |
| `deathSplit.param2Dim` | `internal/config/enemy_config.go` | 67 | Param2Dim   string  `json:"param2Dim"` |
| `deathSplit.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `deathSplit.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `deathSplit.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `deathSplit.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `deathSplit.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `evasion.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `evasion.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `evasion.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `evasion.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `evasion.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `healAura` | `internal/config/enemy_config.go` | 14 | //   - 例: ["stealth", {"type":"healAura","base":0.24,"p... |
| `healAura.param2` | `internal/config/enemy_config.go` | 66 | Param2      float64 `json:"param2"` |
| `healAura.param2Dim` | `internal/config/enemy_config.go` | 67 | Param2Dim   string  `json:"param2Dim"` |
| `healAura.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `healAura.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `healAura.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `healAura.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `healAura.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `phaseShift.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `phaseShift.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `phaseShift.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `phaseShift.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `phaseShift.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `projectileBlock.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `projectileBlock.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `projectileBlock.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `projectileBlock.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `projectileBlock.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `purge.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `purge.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `purge.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `purge.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `purge.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `regen.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `regen.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `regen.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `regen.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `regen.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `slowImmune.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `slowImmune.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `slowImmune.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `slowImmune.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `slowImmune.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `speedAura.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `speedAura.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `speedAura.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `speedAura.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `speedAura.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `stealth` | `internal/config/enemy_config.go` | 14 | //   - 例: ["stealth", {"type":"healAura","base":0.24,"p... |
| `stealth.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `stealth.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `stealth.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `stealth.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `stealth.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `strengthDrain.param2` | `internal/config/enemy_config.go` | 66 | Param2      float64 `json:"param2"` |
| `strengthDrain.param2Dim` | `internal/config/enemy_config.go` | 67 | Param2Dim   string  `json:"param2Dim"` |
| `strengthDrain.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `strengthDrain.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `strengthDrain.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `strengthDrain.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `strengthDrain.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |
| `teleport.paramDim` | `internal/config/enemy_config.go` | 65 | ParamDim    string  `json:"paramDim"` |
| `teleport.potential` | `internal/config/enemy_config.go` | 63 | Potential   float64 `json:"potential"` |
| `teleport.scaleDim` | `internal/config/enemy_config.go` | 61 | ScaleDim    string  `json:"scaleDim"` |
| `teleport.silenceable` | `internal/config/enemy_config.go` | 71 | Silenceable bool    `json:"silenceable"` // 是否可被�... |
| `teleport.visual` | `internal/config/enemy_config.go` | 70 | Visual      string  `json:"visual"`      // 视觉效果�... |

## config/enemies/balance.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `deathSpawn` | `internal/config/balance_config.go` | 126 | DeathSpawn DeathSpawnBalance `json:"deathSpawn"` |
| `deathSpawn.childOffset` | `internal/config/balance_config.go` | 77 | ChildOffset float64 `json:"childOffset"` |
| `deathSpawn.defaultArch` | `internal/config/balance_config.go` | 83 | DefaultArch string  `json:"defaultArch"` |
| `deathSpawn.hpRatio` | `internal/config/balance_config.go` | 73 | HpRatio     float64 `json:"hpRatio"` |
| `deathSpawn.rewardScale` | `internal/config/balance_config.go` | 76 | RewardScale float64 `json:"rewardScale"` // 分裂子体�... |
| `dying.bossDuration` | `internal/config/balance_config.go` | 91 | BossDuration   float64 `json:"bossDuration"` |
| `dying.normalDuration` | `internal/config/balance_config.go` | 90 | NormalDuration float64 `json:"normalDuration"` |
| `split.childOffset` | `internal/config/balance_config.go` | 77 | ChildOffset float64 `json:"childOffset"` |
| `split.hpRatio` | `internal/config/balance_config.go` | 73 | HpRatio     float64 `json:"hpRatio"` |
| `split.radiusRatio` | `internal/config/balance_config.go` | 75 | RadiusRatio float64 `json:"radiusRatio"` |
| `split.rewardScale` | `internal/config/balance_config.go` | 76 | RewardScale float64 `json:"rewardScale"` // 分裂子体�... |
| `split.speedScale` | `internal/config/balance_config.go` | 74 | SpeedScale  float64 `json:"speedScale"` |

## config/gamemodes.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `autoplay.autoStart` | `internal/core/gamemode/mode_config.go` | 20 | AutoStart    bool          `json:"autoStart"` |
| `autoplay.coopEnabled` | `internal/core/gamemode/mode_config.go` | 28 | CoopEnabled  bool          `json:"coopEnabled"` |
| `autoplay.defeat` | `internal/core/gamemode/mode_config.go` | 24 | Defeat       string        `json:"defeat"`       // lives... |
| `autoplay.econID` | `internal/core/gamemode/mode_config.go` | 26 | EconID       string        `json:"econID"` |
| `autoplay.enableEvents` | `internal/core/gamemode/mode_config.go` | 25 | EnableEvents bool          `json:"enableEvents"` |
| `autoplay.endFields` | `internal/core/gamemode/mode_config.go` | 32 | EndFields    []string      `json:"endFields"` |
| `autoplay.initMaxWaves` | `internal/core/gamemode/mode_config.go` | 22 | InitMaxWaves int           `json:"initMaxWaves"` // 0 = �... |
| `autoplay.intermission` | `internal/core/gamemode/mode_config.go` | 21 | Intermission float64       `json:"intermission"` // 0 = �... |
| `autoplay.perfectBonus` | `internal/core/gamemode/mode_config.go` | 27 | PerfectBonus bool          `json:"perfectBonus"` |
| `autoplay.ruleset` | `internal/core/gamemode/mode_config.go` | 29 | Ruleset      RulesetConfig `json:"ruleset"` |
| `autoplay.victory` | `internal/core/gamemode/mode_config.go` | 23 | Victory      string        `json:"victory"`      // allWa... |
| `casual.autoStart` | `internal/core/gamemode/mode_config.go` | 20 | AutoStart    bool          `json:"autoStart"` |
| `casual.coopEnabled` | `internal/core/gamemode/mode_config.go` | 28 | CoopEnabled  bool          `json:"coopEnabled"` |
| `casual.defeat` | `internal/core/gamemode/mode_config.go` | 24 | Defeat       string        `json:"defeat"`       // lives... |
| `casual.econID` | `internal/core/gamemode/mode_config.go` | 26 | EconID       string        `json:"econID"` |
| `casual.enableEvents` | `internal/core/gamemode/mode_config.go` | 25 | EnableEvents bool          `json:"enableEvents"` |
| `casual.endFields` | `internal/core/gamemode/mode_config.go` | 32 | EndFields    []string      `json:"endFields"` |
| `casual.initMaxWaves` | `internal/core/gamemode/mode_config.go` | 22 | InitMaxWaves int           `json:"initMaxWaves"` // 0 = �... |
| `casual.intermission` | `internal/core/gamemode/mode_config.go` | 21 | Intermission float64       `json:"intermission"` // 0 = �... |
| `casual.perfectBonus` | `internal/core/gamemode/mode_config.go` | 27 | PerfectBonus bool          `json:"perfectBonus"` |
| `casual.ruleset` | `internal/core/gamemode/mode_config.go` | 29 | Ruleset      RulesetConfig `json:"ruleset"` |
| `casual.victory` | `internal/core/gamemode/mode_config.go` | 23 | Victory      string        `json:"victory"`      // allWa... |
| `classic` | `internal/config/loader.go` | 60 | //   - modeID == "classic": 只返回 map_cXX 前缀的�... |
| `classic.autoStart` | `internal/core/gamemode/mode_config.go` | 20 | AutoStart    bool          `json:"autoStart"` |
| `classic.coopEnabled` | `internal/core/gamemode/mode_config.go` | 28 | CoopEnabled  bool          `json:"coopEnabled"` |
| `classic.defeat` | `internal/core/gamemode/mode_config.go` | 24 | Defeat       string        `json:"defeat"`       // lives... |
| `classic.econID` | `internal/core/gamemode/mode_config.go` | 26 | EconID       string        `json:"econID"` |
| `classic.enableEvents` | `internal/core/gamemode/mode_config.go` | 25 | EnableEvents bool          `json:"enableEvents"` |
| `classic.endFields` | `internal/core/gamemode/mode_config.go` | 32 | EndFields    []string      `json:"endFields"` |
| `classic.initMaxWaves` | `internal/core/gamemode/mode_config.go` | 22 | InitMaxWaves int           `json:"initMaxWaves"` // 0 = �... |
| `classic.intermission` | `internal/core/gamemode/mode_config.go` | 21 | Intermission float64       `json:"intermission"` // 0 = �... |
| `classic.perfectBonus` | `internal/core/gamemode/mode_config.go` | 27 | PerfectBonus bool          `json:"perfectBonus"` |
| `classic.ruleset` | `internal/core/gamemode/mode_config.go` | 29 | Ruleset      RulesetConfig `json:"ruleset"` |
| `classic.victory` | `internal/core/gamemode/mode_config.go` | 23 | Victory      string        `json:"victory"`      // allWa... |
| `coop.autoStart` | `internal/core/gamemode/mode_config.go` | 20 | AutoStart    bool          `json:"autoStart"` |
| `coop.coopEnabled` | `internal/core/gamemode/mode_config.go` | 28 | CoopEnabled  bool          `json:"coopEnabled"` |
| `coop.defeat` | `internal/core/gamemode/mode_config.go` | 24 | Defeat       string        `json:"defeat"`       // lives... |
| `coop.econID` | `internal/core/gamemode/mode_config.go` | 26 | EconID       string        `json:"econID"` |
| `coop.enableEvents` | `internal/core/gamemode/mode_config.go` | 25 | EnableEvents bool          `json:"enableEvents"` |
| `coop.endFields` | `internal/core/gamemode/mode_config.go` | 32 | EndFields    []string      `json:"endFields"` |
| `coop.initMaxWaves` | `internal/core/gamemode/mode_config.go` | 22 | InitMaxWaves int           `json:"initMaxWaves"` // 0 = �... |
| `coop.intermission` | `internal/core/gamemode/mode_config.go` | 21 | Intermission float64       `json:"intermission"` // 0 = �... |
| `coop.perfectBonus` | `internal/core/gamemode/mode_config.go` | 27 | PerfectBonus bool          `json:"perfectBonus"` |
| `coop.ruleset` | `internal/core/gamemode/mode_config.go` | 29 | Ruleset      RulesetConfig `json:"ruleset"` |
| `coop.victory` | `internal/core/gamemode/mode_config.go` | 23 | Victory      string        `json:"victory"`      // allWa... |
| `simulation.autoStart` | `internal/core/gamemode/mode_config.go` | 20 | AutoStart    bool          `json:"autoStart"` |
| `simulation.coopEnabled` | `internal/core/gamemode/mode_config.go` | 28 | CoopEnabled  bool          `json:"coopEnabled"` |
| `simulation.defeat` | `internal/core/gamemode/mode_config.go` | 24 | Defeat       string        `json:"defeat"`       // lives... |
| `simulation.econID` | `internal/core/gamemode/mode_config.go` | 26 | EconID       string        `json:"econID"` |
| `simulation.enableEvents` | `internal/core/gamemode/mode_config.go` | 25 | EnableEvents bool          `json:"enableEvents"` |
| `simulation.endFields` | `internal/core/gamemode/mode_config.go` | 32 | EndFields    []string      `json:"endFields"` |
| `simulation.initMaxWaves` | `internal/core/gamemode/mode_config.go` | 22 | InitMaxWaves int           `json:"initMaxWaves"` // 0 = �... |
| `simulation.intermission` | `internal/core/gamemode/mode_config.go` | 21 | Intermission float64       `json:"intermission"` // 0 = �... |
| `simulation.perfectBonus` | `internal/core/gamemode/mode_config.go` | 27 | PerfectBonus bool          `json:"perfectBonus"` |
| `simulation.ruleset` | `internal/core/gamemode/mode_config.go` | 29 | Ruleset      RulesetConfig `json:"ruleset"` |
| `simulation.victory` | `internal/core/gamemode/mode_config.go` | 23 | Victory      string        `json:"victory"`      // allWa... |
| `test.autoStart` | `internal/core/gamemode/mode_config.go` | 20 | AutoStart    bool          `json:"autoStart"` |
| `test.coopEnabled` | `internal/core/gamemode/mode_config.go` | 28 | CoopEnabled  bool          `json:"coopEnabled"` |
| `test.defeat` | `internal/core/gamemode/mode_config.go` | 24 | Defeat       string        `json:"defeat"`       // lives... |
| `test.econID` | `internal/core/gamemode/mode_config.go` | 26 | EconID       string        `json:"econID"` |
| `test.enableEvents` | `internal/core/gamemode/mode_config.go` | 25 | EnableEvents bool          `json:"enableEvents"` |
| `test.endFields` | `internal/core/gamemode/mode_config.go` | 32 | EndFields    []string      `json:"endFields"` |
| `test.initMaxWaves` | `internal/core/gamemode/mode_config.go` | 22 | InitMaxWaves int           `json:"initMaxWaves"` // 0 = �... |
| `test.intermission` | `internal/core/gamemode/mode_config.go` | 21 | Intermission float64       `json:"intermission"` // 0 = �... |
| `test.perfectBonus` | `internal/core/gamemode/mode_config.go` | 27 | PerfectBonus bool          `json:"perfectBonus"` |
| `test.ruleset` | `internal/core/gamemode/mode_config.go` | 29 | Ruleset      RulesetConfig `json:"ruleset"` |
| `test.victory` | `internal/core/gamemode/mode_config.go` | 23 | Victory      string        `json:"victory"`      // allWa... |

## config/levels/map_01.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `cellSize` | `internal/config/loader.go` | 156 | CellSize    int                 `json:"cellSize"`        ... |
| `difficulty` | `internal/autoplay/recorder.go` | 80 | Difficulty    string       `json:"difficulty"` |
| `difficulty` | `internal/autoplay/sim_report.go` | 49 | Difficulty string  `json:"difficulty"` |
| `difficulty` | `internal/config/loader.go` | 53 | Difficulty      string `json:"difficulty"` |
| `pathOrder` | `internal/config/loader.go` | 161 | PathOrder   [][2]int            `json:"pathOrder"`       ... |

## config/levels/map_02.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `cellSize` | `internal/config/loader.go` | 156 | CellSize    int                 `json:"cellSize"`        ... |
| `difficulty` | `internal/autoplay/recorder.go` | 80 | Difficulty    string       `json:"difficulty"` |
| `difficulty` | `internal/autoplay/sim_report.go` | 49 | Difficulty string  `json:"difficulty"` |
| `difficulty` | `internal/config/loader.go` | 53 | Difficulty      string `json:"difficulty"` |
| `entries` | `internal/config/loader.go` | 162 | Entries     []MapEntry          `json:"entries,omitempty"... |
| `pathOrder` | `internal/config/loader.go` | 161 | PathOrder   [][2]int            `json:"pathOrder"`       ... |
| `pathOrders` | `internal/config/loader.go` | 163 | PathOrders  map[string][][2]int `json:"pathOrders,omitemp... |
| `pathOrders.bottom` | `internal/config/classic_waves_config.go` | 24 | Path      string `json:"path"`      // 路径 ID（如 "t... |

## config/levels/map_03.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `cellSize` | `internal/config/loader.go` | 156 | CellSize    int                 `json:"cellSize"`        ... |
| `difficulty` | `internal/autoplay/recorder.go` | 80 | Difficulty    string       `json:"difficulty"` |
| `difficulty` | `internal/autoplay/sim_report.go` | 49 | Difficulty string  `json:"difficulty"` |
| `difficulty` | `internal/config/loader.go` | 53 | Difficulty      string `json:"difficulty"` |
| `pathOrder` | `internal/config/loader.go` | 161 | PathOrder   [][2]int            `json:"pathOrder"`       ... |

## config/levels/map_04.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `cellSize` | `internal/config/loader.go` | 156 | CellSize    int                 `json:"cellSize"`        ... |
| `difficulty` | `internal/autoplay/recorder.go` | 80 | Difficulty    string       `json:"difficulty"` |
| `difficulty` | `internal/autoplay/sim_report.go` | 49 | Difficulty string  `json:"difficulty"` |
| `difficulty` | `internal/config/loader.go` | 53 | Difficulty      string `json:"difficulty"` |
| `pathOrder` | `internal/config/loader.go` | 161 | PathOrder   [][2]int            `json:"pathOrder"`       ... |

## config/levels/map_05.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `cellSize` | `internal/config/loader.go` | 156 | CellSize    int                 `json:"cellSize"`        ... |
| `difficulty` | `internal/autoplay/recorder.go` | 80 | Difficulty    string       `json:"difficulty"` |
| `difficulty` | `internal/autoplay/sim_report.go` | 49 | Difficulty string  `json:"difficulty"` |
| `difficulty` | `internal/config/loader.go` | 53 | Difficulty      string `json:"difficulty"` |
| `entries` | `internal/config/loader.go` | 162 | Entries     []MapEntry          `json:"entries,omitempty"... |
| `pathOrder` | `internal/config/loader.go` | 161 | PathOrder   [][2]int            `json:"pathOrder"`       ... |
| `pathOrders` | `internal/config/loader.go` | 163 | PathOrders  map[string][][2]int `json:"pathOrders,omitemp... |
| `pathOrders.bottom` | `internal/config/classic_waves_config.go` | 24 | Path      string `json:"path"`      // 路径 ID（如 "t... |

## config/levels/map_06.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `cellSize` | `internal/config/loader.go` | 156 | CellSize    int                 `json:"cellSize"`        ... |
| `difficulty` | `internal/autoplay/recorder.go` | 80 | Difficulty    string       `json:"difficulty"` |
| `difficulty` | `internal/autoplay/sim_report.go` | 49 | Difficulty string  `json:"difficulty"` |
| `difficulty` | `internal/config/loader.go` | 53 | Difficulty      string `json:"difficulty"` |
| `pathOrder` | `internal/config/loader.go` | 161 | PathOrder   [][2]int            `json:"pathOrder"`       ... |

## config/levels/map_07.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `cellSize` | `internal/config/loader.go` | 156 | CellSize    int                 `json:"cellSize"`        ... |
| `difficulty` | `internal/autoplay/recorder.go` | 80 | Difficulty    string       `json:"difficulty"` |
| `difficulty` | `internal/autoplay/sim_report.go` | 49 | Difficulty string  `json:"difficulty"` |
| `difficulty` | `internal/config/loader.go` | 53 | Difficulty      string `json:"difficulty"` |
| `pathOrder` | `internal/config/loader.go` | 161 | PathOrder   [][2]int            `json:"pathOrder"`       ... |

## config/levels/map_08.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `cellSize` | `internal/config/loader.go` | 156 | CellSize    int                 `json:"cellSize"`        ... |
| `difficulty` | `internal/autoplay/recorder.go` | 80 | Difficulty    string       `json:"difficulty"` |
| `difficulty` | `internal/autoplay/sim_report.go` | 49 | Difficulty string  `json:"difficulty"` |
| `difficulty` | `internal/config/loader.go` | 53 | Difficulty      string `json:"difficulty"` |
| `entries` | `internal/config/loader.go` | 162 | Entries     []MapEntry          `json:"entries,omitempty"... |
| `pathOrder` | `internal/config/loader.go` | 161 | PathOrder   [][2]int            `json:"pathOrder"`       ... |
| `pathOrders` | `internal/config/loader.go` | 163 | PathOrders  map[string][][2]int `json:"pathOrders,omitemp... |

## config/levels/map_c01.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `cellSize` | `internal/config/loader.go` | 156 | CellSize    int                 `json:"cellSize"`        ... |
| `difficulty` | `internal/autoplay/recorder.go` | 80 | Difficulty    string       `json:"difficulty"` |
| `difficulty` | `internal/autoplay/sim_report.go` | 49 | Difficulty string  `json:"difficulty"` |
| `difficulty` | `internal/config/loader.go` | 53 | Difficulty      string `json:"difficulty"` |
| `pathOrder` | `internal/config/loader.go` | 161 | PathOrder   [][2]int            `json:"pathOrder"`       ... |

## config/levels/map_c02.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `cellSize` | `internal/config/loader.go` | 156 | CellSize    int                 `json:"cellSize"`        ... |
| `difficulty` | `internal/autoplay/recorder.go` | 80 | Difficulty    string       `json:"difficulty"` |
| `difficulty` | `internal/autoplay/sim_report.go` | 49 | Difficulty string  `json:"difficulty"` |
| `difficulty` | `internal/config/loader.go` | 53 | Difficulty      string `json:"difficulty"` |
| `pathOrder` | `internal/config/loader.go` | 161 | PathOrder   [][2]int            `json:"pathOrder"`       ... |

## config/levels/map_co01.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `cellSize` | `internal/config/loader.go` | 156 | CellSize    int                 `json:"cellSize"`        ... |
| `coop.playerCount` | `internal/config/loader.go` | 169 | PlayerCount int           `json:"playerCount"` // 玩家�... |
| `coop.sections` | `internal/config/loader.go` | 170 | Sections    []CoopSection `json:"sections"`    // 分区�... |
| `difficulty` | `internal/autoplay/recorder.go` | 80 | Difficulty    string       `json:"difficulty"` |
| `difficulty` | `internal/autoplay/sim_report.go` | 49 | Difficulty string  `json:"difficulty"` |
| `difficulty` | `internal/config/loader.go` | 53 | Difficulty      string `json:"difficulty"` |
| `pathOrder` | `internal/config/loader.go` | 161 | PathOrder   [][2]int            `json:"pathOrder"`       ... |

## config/levels/map_co02.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `cellSize` | `internal/config/loader.go` | 156 | CellSize    int                 `json:"cellSize"`        ... |
| `coop.playerCount` | `internal/config/loader.go` | 169 | PlayerCount int           `json:"playerCount"` // 玩家�... |
| `coop.sections` | `internal/config/loader.go` | 170 | Sections    []CoopSection `json:"sections"`    // 分区�... |
| `difficulty` | `internal/autoplay/recorder.go` | 80 | Difficulty    string       `json:"difficulty"` |
| `difficulty` | `internal/autoplay/sim_report.go` | 49 | Difficulty string  `json:"difficulty"` |
| `difficulty` | `internal/config/loader.go` | 53 | Difficulty      string `json:"difficulty"` |
| `pathOrder` | `internal/config/loader.go` | 161 | PathOrder   [][2]int            `json:"pathOrder"`       ... |

## config/levels/map_co03.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `cellSize` | `internal/config/loader.go` | 156 | CellSize    int                 `json:"cellSize"`        ... |
| `coop.playerCount` | `internal/config/loader.go` | 169 | PlayerCount int           `json:"playerCount"` // 玩家�... |
| `coop.sections` | `internal/config/loader.go` | 170 | Sections    []CoopSection `json:"sections"`    // 分区�... |
| `difficulty` | `internal/autoplay/recorder.go` | 80 | Difficulty    string       `json:"difficulty"` |
| `difficulty` | `internal/autoplay/sim_report.go` | 49 | Difficulty string  `json:"difficulty"` |
| `difficulty` | `internal/config/loader.go` | 53 | Difficulty      string `json:"difficulty"` |
| `pathOrder` | `internal/config/loader.go` | 161 | PathOrder   [][2]int            `json:"pathOrder"`       ... |

## config/levels/map_dummy.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `cellSize` | `internal/config/loader.go` | 156 | CellSize    int                 `json:"cellSize"`        ... |
| `difficulty` | `internal/autoplay/recorder.go` | 80 | Difficulty    string       `json:"difficulty"` |
| `difficulty` | `internal/autoplay/sim_report.go` | 49 | Difficulty string  `json:"difficulty"` |
| `difficulty` | `internal/config/loader.go` | 53 | Difficulty      string `json:"difficulty"` |
| `pathOrder` | `internal/config/loader.go` | 161 | PathOrder   [][2]int            `json:"pathOrder"`       ... |

## config/levels/map_test.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `cellSize` | `internal/config/loader.go` | 156 | CellSize    int                 `json:"cellSize"`        ... |
| `difficulty` | `internal/autoplay/recorder.go` | 80 | Difficulty    string       `json:"difficulty"` |
| `difficulty` | `internal/autoplay/sim_report.go` | 49 | Difficulty string  `json:"difficulty"` |
| `difficulty` | `internal/config/loader.go` | 53 | Difficulty      string `json:"difficulty"` |
| `pathOrder` | `internal/config/loader.go` | 161 | PathOrder   [][2]int            `json:"pathOrder"`       ... |

## config/levels/map_test_large.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `cellSize` | `internal/config/loader.go` | 156 | CellSize    int                 `json:"cellSize"`        ... |
| `difficulty` | `internal/autoplay/recorder.go` | 80 | Difficulty    string       `json:"difficulty"` |
| `difficulty` | `internal/autoplay/sim_report.go` | 49 | Difficulty string  `json:"difficulty"` |
| `difficulty` | `internal/config/loader.go` | 53 | Difficulty      string `json:"difficulty"` |
| `pathOrder` | `internal/config/loader.go` | 161 | PathOrder   [][2]int            `json:"pathOrder"`       ... |

## config/platform/default.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `enemyArmorScale` | `internal/config/platform_config.go` | 21 | EnemyArmorScale     float64 `json:"enemyArmorScale"`     ... |
| `enemyDamageCapScale` | `internal/config/platform_config.go` | 22 | EnemyDamageCapScale float64 `json:"enemyDamageCapScale"` ... |
| `enemyHPScale` | `internal/config/platform_config.go` | 19 | EnemyHPScale        float64 `json:"enemyHPScale"`        ... |
| `enemySpeedScale` | `internal/config/platform_config.go` | 20 | EnemySpeedScale     float64 `json:"enemySpeedScale"`     ... |

## config/platform/web.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `enemyArmorScale` | `internal/config/platform_config.go` | 21 | EnemyArmorScale     float64 `json:"enemyArmorScale"`     ... |
| `enemyDamageCapScale` | `internal/config/platform_config.go` | 22 | EnemyDamageCapScale float64 `json:"enemyDamageCapScale"` ... |
| `enemyHPScale` | `internal/config/platform_config.go` | 19 | EnemyHPScale        float64 `json:"enemyHPScale"`        ... |
| `enemySpeedScale` | `internal/config/platform_config.go` | 20 | EnemySpeedScale     float64 `json:"enemySpeedScale"`     ... |

## config/settings.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `difficulty` | `internal/config/settings_config.go` | 22 | } `json:"difficulty"` |

## config/systems/classic-waves/map_01.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `scaling` | `internal/config/classic_waves_config.go` | 123 | Scaling    ClassicWavesScaling       `json:"scaling"`    ... |
| `scaling.hpBase` | `internal/config/classic_waves_config.go` | 112 | HpBase        float64 `json:"hpBase"`        // 血量基... |
| `scaling.hpPerWave` | `internal/config/classic_waves_config.go` | 113 | HpPerWave     float64 `json:"hpPerWave"`     // 每波血... |
| `scaling.hpQuadratic` | `internal/config/classic_waves_config.go` | 114 | HpQuadratic   float64 `json:"hpQuadratic"`   // 每波血... |
| `scaling.spawnInterval` | `internal/config/classic_waves_config.go` | 117 | SpawnInterval float64 `json:"spawnInterval"` // 出怪间... |
| `scaling.speedBase` | `internal/config/classic_waves_config.go` | 115 | SpeedBase     float64 `json:"speedBase"`     // 速度基... |
| `scaling.speedPerWave` | `internal/config/classic_waves_config.go` | 116 | SpeedPerWave  float64 `json:"speedPerWave"`  // 每波速... |
| `totalWaves` | `internal/config/classic_waves_config.go` | 122 | TotalWaves int                       `json:"totalWaves"` ... |

## config/systems/classic-waves/map_02.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `scaling` | `internal/config/classic_waves_config.go` | 123 | Scaling    ClassicWavesScaling       `json:"scaling"`    ... |
| `scaling.hpBase` | `internal/config/classic_waves_config.go` | 112 | HpBase        float64 `json:"hpBase"`        // 血量基... |
| `scaling.hpPerWave` | `internal/config/classic_waves_config.go` | 113 | HpPerWave     float64 `json:"hpPerWave"`     // 每波血... |
| `scaling.hpQuadratic` | `internal/config/classic_waves_config.go` | 114 | HpQuadratic   float64 `json:"hpQuadratic"`   // 每波血... |
| `scaling.spawnInterval` | `internal/config/classic_waves_config.go` | 117 | SpawnInterval float64 `json:"spawnInterval"` // 出怪间... |
| `scaling.speedBase` | `internal/config/classic_waves_config.go` | 115 | SpeedBase     float64 `json:"speedBase"`     // 速度基... |
| `scaling.speedPerWave` | `internal/config/classic_waves_config.go` | 116 | SpeedPerWave  float64 `json:"speedPerWave"`  // 每波速... |
| `totalWaves` | `internal/config/classic_waves_config.go` | 122 | TotalWaves int                       `json:"totalWaves"` ... |

## config/systems/classic-waves/map_03.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `scaling` | `internal/config/classic_waves_config.go` | 123 | Scaling    ClassicWavesScaling       `json:"scaling"`    ... |
| `scaling.hpBase` | `internal/config/classic_waves_config.go` | 112 | HpBase        float64 `json:"hpBase"`        // 血量基... |
| `scaling.hpPerWave` | `internal/config/classic_waves_config.go` | 113 | HpPerWave     float64 `json:"hpPerWave"`     // 每波血... |
| `scaling.hpQuadratic` | `internal/config/classic_waves_config.go` | 114 | HpQuadratic   float64 `json:"hpQuadratic"`   // 每波血... |
| `scaling.spawnInterval` | `internal/config/classic_waves_config.go` | 117 | SpawnInterval float64 `json:"spawnInterval"` // 出怪间... |
| `scaling.speedBase` | `internal/config/classic_waves_config.go` | 115 | SpeedBase     float64 `json:"speedBase"`     // 速度基... |
| `scaling.speedPerWave` | `internal/config/classic_waves_config.go` | 116 | SpeedPerWave  float64 `json:"speedPerWave"`  // 每波速... |
| `totalWaves` | `internal/config/classic_waves_config.go` | 122 | TotalWaves int                       `json:"totalWaves"` ... |

## config/systems/classic-waves/map_04.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `scaling` | `internal/config/classic_waves_config.go` | 123 | Scaling    ClassicWavesScaling       `json:"scaling"`    ... |
| `scaling.hpBase` | `internal/config/classic_waves_config.go` | 112 | HpBase        float64 `json:"hpBase"`        // 血量基... |
| `scaling.hpPerWave` | `internal/config/classic_waves_config.go` | 113 | HpPerWave     float64 `json:"hpPerWave"`     // 每波血... |
| `scaling.hpQuadratic` | `internal/config/classic_waves_config.go` | 114 | HpQuadratic   float64 `json:"hpQuadratic"`   // 每波血... |
| `scaling.spawnInterval` | `internal/config/classic_waves_config.go` | 117 | SpawnInterval float64 `json:"spawnInterval"` // 出怪间... |
| `scaling.speedBase` | `internal/config/classic_waves_config.go` | 115 | SpeedBase     float64 `json:"speedBase"`     // 速度基... |
| `scaling.speedPerWave` | `internal/config/classic_waves_config.go` | 116 | SpeedPerWave  float64 `json:"speedPerWave"`  // 每波速... |
| `totalWaves` | `internal/config/classic_waves_config.go` | 122 | TotalWaves int                       `json:"totalWaves"` ... |

## config/systems/classic-waves/map_05.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `scaling` | `internal/config/classic_waves_config.go` | 123 | Scaling    ClassicWavesScaling       `json:"scaling"`    ... |
| `scaling.hpBase` | `internal/config/classic_waves_config.go` | 112 | HpBase        float64 `json:"hpBase"`        // 血量基... |
| `scaling.hpPerWave` | `internal/config/classic_waves_config.go` | 113 | HpPerWave     float64 `json:"hpPerWave"`     // 每波血... |
| `scaling.hpQuadratic` | `internal/config/classic_waves_config.go` | 114 | HpQuadratic   float64 `json:"hpQuadratic"`   // 每波血... |
| `scaling.spawnInterval` | `internal/config/classic_waves_config.go` | 117 | SpawnInterval float64 `json:"spawnInterval"` // 出怪间... |
| `scaling.speedBase` | `internal/config/classic_waves_config.go` | 115 | SpeedBase     float64 `json:"speedBase"`     // 速度基... |
| `scaling.speedPerWave` | `internal/config/classic_waves_config.go` | 116 | SpeedPerWave  float64 `json:"speedPerWave"`  // 每波速... |
| `totalWaves` | `internal/config/classic_waves_config.go` | 122 | TotalWaves int                       `json:"totalWaves"` ... |

## config/systems/classic-waves/map_06.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `scaling` | `internal/config/classic_waves_config.go` | 123 | Scaling    ClassicWavesScaling       `json:"scaling"`    ... |
| `scaling.hpBase` | `internal/config/classic_waves_config.go` | 112 | HpBase        float64 `json:"hpBase"`        // 血量基... |
| `scaling.hpPerWave` | `internal/config/classic_waves_config.go` | 113 | HpPerWave     float64 `json:"hpPerWave"`     // 每波血... |
| `scaling.hpQuadratic` | `internal/config/classic_waves_config.go` | 114 | HpQuadratic   float64 `json:"hpQuadratic"`   // 每波血... |
| `scaling.spawnInterval` | `internal/config/classic_waves_config.go` | 117 | SpawnInterval float64 `json:"spawnInterval"` // 出怪间... |
| `scaling.speedBase` | `internal/config/classic_waves_config.go` | 115 | SpeedBase     float64 `json:"speedBase"`     // 速度基... |
| `scaling.speedPerWave` | `internal/config/classic_waves_config.go` | 116 | SpeedPerWave  float64 `json:"speedPerWave"`  // 每波速... |
| `totalWaves` | `internal/config/classic_waves_config.go` | 122 | TotalWaves int                       `json:"totalWaves"` ... |

## config/systems/classic-waves/map_07.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `scaling` | `internal/config/classic_waves_config.go` | 123 | Scaling    ClassicWavesScaling       `json:"scaling"`    ... |
| `scaling.hpBase` | `internal/config/classic_waves_config.go` | 112 | HpBase        float64 `json:"hpBase"`        // 血量基... |
| `scaling.hpPerWave` | `internal/config/classic_waves_config.go` | 113 | HpPerWave     float64 `json:"hpPerWave"`     // 每波血... |
| `scaling.hpQuadratic` | `internal/config/classic_waves_config.go` | 114 | HpQuadratic   float64 `json:"hpQuadratic"`   // 每波血... |
| `scaling.spawnInterval` | `internal/config/classic_waves_config.go` | 117 | SpawnInterval float64 `json:"spawnInterval"` // 出怪间... |
| `scaling.speedBase` | `internal/config/classic_waves_config.go` | 115 | SpeedBase     float64 `json:"speedBase"`     // 速度基... |
| `scaling.speedPerWave` | `internal/config/classic_waves_config.go` | 116 | SpeedPerWave  float64 `json:"speedPerWave"`  // 每波速... |
| `totalWaves` | `internal/config/classic_waves_config.go` | 122 | TotalWaves int                       `json:"totalWaves"` ... |

## config/systems/classic-waves/map_08.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `scaling` | `internal/config/classic_waves_config.go` | 123 | Scaling    ClassicWavesScaling       `json:"scaling"`    ... |
| `scaling.hpBase` | `internal/config/classic_waves_config.go` | 112 | HpBase        float64 `json:"hpBase"`        // 血量基... |
| `scaling.hpPerWave` | `internal/config/classic_waves_config.go` | 113 | HpPerWave     float64 `json:"hpPerWave"`     // 每波血... |
| `scaling.hpQuadratic` | `internal/config/classic_waves_config.go` | 114 | HpQuadratic   float64 `json:"hpQuadratic"`   // 每波血... |
| `scaling.spawnInterval` | `internal/config/classic_waves_config.go` | 117 | SpawnInterval float64 `json:"spawnInterval"` // 出怪间... |
| `scaling.speedBase` | `internal/config/classic_waves_config.go` | 115 | SpeedBase     float64 `json:"speedBase"`     // 速度基... |
| `scaling.speedPerWave` | `internal/config/classic_waves_config.go` | 116 | SpeedPerWave  float64 `json:"speedPerWave"`  // 每波速... |
| `totalWaves` | `internal/config/classic_waves_config.go` | 122 | TotalWaves int                       `json:"totalWaves"` ... |

## config/systems/classic-waves/map_c01.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `scaling` | `internal/config/classic_waves_config.go` | 123 | Scaling    ClassicWavesScaling       `json:"scaling"`    ... |
| `scaling.hpBase` | `internal/config/classic_waves_config.go` | 112 | HpBase        float64 `json:"hpBase"`        // 血量基... |
| `scaling.hpPerWave` | `internal/config/classic_waves_config.go` | 113 | HpPerWave     float64 `json:"hpPerWave"`     // 每波血... |
| `scaling.hpQuadratic` | `internal/config/classic_waves_config.go` | 114 | HpQuadratic   float64 `json:"hpQuadratic"`   // 每波血... |
| `scaling.spawnInterval` | `internal/config/classic_waves_config.go` | 117 | SpawnInterval float64 `json:"spawnInterval"` // 出怪间... |
| `scaling.speedBase` | `internal/config/classic_waves_config.go` | 115 | SpeedBase     float64 `json:"speedBase"`     // 速度基... |
| `scaling.speedPerWave` | `internal/config/classic_waves_config.go` | 116 | SpeedPerWave  float64 `json:"speedPerWave"`  // 每波速... |
| `totalWaves` | `internal/config/classic_waves_config.go` | 122 | TotalWaves int                       `json:"totalWaves"` ... |

## config/systems/classic-waves/map_c02.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `scaling` | `internal/config/classic_waves_config.go` | 123 | Scaling    ClassicWavesScaling       `json:"scaling"`    ... |
| `scaling.hpBase` | `internal/config/classic_waves_config.go` | 112 | HpBase        float64 `json:"hpBase"`        // 血量基... |
| `scaling.hpPerWave` | `internal/config/classic_waves_config.go` | 113 | HpPerWave     float64 `json:"hpPerWave"`     // 每波血... |
| `scaling.hpQuadratic` | `internal/config/classic_waves_config.go` | 114 | HpQuadratic   float64 `json:"hpQuadratic"`   // 每波血... |
| `scaling.spawnInterval` | `internal/config/classic_waves_config.go` | 117 | SpawnInterval float64 `json:"spawnInterval"` // 出怪间... |
| `scaling.speedBase` | `internal/config/classic_waves_config.go` | 115 | SpeedBase     float64 `json:"speedBase"`     // 速度基... |
| `scaling.speedPerWave` | `internal/config/classic_waves_config.go` | 116 | SpeedPerWave  float64 `json:"speedPerWave"`  // 每波速... |
| `totalWaves` | `internal/config/classic_waves_config.go` | 122 | TotalWaves int                       `json:"totalWaves"` ... |

## config/systems/combat.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `critMultiplier` | `internal/config/balance_config.go` | 39 | CritMultiplier            float64 `json:"critMultiplier"`... |
| `defaultProjectileRadius` | `internal/config/balance_config.go` | 41 | DefaultProjectileRadius   float64 `json:"defaultProjectil... |
| `defaultProjectileSpeed` | `internal/config/balance_config.go` | 40 | DefaultProjectileSpeed    float64 `json:"defaultProjectil... |
| `dotTickInterval` | `internal/config/balance_config.go` | 38 | DotTickInterval           float64 `json:"dotTickInterval"... |
| `minSpeedRatio` | `internal/config/balance_config.go` | 37 | MinSpeedRatio             float64 `json:"minSpeedRatio"` ... |
| `wardenMechProjectileSpeed` | `internal/config/balance_config.go` | 42 | WardenMechProjectileSpeed float64 `json:"wardenMechProjec... |

## config/systems/economy.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `global` | `internal/config/economy_spec.go` | 16 | } `json:"global"` |
| `global.killReward` | `internal/config/economy_spec.go` | 14 | KillReward      int     `json:"killReward"` |
| `global.sellRefundRatio` | `internal/config/economy_spec.go` | 15 | SellRefundRatio float64 `json:"sellRefundRatio"` |

## config/systems/gameplay.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `gameplay` | `internal/config/balance_config.go` | 129 | Gameplay   GameplayBalance   `json:"gameplay"` |
| `gameplay.multiKillWindow` | `internal/config/balance_config.go` | 115 | MultiKillWindow     float64 `json:"multiKillWindow"` |
| `gameplay.starRatingThreshold` | `internal/config/balance_config.go` | 114 | StarRatingThreshold float64 `json:"starRatingThreshold"` |
| `itemDrop` | `internal/config/balance_config.go` | 130 | ItemDrop   ItemDropBalance   `json:"itemDrop"` |
| `itemDrop.cycleWaves` | `internal/config/balance_config.go` | 105 | CycleWaves  int     `json:"cycleWaves"`  // 掉落周期�... |
| `itemDrop.dropChance` | `internal/config/balance_config.go` | 106 | DropChance  float64 `json:"dropChance"`  // 每次击杀�... |
| `itemDrop.flySec` | `internal/config/balance_config.go` | 109 | FlySec      float64 `json:"flySec"`      // 道具飞向�... |
| `itemDrop.groundSec` | `internal/config/balance_config.go` | 108 | GroundSec   float64 `json:"groundSec"`   // 道具落地�... |
| `itemDrop.maxPerCycle` | `internal/config/balance_config.go` | 107 | MaxPerCycle int     `json:"maxPerCycle"` // 每周期最�... |

## config/systems/spawner.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `boss.dyingDuration` | `internal/config/spawner_config.go` | 56 | DyingDuration    float64 `json:"dyingDuration"` // 死亡... |
| `boss.entranceDelay` | `internal/config/spawner_config.go` | 53 | EntranceDelay    float64 `json:"entranceDelay"` |
| `boss.everyNWaves` | `internal/config/spawner_config.go` | 50 | EveryNWaves      int     `json:"everyNWaves"` |
| `boss.hpMultBase` | `internal/config/spawner_config.go` | 51 | HpMultBase       float64 `json:"hpMultBase"` |
| `boss.percentHpCap` | `internal/config/spawner_config.go` | 55 | PercentHpCap     float64 `json:"percentHpCap"`  // %HP �... |
| `boss.radiusScale` | `internal/config/spawner_config.go` | 52 | RadiusScale      float64 `json:"radiusScale"` |
| `boss.rewardMultiplier` | `internal/config/spawner_config.go` | 54 | RewardMultiplier float64 `json:"rewardMultiplier"` |
| `compositions` | `internal/config/spawner_config.go` | 100 | Compositions []WaveComposition             `json:"composi... |
| `coopScaling` | `internal/config/spawner_config.go` | 97 | CoopScaling  map[string]CoopScalingEntry   `json:"coopSca... |
| `scaling` | `internal/config/spawner_config.go` | 96 | Scaling      SpawnerScaling               `json:"scaling"` |
| `scaling.enemiesPerWave` | `internal/config/spawner_config.go` | 35 | EnemiesPerWave    int     `json:"enemiesPerWave"`    // �... |
| `scaling.hpBase` | `internal/config/spawner_config.go` | 30 | HpBase            float64 `json:"hpBase"`            // �... |
| `scaling.hpPerWave` | `internal/config/spawner_config.go` | 31 | HpPerWave         float64 `json:"hpPerWave"`         // �... |
| `scaling.hpQuadratic` | `internal/config/spawner_config.go` | 32 | HpQuadratic       float64 `json:"hpQuadratic"`       // �... |
| `scaling.spawnBaseInterval` | `internal/config/spawner_config.go` | 37 | SpawnBaseInterval float64 `json:"spawnBaseInterval"` // �... |
| `scaling.spawnDecayPerWave` | `internal/config/spawner_config.go` | 39 | SpawnDecayPerWave float64 `json:"spawnDecayPerWave"` // �... |
| `scaling.spawnInterval` | `internal/config/spawner_config.go` | 36 | SpawnInterval     float64 `json:"spawnInterval"`     // �... |
| `scaling.spawnMinInterval` | `internal/config/spawner_config.go` | 38 | SpawnMinInterval  float64 `json:"spawnMinInterval"`  // �... |
| `scaling.speedBase` | `internal/config/spawner_config.go` | 33 | SpeedBase         float64 `json:"speedBase"`         // �... |
| `scaling.speedPerWave` | `internal/config/spawner_config.go` | 34 | SpeedPerWave      float64 `json:"speedPerWave"`      // �... |
| `squads` | `internal/config/spawner_config.go` | 102 | Squads       SquadsConfig                  `json:"squads"` |
| `squads.templates` | `internal/config/spawner_config.go` | 84 | Templates []SquadTemplate `json:"templates"` // 所有可... |
| `timing` | `internal/config/spawner_config.go` | 98 | Timing       SpawnerTiming                 `json:"timing"` |
| `timing.firstWaveInterval` | `internal/config/spawner_config.go` | 45 | FirstWaveInterval float64 `json:"firstWaveInterval"` // �... |
| `timing.waveInterval` | `internal/config/spawner_config.go` | 44 | WaveInterval      float64 `json:"waveInterval"`      // �... |
| `waveBuffs` | `internal/config/spawner_config.go` | 101 | WaveBuffs    WaveBuffConfig                `json:"waveBuf... |
| `waveBuffs.chance` | `internal/config/spawner_config.go` | 70 | Chance float64        `json:"chance"` |

## config/systems/tower-randomize.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `formula.potential` | `internal/core/tower/descriptor/derive_ability_table.go` | 278 | Potential float64 `json:"potential"` |
| `formula.potential` | `internal/core/tower/descriptor/scaler.go` | 39 | Potential float64 `json:"potential"` |
| `specialty` | `internal/config/scenario_config.go` | 43 | Specialty       int       `json:"specialty,omitempty"` //... |
| `specialty` | `internal/core/tower/descriptor/blueprint.go` | 39 | Specialty string `json:"specialty"` |

## config/towers/abilities.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `attackSpeedAura.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `attackSpeedAura.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `attackSpeedAura.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `attackSpeedAura.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `attackSpeedAura.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `barrage.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `barrage.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `barrage.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `barrage.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `barrage.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `bleedDot.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `bleedDot.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `bleedDot.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `bleedDot.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `bleedDot.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `bounce.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `bounce.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `bounce.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `bounce.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `bounce.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `burn.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `burn.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `burn.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `burn.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `burn.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `crit.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `crit.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `crit.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `crit.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `crit.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `critAura.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `critAura.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `critAura.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `critAura.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `critAura.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `curseZone.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `curseZone.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `curseZone.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `curseZone.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `curseZone.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `damageUpAura.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `damageUpAura.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `damageUpAura.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `damageUpAura.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `damageUpAura.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `distanceDamage.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `distanceDamage.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `distanceDamage.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `distanceDamage.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `distanceDamage.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `enhance.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `enhance.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `enhance.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `enhance.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `enhance.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `executionBonus.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `executionBonus.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `executionBonus.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `executionBonus.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `executionBonus.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `flatDamage.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `flatDamage.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `flatDamage.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `flatDamage.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `flatDamage.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `goldPassive.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `goldPassive.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `goldPassive.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `goldPassive.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `goldPassive.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `momentum.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `momentum.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `momentum.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `momentum.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `momentum.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `multiTarget.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `multiTarget.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `multiTarget.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `multiTarget.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `multiTarget.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `poison.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `poison.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `poison.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `poison.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `poison.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `poisonZone.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `poisonZone.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `poisonZone.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `poisonZone.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `poisonZone.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `radial.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `radial.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `radial.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `radial.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `radial.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `rangeAura.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `rangeAura.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `rangeAura.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `rangeAura.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `rangeAura.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `scatter.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `scatter.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `scatter.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `scatter.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `scatter.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `silenceZone.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `silenceZone.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `silenceZone.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `silenceZone.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `silenceZone.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `slowDuration.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `slowDuration.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `slowDuration.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `slowDuration.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `slowDuration.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `slowPower.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `slowPower.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `slowPower.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `slowPower.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `slowPower.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `soloBoost.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `soloBoost.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `soloBoost.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `soloBoost.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `soloBoost.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `spinAoe.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `spinAoe.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `spinAoe.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `spinAoe.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `spinAoe.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `splash.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `splash.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `splash.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `splash.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `splash.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `stunChance.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `stunChance.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `stunChance.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `stunChance.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `stunChance.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `stunDuration.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `stunDuration.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `stunDuration.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `stunDuration.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `stunDuration.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `weaken.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `weaken.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `weaken.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `weaken.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `weaken.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `weakenZone.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `weakenZone.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `weakenZone.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `weakenZone.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `weakenZone.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |
| `wideBeam.param2` | `internal/config/ability_config.go` | 26 | Param2    float64 `json:"param2"`    // 第二固定参�... |
| `wideBeam.param2Dim` | `internal/config/ability_config.go` | 27 | Param2Dim string  `json:"param2Dim"` // 第二固定参�... |
| `wideBeam.paramDim` | `internal/config/ability_config.go` | 25 | ParamDim  string  `json:"paramDim"`  // 固定参数的�... |
| `wideBeam.potential` | `internal/config/ability_config.go` | 23 | Potential float64 `json:"potential"` // 缩放维度的�... |
| `wideBeam.scaleDim` | `internal/config/ability_config.go` | 21 | ScaleDim  string  `json:"scaleDim"`  // 可提升维度�... |

## config/towers/ability-descriptors.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `descriptors` | `internal/core/tower/descriptor/loader.go` | 23 | Descriptors []json.RawMessage `json:"descriptors"` |

## config/towers/balance.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `chain.distance` | `internal/config/balance_config.go` | 58 | Distance         float64 `json:"distance"`         // 连... |
| `chain.strengthPerTower` | `internal/config/balance_config.go` | 59 | StrengthPerTower float64 `json:"strengthPerTower"` // 每... |
| `tower.attackSpeedFloor` | `internal/config/balance_config.go` | 52 | AttackSpeedFloor  float64 `json:"attackSpeedFloor"`  // �... |
| `tower.choicesPerUnlock` | `internal/config/balance_config.go` | 51 | ChoicesPerUnlock  int     `json:"choicesPerUnlock"`  // �... |
| `tower.strengthBuyAmount` | `internal/config/balance_config.go` | 49 | StrengthBuyAmount float64 `json:"strengthBuyAmount"` // �... |
| `tower.strengthBuyCost` | `internal/config/balance_config.go` | 48 | StrengthBuyCost   int     `json:"strengthBuyCost"`   // �... |
| `tower.wavesPerUnlock` | `internal/config/balance_config.go` | 50 | WavesPerUnlock    int     `json:"wavesPerUnlock"`    // �... |

## config/towers/budget-rules.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `attackStyleCosts` | `internal/core/tower/descriptor/budget.go` | 12 | AttackStyleCosts map[string]int `json:"attackStyleCosts"`... |
| `attackStyleCosts.projectile` | `internal/config/tower_config.go` | 32 | AttackStyle     string  `json:"attackStyle"`     // "proj... |
| `attackStyleCosts.scatter` | `internal/config/tower_config.go` | 32 | AttackStyle     string  `json:"attackStyle"`     // "proj... |
| `attackStyleCosts.spin_aoe` | `internal/config/tower_config.go` | 32 | AttackStyle     string  `json:"attackStyle"`     // "proj... |
| `attackStyleCosts.wideBeam` | `internal/config/tower_config.go` | 32 | AttackStyle     string  `json:"attackStyle"`     // "proj... |
| `baseBuildCost` | `internal/core/tower/descriptor/budget.go` | 15 | BaseBuildCost    int            `json:"baseBuildCost"`   ... |
| `baseCap` | `internal/core/tower/descriptor/budget.go` | 10 | BaseCap          int            `json:"baseCap"`         ... |
| `costPerPoint` | `internal/core/tower/descriptor/budget.go` | 16 | CostPerPoint     float64        `json:"costPerPoint"`    ... |
| `maxSlots` | `internal/core/tower/descriptor/budget.go` | 11 | MaxSlots         int            `json:"maxSlots"`        ... |
| `specialtyCost` | `internal/core/tower/descriptor/budget.go` | 14 | SpecialtyCost    int            `json:"specialtyCost"`   ... |
| `tierCosts` | `internal/core/tower/descriptor/budget.go` | 13 | TierCosts        map[string]int `json:"tierCosts"`       ... |

## config/towers/classic-presets.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `defaults` | `internal/config/classic_preset_config.go` | 48 | Defaults classicPresetsDefaults `json:"defaults"` |
| `defaults.abilityMode` | `internal/config/classic_preset_config.go` | 29 | AbilityMode string            `json:"abilityMode"` // 能... |
| `defaults.buildCost` | `internal/config/classic_preset_config.go` | 34 | BuildCost   int               `json:"buildCost"`   // 建... |
| `defaults.strength` | `internal/config/classic_preset_config.go` | 32 | Strength    StrengthConfig    `json:"strength"`    // 强... |
| `towers` | `internal/config/classic_preset_config.go` | 49 | Towers   []ClassicPreset        `json:"towers"` |

## config/towers/prebuilt-blueprints.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `blueprints` | `internal/core/gamemode/mode_config.go` | 43 | Blueprints     bool   `json:"blueprints"`     // 允许�... |
| `blueprints` | `internal/core/tower/descriptor/blueprint_store.go` | 23 | Blueprints []TowerBlueprint `json:"blueprints"` |
| `blueprints` | `internal/core/tower/descriptor/prebuilt_blueprints.go` | 23 | Blueprints []TowerBlueprint `json:"blueprints"` |

## config/towers/tier-presets.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `attackSpeed` | `internal/config/tier_presets.go` | 25 | AttackSpeed AttrTiers `json:"attackSpeed"` |
| `attackSpeed.basePotential` | `internal/config/tier_presets.go` | 18 | BasePotential float64              `json:"basePotential"`... |
| `damage` | `internal/config/tier_presets.go` | 24 | Damage      AttrTiers `json:"damage"` |
| `damage.basePotential` | `internal/config/tier_presets.go` | 18 | BasePotential float64              `json:"basePotential"`... |
| `range.basePotential` | `internal/config/tier_presets.go` | 18 | BasePotential float64              `json:"basePotential"`... |

## config/towers/towers.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `basic.abilities` | `internal/config/tower_config.go` | 30 | Abilities      []string            `json:"abilities"`    ... |
| `basic.abilities` | `internal/core/tower/descriptor/ability_store.go` | 33 | Abilities []CustomAbility `json:"abilities"` |
| `basic.abilities` | `internal/core/tower/descriptor/blueprint.go` | 42 | Abilities []string `json:"abilities"` |
| `basic.abilityMode` | `internal/config/tower_config.go` | 36 | AbilityMode string         `json:"abilityMode"` // 能力... |
| `basic.attackStyle` | `internal/config/tower_config.go` | 32 | AttackStyle     string  `json:"attackStyle"`     // "proj... |
| `basic.attackStyle` | `internal/core/tower/descriptor/blueprint.go` | 30 | AttackStyle string `json:"attackStyle"` |
| `basic.attackStyle` | `internal/core/tower/descriptor/descriptor.go` | 28 | AttackStyle string            `json:"attackStyle,omitempty"` |
| `basic.buildCost` | `internal/config/tower_config.go` | 20 | BuildCost   int    `json:"buildCost"`   // 建造费用�... |
| `basic.buildCost` | `internal/config/validate.go` | 35 | Field:   "buildCost", |
| `basic.buildCost` | `internal/core/tower/descriptor/blueprint.go` | 45 | BuildCost int            `json:"buildCost"` |
| `basic.projectileSpeed` | `internal/config/tower_config.go` | 33 | ProjectileSpeed float64 `json:"projectileSpeed"` // 弹�... |
| `basic.shortLabel` | `internal/config/tower_config.go` | 18 | ShortLabel  string `json:"shortLabel"`  // 塔简称（HU... |
| `basic.strength` | `internal/config/scenario_config.go` | 44 | Strength        float64   `json:"strength,omitempty"`  //... |
| `basic.strength` | `internal/config/tower_config.go` | 37 | Strength    StrengthConfig `json:"strength"`     // 强�... |
| `basic.strength` | `internal/core/tower/descriptor/blueprint.go` | 46 | Strength  StrengthConfig `json:"strength"` |
| `basic.strength` | `internal/core/tower/descriptor/scaler.go` | 77 | Strength float64 `json:"strength"` |
| `basic.upgradeCosts` | `internal/config/tower_config.go` | 40 | UpgradeCosts []int `json:"upgradeCosts"` // 每次升级�... |

## config/visuals/vfx.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `categories` | `internal/config/vfx_config.go` | 29 | Categories []VFXCategory `json:"categories"` |

## config/wardens/balance.json

| JSON 字段 | Go 文件 | 行 | 上下文 |
|-----------|---------|-----|--------|
| `defaultGrowthOnKill` | `internal/config/balance_config.go` | 97 | DefaultGrowthOnKill      float64 `json:"defaultGrowthOnKi... |
| `defaultGrowthOnWaveClear` | `internal/config/balance_config.go` | 98 | DefaultGrowthOnWaveClear float64 `json:"defaultGrowthOnWa... |
| `initialStrength` | `internal/config/balance_config.go` | 96 | InitialStrength          float64 `json:"initialStrength"` |

