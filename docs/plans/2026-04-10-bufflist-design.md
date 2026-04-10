# BuffList System Design

> Date: 2026-04-10
> Status: Approved
> Goal: Migrate enemy/tower status effects from direct struct fields to a unified BuffList system with stacking rules and callback mechanisms.

## Motivation

1. **Extensibility**: Add new buff types (empBurst, timewarp, blink) via JSON config without changing Go code
2. **Unified stacking rules**: Make `buff-stack.json` the single source of truth, eliminate hardcoded stacking logic scattered across 6+ files
3. **HUD visibility**: Expose all active buffs on enemies/towers for UI display (icons, remaining time, source)

## Core Data Structure

### Buff

```go
// internal/core/buff/buff.go

type Category int
const (
    CatCC       Category = iota // stun, slow, root
    CatDoT                      // bleed, burn, poison
    CatDefense                  // damageReduce, invincible, damageImmune, controlImmune
    CatDebuff                   // weaken(damageAmplify), silence
    CatBehavior                 // berserk, regen, healAura, buffer, stealth, phase, purge
    CatAura                     // tower auras (damageUp, attackSpeed, range, crit)
)

type StackMode int
const (
    Strongest         StackMode = iota // take strongest value
    Additive                           // multiple sources sum
    Multiplicative                     // multiple sources multiply
    Override                           // latest overwrites
    Independent                        // each calculated independently
    IndependentPerSource               // same-source overwrites, different-source coexist
)

type Buff struct {
    ID        string    // matches buff-stack.json key: "slow", "stun", "bleed"
    Category  Category
    Source    string    // "tower_3_slow", "wave_berserk", "warden_envoy"
    Value     float64   // primary value (slow factor, DPS, threshold...)
    Value2    float64   // secondary value (few buffs need this)
    Duration  float64   // total duration (-1 = permanent)
    Remaining float64   // remaining time
    StackMode StackMode // loaded from buff-stack.json
    Priority  int       // for override mode
    Paused    bool      // pause countdown
    TickFunc  func(b *Buff, dt float64) // optional custom tick (behavior buffs)
}
```

### BuffList

```go
// internal/core/buff/list.go

type StackRule struct {
    Mode     StackMode
    Cap      float64 // 0 = no cap
    Floor    float64 // 0 = no floor
    Priority int
}

type BuffList struct {
    active []Buff
    rules  map[string]StackRule // loaded from buff-stack.json
}

// Core API
func (bl *BuffList) Add(b Buff) bool               // apply stacking rules, return success
func (bl *BuffList) Remove(id, source string)       // remove by ID+Source
func (bl *BuffList) RemoveByID(id string)           // remove all with ID
func (bl *BuffList) Tick(dt float64)                // countdown, cleanup expired
func (bl *BuffList) Has(id string) bool             // query existence
func (bl *BuffList) Get(id string) (Buff, bool)     // get effective value (merged by stack rules)
func (bl *BuffList) GetAll(id string) []Buff        // all same-ID buffs (independentPerSource)
func (bl *BuffList) Active() []Buff                 // HUD: snapshot of all active buffs
func (bl *BuffList) Clear()                         // purge: remove all
func (bl *BuffList) ClearByCategory(cats ...Category) // purge by category
```

Stacking dispatch in `Add()` by `rules[id].StackMode`:
- `Strongest`: same ID, keep higher Value
- `Override`: same ID, replace
- `Additive`: same ID coexist, `Get()` returns sum
- `IndependentPerSource`: same Source overwrites, different Source coexist

## Migration Strategy: 3 Phases

### Phase 1: CC + DoT (this phase)

Migrate the most regular, rule-driven fields first.

#### Fields to migrate

| Old Field | Buff ID | Category | Value Meaning | StackMode |
|-----------|---------|----------|---------------|-----------|
| StunTimer | `stun` | CC | (none, pure control) | override |
| SlowTimer + SlowFactor | `slow` | CC | Value=factor | strongest |
| RootTimer | `root` | CC | (none, pure control) | override |
| BleedTimer + BleedDPS | `bleed` | DoT | Value=DPS | independentPerSource |
| BurnTimer + BurnDPS | `burn` | DoT | Value=DPS | independentPerSource |
| PoisonTimer + PoisonDPS | `poison` | DoT | Value=DPS | independentPerSource |
| DamageAmplifyTimer + DamageAmplify | `weaken` | Debuff | Value=amplify ratio | strongest |
| ControlImmuneTimer + IsControlImmune | `controlImmune` | Defense | (none) | override |

#### Fields NOT migrated (stay as direct fields)

- Immunity flags (IsStunImmune/IsSlowImmune/IsRootImmune) — archetype properties, not buffs
- Tenacity — archetype property
- DotTickTimer/LastDotDmg/ZoneDmgAccum — pipeline internal counters
- Behavior fields (berserk/regen/healer/stealth/buffer) — Phase 2
- Silenced/AbilitySilenced — per-frame zone effects, Phase 2

#### Files to change

| File | Change |
|------|--------|
| **NEW** `internal/core/buff/` | buff.go, list.go, stack.go, rules.go |
| `enemy/enemy.go` | Add `Buffs buff.BuffList`, remove migrated fields |
| `combat/crowd_control.go` | ApplyStun/ApplySlow → `enemy.Buffs.Add()` |
| `combat/apply_hit.go` | bleed/burn/poison → `enemy.Buffs.Add()` |
| `combat/damage_pipeline.go` | Read `enemy.Buffs.Get("weaken")` instead of direct field |
| `enemy/enemy.go` TickStatusEffects | Remove manual countdown, use `Buffs.Tick(dt)` + DoT callback |
| `enemy/movement.go` | Read `Buffs.Has("stun")` / `Buffs.Get("slow")` |
| `enemy/behaviors.go` Purge | `Buffs.ClearByCategory(CatCC, CatDoT, CatDebuff)` |
| `pipeline/tick_abilities.go` | Remove per-frame DamageAmplify reset (BuffList self-manages) |
| `config/systems/buff-stack.json` | Load as runtime StackRule map |

#### DoT handling

- `BuffList.Tick(dt)` iterates DoT buffs, each has internal tick accumulator
- Every 0.5s (DotTickInterval) fires damage via callback `OnDotTick func(dmg float64)`
- Multiple same-type DoTs (independentPerSource) tick independently

### Phase 2: Behavior buffs (Enemy)

| Old Field Group | Buff ID | Notes |
|-----------------|---------|-------|
| Berserk(Threshold/SpeedScale/Triggered) | `berserk` | Permanent after trigger, Value=speedScale |
| RegenPerSec | `regen` | Permanent, Value=regenPerSec |
| HealPower/Radius/Interval/Cooldown | `healAura` | Permanent, Value=power, Value2=radius |
| SpeedBuff/BuffRadius/BuffAmount | `bufferAura` | Permanent, grants `speedUp` buff to nearby |
| Stealthed/StealthTimer | `stealth` | Timed, Value=alpha |
| PhaseDuration/PhaseActive | `phaseShift` | Cycling, Value=duration |
| Silenced/AbilitySilenced | `silence` | Per-frame zone |
| DamageReduceRatio | `damageReduce` | Permanent or timed |

Behavior buffs have active tick logic (healer periodic heal, buffer periodic speed). Solution: use `TickFunc` on Buff struct, `BuffList.Tick()` dispatches uniformly.

### Phase 3: Tower side

Unify Tower's `[]TowerBuff` + `AttrMods` + `StrengthData` into BuffList:
- Aura effects (damageUp/attackSpeed/range/crit) → buffs
- Warden buffs → buffs
- `Mods` and `CritBonus`/`DamageAmp` replaced by `BuffList.Get()` aggregation

Phase 3 is the largest but fully independent from Phase 1/2.

## HUD Visualization

`BuffList.Active()` returns read-only snapshots (pure value structs) for HUD ViewModel:

```go
// Used by HUD layer (zero core dependency)
type BuffInfo struct {
    ID        string
    Category  string   // "cc", "dot", "defense", "debuff", "behavior", "aura"
    Source    string
    Value     float64
    Duration  float64
    Remaining float64
    Stacks    int      // stack count for additive mode
}
```

Rendering: small icon row below enemy HP bar. Tower selected panel shows buff list (reuse existing TowerBuff display, unified after Phase 3).

## Config Integration

`buff-stack.json` becomes the runtime source of truth:

```json
{
  "slow": { "mode": "strongest", "cap": 0.8 },
  "stun": { "mode": "override" },
  "bleed": { "mode": "independentPerSource" },
  "dot": { "mode": "independentPerSource" }
}
```

Loaded at init via `buff.LoadRules(jsonBytes)` → `map[string]StackRule`.

New buff types can be added by:
1. Adding entry to `buff-stack.json`
2. Adding trigger point in ability/behavior code
3. No new Go types needed
