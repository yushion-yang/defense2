# BuffList Phase 2/3 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Migrate enemy stealth to BuffList, add behavioral markers, unify tower buffs under the same `buff.BuffList` system with short-duration aura pattern.

**Architecture:** Phase 2 replaces `Stealthed/StealthTimer` with `Buffs.Has("stealth")`. Phase 3 replaces `[]TowerBuff` + `AttrMods/CritBonus/DamageAmp` with `*buff.BuffList` + `SumByID()` aggregation, using 0.1s refresh aura buffs.

**Tech Stack:** Go 1.24+, buff package (`internal/core/buff/`), config `buff-stack.json`

**Design doc:** `docs/plans/2026-04-10-bufflist-phase2-3-design.md`

---

## Task 1: Add SumByID to BuffList + stealth stacking rule

**Purpose:** Infrastructure for Phase 3 RecalcStats, and stealth rule for Phase 2.

**Files:**
- Modify: `internal/core/buff/list.go` (add SumByID method)
- Modify: `config/systems/buff-stack.json` (add stealth + tower aura rules)
- Test: `tests/core/buff_test.go` (add SumByID tests)

**Design decisions:**
- `SumByID(id string) float64` iterates active buffs, sums Value for matching ID. Respects cap/floor from rules.
- Stacking rules added: `stealth` (override), `aura:damage/attackSpeed/range/crit/damageAmp` (additive), `towerStrength` (override).

**Test cases:**
- SumByID with no matching buffs returns 0.
- SumByID with one buff returns its Value.
- SumByID with multiple additive buffs returns sum.
- SumByID respects cap from rules.

---

## Task 2: Stealth migration — add IsStealthed() helper

**Purpose:** Replace `e.Stealthed` field reads with BuffList query.

**Files:**
- Modify: `internal/core/enemy/enemy.go:193-194` (remove fields, add helper)

**Implementation:**
- Remove `Stealthed bool` and `StealthTimer float64` from Enemy struct.
- Add method:
  ```go
  func (e *Enemy) IsStealthed() bool { return e.Buffs != nil && e.Buffs.Has("stealth") }
  ```
- Add method for tooltip timer display:
  ```go
  func (e *Enemy) StealthRemaining() float64 {
      if b, ok := e.Buffs.Get("stealth"); ok { return b.Remaining }
      return 0
  }
  ```

---

## Task 3: Stealth migration — update Spawn + tickStealth

**Files:**
- Modify: `internal/core/enemy/pool.go:100-103` (Spawn stealth init)
- Modify: `internal/core/enemy/behaviors.go:203-213` (tickStealth)

**pool.go change:**
```go
// Replace:
if cfg.StealthDuration > 0 {
    e.Stealthed = true
    e.StealthTimer = cfg.StealthDuration
}
// With:
if cfg.StealthDuration > 0 {
    e.Buffs.Add(buff.Buff{
        ID: "stealth", Category: buff.CatBehavior,
        Source: "archetype", Value: 1,
        Duration: cfg.StealthDuration, Remaining: cfg.StealthDuration,
    })
}
```

**behaviors.go change:** `tickStealth` reads from BuffList:
```go
func tickStealth(e *Enemy, dt float64, events *BehaviorEvents) {
    if !e.IsStealthed() {
        return
    }
    // Break on hit (HitFlash set by apply_hit)
    if e.HitFlash > 0 {
        e.Buffs.RemoveByID("stealth")
        events.Reveals = append(events.Reveals, RevealEvent{X: e.X, Y: e.Y})
        return
    }
    // Duration managed by BuffList.Tick() — check if just expired
    // (Tick is called before behaviors in pipeline, so if stealth was ticking
    //  and just expired, IsStealthed() would already be false and we'd return above)
}
```
Actually simpler: `BuffList.Tick(dt)` already decrements and removes expired buffs. `tickStealth` only needs to handle the break-on-hit case. The normal expiry is handled by Tick.

But we need the reveal event when stealth expires naturally. Two options:
- Check before+after: if was stealthed before Tick, and not after, emit reveal.
- Better: handle in `tickStealth` — check if buff just expired this frame (Remaining very small).

**Chosen approach:** Let `tickStealth` only handle break-on-hit. For natural expiry, check in the behaviors dispatch: if behavior=="stealth" and `!e.IsStealthed()`, emit reveal once.

Add a `StealthRevealed bool` one-shot flag to avoid repeat events, or simpler: once `tickStealth` sees `!IsStealthed()`, emit reveal and switch behavior to "" (no behavior).

---

## Task 4: Stealth migration — update all read sites

**Files:**
- Modify: `internal/core/tower/targeting.go:15,42,65` (3 sites)
- Modify: `internal/render/draw_enemy.go:153,173,208` (3 sites)
- Modify: `internal/scene/stage.go:1163,1199` (2 sites)

**All changes:** Replace `e.Stealthed` → `e.IsStealthed()`, `e.StealthTimer` → `e.StealthRemaining()`.

**After this task:** `go build ./...` must pass, `make test` must pass.

---

## Task 5: Marker buffs for berserk/regen/healer/buffer

**Purpose:** Display-only markers in BuffList for HUD tooltip consistency.

**Files:**
- Modify: `internal/core/enemy/behaviors.go` (add marker buffs)
- Modify: `internal/core/enemy/pool.go` (add regen/healer/buffer markers at spawn)
- Modify: `config/systems/buff-stack.json` (add marker rules)

**Implementation:**
- In `pool.Spawn()`, after behavior setup, add permanent markers:
  ```go
  if cfg.RegenPerSec > 0 {
      e.Buffs.Add(buff.Buff{ID: "regen", Category: buff.CatBehavior, Source: "archetype", Duration: -1, Remaining: -1})
  }
  ```
  Same for healAura, speedAura (buffer behavior).
- In `tickBerserk`, when `BerserkTriggered` flips to true:
  ```go
  e.Buffs.Add(buff.Buff{ID: "berserk", Category: buff.CatBehavior, Source: "archetype", Duration: -1, Remaining: -1})
  ```
- Rules in JSON: all `"override"` mode (single instance per enemy).

---

## Task 6: Phase 3a — Replace Tower.Buffs container

**Purpose:** Swap `[]TowerBuff` → `*buff.BuffList` on Tower.

**Files:**
- Modify: `internal/core/tower/tower.go:80` (change field type)
- Delete: `internal/core/tower/tower_buff.go` (91 lines)
- Modify: `internal/core/tower/pool.go:41-69` (init BuffList in initTower)
- Modify: `internal/core/pipeline/tick_abilities.go:45-47` (Tick call)
- Modify: `internal/core/warden/types/envoy.go:147,160-167` (ApplyBuff → Buffs.Add)
- Modify: `internal/core/tower/abilities/config_ability.go:349-371` (applyBuffDisplay/removeBuffDisplay)
- Modify: `internal/render/hud/info_panel.go` (read Buffs.Active())
- Modify: `internal/scene/stage.go` (BuildInfoPanelVM)
- Test: `tests/core/tower_buff_test.go` or existing tests

**Key concern:** `ApplyBuff` syncs to `Strength.SetTemp(key, value)`. The new flow:
- `Buffs.Add(buff.Buff{ID: key, Value: value, ...})` handles stacking
- A new helper `syncStrengthFromBuffs(t)` scans `t.Buffs.Active()` for `CatBuff` entries and calls `Strength.SetTemp` accordingly
- Or simpler: the envoy code calls both `Buffs.Add` and `Strength.SetTemp` explicitly (keep coupling explicit)

**Chosen:** Keep explicit dual-write for now (Buffs.Add + Strength.SetTemp). Phase 3c will clean this up when RecalcStats reads from BuffList.

---

## Task 7: Phase 3b — Migrate aura abilities to 0.1s buffs

**Purpose:** Replace per-frame `AttrMods/CritBonus/DamageAmp` writes with short-duration BuffList entries.

**Files:**
- Modify: `internal/core/tower/abilities/config_ability.go:206-274` (5 aura OnTick functions)
- Modify: `internal/core/pipeline/tick_abilities.go:89-94` (resetTowerStats)

**Per aura ability change pattern:**
```go
// Before (damageUpAura):
other.DamageAmp += sv
applyBuffDisplay(other, key, srcLabel, desc)
other.RecalcStats()

// After:
other.Buffs.Add(buff.Buff{
    ID: "aura:damageAmp", Category: buff.CatAura,
    Source: key, Value: sv,
    Duration: 0.15, Remaining: 0.15,
})
// RecalcStats will be called after all auras in the pipeline
```

**resetTowerStats change:**
```go
// Before:
t.Mods = tower.AttrMods{}
t.CritBonus = 0
t.DamageAmp = 0
t.RecalcStats()

// After:
// Aura buffs expire naturally via Tick(dt) called earlier in pipeline
// No need to zero anything — just RecalcStats
t.RecalcStats()
```

**Important:** `Buffs.Tick(dt)` must be called BEFORE aura OnTick in the pipeline, so expired aura buffs are cleaned up before new ones are added. Current order in `tick_abilities.go`: Phase 1.1 (TickBuffs) → Phase 2 (OnTick). This is correct.

---

## Task 8: Phase 3c — RecalcStats from BuffList

**Purpose:** Eliminate AttrMods/CritBonus/DamageAmp fields, read from BuffList.

**Files:**
- Modify: `internal/core/tower/tower.go:83-87,112-119,136-161` (remove fields, update RecalcStats)

**RecalcStats new implementation:**
```go
func (t *Tower) RecalcStats() {
    ratio := t.Strength.Ratio()
    bal := config.GlobalBalance()

    // Base + potential scaled by strength
    t.Damage = t.BaseDamage + t.PotentialDamage*ratio
    t.AttackSpeed = t.BaseSpeed + t.PotentialSpeed*ratio
    t.Range = t.BaseRange + t.PotentialRange*ratio

    // Aura modifiers from BuffList
    if t.Buffs != nil {
        t.Damage *= 1 + t.Buffs.SumByID("aura:pctDamage")
        t.Damage += t.Buffs.SumByID("aura:flatDamage")
        t.AttackSpeed *= 1 + t.Buffs.SumByID("aura:pctSpeed")
        t.AttackSpeed += t.Buffs.SumByID("aura:flatSpeed")
        t.Range += t.Buffs.SumByID("aura:flatRange")
        t.CritChance = t.Buffs.SumByID("aura:crit")
        t.DamageAmpFactor = 1 + t.Buffs.SumByID("aura:damageAmp")
    }

    // Floors
    if t.Damage < t.BaseDamage { t.Damage = t.BaseDamage }
    if t.AttackSpeed < bal.Tower.AttackSpeedFloor { t.AttackSpeed = bal.Tower.AttackSpeedFloor }
    if t.Range < t.BaseRange { t.Range = t.BaseRange }
}
```

**Remove:** `AttrMods` struct, `Mods AttrMods` field, `CritBonus float64`, `DamageAmp float64`.
**Add:** `CritChance float64`, `DamageAmpFactor float64` (computed from BuffList each RecalcStats call).

**Update callers** of `t.CritBonus` → `t.CritChance`, `t.DamageAmp` → `t.DamageAmpFactor` (grep all references).

---

## Task 9: Dead code cleanup

**Purpose:** Remove unused TickResult fields.

**Files:**
- Modify: `internal/core/tower/ability.go:77-82` (remove DamageBoost/SpeedBoost/RangeBoost)
- Modify: `internal/core/tower/abilities/scaling.go` (remove dead returns)
- Modify: `internal/core/pipeline/tick_abilities.go` (remove dead aggregation if any)

---

## Implementation Order & Dependencies

```
Task 1: SumByID + rules          (foundation, no deps)
Task 2: IsStealthed helper       (depends on Task 1 for stealth rule)
Task 3: Spawn + tickStealth      (depends on Task 2)
Task 4: Read site migration      (depends on Task 2)
Task 5: Marker buffs             (independent of Tasks 2-4)
Task 6: Tower.Buffs container    (independent of Phase 2)
Task 7: Aura → 0.1s buffs       (depends on Task 6)
Task 8: RecalcStats unification  (depends on Tasks 1, 6, 7)
Task 9: Dead code cleanup        (independent)
```

**Recommended batches:**
- Batch A: Tasks 1-4 (stealth migration) → commit
- Batch B: Task 5 (marker buffs) → commit
- Batch C: Tasks 6-8 (tower unification) → commit per task
- Batch D: Task 9 (cleanup) → commit

**After each task:** `make test` to verify no regressions.

---

## Verification Checklist

After all tasks:
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] `go run cmd/autoplay/main.go --scenario attack-style-coverage` — no anomalies
- [ ] Visual check: stealth enemies fade in/out correctly
- [ ] Visual check: tower aura indicators display correctly
- [ ] Visual check: tower info panel shows buffs correctly
- [ ] Visual check: envoy warden buff appears in tower info panel
- [ ] Confirm `e.Stealthed` and `e.StealthTimer` fields no longer exist
- [ ] Confirm `tower_buff.go` file deleted
- [ ] Confirm `AttrMods` struct removed from tower.go
