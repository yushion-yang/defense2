# 定制炮塔系统设计

> 日期: 2026-04-15
> 状态: 已确认，待实施

## 概述

构建一个**原语驱动的能力描述符引擎**，将现有 32 个能力拆解为更底层的构建块（触发器/条件/目标选择器/效果/缩放器），
让玩家用这些原语自由组合出自定义能力，进而组装成定制炮塔蓝图，跨所有游戏模式使用。

### 设计目标

- 玩家定制的能力与官方预制能力在引擎层**完全平等**
- 交互形式：**槽位拼装**（模块化硬件式，移动端友好）
- 平衡约束：**预算系统**（每个原语有费用，塔有总预算上限）
- 落地节奏：**渐进式**（中粒度验证 → 细粒度 → 极细粒度）

### 现状

| 模式 | 塔来源 | 能力获取 |
|------|--------|---------|
| 战役/测试 | 放置 "basic" 塔，随机 roll S/B/D | 每 2 波解锁 1 槽，6 类各 3 选 1 |
| 经典 | 11 种预设成品塔 | 固定 2 个能力，不可更改 |
| **新: 定制** | 玩家蓝图库 | 出厂内置，固定能力 |

---

## 架构：三层原语引擎

```
Layer 3: 炮塔模板（官方预制 / 经典预设 / 玩家自定义蓝图）
Layer 2: 描述符引擎（触发→条件→目标→效果 管线，JSON 驱动）
Layer 1: 原语注册表（Trigger / Condition / Selector / Effect / Scaler 五类原语）
```

---

## Layer 1：原语注册表

每个原语是一个 Go 接口实现 + JSON schema，通过 `init()` 注册到全局表。

### 1.1 Trigger（触发器）

| ID | 触发时机 | 上下文参数 | Phase |
|---|---|---|---|
| `onHit` | 弹道/光束命中敌人 | tower, projectile, enemy | 1 |
| `onTick` | 每帧（60fps） | tower, dt | 1 |
| `onKill` | 击杀敌人时 | tower, enemy | 1 |
| `onPlace` | 塔放置时（一次性） | tower | 1 |
| `onSell` | 塔出售时（一次性） | tower | 2 |
| `onWaveStart` | 波次开始 | tower, waveNum | 2 |
| `onWaveEnd` | 波次结束 | tower, waveNum | 2 |
| `onDamageTaken` | 附近友方受击 | tower, source, target | 3 |

### 1.2 Condition（条件门）

| ID | 含义 | 参数 | Phase |
|---|---|---|---|
| `chance` | 概率触发 | `rate: Scaler` | 1 |
| `cooldown` | 冷却时间 | `seconds: float` | 1 |
| `hpBelow` | 目标 HP 低于阈值 | `threshold: Scaler (0-1)` | 1 |
| `hpAbove` | 目标 HP 高于阈值 | `threshold: Scaler (0-1)` | 1 |
| `distanceMin` | 与目标距离 >= N | `distance: float` | 1 |
| `noNearbyTower` | 周围无友方塔 | `radius: float` | 1 |
| `isBoss` | 目标是 Boss | — | 2 |
| `notBoss` | 目标非 Boss | — | 2 |
| `every` | 每 N 次触发执行一次 | `n: int` | 2 |
| `buffActive` | 目标身上有某 buff | `buffID: string` | 2 |
| `buffAbsent` | 目标身上无某 buff | `buffID: string` | 2 |

条件支持 AND 组合（数组内全部满足）。Phase 2 加 OR/NOT 逻辑运算符。

### 1.3 Selector（目标选择器）

| ID | 选择目标 | 参数 | Phase |
|---|---|---|---|
| `currentTarget` | 当前攻击目标 | — | 1 |
| `aoeRadius` | 命中点范围内敌人 | `radius: Scaler` | 1 |
| `chain` | 链式弹跳 | `maxBounce: Scaler, range: float, decayRatio: float` | 1 |
| `allInRange` | 塔射程内所有敌人 | — | 1 |
| `nearbyAllies` | 附近友方塔 | `radius: float` | 1 |
| `selfTower` | 自身塔 | — | 1 |
| `cone` | 扇形范围 | `angle: float, radius: Scaler` | 2 |
| `ring360` | 360度均匀方向 | `count: Scaler` | 2 |
| `random` | 范围内随机 N 个 | `count: Scaler, radius: float` | 2 |

### 1.4 Effect（效果）

| ID | 参数 | 说明 | Phase |
|---|---|---|---|
| `damage` | `mode: flat/ratio/hpPercent`, `value: Scaler` | 伤害 | 1 |
| `slow` | `factor: Scaler`, `duration: Scaler` | 减速 | 1 |
| `stun` | `duration: Scaler` | 眩晕 | 1 |
| `root` | `duration: Scaler` | 定身 | 1 |
| `dot` | `subtype: burn/bleed/poison`, `value: Scaler`, `duration: Scaler`, `mode` | 持续伤害 | 1 |
| `weaken` | `amplify: Scaler`, `duration: Scaler` | 易伤 | 1 |
| `silence` | — | 沉默 | 1 |
| `buff` | `stat: damage/speed/range/crit`, `bonus: Scaler` | 友方增益 | 1 |
| `selfBuff` | `stat`, `bonus: Scaler` | 自身增益 | 1 |
| `gold` | `amount: Scaler` | 产金 | 1 |
| `modifyStat` | `stat`, `multiplier: float` | 一次性改属性 | 1 |
| `crit` | `multiplier: float` | 暴击（独立 effect） | 2 |
| `purge` | `count: int` | 移除敌人 buff | 2 |
| `teleport` | `distance: float` | 沿路径回推 | 3 |

### 1.5 Scaler（缩放器）

| ID | 公式 | 参数 | Phase |
|---|---|---|---|
| `linear` | `base + potential * (str/100)` | `base, potential` | 1 |
| `fixed` | `value` | `value` | 1 |
| `diminishing` | `base + potential * (1 - e^(-str/k))` | `base, potential, k` | 2 |
| `capped` | `min(linear(b,p,str), cap)` | `base, potential, cap` | 2 |
| `stepped` | 按阈值分段 | `thresholds: [{str, value}]` | 3 |

### Go 接口设计

```go
type PrimitiveID = string

type TriggerDef struct {
    ID       PrimitiveID
    Label    string
    Cost     int
    CtxType  TriggerCtxType
}

type ConditionDef struct {
    ID     PrimitiveID
    Label  string
    Cost   int
    Params map[string]ParamSchema
}

// Selector, Effect, Scaler 类似结构

// 全局注册表
var Triggers   = map[PrimitiveID]*TriggerDef{}
var Conditions = map[PrimitiveID]*ConditionDef{}
var Selectors  = map[PrimitiveID]*SelectorDef{}
var Effects    = map[PrimitiveID]*EffectDef{}
var Scalers    = map[PrimitiveID]*ScalerDef{}
```

---

## Layer 2：描述符引擎

### 2.1 能力描述符（AbilityDescriptor）

一个能力 = 一组 Pipeline（管线），每条管线是一个「触发→条件→目标→效果」链路。

```json
{
  "id": "stunChance",
  "label": "震慑",
  "icon": "stun",
  "cost": 8,
  "tags": ["cc", "single_target"],
  "pipelines": [
    {
      "trigger": "onHit",
      "conditions": [
        {"type": "chance", "rate": {"scaler": "linear", "base": 0.10, "potential": 0.05}}
      ],
      "selector": {"type": "currentTarget"},
      "effects": [
        {"type": "stun", "duration": {"scaler": "fixed", "value": 0.5}}
      ]
    }
  ]
}
```

复杂例子 — splash（溅射）：

```json
{
  "id": "splash",
  "label": "溅射",
  "icon": "splash",
  "cost": 10,
  "tags": ["attack", "aoe"],
  "attackStyle": "projectile",
  "spriteKey": "mortar",
  "pipelines": [
    {
      "trigger": "onHit",
      "conditions": [],
      "selector": {"type": "aoeRadius", "radius": {"scaler": "fixed", "value": 50}},
      "effects": [
        {"type": "damage", "mode": "ratio", "value": {"scaler": "linear", "base": 0.75, "potential": 0.05}}
      ]
    }
  ]
}
```

玩家自创 — 冰火连击（多管线）：

```json
{
  "id": "user_frostfire",
  "label": "冰火连击",
  "cost": 15,
  "tags": ["cc", "dot"],
  "pipelines": [
    {
      "trigger": "onHit",
      "conditions": [{"type": "chance", "rate": {"scaler": "fixed", "value": 0.4}}],
      "selector": {"type": "currentTarget"},
      "effects": [
        {"type": "slow", "factor": {"scaler": "linear", "base": 0.30, "potential": 0.05}, "duration": {"scaler": "fixed", "value": 1.5}}
      ]
    },
    {
      "trigger": "onHit",
      "conditions": [],
      "selector": {"type": "currentTarget"},
      "effects": [
        {"type": "dot", "subtype": "burn", "mode": "ratio", "value": {"scaler": "linear", "base": 0.08, "potential": 0.03}, "duration": {"scaler": "fixed", "value": 2.0}}
      ]
    }
  ]
}
```

### 2.2 攻击模式描述符

攻击模式（scatter/wideBeam/spinAoe/radial/barrage）改变弹道行为本身，不走 pipeline。
用 `attackStyle` + `attackParams` 字段声明：

```json
{
  "id": "scatter",
  "label": "散射",
  "cost": 12,
  "tags": ["attack"],
  "attackStyle": "scatter",
  "spriteKey": "shotgun",
  "attackParams": {
    "pellets": {"scaler": "linear", "base": 3, "potential": 0.5},
    "coneAngle": {"scaler": "fixed", "value": 60}
  },
  "pipelines": []
}
```

**规则**：一个炮塔最多 1 个带 `attackStyle` 的描述符（唯一硬性互斥约束）。
无 attackStyle 则默认 `projectile`。

### 2.3 运行时解释器

```go
// Interpreter 挂载在每个 Tower 上，塔放置时从描述符预编译。
type Interpreter struct {
    pipelines []compiledPipeline
}

// Exec 由 combat pipeline 调用。
// 匹配 trigger → 逐条 condition 求值 → selector 选目标 → 逐 effect 执行。
func (interp *Interpreter) Exec(ctx TriggerContext) []EffectResult {
    var results []EffectResult
    for _, p := range interp.pipelines {
        if p.trigger.Type() != ctx.Type() {
            continue
        }
        if !p.evalConditions(ctx) {
            continue
        }
        targets := p.selector.Select(ctx)
        for _, eff := range p.effects {
            for _, t := range targets {
                results = append(results, eff.Apply(ctx, t))
            }
        }
    }
    return results
}
```

关键设计：
- **预编译**：放置时 JSON → `compiledPipeline`（接口指针数组），运行时零反射零 map 查找
- **EffectResult**：替代现有 `HitResult`，统一的效果输出结构
- **衔接点**：`Interpreter.Exec()` 在 `combat/apply_hit.go` 中替代 `ability.OnHit()` 循环；
  在 `pipeline/sys_tower.go` 中替代 `ability.(Ticker).OnTick()` 调用

### 2.4 EffectResult 统一输出

```go
type EffectResult struct {
    Type     EffectType

    // 伤害类
    Damage      float64
    DamageMode  DamageMode // flat/ratio/hpPercent/separate
    IsCrit      bool
    CritMult    float64

    // CC 类
    SlowFactor  float64
    StunDur     float64
    RootDur     float64
    Duration    float64

    // DoT 类
    DotSubtype  string
    DotValue    float64
    DotDuration float64
    DotMode     DamageMode

    // Buff 类
    BuffStat    string
    BuffBonus   float64

    // 经济类
    GoldAmount  float64

    // 目标
    Target      *enemy.Enemy
}
```

### 2.5 现有 32 能力迁移映射

| 现有能力 | 描述符管线 |
|---------|-----------|
| `stunChance` | `onHit → chance(linear 0.1,0.05) → currentTarget → stun(fixed 0.5)` |
| `splash` | `onHit → (none) → aoeRadius(fixed 50) → damage(ratio, linear 0.75,0.05)` |
| `damageUpAura` | `onTick → (none) → nearbyAllies(150) → buff(damage, linear 0.05,0.02)` |
| `goldPassive` | `onTick → cooldown(3s) → selfTower → gold(linear 1,1)` |
| `executionBonus` | `onHit → hpBelow(fixed 0.5) → currentTarget → damage(ratio, linear 0.20,0.05)` |
| `soloBoost` | `onTick → noNearbyTower(120) → selfTower → selfBuff(damage, linear 0.05,0.05)` |
| `burn` | `onHit → (none) → currentTarget → dot(burn, ratio, linear 0.10,0.05, dur=2.0)` |
| `bleedDot` | `onHit → notBoss → currentTarget → dot(bleed, hpPercent, linear 0.01,0.004, dur=3.0)` |
| `poison` | `onHit → (none) → currentTarget → dot(poison, flat, linear 3,5, dur=4.0)` |
| `weaken` | `onHit → (none) → currentTarget → weaken(linear 0.15,0.05, dur=3.0)` |
| `crit` | `onHit → chance(linear 0.10,0.05) → currentTarget → crit(mult=2.0)` |
| `distanceDamage` | `onHit → distanceMin(150) → currentTarget → damage(ratio, linear 0.10,0.05 * steps)` |
| `flatDamage` | `onHit → (none) → currentTarget → damage(flat, linear 2,5)` |
| `momentum` | `onHit → (none) → currentTarget → damage(ratio, linear 0.05,0.05 * hitCount)` |
| `slowPower` | `onHit → (none) → currentTarget → slow(linear 0.20,0.05, fixed 1.0)` |
| `slowDuration` | `onHit → (none) → currentTarget → slow(fixed 0.35, linear 0.5,0.1)` |
| `stunDuration` | `onHit → chance(fixed 0.25) → currentTarget → stun(linear 0.2,0.1)` |
| `poisonZone` | `onTick → (none) → allInRange → dot(poison, flat, linear 3,5)` |
| `silenceZone` | `onTick → (none) → allInRange → silence` |
| `curseZone` | `onTick → (none) → allInRange → damage(hpPercent, linear 0.01,0.002)` |
| `weakenZone` | `onTick → (none) → allInRange → weaken(linear 0.05,0.05)` |
| `attackSpeedAura` | `onTick → (none) → nearbyAllies(150) → buff(speed, linear 0.08,0.02)` |
| `rangeAura` | `onTick → (none) → nearbyAllies(150) → buff(range, linear 10,2)` |
| `critAura` | `onTick → (none) → nearbyAllies(150) → buff(crit, linear 0.04,0.02)` |
| `enhance` | `onPlace → (none) → selfTower → modifyStat(damage, 1.8) + modifyStat(speed, 1.8) + modifyStat(range, 1.2)` |

攻击模式类（scatter/wideBeam/spinAoe/radial/barrage/bounce/multiTarget）用
`attackStyle` + `attackParams` 表示，不走 pipeline。

---

## Layer 3：炮塔模板 + 预算系统

### 3.1 炮塔蓝图（TowerBlueprint）

```json
{
  "id": "bp_frostfire_001",
  "name": "霜焰使者",
  "author": "player",
  "createdAt": "2026-04-15T10:30:00Z",

  "attackStyle": "scatter",
  "attackParams": {
    "pellets": {"scaler": "linear", "base": 3, "potential": 0.5},
    "coneAngle": {"scaler": "fixed", "value": 60}
  },
  "spriteKey": "shotgun",

  "tiers": {"damage": "B", "atkSpeed": "S", "range": "B"},
  "specialty": "atkSpeed",

  "abilities": ["user_frostfire", "crit", "poisonZone"],

  "budget": {
    "cap": 50,
    "used": 38,
    "breakdown": {
      "scatter": 12,
      "user_frostfire": 15,
      "crit": 6,
      "poisonZone": 5
    }
  },

  "buildCost": 59,
  "strength": {
    "cost": 50,
    "amount": 50,
    "maxPurchases": -1
  }
}
```

### 3.2 预算系统

#### 费用分层

| 层级 | 说明 | 费用来源 |
|---|---|---|
| 攻击模式 | scatter/beam/radial 等 | attackStyle 固定 cost |
| 能力描述符 | 每个能力的整体 cost | 描述符 `cost` 字段 |
| 原语级（Phase 2+） | 自组合能力时 cost = 各原语 cost 之和 | 各原语的 `cost` |

#### 配置

```json
{
  "budgetRules": {
    "baseCap": 50,
    "maxSlots": 6,
    "attackStyleCosts": {
      "projectile": 0,
      "scatter": 12,
      "wideBeam": 14,
      "spin_aoe": 10,
      "radial": 11,
      "barrage": 13
    },
    "tierCosts": {
      "S": 4,
      "B": 2,
      "D": 0
    },
    "specialtyCost": 2,
    "baseBuildCost": 40,
    "costPerPoint": 0.5
  }
}
```

#### 核心公式

- `usedBudget = attackStyleCost + sum(abilityCosts) + sum(tierCosts) + specialtyCost`
- `usedBudget <= budgetCap` 才合法
- `buildCost = baseBuildCost + usedBudget * costPerPoint`

设计意图：
- D 档免费、S 档 4 点 → SSS 花 12 点预算，留给能力的空间少
- 攻击模式占大头（10-14 点）→ 强攻击模式 + 强能力不可兼得
- `projectile` 免费 → bounce/splash/multiTarget 等衍生能力零攻击模式开销

#### 校验逻辑

```go
type BudgetValidator struct{}

func (v *BudgetValidator) Validate(bp *TowerBlueprint) []ValidationError {
    // 1. attackStyle 最多 1 个
    // 2. usedBudget <= budgetCap
    // 3. 所有引用的 ability ID 存在于注册表
    // 4. 无重复 ability
    // 5. tags 互斥检查（Phase 2）
}
```

### 3.3 与现有模式整合

#### TowerRuleset 扩展

```go
type TowerRuleset interface {
    // 现有方法不变
    AllowCustomBlueprints() bool // 新增
    CustomBudgetCap() int        // 新增（-1 = 用默认）
}
```

| 方法 | Campaign | Test | Classic |
|---|---|---|---|
| AllowCustomBlueprints | true | true | false |
| CustomBudgetCap | -1 | -1 | — |

整合方式：
- 建塔菜单新增「自定义」tab，列出蓝图库中的塔
- 自定义塔放置后固定能力，只能买 Strength
- 与原有 "basic" 塔共存混用
- 经典模式不允许（保持纯粹体验）

#### 经济权衡

自定义塔花钱多但确定性高（能力固定），基础塔花钱少但看运气（随机 roll）。

### 3.4 持久化

```go
const blueprintsKey = "tower_blueprints"

type BlueprintStore struct {
    Blueprints []TowerBlueprint `json:"blueprints"`
    Version    int              `json:"version"`
}
```

- 桌面：`~/.defense2/tower_blueprints.json`
- WASM：localStorage `tower_blueprints` key
- 上限：20 个蓝图

### 3.5 现有 32 能力 cost 预估

| 费用档 | cost | 能力 |
|---|---|---|
| 低 (4-6) | 4-6 | goldPassive, soloBoost, rangeAura, poison |
| 中 (7-9) | 7-9 | crit, slowPower, slowDuration, stunChance, stunDuration, burn, bleedDot, weaken, damageUpAura, attackSpeedAura, critAura |
| 高 (10-13) | 10-13 | executionBonus, flatDamage, momentum, distanceDamage, weakenZone, curseZone, poisonZone, silenceZone |
| 攻击 (10-14) | 10-14 | scatter, wideBeam, spinAoe, radial, barrage, bounce, splash, multiTarget, enhance |

精确数值在实现阶段通过 autoplay 平衡测试调整。

---

## 渐进式落地

### Phase 1：中粒度引擎（基础可用）

**目标**：描述符引擎上线，现有 32 能力全部迁移为描述符驱动，autoplay 全量回归通过。

**文件结构**：

```
internal/core/tower/descriptor/
├── primitives.go      — 五类原语接口 + 注册表
├── trigger.go         — 4 种 Trigger (onHit/onTick/onKill/onPlace)
├── condition.go       — 6 种 Condition (chance/cooldown/hpBelow/hpAbove/distanceMin/noNearbyTower)
├── selector.go        — 6 种 Selector (currentTarget/aoeRadius/chain/allInRange/nearbyAllies/selfTower)
├── effect.go          — 11 种 Effect (damage/slow/stun/root/dot/weaken/silence/buff/selfBuff/gold/modifyStat)
├── scaler.go          — 2 种 Scaler (linear/fixed)
├── descriptor.go      — AbilityDescriptor + JSON 反序列化
├── interpreter.go     — 预编译 + 运行时执行
├── validator.go       — 描述符合法性校验
├── budget.go          — 预算计算与校验
└── migrate.go         — 老 AbilityDef → 描述符 自动转换

config/towers/
├── ability-descriptors.json  — 32 个能力的描述符定义
├── budget-rules.json         — 预算规则
├── abilities.json            — 保留（fallback）

internal/core/tower/
├── blueprint.go       — TowerBlueprint + Blueprint→TowerDef
└── blueprint_store.go — 持久化 CRUD
```

**不做的事**：
- 不做 UI 编辑器（JSON 手写蓝图，测试模式验证）
- 不做 Phase 2/3 细粒度原语
- 不做条件 OR/NOT 组合
- 不改建塔菜单 UI

**验收标准**：
1. `make test` 全通过
2. `autoplay --sweep` 68 局零异常，行为等价
3. 现有存档/配置零破坏

### Phase 2：细粒度 + 建塔菜单

**目标**：更多原语 + 条件组合逻辑 + 建塔菜单整合自定义塔。

**新增原语**：Trigger(onWaveStart/onWaveEnd/onSell), Condition(isBoss/notBoss/every/buffActive/buffAbsent + AND/OR/NOT),
Selector(cone/ring360/random), Effect(crit/purge), Scaler(diminishing/capped)

**UI 变更**：
- 建塔菜单新增「自定义」tab
- 蓝图编辑场景（从预制能力库挑选，不开放原语组合）

**验收标准**：能在 Campaign 中混用自定义塔通关 map_01-03

### Phase 3：极细粒度 + 槽位拼装编辑器

**目标**：完整的原语级自由组合 UI。

**能力编辑器**（新场景）：
- 添加/删除管线
- 每管线选择触发器/条件/目标/效果
- 参数滑块调节 + 实时费用计算

**蓝图编辑器**（新场景）：
- 攻击模式选择
- 属性档位 + 专精选择
- 能力槽拖拽管理
- 预算进度条

**新增**：Scaler(stepped), Effect(teleport)

**验收标准**：玩家从零创建能力+蓝图，在 Campaign 中使用并通关

### Phase 前置关系

```
Phase 1 (引擎 + 迁移)
   │ 验收: autoplay --sweep 零差异
   ▼
Phase 2 (更多原语 + 建塔菜单)
   │ 验收: Campaign 混用自定义塔通关 map_01-03
   ▼
Phase 3 (槽位拼装编辑器)
   │ 验收: 从零创建能力+蓝图并通关
```

---

## 向后兼容

| 场景 | 处理方式 |
|---|---|
| 老存档加载 | progress.json 无蓝图字段 → 正常，蓝图库为空 |
| 老配置 abilities.json | Phase 1 保留，migrate.go 自动转描述符。Phase 2 删除 |
| 经典预设塔 | 不改格式，加载时内部转为 Blueprint → 描述符 |
| 战役模式 3 选 1 | 候选池从描述符注册表按 tags 筛选，替代按 category 筛选 |
| ConfigAbility 大 switch | Phase 1 并行两套，autoplay 验证后 Phase 2 删除 |
| HitResult 结构 | Phase 1 保留 + 适配层。Phase 2 统一为 EffectResult |
| WASM 构建 | 描述符 JSON 通过 `//go:embed` 打入 |
