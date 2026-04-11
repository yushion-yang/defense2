# BuffList Phase 2 Design — 行为类 Buff 迁移

> Date: 2026-04-11
> Status: Approved
> Goal: 将敌人行为字段 (berserk/regen/healer/stealth/buffer/phaseShift/damageReduce) 迁移到 BuffList

## 核心原则

**BuffList 管数据，behaviors.go 管执行。**

- BuffList 只存储 buff 的存在性和参数（Has/Get/Add/Remove）
- 行为 tick 逻辑仍在 `behaviors.go`，通过 `e.Buffs.Has("healAura")` 判断是否执行
- buff 包保持零 core 依赖

## 迁移清单

| 旧字段 | Buff ID | Category | Value | Value2 | Duration |
|--------|---------|----------|-------|--------|----------|
| BerserkThreshold/SpeedScale/Triggered | `berserk` | Behavior | speedScale | threshold | -1 (永久) |
| RegenPerSec | `regen` | Behavior | regenPerSec | — | -1 |
| HealPower/Radius/Interval/Cooldown | `healAura` | Behavior | power | radius | -1 |
| BuffRadius/BuffAmount | `bufferAura` | Behavior | amount | radius | -1 |
| Stealthed/StealthTimer | `stealth` | Behavior | — | — | 有时长 |
| PhaseDuration/Cooldown/Active | `phaseShift` | Behavior | duration | cooldown | -1 |
| DamageReduceRatio | `damageReduce` | Defense | ratio | — | -1 |
| SpeedBuff（光环受益方） | `speedUp` | Behavior | factor | — | 0.05 (1帧过期) |

### 不迁移

- `Silenced/AbilitySilenced` — 每帧 zone reset，不是 buff
- `DamageCap/DamageCapPercent/ArmorFlat/EvasionChance/ProjectileBlockChance` — 原型固有能力，不是 buff
- `Dash*/StrDrain*/Teleport*/Split*/DeathSpawn*/Purge*` — 复杂能力机制，保留直接字段
- `Behavior` string — 保留，用于 switch 分发

## 改动设计

### behaviors.go 改造

**Before** (直接读字段):
```go
if e.RegenPerSec > 0 {
    TickRegeneration(e, dt)
}
```

**After** (从 BuffList 读):
```go
if b, ok := e.Buffs.Get("regen"); ok {
    regenHP(e, b.Value, dt)
}
```

每个行为的改造：

**berserk**: 触发时 Add `berserk` buff (永久)。TickBerserk 改为：检查 threshold buff 参数，触发后标记（仍用 BerserkTriggered 避免重复触发），执行 `BaseSpeed *= speedScale`。实际上 berserk 触发后就是永久速度变化，不需要每帧 tick。保留 BerserkTriggered 标志防止重复。

**regen**: 从 `e.Buffs.Get("regen").Value` 读 regenPerSec，执行回血逻辑。

**healAura**: 从 `e.Buffs.Get("healAura")` 读 power/radius。冷却计时器放 buff 外部（HealCooldown 保留），因为 BuffList 没有 per-buff 自定义计时。或者用 Buff.Remaining 做冷却 —— 不行，Remaining 是 buff 剩余时间不是冷却。保留 HealCooldown 直接字段。

**buffer**: 从 `e.Buffs.Get("bufferAura")` 读 amount/radius。给范围内敌人 Add `speedUp` buff (Duration=0.05，1帧后过期)。

**stealth**: 直接用 BuffList 的 `stealth` buff，Has("stealth") 替代 Stealthed 字段。受击破隐 = Remove("stealth")。

**phaseShift**: 从 `e.Buffs.Get("phaseShift")` 读 duration/cooldown。PhaseTimer/PhaseActive 保留为运行时状态（不是 buff 参数）。

**damageReduce**: 纯被动。`e.Buffs.Get("damageReduce").Value` 在 damage_pipeline.go 中读取。

### pool.go Spawn 改造

wave buff `applyWaveBuff` 改为 Add buff 到 BuffList，不再设直接字段：
```go
case "berserk":
    e.Buffs.Add(buff.Buff{ID: "berserk", Category: buff.CatBehavior,
        Value: 1.5, Value2: 0.5, Duration: -1, Remaining: -1})
case "regen":
    e.Buffs.Add(buff.Buff{ID: "regen", Category: buff.CatBehavior,
        Value: e.MaxHP * 0.02, Duration: -1, Remaining: -1})
```

### 新增 Helper 方法

```go
func (e *Enemy) HasBerserk() bool    { return e.Buffs != nil && e.Buffs.Has("berserk") }
func (e *Enemy) HasRegen() bool      { return e.Buffs != nil && e.Buffs.Has("regen") }
func (e *Enemy) HasHealAura() bool   { return e.Buffs != nil && e.Buffs.Has("healAura") }
func (e *Enemy) HasBufferAura() bool { return e.Buffs != nil && e.Buffs.Has("bufferAura") }
func (e *Enemy) IsStealthed() bool   { return e.Buffs != nil && e.Buffs.Has("stealth") }
func (e *Enemy) HasPhaseShift() bool { return e.Buffs != nil && e.Buffs.Has("phaseShift") }
func (e *Enemy) GetDamageReduce() float64 {
    if e.Buffs == nil { return 0 }
    b, ok := e.Buffs.Get("damageReduce")
    if !ok { return 0 }
    return b.Value
}
func (e *Enemy) HasSpeedUp() bool { return e.Buffs != nil && e.Buffs.Has("speedUp") }
func (e *Enemy) GetSpeedUp() float64 {
    if e.Buffs == nil { return 0 }
    b, ok := e.Buffs.Get("speedUp")
    if !ok { return 0 }
    return b.Value
}
```

### 保留的运行时字段

这些字段是行为的运行时状态，不是 buff 参数，保留在 Enemy struct：
- `BerserkTriggered bool` — 防止重复触发
- `HealCooldown float64` — healer 冷却计时
- `PhaseTimer float64` — phaseShift 当前阶段计时
- `PhaseActive bool` — 当前是否免伤

### 删除的字段

| 字段 | 替代 |
|------|------|
| BerserkThreshold, BerserkSpeedScale | Buff Value/Value2 |
| RegenPerSec | Buff Value |
| HealPower, HealRadius, HealInterval | Buff Value/Value2 + buff-stack.json |
| Stealthed, StealthTimer | Buff Has/Remaining |
| BuffRadius, BuffAmount, SpeedBuff, AuraRange, AuraSpeedUp | Buff Value/Value2 + speedUp buff |
| DamageReduceRatio | Buff Value |
| PhaseDuration, PhaseCooldown | Buff Value/Value2 |

### buff-stack.json 新增规则

```json
"berserk": {"mode": "override"},
"regen": {"mode": "override"},
"healAura": {"mode": "override"},
"bufferAura": {"mode": "override"},
"stealth": {"mode": "override"},
"phaseShift": {"mode": "override"},
"damageReduce": {"mode": "strongest", "cap": 0.8},
"speedUp": {"mode": "strongest", "cap": 1.4}
```

## 文件影响

| 文件 | 改动 |
|------|------|
| `internal/core/enemy/enemy.go` | 删除迁移字段，添加 helper 方法 |
| `internal/core/enemy/behaviors.go` | 从 BuffList 读参数替代直接字段 |
| `internal/core/enemy/pool.go` | Spawn 不再设迁移字段 |
| `internal/core/enemy/spawner.go` | applyWaveBuff 改为 Buffs.Add |
| `internal/core/enemy/movement.go` | SpeedBuff 改为 GetSpeedUp() |
| `internal/core/combat/damage_pipeline.go` | DamageReduceRatio 改为 GetDamageReduce() |
| `internal/render/draw_enemy.go` | Stealthed 改为 IsStealthed() |
| `internal/scene/stage.go` | HUD/遥测读取点迁移 |
| `config/systems/buff-stack.json` | 新增 8 条规则 |

## HUD 可视化

`BuffList.Active()` 现在会返回行为类 buff，HUD 可以展示：
- 敌人头顶：狂暴图标、回血图标、隐身状态、免伤相位
- 选中敌人面板：所有 buff 列表含参数
