# Experience Polish P0 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement 9 pure-code experience polish features (zero external asset dependency).

**Architecture:** All features layer onto existing systems — TimeScale on `gameDT` in stage.go, Effects on postprocess pipeline, spawn animation on enemy pool, etc. No new packages; extend existing structs and functions.

**Tech Stack:** Go 1.24+, Ebitengine v2.9.9, Kage shaders

**Design doc:** `docs/plans/2026-04-09-experience-polish-design.md`

---

## Task 1: Time-Scale System

**Purpose:** Global slow-motion for boss kills, wave clears, clutch moments.

**Files:**
- Create: `internal/core/timescale/timescale.go`
- Modify: `internal/scene/stage.go:1656` (gameDT calculation)
- Modify: `internal/scene/stage.go:1659-1664` (VFX continues during slow-mo)
- Test: `tests/core/timescale_test.go`

**Design decisions:**
- Standalone package `timescale` — no scene/pipeline dependency, pure math.
- `Controller` struct holds: `scale float64`, `targetScale float64`, `easeInDur/holdDur/easeOutDur`, `elapsed float64`, `phase` (idle/easeIn/hold/easeOut).
- `Trigger(scale, easeIn, hold, easeOut float64)` starts a slow-mo sequence.
- `Update(dt float64) float64` returns current scale (1.0 when idle). Uses wall-clock dt (NOT gameDT) so the slow-mo itself doesn't slow down.
- stage.go line 1656 becomes: `gameDT := dt * float64(s.gameSpeed) * s.timeScale.Update(dt)`
- VFX (particles, float text, shake) receive `gameDT` as-is — they slow down with the game for cinematic feel.
- HitStop takes priority: if HitStop is active, TimeScale is ignored (existing early-return at line 1659).

**Integration in stage.go:**
- Add `timeScale *timescale.Controller` field to `StageScene` (near line 99).
- Init in `NewStageScene()`.
- Trigger points (added in Task 9 after kill feedback tiers are built).

**Test cases:**
- Idle state returns scale 1.0.
- Trigger → easeIn interpolates from 1.0 to target scale.
- Hold phase maintains target scale for specified duration.
- EaseOut interpolates back to 1.0.
- Multiple triggers: stronger (lower scale) overrides weaker.
- Update with wall-clock dt, not affected by own output.

---

## Task 2: SFX Pitch Variation

**Purpose:** Prevent auditory fatigue from repeated identical sounds.

**Files:**
- Modify: `internal/audio/manager.go:108-127` (PlayAt method)
- Create: `internal/audio/resample.go` (pitch-shift via resampling)
- Test: `tests/core/audio_pitch_test.go`

**Design decisions:**
- Ebitengine `audio.Player` has no pitch API. Must resample PCM data.
- For subtle variation (±8%), linear interpolation resampling is sufficient.
- `resamplePCM(pcm []byte, pitchFactor float64) []byte` — takes 16-bit stereo 44100Hz PCM, resamples to produce playback at altered pitch.
- pitchFactor > 1.0 = higher pitch (shorter buffer), < 1.0 = lower pitch (longer buffer).
- `PlayAt` gains optional pitch randomization: add `PitchVariation float64` field to Manager (default 0.08). Each call computes `pitch = 1.0 + (rand.Float64()*2-1)*PitchVariation`.
- BGM, UI SFX, achievement SFX bypass pitch variation (controlled by a `noPitch` set or by using `PlayExact` for fixed-pitch playback).
- Performance: resampled buffer is small (most SFX < 100KB), allocation is acceptable for throttled playback (~50ms intervals).

**Test cases:**
- `resamplePCM` with factor 1.0 returns identical buffer.
- Factor 2.0 produces buffer half the length.
- Factor 0.5 produces buffer double the length.
- Stereo sample pairs maintained (even byte boundaries).

---

## Task 3: Enemy Spawn Animation

**Purpose:** Enemies materialize with scale+alpha transition instead of popping in.

**Files:**
- Modify: `internal/core/enemy/enemy.go` (add SpawnTimer/SpawnDuration fields)
- Modify: `internal/core/enemy/pool.go:34` (Spawn sets SpawnTimer)
- Modify: `internal/render/draw_enemy.go:74-342` (render spawn state)
- Test: `tests/core/enemy_spawn_anim_test.go`

**Design decisions:**
- Add to Enemy struct: `SpawnTimer float64`, `SpawnDuration float64`.
- `Pool.Spawn()` sets `SpawnTimer = 0.3` (normal) or `0.5` (boss). `SpawnDuration = SpawnTimer`.
- `IsSpawning() bool` method: `return e.SpawnTimer > 0`.
- SpawnTimer decremented in `updatePlaying()` alongside DyingTimer (stage.go ~line 1740 area).
- During spawning: enemy is visible but **not targetable** — `targeting.go` skips `IsSpawning()` enemies (same pattern as `IsDying()`).
- Draw: compute `progress = 1 - SpawnTimer/SpawnDuration` (0→1). Scale = `1.2 - 0.2*easeOutBack(progress)` for overshoot feel. Alpha = `easeOutQuad(progress)`. Use same rendering path as active enemies but with modified scale/alpha.
- Spawn point particle burst: call `particle.EmitSpawnBurst(pool, x, y)` — new emitter preset, 6-8 white/cyan upward particles.

**Test cases:**
- New enemy has SpawnTimer > 0 after Spawn().
- IsSpawning() returns true during timer, false after.
- SpawnTimer decreases by dt each tick.
- Enemy not targetable while spawning.
- Progress calculation: 0 at start, 1 at end.

---

## Task 4: Boss Entrance Sequence

**Purpose:** 3-second tension buildup before boss wave enemies start spawning.

**Files:**
- Modify: `internal/core/enemy/spawner.go:207-213` (add pre-boss delay)
- Modify: `internal/scene/stage.go:401-411` (EvtWaveStarted handler)
- Modify: `internal/render/postprocess/effects.go` (add screen edge pulse)
- Test: `tests/core/boss_entrance_test.go`

**Design decisions:**
- In `Spawner.startWave()`: if `bossQueued`, set `SpawnTimer = 3.0` (new field `BossEntranceDelay float64`). The existing `SpawnTimer` field already gates spawning — enemies don't spawn until timer reaches 0.
  - Actually check spawner: `SpawnTimer` is the inter-enemy timer. Need a new `EntranceDelay float64` field. In `Update()`, if `EntranceDelay > 0`, decrement and skip spawning.
- On `EvtWaveStarted` with `IsBoss`: already plays `SFXBossEnter` and triggers wave announce. Add:
  - `render.TriggerShake(4.0, 0.5)` — medium shake.
  - `effects.TriggerEdgePulse(3.0)` — new: red border pulse that fades over 3s (reuse the `drawWarningFlash` pattern from wave_announce but as a continuous effect via postprocess).
  - Alternatively: reuse the existing `waveAnnounce.warningFlashes` system but extend to more flashes and longer duration. This is simpler — the wave announce already does 2 red flashes for boss. Extend to 4 flashes covering the 3s delay.
- **Chosen approach**: Extend wave_announce to handle the 3s boss entrance:
  - New phase `announceEntrance` before `announceSlideIn`. Duration: 2.5s. During this phase: screen edge red pulses (6 pulses @ 0.4s interval), increasing intensity. Then normal slide-in/hold/slide-out follows.
  - Spawner `EntranceDelay` synced to total announce duration (~4s) so no enemies spawn during buildup.

**Test cases:**
- Boss wave has EntranceDelay set to expected value.
- Non-boss wave has EntranceDelay = 0.
- EntranceDelay decrements each Update.
- No enemies spawn while EntranceDelay > 0.
- WaveAnnounce entrance phase duration correct.

---

## Task 5: Pause/Defeat Desaturation Shader

**Purpose:** Desaturate screen on pause, red-tint desaturation on defeat.

**Files:**
- Create: `internal/render/postprocess/shaders/desaturate.kage`
- Modify: `internal/render/postprocess/shaders.go` (embed + compile)
- Modify: `internal/render/postprocess/effects.go` (add DesatStrength field)
- Modify: `internal/render/postprocess/pipeline.go:140-341` (add desat pass)
- Modify: `internal/scene/stage.go` (set desat on pause/defeat state change)
- Test: `tests/core/desaturate_test.go`

**Design decisions:**
- New Kage shader `desaturate.kage`:
  ```
  var Strength float  // 0=normal, 1=full grayscale
  var TintR, TintG, TintB float  // optional color tint

  Fragment: luminance = dot(rgb, vec3(0.299,0.587,0.114))
            gray = vec3(luminance)
            mixed = mix(rgb, gray, Strength)
            mixed = mix(mixed, vec3(TintR,TintG,TintB), Strength*0.2)
  ```
- Effects struct adds: `DesatStrength float64`, `DesatTarget float64`, `DesatSpeed float64`, `DesatTintR/G/B float64`.
- `Effects.SetDesaturation(target, speed, r, g, b float64)` — animate toward target.
- `Effects.Update()` lerps `DesatStrength` toward `DesatTarget` at `DesatSpeed`.
- Pipeline adds pass after color_grade, before radial_blur. Guarded by `needDesat = fx.DesatStrength > 0.01`.
- stage.go: on enter `modePaused` → `SetDesaturation(0.7, 3.0, 0.5,0.5,0.6)`. On resume → `SetDesaturation(0.0, 4.0, ...)`. On `stateDefeat` → `SetDesaturation(0.8, 1.5, 0.8,0.2,0.2)`.

**Test cases:**
- NewEffects has DesatStrength = 0.
- SetDesaturation sets target correctly.
- Update lerps strength toward target.
- Strength clamps to [0,1].
- Pipeline includes desat pass when strength > 0.

---

## Task 6: Button Micro-Interactions

**Purpose:** Press/release/disabled feedback on all buttons.

**Files:**
- Modify: `internal/render/ui/widget.go:83-118` (Button → InteractiveButton)
- Modify: callers in `internal/scene/*.go` and `internal/render/hud/*.go` (adopt new API)
- Test: `tests/core/button_interaction_test.go`

**Design decisions:**
- Current `Button()` is a stateless draw function. Keep it for simple cases.
- Add `ButtonState` struct:
  ```go
  type ButtonState struct {
      ScaleT   float64 // animation timer for scale bounce
      Pressed  bool
      Disabled bool
  }
  ```
- Add `ButtonState.Update(dt float64, hovered, pressed bool)` — manages scale animation:
  - On press: `ScaleT = 1.0` (start bounce).
  - Each frame: `ScaleT` decays toward 0.
  - Scale = `1.0 - 0.07*ScaleT` when pressed, `1.0 + 0.03*ScaleT` on release bounce.
- Add `ButtonWithState(screen, x, y, w, h, label, style, state)` — applies scale transform and color modification before delegating to `Button()`.
  - Pressed: darken BgColor by 15%.
  - Disabled: reduce alpha to 40%, on tap trigger horizontal shake (`ShakeT` field).
- Adopt incrementally: start with ActionBar and PauseMenu buttons (highest visibility), then expand.
- Each scene/HUD component that already has hover logic keeps it — ButtonState adds press/disabled on top.

**Test cases:**
- Default ButtonState: ScaleT=0, Pressed=false, Disabled=false.
- Update with pressed=true sets Pressed, starts ScaleT.
- ScaleT decays toward 0 over frames.
- Disabled state prevents press animation, starts shake.
- Scale calculation produces correct values for pressed/released states.

---

## Task 7: Damage Number Filtering

**Purpose:** Reduce visual noise by suppressing minor damage numbers on lower quality.

**Files:**
- Modify: `internal/render/floattext.go:32-44` (SpawnDamageText)
- Modify: `internal/core/game/quality.go` (add DamageTextThreshold to presets)
- Modify: `internal/scene/stage.go` call sites (pass isBoss flag)
- Test: `tests/core/floattext_filter_test.go`

**Design decisions:**
- Extend `SpawnDamageText` signature: `SpawnDamageText(x, y, damage float64, crit, isBoss bool)`.
- Inside `SpawnDamageText`: check `game.CurrentQuality`:
  - `QualityHigh`: show all (no change).
  - `QualityMedium`: skip if `!crit && !isBoss && damage < maxHP*0.05`. Since maxHP isn't passed, use absolute threshold: skip if `damage < 3`.
  - `QualityLow`: only show if `crit || isBoss || damage >= 20`.
- Always show: `SpawnGoldText`, `SpawnKillText`, `SpawnText` (unfiltered).
- Update 4 call sites in stage.go to pass `e.Boss` as the new parameter.
- Add `DamageTextMinDmg float64` to `QualitySettings` struct for the threshold values.

**Test cases:**
- High quality: all damage numbers shown.
- Medium quality: damage < 3 suppressed.
- Low quality: only crit/boss/large damage shown.
- Boss damage always shown regardless of quality.
- Crit damage always shown regardless of quality.
- Gold/kill text unaffected.

---

## Task 8: Bloom Alternative (Additive Glow)

**Purpose:** Localized glow effects using additive blending instead of disabled bloom shader.

**Files:**
- Modify: `internal/render/draw/circle.go:26-40` (add GlowAdditive)
- Modify: `internal/render/draw_projectile.go` (switch key visuals to additive)
- Modify: `internal/render/particle/particle.go:189` (add additive blend mode option)
- Modify: `internal/render/draw_beam.go` (outer glow layer additive)
- Test: `tests/core/glow_test.go`

**Design decisions:**
- Add `GlowAdditive(screen, cx, cy, innerR, outerR, clr)` in `draw/circle.go`:
  - Draw outer circle using a temporary image + `ebiten.DrawImageOptions{Blend: ebiten.BlendLighter}`.
  - This creates real additive blending (color values add up, brighter = more glow).
  - Performance: temp image per-call is expensive. Better approach: batch all additive draws to a shared offscreen image, then composite once with `BlendLighter`.
- **Chosen approach**: Add a `GlowLayer` concept:
  - `draw.BeginGlowPass(screen)` — creates/clears an offscreen image the size of screen.
  - All `draw.Glow*` calls during glow pass draw to the offscreen image with normal blend.
  - `draw.EndGlowPass(screen)` — composites the offscreen image onto screen with `BlendLighter`.
  - This batches all glows into one additive composite — efficient.
- Update `draw_projectile.go`: wrap existing Glow calls in glow pass for charge/penetrate/sniper/default projectiles.
- Update `draw_beam.go`: outer glow layers (lines 45-52) go through glow pass.
- Particle system: add `Additive bool` to `Emitter` config. When true, render that emitter's particles with `BlendLighter` in a separate `DrawTriangles` call.
- Impact VFX: impact rings already use alpha blending — optionally switch to additive for brighter feel.

**Test cases:**
- GlowLayer can be created and cleared.
- BeginGlowPass/EndGlowPass don't panic.
- Particle emitter respects Additive flag (unit test with mock).
- Glow pass offscreen image size matches screen.

---

## Task 9: Kill Feedback Tiers

**Purpose:** Normal/multi-kill/overkill/boss kills feel distinct.

**Files:**
- Modify: `internal/scene/stage.go:1975-2009` (projectile kill handler)
- Modify: `internal/scene/stage.go:1780-1795` (warden kill handler)
- Modify: `internal/scene/stage.go:1982-2002` (multi-kill logic enhancement)
- Modify: `internal/render/particle/emitter.go` (add EmitBossDeathBurst)
- Test: `tests/core/kill_feedback_test.go`

**Design decisions:**
- **Normal kill** (current behavior, no change): `EmitDeathBurst` (8-16 particles) + `EmitGoldCollect` + `SFXEnemyDeath` throttled.
- **Multi-kill tier** (multiKillCount >= 3):
  - EmitDeathBurst with doubled particle count: add `EmitDeathBurstLarge(pool, x, y)` preset (16-24 particles, larger radius).
  - Multi-kill counter text at more thresholds: 3=white, 5=yellow, 10=orange, 20=red (extend existing x5/x10 logic).
  - Bonus gold at milestones (already in design, add later in P1).
- **Overkill** (damage > 2× enemy MaxHP):
  - Need to pass `overkillRatio` from apply_hit to the kill callback. Currently the callback receives `killed bool` — extend to `KillInfo{Killed bool, OverkillRatio float64}` or add `e.LastDamageRatio` to enemy.
  - Simpler: compute `overkillRatio = damage / e.MaxHP` at the kill callback site (damage and e.MaxHP are available in stage.go context).
  - If overkill > 2.0: `SpawnText` with gold color "OVERKILL" + EmitDeathBurstLarge + screen shake(2.0, 0.15).
- **Boss kill**:
  - Existing: HitStop(3) + SFXBossDeathBoss.
  - Add: `TriggerSlowMo(0.2, 0.1, 0.4, 0.3)` (from Task 1), `TriggerShake(5.0, 0.4)`, `TriggerRadialBlur(e.X, e.Y, 0.04, 0.6)`, `EmitBossDeathBurst(pool, e.X, e.Y)` (new preset: 30-40 particles, multi-color, larger radius), `TriggerRipple(e.X, e.Y, 20)`.
  - This combines slow-mo + shake + blur + ripple + massive particles — maximum drama.
- **Screen shake on all kills** (light):
  - Normal kill: no shake (too frequent).
  - Multi-kill x5+: `TriggerShake(1.5, 0.1)`.
  - Boss kill: `TriggerShake(5.0, 0.4)`.
  - Overkill: `TriggerShake(2.0, 0.15)`.

**Test cases:**
- Multi-kill threshold increments correctly.
- Multi-kill text spawns at thresholds 3, 5, 10, 20.
- Overkill detection: damage/MaxHP > 2.0 triggers overkill.
- Boss kill triggers slow-mo + shake + radial blur + ripple.
- Normal kill does not trigger shake.
- Kill feedback works in combination (boss + overkill + multi-kill).

---

## Implementation Order & Dependencies

```
Task 1: Time-Scale System          (independent, foundation for Task 9)
Task 2: SFX Pitch Variation        (independent)
Task 3: Enemy Spawn Animation      (independent)
Task 4: Boss Entrance Sequence     (independent)
Task 5: Desaturation Shader        (independent)
Task 6: Button Micro-Interactions  (independent)
Task 7: Damage Number Filtering    (independent)
Task 8: Bloom Alternative Glow     (independent)
Task 9: Kill Feedback Tiers        (depends on Task 1 for slow-mo)
```

**Recommended execution order:**
1. Task 1 (Time-Scale) — foundation, other tasks reference it
2. Task 5 (Desaturation) — cleanest postprocess change, validates shader pipeline
3. Task 3 (Spawn Animation) — isolated enemy system change
4. Task 4 (Boss Entrance) — builds on spawner understanding from Task 3
5. Task 2 (SFX Pitch) — isolated audio change
6. Task 7 (Damage Numbers) — small, isolated
7. Task 6 (Button Interactions) — UI layer, test visually
8. Task 8 (Additive Glow) — rendering layer, test visually
9. Task 9 (Kill Feedback) — integrates TimeScale + particles + shake + effects

**After each task:** `make test` to verify no regressions. After Tasks 3, 4, 8, 9: `make run` for visual verification.

---

## Verification Checklist

After all 9 tasks:
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] `go run cmd/autoplay/main.go --scenario attack-style-coverage` — no anomalies
- [ ] Visual check: run game, observe slow-mo on boss kill, spawn animations, pause gray-out
- [ ] Audio check: SFX sounds slightly different each play
- [ ] Performance: no FPS drop on High quality (F2 debug overlay)
