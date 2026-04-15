# Custom Tower Presets — Phase 1 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the descriptor engine (Layer 1 + Layer 2), migrate all 32 existing abilities to descriptor-driven execution, and add the blueprint/budget foundation — with zero behavioral change verified by autoplay.

**Architecture:** A new `internal/core/tower/descriptor/` package implements 5 primitive types (Trigger/Condition/Selector/Effect/Scaler), a JSON-driven AbilityDescriptor format, and a precompiled Interpreter that replaces the current `ConfigAbility` big-switch dispatch. Existing `abilities.json` is mechanically converted to `ability-descriptors.json`. A `TowerBlueprint` struct + `BudgetValidator` + persistence store provide the foundation for player-customized towers (UI deferred to Phase 2).

**Tech Stack:** Go 1.24+, Ebitengine v2.9.9, JSON config via `//go:embed`, table-driven tests with `-race`

**Design doc:** `docs/plans/2026-04-15-custom-tower-presets-design.md`

---

## Task 1: Scaler primitives

最底层的缩放计算器。无任何外部依赖，纯数学。

**Files:**
- Create: `internal/core/tower/descriptor/scaler.go`
- Test: `tests/core/descriptor_scaler_test.go`

**Step 1: Write failing tests**

```go
// tests/core/descriptor_scaler_test.go
package core_test

import (
    "testing"
    "defense2/internal/core/tower/descriptor"
)

func TestLinearScaler(t *testing.T) {
    tests := []struct {
        name     string
        base     float64
        pot      float64
        str      float64
        want     float64
    }{
        {"base only", 10, 0, 100, 10},
        {"str=100", 10, 5, 100, 15},
        {"str=200", 10, 5, 200, 20},
        {"str=0", 10, 5, 0, 10},
        {"negative pot", 10, -2, 100, 8},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            s := descriptor.LinearScaler{Base: tt.base, Potential: tt.pot}
            got := s.Calc(tt.str)
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}

func TestFixedScaler(t *testing.T) {
    s := descriptor.FixedScaler{Value: 42}
    if got := s.Calc(0); got != 42 {
        t.Errorf("str=0: got %v", got)
    }
    if got := s.Calc(999); got != 42 {
        t.Errorf("str=999: got %v", got)
    }
}

func TestParseScaler(t *testing.T) {
    tests := []struct {
        name string
        json string
        str  float64
        want float64
    }{
        {"linear", `{"scaler":"linear","base":10,"potential":5}`, 100, 15},
        {"fixed", `{"scaler":"fixed","value":42}`, 999, 42},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            s, err := descriptor.ParseScaler([]byte(tt.json))
            if err != nil {
                t.Fatal(err)
            }
            if got := s.Calc(tt.str); got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

**Step 2: Run test, verify FAIL**

```bash
go test ./tests/core/ -run TestLinearScaler -v
# Expected: FAIL — package not found
```

**Step 3: Implement**

```go
// internal/core/tower/descriptor/scaler.go
// scaler.go — 缩放器原语。
//
// 定义 Scaler 接口及两种实现（linear/fixed），
// 控制能力参数如何随塔 Strength 值变化。
// 公式沿用现有 CalcScale: base + potential * (str/100)。
package descriptor

import (
    "encoding/json"
    "fmt"
)

// Scaler 缩放器接口 — 将 Strength 映射为最终参数值。
type Scaler interface {
    Calc(strength float64) float64
}

// LinearScaler 线性缩放：base + potential * (strength / 100)。
// 覆盖现有 AbilityDef.CalcScale 的全部用例。
type LinearScaler struct {
    Base      float64 `json:"base"`
    Potential float64 `json:"potential"`
}

func (s LinearScaler) Calc(strength float64) float64 {
    return s.Base + s.Potential*(strength/100.0)
}

// FixedScaler 固定值，不随 Strength 变化。
type FixedScaler struct {
    Value float64 `json:"value"`
}

func (s FixedScaler) Calc(_ float64) float64 {
    return s.Value
}

// scalerJSON JSON 反序列化中间结构。
type scalerJSON struct {
    Type string  `json:"scaler"`
    Base float64 `json:"base"`
    Pot  float64 `json:"potential"`
    Val  float64 `json:"value"`
}

// ParseScaler 从 JSON 字节解析 Scaler。
func ParseScaler(data []byte) (Scaler, error) {
    var raw scalerJSON
    if err := json.Unmarshal(data, &raw); err != nil {
        return nil, fmt.Errorf("parse scaler: %w", err)
    }
    switch raw.Type {
    case "linear":
        return LinearScaler{Base: raw.Base, Potential: raw.Pot}, nil
    case "fixed":
        return FixedScaler{Value: raw.Val}, nil
    default:
        return nil, fmt.Errorf("unknown scaler type: %q", raw.Type)
    }
}
```

**Step 4: Run test, verify PASS**

```bash
go test ./tests/core/ -run "Test(Linear|Fixed|Parse)Scaler" -v -race
```

**Step 5: Commit**

```bash
git add internal/core/tower/descriptor/scaler.go tests/core/descriptor_scaler_test.go
git commit -m "feat: add Scaler primitives (linear/fixed) for descriptor engine"
```

---

## Task 2: Condition primitives

条件门实现。依赖 Scaler（Task 1）。

**Files:**
- Create: `internal/core/tower/descriptor/condition.go`
- Test: `tests/core/descriptor_condition_test.go`

**Step 1: Write failing tests**

```go
// tests/core/descriptor_condition_test.go
package core_test

import (
    "testing"
    "defense2/internal/core/tower/descriptor"
)

func TestChanceCondition(t *testing.T) {
    // chance=1.0 固定通过
    c := descriptor.ChanceCondition{Rate: descriptor.FixedScaler{Value: 1.0}}
    ctx := descriptor.ConditionCtx{Strength: 100}
    for i := 0; i < 100; i++ {
        if !c.Eval(ctx) {
            t.Fatal("chance=1.0 should always pass")
        }
    }
    // chance=0.0 固定不通过
    c2 := descriptor.ChanceCondition{Rate: descriptor.FixedScaler{Value: 0.0}}
    for i := 0; i < 100; i++ {
        if c2.Eval(ctx) {
            t.Fatal("chance=0.0 should never pass")
        }
    }
}

func TestCooldownCondition(t *testing.T) {
    c := descriptor.CooldownCondition{Seconds: 3.0}
    ctx := descriptor.ConditionCtx{Strength: 100, Elapsed: 0}
    // 首次调用应通过（从未触发过）
    if !c.Eval(ctx) {
        t.Fatal("first call should pass")
    }
    // 立刻再调用不应通过
    if c.Eval(ctx) {
        t.Fatal("immediate second call should fail")
    }
    // 3 秒后应通过
    ctx.Elapsed = 3.1
    if !c.Eval(ctx) {
        t.Fatal("after cooldown should pass")
    }
}

func TestHpBelowCondition(t *testing.T) {
    c := descriptor.HpBelowCondition{Threshold: descriptor.FixedScaler{Value: 0.5}}
    // HP 30% < 50% → 通过
    ctx := descriptor.ConditionCtx{TargetHpRatio: 0.3}
    if !c.Eval(ctx) {
        t.Fatal("0.3 < 0.5 should pass")
    }
    // HP 70% > 50% → 不通过
    ctx.TargetHpRatio = 0.7
    if c.Eval(ctx) {
        t.Fatal("0.7 > 0.5 should fail")
    }
}

func TestNoNearbyTowerCondition(t *testing.T) {
    c := descriptor.NoNearbyTowerCondition{Radius: 120}
    // 无邻居 → 通过
    ctx := descriptor.ConditionCtx{NearestAllyDist: 200}
    if !c.Eval(ctx) {
        t.Fatal("no nearby tower should pass")
    }
    // 有邻居 → 不通过
    ctx.NearestAllyDist = 80
    if c.Eval(ctx) {
        t.Fatal("nearby tower should fail")
    }
}

func TestConditionAND(t *testing.T) {
    // chance=1.0 AND hpBelow=0.5 → 目标 HP 30% → 通过
    conds := []descriptor.Condition{
        &descriptor.ChanceCondition{Rate: descriptor.FixedScaler{Value: 1.0}},
        &descriptor.HpBelowCondition{Threshold: descriptor.FixedScaler{Value: 0.5}},
    }
    ctx := descriptor.ConditionCtx{TargetHpRatio: 0.3}
    if !descriptor.EvalAll(conds, ctx) {
        t.Fatal("AND should pass when all conditions pass")
    }
    // chance=1.0 AND hpBelow=0.5 → 目标 HP 70% → 不通过
    ctx.TargetHpRatio = 0.7
    if descriptor.EvalAll(conds, ctx) {
        t.Fatal("AND should fail when one condition fails")
    }
}
```

**Step 2: Run test, verify FAIL**

```bash
go test ./tests/core/ -run TestChanceCondition -v
```

**Step 3: Implement**

```go
// internal/core/tower/descriptor/condition.go
// condition.go — 条件门原语。
//
// 条件门决定管线是否执行。多个条件之间为 AND 关系（全部满足才执行）。
// ConditionCtx 由解释器在运行时填充，包含判断所需的上下文信息。
package descriptor

import "math/rand"

// Condition 条件门接口。
type Condition interface {
    Eval(ctx ConditionCtx) bool
}

// ConditionCtx 条件求值上下文。
// 由 Interpreter 在触发时从 Tower/Enemy/Pipeline 上下文中提取并填充。
type ConditionCtx struct {
    Strength        float64 // 塔当前 Strength
    TargetHpRatio   float64 // 目标当前 HP / 最大 HP (0-1)
    TargetDistance   float64 // 目标与塔的距离
    NearestAllyDist float64 // 最近友方塔的距离（无友方时为 math.MaxFloat64）
    Elapsed         float64 // 自管线上次触发以来的秒数（用于 cooldown）
}

// EvalAll AND 组合：全部条件通过则返回 true。空列表返回 true。
func EvalAll(conds []Condition, ctx ConditionCtx) bool {
    for _, c := range conds {
        if !c.Eval(ctx) {
            return false
        }
    }
    return true
}

// ChanceCondition 概率门。
type ChanceCondition struct {
    Rate Scaler // 触发概率 (0-1)
}

func (c *ChanceCondition) Eval(ctx ConditionCtx) bool {
    rate := c.Rate.Calc(ctx.Strength)
    if rate >= 1.0 {
        return true
    }
    if rate <= 0.0 {
        return false
    }
    return rand.Float64() < rate
}

// CooldownCondition 冷却时间门（有状态）。
type CooldownCondition struct {
    Seconds    float64
    lastFired  float64
    firstCall  bool
}

func NewCooldownCondition(seconds float64) *CooldownCondition {
    return &CooldownCondition{Seconds: seconds, firstCall: true}
}

func (c *CooldownCondition) Eval(ctx ConditionCtx) bool {
    if c.firstCall {
        c.firstCall = false
        c.lastFired = ctx.Elapsed
        return true
    }
    if ctx.Elapsed-c.lastFired >= c.Seconds {
        c.lastFired = ctx.Elapsed
        return true
    }
    return false
}

// HpBelowCondition 目标 HP 低于阈值。
type HpBelowCondition struct {
    Threshold Scaler
}

func (c *HpBelowCondition) Eval(ctx ConditionCtx) bool {
    return ctx.TargetHpRatio < c.Threshold.Calc(ctx.Strength)
}

// HpAboveCondition 目标 HP 高于阈值。
type HpAboveCondition struct {
    Threshold Scaler
}

func (c *HpAboveCondition) Eval(ctx ConditionCtx) bool {
    return ctx.TargetHpRatio > c.Threshold.Calc(ctx.Strength)
}

// DistanceMinCondition 与目标距离 >= N。
type DistanceMinCondition struct {
    Distance float64
}

func (c *DistanceMinCondition) Eval(ctx ConditionCtx) bool {
    return ctx.TargetDistance >= c.Distance
}

// NoNearbyTowerCondition 周围无友方塔。
type NoNearbyTowerCondition struct {
    Radius float64
}

func (c *NoNearbyTowerCondition) Eval(ctx ConditionCtx) bool {
    return ctx.NearestAllyDist > c.Radius
}
```

**Step 4: Run test, verify PASS**

```bash
go test ./tests/core/ -run "Test(Chance|Cooldown|HpBelow|NoNearby|ConditionAND)" -v -race
```

**Step 5: Commit**

```bash
git add internal/core/tower/descriptor/condition.go tests/core/descriptor_condition_test.go
git commit -m "feat: add Condition primitives (chance/cooldown/hpBelow/hpAbove/distanceMin/noNearbyTower)"
```

---

## Task 3: Effect primitives

效果原语。返回统一的 EffectResult。

**Files:**
- Create: `internal/core/tower/descriptor/effect.go`
- Create: `internal/core/tower/descriptor/effect_result.go`
- Test: `tests/core/descriptor_effect_test.go`

**Step 1: Write failing tests**

```go
// tests/core/descriptor_effect_test.go
package core_test

import (
    "testing"
    "defense2/internal/core/tower/descriptor"
)

func TestDamageEffect(t *testing.T) {
    // flat damage
    e := descriptor.DamageEffect{
        Mode:  descriptor.DmgFlat,
        Value: descriptor.FixedScaler{Value: 10},
    }
    ctx := descriptor.EffectCtx{Strength: 100, TowerDamage: 50}
    r := e.Apply(ctx)
    if r.Type != descriptor.EffTypeDamage {
        t.Fatalf("type=%v", r.Type)
    }
    if r.Damage != 10 {
        t.Errorf("flat damage got %v, want 10", r.Damage)
    }

    // ratio damage (50% of tower damage 50 = 25)
    e2 := descriptor.DamageEffect{
        Mode:  descriptor.DmgRatio,
        Value: descriptor.FixedScaler{Value: 0.5},
    }
    r2 := e2.Apply(ctx)
    if r2.Damage != 25 {
        t.Errorf("ratio damage got %v, want 25", r2.Damage)
    }
}

func TestSlowEffect(t *testing.T) {
    e := descriptor.SlowEffect{
        Factor:   descriptor.LinearScaler{Base: 0.3, Potential: 0.05},
        Duration: descriptor.FixedScaler{Value: 1.0},
    }
    ctx := descriptor.EffectCtx{Strength: 200}
    r := e.Apply(ctx)
    if r.Type != descriptor.EffTypeSlow {
        t.Fatalf("type=%v", r.Type)
    }
    if r.SlowFactor != 0.4 { // 0.3 + 0.05*(200/100)
        t.Errorf("slow factor got %v, want 0.4", r.SlowFactor)
    }
    if r.Duration != 1.0 {
        t.Errorf("duration got %v, want 1.0", r.Duration)
    }
}

func TestStunEffect(t *testing.T) {
    e := descriptor.StunEffect{Duration: descriptor.FixedScaler{Value: 0.5}}
    r := e.Apply(descriptor.EffectCtx{Strength: 100})
    if r.Type != descriptor.EffTypeStun || r.StunDur != 0.5 {
        t.Errorf("stun: type=%v dur=%v", r.Type, r.StunDur)
    }
}

func TestDotEffect(t *testing.T) {
    e := descriptor.DotEffect{
        Subtype:  "burn",
        Mode:     descriptor.DmgRatio,
        Value:    descriptor.FixedScaler{Value: 0.1},
        Duration: descriptor.FixedScaler{Value: 2.0},
    }
    ctx := descriptor.EffectCtx{Strength: 100, TowerDamage: 50}
    r := e.Apply(ctx)
    if r.Type != descriptor.EffTypeDot {
        t.Fatalf("type=%v", r.Type)
    }
    if r.DotSubtype != "burn" {
        t.Errorf("subtype=%v", r.DotSubtype)
    }
    // ratio 0.1 * 50 = 5 DPS
    if r.DotValue != 5 {
        t.Errorf("dot value got %v, want 5", r.DotValue)
    }
}

func TestGoldEffect(t *testing.T) {
    e := descriptor.GoldEffect{Amount: descriptor.LinearScaler{Base: 1, Potential: 1}}
    r := e.Apply(descriptor.EffectCtx{Strength: 200})
    if r.Type != descriptor.EffTypeGold || r.GoldAmount != 3 { // 1 + 1*(200/100)
        t.Errorf("gold: type=%v amount=%v", r.Type, r.GoldAmount)
    }
}

func TestModifyStatEffect(t *testing.T) {
    e := descriptor.ModifyStatEffect{Stat: "damage", Multiplier: 1.8}
    r := e.Apply(descriptor.EffectCtx{})
    if r.Type != descriptor.EffTypeModifyStat || r.BuffStat != "damage" || r.BuffBonus != 1.8 {
        t.Errorf("modifyStat: %+v", r)
    }
}
```

**Step 2: Run test, verify FAIL**

**Step 3: Implement effect_result.go + effect.go**

`effect_result.go` 定义 `EffectResult` 结构和类型常量。
`effect.go` 实现 11 种 Effect（damage/slow/stun/root/dot/weaken/silence/buff/selfBuff/gold/modifyStat）。

每种 Effect 是一个 struct，实现 `Apply(EffectCtx) EffectResult` 方法。
`EffectCtx` 包含 `Strength`, `TowerDamage`, `TargetMaxHp` 等运行时信息。

**Step 4: Run test, verify PASS**

```bash
go test ./tests/core/ -run "Test(Damage|Slow|Stun|Dot|Gold|ModifyStat)Effect" -v -race
```

**Step 5: Commit**

```bash
git add internal/core/tower/descriptor/effect.go internal/core/tower/descriptor/effect_result.go tests/core/descriptor_effect_test.go
git commit -m "feat: add Effect primitives (damage/slow/stun/root/dot/weaken/silence/buff/gold/modifyStat)"
```

---

## Task 4: Trigger + Selector primitives

触发器和目标选择器。Selector 依赖 enemy.Pool / tower.Pool 接口。

**Files:**
- Create: `internal/core/tower/descriptor/trigger.go`
- Create: `internal/core/tower/descriptor/selector.go`
- Test: `tests/core/descriptor_trigger_test.go`
- Test: `tests/core/descriptor_selector_test.go`

**Step 1: Write failing tests**

测试 trigger 类型匹配（onHit/onTick/onKill/onPlace）和 selector 目标选择（currentTarget 返回单个目标、aoeRadius 返回范围内多个、selfTower 返回自身等）。

Selector 测试使用 mock enemy/tower 列表，验证 `Select()` 返回正确数量和 ID 的目标。

**Step 2: Run test, verify FAIL**

**Step 3: Implement**

trigger.go：4 种 Trigger 实现，核心是 `Type() TriggerType` 匹配。

selector.go：6 种 Selector 实现。关键接口：

```go
type Selector interface {
    Select(ctx SelectorCtx) []Target
}

type Target struct {
    Enemy *enemy.Enemy // 敌人目标（CC/伤害类效果）
    Tower *tower.Tower // 友方塔目标（buff 类效果）
}

type SelectorCtx struct {
    CurrentEnemy *enemy.Enemy
    HitX, HitY   float64
    Tower        *tower.Tower
    Enemies      EnemyQuerier   // 接口：QueryRadius(x,y,r) []Enemy
    Towers       TowerQuerier   // 接口：QueryRadius(x,y,r) []Tower
    Strength     float64
}
```

用接口（`EnemyQuerier`/`TowerQuerier`）而非直接依赖 Pool，方便测试 mock。

**Step 4: Run test, verify PASS**

**Step 5: Commit**

```bash
git commit -m "feat: add Trigger (onHit/onTick/onKill/onPlace) and Selector (currentTarget/aoeRadius/chain/allInRange/nearbyAllies/selfTower) primitives"
```

---

## Task 5: AbilityDescriptor JSON schema + parser

描述符结构定义和 JSON 反序列化。

**Files:**
- Create: `internal/core/tower/descriptor/descriptor.go`
- Create: `config/towers/ability-descriptors.json` (先写 3 个代表性能力验证 parser)
- Test: `tests/core/descriptor_parse_test.go`

**Step 1: Write failing tests**

```go
func TestParseDescriptor_StunChance(t *testing.T) {
    raw := `{
        "id": "stunChance",
        "label": "震慑",
        "cost": 8,
        "tags": ["cc"],
        "pipelines": [{
            "trigger": "onHit",
            "conditions": [{"type":"chance","rate":{"scaler":"linear","base":0.10,"potential":0.05}}],
            "selector": {"type":"currentTarget"},
            "effects": [{"type":"stun","duration":{"scaler":"fixed","value":0.5}}]
        }]
    }`
    d, err := descriptor.ParseDescriptor([]byte(raw))
    if err != nil {
        t.Fatal(err)
    }
    if d.ID != "stunChance" {
        t.Errorf("id=%v", d.ID)
    }
    if len(d.Pipelines) != 1 {
        t.Fatalf("pipelines=%d", len(d.Pipelines))
    }
    if d.Cost != 8 {
        t.Errorf("cost=%d", d.Cost)
    }
}

func TestParseDescriptor_Splash(t *testing.T) {
    // 验证 attackStyle 字段和 aoeRadius selector
    raw := `{
        "id": "splash",
        "label": "溅射",
        "cost": 10,
        "tags": ["attack","aoe"],
        "attackStyle": "projectile",
        "spriteKey": "mortar",
        "pipelines": [{
            "trigger": "onHit",
            "conditions": [],
            "selector": {"type":"aoeRadius","radius":{"scaler":"fixed","value":50}},
            "effects": [{"type":"damage","mode":"ratio","value":{"scaler":"linear","base":0.75,"potential":0.05}}]
        }]
    }`
    d, err := descriptor.ParseDescriptor([]byte(raw))
    if err != nil {
        t.Fatal(err)
    }
    if d.AttackStyle != "projectile" {
        t.Errorf("attackStyle=%v", d.AttackStyle)
    }
    if d.SpriteKey != "mortar" {
        t.Errorf("spriteKey=%v", d.SpriteKey)
    }
}

func TestParseDescriptor_MultiPipeline(t *testing.T) {
    // 多管线描述符
    raw := `{
        "id": "user_frostfire",
        "label": "冰火连击",
        "cost": 15,
        "pipelines": [
            {"trigger":"onHit","conditions":[{"type":"chance","rate":{"scaler":"fixed","value":0.4}}],"selector":{"type":"currentTarget"},"effects":[{"type":"slow","factor":{"scaler":"fixed","value":0.3},"duration":{"scaler":"fixed","value":1.5}}]},
            {"trigger":"onHit","conditions":[],"selector":{"type":"currentTarget"},"effects":[{"type":"dot","subtype":"burn","mode":"ratio","value":{"scaler":"fixed","value":0.08},"duration":{"scaler":"fixed","value":2.0}}]}
        ]
    }`
    d, err := descriptor.ParseDescriptor([]byte(raw))
    if err != nil {
        t.Fatal(err)
    }
    if len(d.Pipelines) != 2 {
        t.Fatalf("pipelines=%d, want 2", len(d.Pipelines))
    }
}
```

**Step 2: Run test, verify FAIL**

**Step 3: Implement descriptor.go**

```go
// AbilityDescriptor 一个能力的完整描述。
type AbilityDescriptor struct {
    ID          string     `json:"id"`
    Label       string     `json:"label"`
    Icon        string     `json:"icon"`
    Cost        int        `json:"cost"`
    Tags        []string   `json:"tags"`
    AttackStyle string     `json:"attackStyle,omitempty"`
    SpriteKey   string     `json:"spriteKey,omitempty"`
    AttackParams json.RawMessage `json:"attackParams,omitempty"`
    Pipelines   []PipelineDesc  `json:"pipelines"`
}

type PipelineDesc struct {
    Trigger    string              `json:"trigger"`
    Conditions []json.RawMessage   `json:"conditions"`
    Selector   json.RawMessage     `json:"selector"`
    Effects    []json.RawMessage   `json:"effects"`
}
```

`ParseDescriptor()` 做二阶段解析：先 unmarshal 结构，再按 type 字段分派解析各原语为具体 struct。

**Step 4: Run test, verify PASS**

**Step 5: Commit**

```bash
git commit -m "feat: add AbilityDescriptor JSON schema and parser"
```

---

## Task 6: Interpreter (precompile + execute)

核心运行时。将描述符预编译为可执行管线，替代 ConfigAbility 的 switch dispatch。

**Files:**
- Create: `internal/core/tower/descriptor/interpreter.go`
- Test: `tests/core/descriptor_interpreter_test.go`

**Step 1: Write failing tests**

测试场景：
1. `stunChance` 描述符 + strength=100 + chance=1.0(测试用) → 应产生 StunEffect
2. `splash` 描述符 + 3 个范围内敌人 → 应产生 3 个 DamageEffect
3. `goldPassive` 描述符 (onTick + cooldown) → 第一次 tick 产金，3 秒内再 tick 不产金
4. `soloBoost` 描述符 (onTick + noNearbyTower) → 无邻居时产生 selfBuff，有邻居时不产生
5. 多管线描述符 → 所有管线的效果都被收集

**Step 2: Run test, verify FAIL**

**Step 3: Implement interpreter.go**

```go
type Interpreter struct {
    pipelines []compiledPipeline
}

type compiledPipeline struct {
    trigger    TriggerType
    conditions []Condition
    selector   Selector
    effects    []Effect
}

// Compile 从描述符预编译为运行时管线。
func Compile(desc *AbilityDescriptor) (*Interpreter, error) { ... }

// ExecOnHit 命中时调用，过滤 trigger=onHit 的管线。
func (interp *Interpreter) ExecOnHit(ctx TriggerContext) []EffectResult { ... }

// ExecOnTick 每帧调用，过滤 trigger=onTick 的管线。
func (interp *Interpreter) ExecOnTick(ctx TriggerContext) []EffectResult { ... }
```

`TriggerContext` 统一上下文，包含 Tower/Enemy/Projectile/DT/Elapsed 等运行时数据。
内部转换为 ConditionCtx/SelectorCtx/EffectCtx 传给各原语。

**Step 4: Run test, verify PASS**

**Step 5: Commit**

```bash
git commit -m "feat: add Interpreter with precompile + execute for descriptor engine"
```

---

## Task 7: Migrate existing 32 abilities to descriptors

将 `config/towers/abilities.json` 的 32 个 AbilityDef 机械转换为 `ability-descriptors.json`。

**Files:**
- Create: `internal/core/tower/descriptor/migrate.go`
- Create: `config/towers/ability-descriptors.json`
- Test: `tests/core/descriptor_migrate_test.go`
- Test: `tests/contracts/descriptor_contracts_test.go`

**Step 1: Write failing contract tests**

```go
// tests/contracts/descriptor_contracts_test.go
func TestAllAbilitiesHaveDescriptors(t *testing.T) {
    // 每个 abilities.json 中的能力都必须有对应的描述符
    table := config.GlobalAbilityTable()
    descriptors := descriptor.GlobalDescriptorTable()
    for name := range table {
        if _, ok := descriptors[name]; !ok {
            t.Errorf("ability %q has no descriptor", name)
        }
    }
}

func TestDescriptorCountMatchesAbilities(t *testing.T) {
    table := config.GlobalAbilityTable()
    descriptors := descriptor.GlobalDescriptorTable()
    if len(descriptors) != len(table) {
        t.Errorf("descriptors=%d, abilities=%d", len(descriptors), len(table))
    }
}
```

**Step 2: Run test, verify FAIL**

**Step 3: Implement**

`migrate.go`：`MigrateAbilityDef(def *config.AbilityDef) *AbilityDescriptor` 函数，按 category + type 映射为描述符。每种能力的管线模式在设计文档 Section 2.5 的映射表中已完整列出。

`ability-descriptors.json`：32 个描述符的完整 JSON 文件。由 migrate 函数输出验证后手工固化为配置文件。

**Step 4: Run test, verify PASS**

同时运行现有契约测试确保不破坏：

```bash
go test ./tests/contracts/ -v -race
go test ./tests/core/ -v -race
```

**Step 5: Commit**

```bash
git commit -m "feat: migrate all 32 abilities to descriptor format"
```

---

## Task 8: EffectResult → HitResult/TickResult 适配层

描述符引擎输出 EffectResult，现有战斗管线消费 HitResult/TickResult。写适配层桥接。

**Files:**
- Create: `internal/core/tower/descriptor/adapter.go`
- Test: `tests/core/descriptor_adapter_test.go`

**Step 1: Write failing tests**

```go
func TestAdaptToHitResult(t *testing.T) {
    // 一组 EffectResult → 合并为一个 HitResult
    results := []descriptor.EffectResult{
        {Type: descriptor.EffTypeDamage, Damage: 10, DamageMode: descriptor.DmgFlat},
        {Type: descriptor.EffTypeSlow, SlowFactor: 0.3, Duration: 1.0},
        {Type: descriptor.EffTypeStun, StunDur: 0.5},
    }
    hr := descriptor.AdaptToHitResult(results)
    if hr.SeparateDamage != 10 {
        t.Errorf("separateDmg=%v", hr.SeparateDamage)
    }
    if hr.Slow == nil || hr.Slow.Factor != 0.3 {
        t.Errorf("slow=%+v", hr.Slow)
    }
    if hr.Stun == nil || hr.Stun.Duration != 0.5 {
        t.Errorf("stun=%+v", hr.Stun)
    }
}

func TestAdaptToTickResult(t *testing.T) {
    results := []descriptor.EffectResult{
        {Type: descriptor.EffTypeGold, GoldAmount: 3},
    }
    tr := descriptor.AdaptToTickResult(results)
    if tr.GoldEarned != 3 {
        t.Errorf("gold=%d", tr.GoldEarned)
    }
}
```

**Step 2: Run test, verify FAIL**

**Step 3: Implement adapter.go**

`AdaptToHitResult([]EffectResult) *tower.HitResult` — 遍历 EffectResult，填充 HitResult 各字段。
`AdaptToTickResult([]EffectResult) *tower.TickResult` — 同理。

映射规则：
- `EffTypeDamage` + `DmgFlat` → `hr.SeparateDamage`
- `EffTypeDamage` + `DmgRatio` → `hr.BonusDamage`（按比例由调用方计算）
- `EffTypeSlow` → `hr.Slow`
- `EffTypeStun` → `hr.Stun`
- `EffTypeDot` + subtype=bleed → `hr.Bleed`
- `EffTypeDot` + subtype=burn → `hr.Burn`
- `EffTypeGold` → `tr.GoldEarned`

**Step 4: Run test, verify PASS**

**Step 5: Commit**

```bash
git commit -m "feat: add EffectResult → HitResult/TickResult adapter layer"
```

---

## Task 9: DescriptorAbility — 实现 Ability/Ticker 接口的包装器

用 Interpreter 实现现有 `tower.Ability` 和 `tower.Ticker` 接口，可以直接注册到 `tower.Registry`。

**Files:**
- Create: `internal/core/tower/descriptor/descriptor_ability.go`
- Test: `tests/core/descriptor_ability_test.go`

**Step 1: Write failing tests**

```go
func TestDescriptorAbility_ImplementsAbility(t *testing.T) {
    // 加载 stunChance 描述符，构造 DescriptorAbility，调用 OnHit
    desc := loadTestDescriptor(t, "stunChance")
    da := descriptor.NewDescriptorAbility(desc)

    // 验证接口
    var _ tower.Ability = da
    if da.Name() != "stunChance" {
        t.Errorf("name=%v", da.Name())
    }
}

func TestDescriptorAbility_ImplementsTicker(t *testing.T) {
    // goldPassive 是 onTick 类能力
    desc := loadTestDescriptor(t, "goldPassive")
    da := descriptor.NewDescriptorAbility(desc)

    _, ok := da.(tower.Ticker)
    if !ok {
        t.Fatal("goldPassive should implement Ticker")
    }
}

func TestDescriptorAbility_OnHitNotTicker(t *testing.T) {
    // stunChance 只有 onHit 管线，不应实现 Ticker
    desc := loadTestDescriptor(t, "stunChance")
    da := descriptor.NewDescriptorAbility(desc)

    _, ok := da.(tower.Ticker)
    if ok {
        t.Fatal("stunChance should NOT implement Ticker")
    }
}
```

**Step 2: Run test, verify FAIL**

**Step 3: Implement**

```go
// DescriptorAbility 描述符驱动的能力，实现 tower.Ability（+ 可选 tower.Ticker）。
// 作为从描述符引擎到现有注册表系统的桥接。
type DescriptorAbility struct {
    desc *AbilityDescriptor
    interp *Interpreter
    hasTick bool
}

func (da *DescriptorAbility) Name() string { return da.desc.ID }

func (da *DescriptorAbility) OnHit(t *tower.Tower, p *projectile.Projectile, e *enemy.Enemy) *tower.HitResult {
    ctx := buildHitContext(t, p, e)
    results := da.interp.ExecOnHit(ctx)
    if len(results) == 0 { return nil }
    return AdaptToHitResult(results)
}

// 仅当描述符包含 onTick 管线时，动态实现 Ticker 接口。
// 用两个 struct 实现（有/无 Ticker），NewDescriptorAbility 根据描述符选择。
```

技巧：Go 不支持运行时决定是否实现接口，用两个 struct 解决：
- `descriptorAbilityHit` — 只实现 Ability
- `descriptorAbilityFull` — 实现 Ability + Ticker

`NewDescriptorAbility()` 根据描述符是否含 onTick 管线返回不同的 struct。

**Step 4: Run test, verify PASS**

**Step 5: Commit**

```bash
git commit -m "feat: add DescriptorAbility wrapper implementing tower.Ability/Ticker interfaces"
```

---

## Task 10: 双轨注册 — 描述符与 ConfigAbility 并行

修改初始化流程，同时注册两套能力实现。描述符优先，ConfigAbility 做 fallback。

**Files:**
- Modify: `internal/core/tower/abilities/config_ability.go` (InitConfigAbilities)
- Create: `internal/core/tower/descriptor/init.go`
- Modify: `internal/core/tower/ability.go` (Registry lookup 逻辑)
- Test: `tests/contracts/descriptor_contracts_test.go` (补充双轨验证)

**Step 1: Write failing tests**

```go
func TestDualRegistration(t *testing.T) {
    // 初始化后，每个能力在 Registry 中都应该是 DescriptorAbility 类型
    for name := range config.GlobalAbilityTable() {
        ab, ok := tower.Lookup(name)
        if !ok {
            t.Errorf("%q not in registry", name)
            continue
        }
        if _, ok := ab.(*descriptor.DescriptorAbilityHit); !ok {
            if _, ok := ab.(*descriptor.DescriptorAbilityFull); !ok {
                t.Errorf("%q is %T, want DescriptorAbility", name, ab)
            }
        }
    }
}
```

**Step 2: Run test, verify FAIL** (目前 Registry 中是 ConfigAbility)

**Step 3: Implement**

`descriptor/init.go`：
```go
func InitDescriptorAbilities() error {
    // 1. 加载 ability-descriptors.json
    // 2. ParseDescriptor 每个
    // 3. NewDescriptorAbility 每个
    // 4. tower.Register 覆盖 ConfigAbility 的注册
}
```

调用顺序（在 loading.go / game.go 中）：
1. `abilities.InitConfigAbilities()` — 先注册 ConfigAbility（兼容）
2. `descriptor.InitDescriptorAbilities()` — 再注册 DescriptorAbility（覆盖）

覆盖后，Registry 中每个 key 指向 DescriptorAbility。如果某个描述符解析失败，保留 ConfigAbility fallback 并 log warning。

**Step 4: Run test, verify PASS**

**Step 5: Commit**

```bash
git commit -m "feat: dual-track registration — descriptors override ConfigAbility in Registry"
```

---

## Task 11: Autoplay 全量回归验证

**不写新代码**，只运行 autoplay 验证描述符引擎行为等价。

**Step 1: 运行 autoplay 全量回归**

```bash
go run cmd/autoplay/main.go --sweep --json-dir docs/autotest/descriptor-M1 --png-dir docs/autotest/descriptor-M2
```

**Step 2: 对比结果**

```bash
# 对比 coverage_summary.json 中的 anomaly 计数
diff <(jq '.anomalies' docs/autotest/M1/coverage_summary.json) <(jq '.anomalies' docs/autotest/descriptor-M1/coverage_summary.json)
```

预期：零差异（anomaly 数量和类型完全一致）。

**Step 3: 如有差异，定位并修复**

对比具体场景 JSON，找出哪个能力的行为发生了变化。常见问题：
- Scaler 精度差异（float64 舍入）
- Condition 判断顺序与原 switch 不一致
- 攻击模式类能力的 attackParams 未正确传递

**Step 4: Commit 验证结果**

```bash
git add docs/autotest/descriptor-M1/
git commit -m "test: autoplay full sweep — descriptor engine behavioral equivalence verified"
```

---

## Task 12: Budget system + Blueprint struct

预算系统和蓝图结构。纯数据+校验，不涉及 UI。

**Files:**
- Create: `internal/core/tower/descriptor/budget.go`
- Create: `internal/core/tower/descriptor/blueprint.go`
- Create: `config/towers/budget-rules.json`
- Test: `tests/core/descriptor_budget_test.go`
- Test: `tests/core/descriptor_blueprint_test.go`

**Step 1: Write failing tests**

```go
func TestBudgetCalculation(t *testing.T) {
    bp := descriptor.TowerBlueprint{
        AttackStyle: "scatter",                              // cost 12
        Tiers:       map[string]string{"damage":"S","atkSpeed":"B","range":"D"}, // 4+2+0=6
        Specialty:   "damage",                               // cost 2
        Abilities:   []string{"stunChance", "burn"},         // 8+7=15
    }
    budget := descriptor.CalcBudget(bp)
    if budget.Used != 35 { // 12+6+2+15
        t.Errorf("used=%d, want 35", budget.Used)
    }
    if budget.Used > 50 { // default cap
        t.Error("over budget")
    }
}

func TestBudgetOverflow(t *testing.T) {
    bp := descriptor.TowerBlueprint{
        AttackStyle: "wideBeam",                             // 14
        Tiers:       map[string]string{"damage":"S","atkSpeed":"S","range":"S"}, // 12
        Specialty:   "damage",                               // 2
        Abilities:   []string{"executionBonus","poisonZone","silenceZone"}, // 12+12+12=36
    }
    errs := descriptor.ValidateBlueprint(bp)
    if len(errs) == 0 {
        t.Fatal("should fail budget check")
    }
}

func TestBuildCostFormula(t *testing.T) {
    // buildCost = 40 + usedBudget * 0.5
    bp := descriptor.TowerBlueprint{Tiers: map[string]string{"damage":"B","atkSpeed":"B","range":"B"}} // 6
    cost := descriptor.CalcBuildCost(bp)
    if cost != 43 { // 40 + 6*0.5 = 43
        t.Errorf("buildCost=%d, want 43", cost)
    }
}
```

**Step 2: Run test, verify FAIL**

**Step 3: Implement**

budget.go：`CalcBudget(bp) BudgetResult`、`CalcBuildCost(bp) int`
blueprint.go：`TowerBlueprint` struct、`ValidateBlueprint(bp) []ValidationError`

budget-rules.json：从设计文档 Section 3.2 的配置直接写入。

**Step 4: Run test, verify PASS**

**Step 5: Commit**

```bash
git commit -m "feat: add budget system + TowerBlueprint struct with validation"
```

---

## Task 13: Blueprint persistence (CRUD)

蓝图持久化。复用现有 `persistence.Storage` 接口。

**Files:**
- Create: `internal/core/tower/descriptor/blueprint_store.go`
- Test: `tests/core/descriptor_blueprint_store_test.go`

**Step 1: Write failing tests**

```go
func TestBlueprintStore_CRUD(t *testing.T) {
    store := descriptor.NewBlueprintStore(persistence.NewMemoryStorage())

    bp := descriptor.TowerBlueprint{ID: "bp_001", Name: "测试塔"}

    // Create
    if err := store.Save(bp); err != nil {
        t.Fatal(err)
    }

    // Read
    got, err := store.Get("bp_001")
    if err != nil {
        t.Fatal(err)
    }
    if got.Name != "测试塔" {
        t.Errorf("name=%v", got.Name)
    }

    // List
    all := store.List()
    if len(all) != 1 {
        t.Fatalf("list=%d", len(all))
    }

    // Delete
    if err := store.Delete("bp_001"); err != nil {
        t.Fatal(err)
    }
    if len(store.List()) != 0 {
        t.Error("should be empty after delete")
    }
}

func TestBlueprintStore_MaxLimit(t *testing.T) {
    store := descriptor.NewBlueprintStore(persistence.NewMemoryStorage())
    for i := 0; i < 20; i++ {
        store.Save(descriptor.TowerBlueprint{ID: fmt.Sprintf("bp_%03d", i)})
    }
    err := store.Save(descriptor.TowerBlueprint{ID: "bp_overflow"})
    if err == nil {
        t.Fatal("should reject 21st blueprint")
    }
}
```

**Step 2: Run test, verify FAIL**

**Step 3: Implement**

```go
type BlueprintStore struct {
    storage persistence.Storage
    data    blueprintStoreData
}

type blueprintStoreData struct {
    Blueprints []TowerBlueprint `json:"blueprints"`
    Version    int              `json:"version"`
}

const maxBlueprints = 20
const blueprintsKey = "tower_blueprints"

func (s *BlueprintStore) Save(bp TowerBlueprint) error { ... }
func (s *BlueprintStore) Get(id string) (*TowerBlueprint, error) { ... }
func (s *BlueprintStore) Delete(id string) error { ... }
func (s *BlueprintStore) List() []TowerBlueprint { ... }
```

**Step 4: Run test, verify PASS**

**Step 5: Commit**

```bash
git commit -m "feat: add BlueprintStore with CRUD and 20-blueprint limit"
```

---

## Task 14: Blueprint → TowerDef conversion

让蓝图可以转换为 TowerDef，从而被 Pool.Place() 使用。

**Files:**
- Create: `internal/core/tower/descriptor/blueprint_to_def.go`
- Test: `tests/core/descriptor_blueprint_to_def_test.go`

**Step 1: Write failing tests**

验证：
1. Blueprint 的 tiers/specialty 正确映射到 TowerDef 的 CfgBase*/Potential* 字段
2. Blueprint 的 abilities 映射到 TowerDef 的 PresetAbilities
3. Blueprint 的 attackStyle 映射到 TowerDef 的 AttackStyleID + SpriteKeyOverride
4. FixedTiers=true, AbilityAcquireMode="preset"

**Step 2: Run test, verify FAIL**

**Step 3: Implement**

```go
func BlueprintToTowerDef(bp *TowerBlueprint, tierPresets *config.TierPresets) tower.TowerDef {
    // 复用 loadClassicTowerDefs 的逻辑（stage.go:3727-3785）
    // 但从 Blueprint 而非 ClassicPreset 读取参数
}
```

**Step 4: Run test, verify PASS**

**Step 5: Commit**

```bash
git commit -m "feat: add Blueprint → TowerDef conversion for Pool.Place() integration"
```

---

## Task 15: TowerRuleset 扩展 + loadTowerDefsForMode 整合

让 Campaign/Test 模式的建塔列表能加载蓝图塔。

**Files:**
- Modify: `internal/core/gamemode/tower_ruleset.go` (添加 AllowCustomBlueprints/CustomBudgetCap)
- Modify: `internal/core/gamemode/campaign_ruleset.go`
- Modify: `internal/core/gamemode/test_ruleset.go`
- Modify: `internal/core/gamemode/classic_ruleset.go`
- Modify: `internal/scene/stage.go` (loadTowerDefsForMode, ~line 3709)
- Test: `tests/core/gamemode_test.go` (补充新方法测试)

**Step 1: Write failing tests**

```go
func TestCampaignRuleset_AllowCustomBlueprints(t *testing.T) {
    r := gamemode.CampaignRuleset{}
    if !r.AllowCustomBlueprints() {
        t.Error("campaign should allow custom blueprints")
    }
}

func TestClassicRuleset_DisallowCustomBlueprints(t *testing.T) {
    r := gamemode.ClassicRuleset{}
    if r.AllowCustomBlueprints() {
        t.Error("classic should NOT allow custom blueprints")
    }
}
```

**Step 2: Run test, verify FAIL**

**Step 3: Implement**

接口添加两个方法 + baseTowerRuleset 默认实现（AllowCustomBlueprints=true, CustomBudgetCap=-1）。
ClassicRuleset override AllowCustomBlueprints=false。

`loadTowerDefsForMode` 增加：
```go
if ruleset.AllowCustomBlueprints() {
    bpDefs := loadBlueprintTowerDefs(blueprintStore, tierPresets)
    defs = append(defs, bpDefs...)
}
```

**Step 4: Run test, verify PASS**

同时运行 autoplay 快速验证（经典模式不受影响）：

```bash
go run cmd/autoplay/main.go --scenario attack-style-coverage
```

**Step 5: Commit**

```bash
git commit -m "feat: extend TowerRuleset with AllowCustomBlueprints, integrate blueprints into loadTowerDefsForMode"
```

---

## Task 16: Final autoplay sweep + cleanup

最终验证 + 清理。

**Step 1: 全量 autoplay 回归**

```bash
go run cmd/autoplay/main.go --sweep --json-dir docs/autotest/phase1-final --png-dir docs/autotest/phase1-final-png
```

**Step 2: make check-all**

```bash
make check-all
```

预期：lint 通过，所有测试通过。

**Step 3: 验证现有契约测试全部通过**

```bash
go test ./tests/contracts/ -v -race
go test ./tests/regression/ -v -race
```

**Step 4: 清理临时文件，最终 commit**

```bash
git add -A
git commit -m "chore: Phase 1 complete — descriptor engine, 32 ability migration, budget system, blueprint persistence"
```

---

## 依赖图

```
Task 1 (Scaler) ─────────┐
Task 2 (Condition) ───────┤
Task 3 (Effect) ──────────┼─→ Task 5 (Descriptor JSON) → Task 6 (Interpreter) → Task 7 (Migrate 32)
Task 4 (Trigger+Selector) ┘                                      ↓
                                                          Task 8 (Adapter) → Task 9 (DescriptorAbility)
                                                                                      ↓
                                                                              Task 10 (Dual Registration)
                                                                                      ↓
                                                                              Task 11 (Autoplay验证)
                                                                                      ↓
Task 12 (Budget) ─→ Task 13 (Persistence) ─→ Task 14 (Blueprint→Def) ─→ Task 15 (Ruleset整合)
                                                                                      ↓
                                                                              Task 16 (Final验证)
```

Tasks 1-4 可并行执行（无依赖）。Tasks 12-14 可与 Tasks 7-11 并行（独立子系统）。
