# BuffList Phase 2/3 Design

> Unified buff system for both enemies and towers.

Date: 2026-04-10
Status: Approved

---

## Phase 1 Recap (Done)

Migrated 14 enemy status effect fields (stun/slow/root/bleed/burn/poison/weaken/controlImmune) to `buff.BuffList`. Enemy `Buffs *buff.BuffList` initialized on Spawn. Query helpers: `IsStunned()/IsSlowed()/IsBleeding()` etc.

---

## Phase 2: Enemy Behavioral Buffs

### 2.1 Stealth Migration (Full)

**Current**: `Stealthed bool` + `StealthTimer float64` direct fields. Targeting skips stealthed enemies. Break on hit. Alpha=15% rendering.

**Target**: `buff.BuffList` with ID `"stealth"`, Category `CatBehavior`.

**Changes**:
- Remove `Stealthed` / `StealthTimer` from Enemy struct
- Add `IsStealthed() bool` helper → `e.Buffs.Has("stealth")`
- `pool.Spawn()`: add stealth buff with configured duration instead of setting fields
- `behaviors.go TickStealth()`: use `Buffs.Get("stealth")` for timer check, remove buff on expire
- Break on hit: `combat/apply_hit.go` calls `e.Buffs.Remove("stealth")` instead of `e.Stealthed = false`
- Rendering (`draw_enemy.go`): read `e.IsStealthed()` instead of `e.Stealthed`
- Targeting (`combat/targeting.go`): read `e.IsStealthed()`

**Call sites (~15)**: targeting.go, draw_enemy.go, behaviors.go, pool.go, stage.go (rendering/audio), apply_hit.go

### 2.2 Marker Buffs for Other Behaviors (Display Only)

Add non-functional marker buffs at Spawn time for HUD tooltip display:

| Behavior | Buff ID | Duration | Effect |
|----------|---------|----------|--------|
| berserk (when triggered) | `"berserk"` | Permanent (MaxDuration) | Display: "Berserk: +speed" |
| regen | `"regen"` | Permanent | Display: "Regenerating" |
| healer aura | `"healAura"` | Permanent | Display: "Healer: healing nearby" |
| buffer aura | `"speedAura"` | Permanent | Display: "Buffer: speed aura" |

These markers use `CatBehavior` and `ModeRefresh`. No logic changes. berserk marker added when `BerserkTriggered` flips to true (in `TickBerserk`).

---

## Phase 3: Tower Buff Unification

### 3a: Replace Container ([]TowerBuff → *buff.BuffList)

**Current**: `Tower.Buffs []TowerBuff` with custom `ApplyBuff/RemoveBuff/TickBuffs`.

**Target**: `Tower.Buffs *buff.BuffList` using the same `buff.BuffList` as enemies.

**Changes**:
- Remove `tower_buff.go` (91 lines)
- Add `Tower.Buffs *buff.BuffList` initialized in `pool.Place()`
- Migrate `ApplyBuff(key, source, desc, value, duration)` → `Buffs.Add(buff.Buff{ID: key, ...})`
- Migrate `RemoveBuff(key)` → `Buffs.Remove(key)`
- Migrate `TickBuffs(dt)` → `Buffs.Tick(dt)`
- Gold warden buff: `envoy.go` calls `ApplyBuff` → `Buffs.Add` with `CatBuff`
- HUD info_panel: read `Buffs.Active()` instead of iterating `[]TowerBuff`
- Add tower-specific stacking rules to `buff-stack.json` (strengthUp: ModeRefresh, aura types: ModeRefresh)

**Compatibility**: `TowerBuff.Key` → `Buff.ID`, `TowerBuff.Value` → `Buff.Value`, `TowerBuff.Remaining` → `Buff.Remaining`.

### 3b: Aura Abilities → Short-Duration Buffs (0.1s)

**Current**: Per-frame pattern in `tick_abilities.go`:
1. `resetTowerStats()` → zero `AttrMods`, `CritBonus`, `DamageAmp`
2. Aura `OnTick()` → write `t.AttrMods[dim] += bonus`, `t.CritBonus += x`
3. `RecalcStats()` reads `AttrMods` + `CritBonus` + `DamageAmp`

**Target**: Aura abilities add 0.1s buff each tick. RecalcStats aggregates from BuffList.

**Changes per aura ability** (damageUpAura/attackSpeedAura/rangeAura/critAura/soloBoost):
- `OnTick()`: instead of `t.AttrMods[dim] += bonus`, call `t.Buffs.Add(buff.Buff{ID: "aura:damageUp", Category: CatAura, Value: bonus, Duration: 0.1, Remaining: 0.1, Source: sourceID})`
- Stacking: `ModeAdditive` for same-ID auras (multiple towers stacking damage auras)
- `resetTowerStats()`: remove `AttrMods/CritBonus/DamageAmp` zeroing (BuffList Tick handles expiry)

**Buff IDs for auras**:
- `"aura:damage"` — flat damage add
- `"aura:attackSpeed"` — flat speed add
- `"aura:range"` — flat range add
- `"aura:crit"` — crit chance add
- `"aura:damageAmp"` — damage multiplier (soloBoost / gold warden)

### 3c: RecalcStats from BuffList

**Current**: `RecalcStats()` reads from `Strength.Ratio()` + `AttrMods` + `CritBonus` + `DamageAmp`.

**Target**: `RecalcStats()` aggregates all modifiers from `Buffs`:

```
damage  = (BaseDamage + PotentialDamage * ratio) + Buffs.SumByID("aura:damage")
speed   = (BaseSpeed  + PotentialSpeed  * ratio) + Buffs.SumByID("aura:attackSpeed")
range   = (BaseRange  + PotentialRange  * ratio) + Buffs.SumByID("aura:range")
crit    = BaseCrit + Buffs.SumByID("aura:crit")
dmgAmp  = Buffs.SumByID("aura:damageAmp")
```

**New BuffList method**: `SumByID(id string) float64` — sum all active buffs with matching ID.

**Eliminate**:
- `AttrMods [3]float64` field
- `CritBonus float64` field
- `DamageAmp float64` field
- `resetTowerStats()` zeroing of above

**Retain**:
- `Strength` system (base + permanent + temp - enemy drain) → affects `ratio`
- `AttrMods` concept replaced by BuffList queries

### Bonus: Dead Code Cleanup

Remove `TickResult.DamageBoost/SpeedBoost/RangeBoost` (never consumed by pipeline).

---

## Execution Order

1. Phase 2.1 — Stealth migration (~1 day)
2. Phase 2.2 — Marker buffs (~0.5 day)
3. Phase 3a — Container replacement (~1-2 days)
4. Phase 3b — Aura migration (~2-3 days)
5. Phase 3c — RecalcStats unification (~1-2 days)
6. Bonus — Dead code cleanup (~0.5 day)

Total: ~7-9 days

Each phase: write tests first → implement → `make test` → commit → next phase.

---

## Verification

After each phase:
- `make test` — all green
- `make run` — visual spot check (tower stats, aura indicators, stealth rendering)
- `go run cmd/autoplay/main.go --scenario attack-style-coverage` — no anomalies
