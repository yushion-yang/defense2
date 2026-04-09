# Experience Polish P1 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement 11 experience features that need minimal external assets or have asset-independent code scaffolding.

**Architecture:** Extends audio manager (crossfade, ducking), adds animation state to HUD components (following WaveAnnounce pattern), upgrades scene transition system, adds wave rhythm to spawner.

**Tech Stack:** Go 1.24+, Ebitengine v2.9.9, Kage shaders

**Depends on:** P0 tasks completed (especially Task 1 TimeScale, Task 5 Desaturation shader pipeline)

---

## Task 1: BGM Crossfade System

**Purpose:** Smooth transitions between menu/battle/boss music tracks.

**Files:**
- Modify: `internal/audio/manager.go:319-395` (PlayBGM/StopBGM/SetBGMVolume)
- Test: `tests/core/bgm_crossfade_test.go`

**Design:**
- Add fields to Manager: `prevBGMPlayer *audio.Player`, `fadeOutVol float64`, `fadeInVol float64`, `crossfadeTimer float64`, `crossfadeDuration float64` (default 1.0s).
- New `PlayBGMCrossfade(name string, duration float64)`: keeps old player as `prevBGMPlayer`, starts new player at volume 0, begins crossfade.
- New `UpdateBGM(dt float64)`: called from game loop each frame. Ramps `prevBGMPlayer` volume down and new `bgmPlayer` volume up linearly over `crossfadeDuration`. On completion, calls `prevBGMPlayer.Close()`, nils it.
- Modify existing `PlayBGM` to call `PlayBGMCrossfade` with default 1.0s duration.
- Between-wave volume dip: `SetBGMDucked(ducked bool)` — when ducked, target volume = bgmVolume * 0.7. Smooth transition in UpdateBGM.

**Integration in stage.go:**
- `EvtWaveStarted` handler (line 401): already calls `PlayBGM`. Replace with `PlayBGMCrossfade`.
- `EvtWaveCleared` handler (line 412): already calls `PlayBGM(BGMBattle)`. Keep as crossfade.
- Wave timer idle (spawner not active): call `audioMgr.SetBGMDucked(true)`. On wave start: `SetBGMDucked(false)`.
- Call `audioMgr.UpdateBGM(dt)` from `updatePlaying()` (alongside other per-frame updates ~line 2040).

**Asset note:** Requires 3 OGG/WAV files. Code works without them (PlayBGM silently returns on missing file).

**Test cases:**
- Crossfade timer decrements correctly.
- Old player volume reaches 0 at end of crossfade.
- New player volume reaches target at end of crossfade.
- Same track name skips crossfade.
- BGM ducked mode reduces volume to 70%.

---

## Task 2: Audio Ducking

**Purpose:** Lower background audio during boss entrance and major events.

**Files:**
- Modify: `internal/audio/manager.go` (add duck fields + UpdateDuck)
- Modify: `internal/scene/stage.go` (trigger duck on boss events)
- Test: `tests/core/audio_duck_test.go`

**Design:**
- Add to Manager: `duckFactor float64` (default 1.0), `duckTarget float64`, `duckSpeed float64`.
- `TriggerDuck(factor, duration float64)`: sets `duckTarget = factor`, `duckSpeed = (1.0 - factor) / duration`.
- `UpdateDuck(dt float64)`: lerps `duckFactor` toward `duckTarget` at `duckSpeed`. When target reached and target < 1.0, starts ramp-back to 1.0.
- Apply duck in `PlayAt`: `vol := m.volume * scale * m.duckFactor`.
- Apply duck to live BGM in `UpdateBGM`: `bgmPlayer.SetVolume(m.bgmVolume * m.duckFactor)`.

**Trigger points:**
- Boss wave `EvtWaveStarted` with `IsBoss`: `TriggerDuck(0.3, 0.5)` — 0.5s duck, auto-ramp-back.
- Boss kill: `TriggerDuck(0.4, 0.3)` — brief duck during slow-mo.

**Test cases:**
- Duck factor starts at 1.0.
- TriggerDuck sets target and speed.
- UpdateDuck interpolates toward target.
- Auto-ramp-back to 1.0 after target reached.
- PlayAt multiplies by duckFactor.

---

## Task 3: Ambient Environment Audio

**Purpose:** Per-map-theme background ambient sound loop.

**Files:**
- Modify: `internal/audio/manager.go` (add ambient player management)
- Modify: `internal/scene/stage.go` (start ambient on stage init)
- Create: `internal/audio/ambient.go` (ambient sound mapping)
- Test: `tests/core/ambient_audio_test.go`

**Design:**
- New `ambientPlayer *audio.Player` and `ambientName string` in Manager.
- `PlayAmbient(name string)`: loads WAV/OGG loop, plays at low volume (bgmVolume * 0.3).
- `StopAmbient()`: fades out and closes.
- Ambient mapping: `themeToAmbient map[string]string` — maps theme name to audio file name.
  - desert → "ambient-wind", forest → "ambient-forest", tech → "ambient-tech", etc.
- Ambient follows BGM duck factor for consistency.

**Integration:**
- stage.go `NewStageScene()`: after map load, look up theme, call `audioMgr.PlayAmbient(ambient)`.
- On scene exit: `StopAmbient()`.

**Asset note:** Requires 8 OGG loops. Code is safe without them.

**Test cases:**
- PlayAmbient stores name, prevents restart of same track.
- StopAmbient clears player.
- Volume follows BGM volume * 0.3.

---

## Task 4: Warden Entrance Animation

**Purpose:** Dramatic hero summon with light pillar and particles.

**Files:**
- Modify: `internal/core/warden/state.go` (add EntranceTimer/EntranceDuration)
- Modify: `internal/render/draw_warden.go` (entrance rendering)
- Modify: `internal/scene/stage.go:2954-3003` (activateWarden triggers entrance)
- Modify: `internal/render/particle/emitter.go` (add EmitWardenEntrance preset)
- Test: `tests/core/warden_entrance_test.go`

**Design:**
- Add to WardenState: `EntranceTimer float64`, `EntranceDuration float64` (1.5s).
- `IsEntering() bool`: `return EntranceTimer > 0`.
- On `activateWarden()`: set `EntranceTimer = 1.5`, position warden at a fixed entrance point (first tower slot center, or map center).
- During entrance: warden is visible but doesn't attack or move.
- WardenState tick: skip movement/combat if `IsEntering()`, only decrement timer.

**Rendering phases (1.5s total):**
- Phase 1 (0-0.5s): Vertical light beam at warden position — two tall thin triangles, alpha 0→1. No warden visible yet.
- Phase 2 (0.5-1.2s): Warden fades in (alpha 0→1, scale 0.5→1.0 with overshoot). Light beam peaks then fades. Type-themed particle burst.
- Phase 3 (1.2-1.5s): Everything settles. Trail starts recording.

**Particle presets per warden type:**
- Prince (fire): orange/red upward sparks.
- Core (mech): gray/blue mechanical debris.
- Chain: yellow electric sparks (reuse EmitElectricSparks with adjusted count).
- Skystrike (water): blue droplets outward.
- Envoy (gold): gold coin sparkles (reuse EmitGoldCollect pattern).

**Test cases:**
- EntranceTimer set on activation.
- IsEntering() returns true during timer.
- Movement/combat skipped during entrance.
- Timer decrements each tick.
- Normal behavior resumes after timer expires.

---

## Task 5: Scene Transition Upgrade

**Purpose:** Replace black fade with iris/diamond wipe.

**Files:**
- Modify: `internal/scene/game.go:24-203` (transition system)
- Create: `internal/render/transition/iris.go` (iris wipe renderer)
- Test: `tests/core/transition_test.go`

**Design:**
- New package `internal/render/transition` with `Transition` interface:
  ```go
  type Transition interface {
      Update(dt float64) bool // returns true when complete
      Draw(screen *ebiten.Image)
  }
  ```
- `IrisWipe` struct: circle that shrinks to center (fade-out) then expands from center (fade-in).
  - Radius interpolated from `maxR` (screen diagonal / 2) to 0 (fade-out) or 0 to maxR (fade-in).
  - Drawn by filling screen with black, then cutting out a circle using stencil-like technique: draw a filled circle of the scene content over the black.
  - Simpler: render scene to offscreen, then draw a circular mask. Ebitengine doesn't have stencil, so use a shader or draw the scene into a circular clip region via `DrawTriangles` with UV mapping.
  - Simplest practical approach: draw full-screen black rect, then draw a filled circle of screen content using `DrawRectShader` with a circle-clip shader.
- `DiamondWipe`: same principle but diamond shape. Add later as variant.
- game.go: replace `draw.FilledRect` overlay with `transition.Draw(screen)`.
- Low quality fallback: keep simple black fade (no shader).
- Easing: ease-in-out-cubic for smooth acceleration.

**Test cases:**
- IrisWipe starts with max radius.
- Radius decreases to 0 during fade-out.
- Radius increases from 0 during fade-in.
- Update returns true when transition complete.
- Easing produces correct intermediate values.

---

## Task 6: HUD Entry Animations

**Purpose:** TopBar slides down, ActionBar slides up, Minimap fades in on battle start.

**Files:**
- Modify: `internal/render/hud/top_bar.go` (add offset parameter)
- Modify: `internal/render/hud/action_bar.go` (add offset parameter)
- Modify: `internal/render/hud/minimap.go` (add alpha parameter)
- Create: `internal/render/hud/hud_anim.go` (shared HUD animation controller)
- Modify: `internal/scene/stage.go` (init and update HUD animations)
- Test: `tests/core/hud_anim_test.go`

**Design:**
- `HUDAnimController` struct: manages staggered entry for all HUD elements.
  - Fields: `Timer float64`, `Duration float64` (0.8s total), per-element offsets calculated from timer.
  - `TopBarOffsetY(t) float64`: slides from -60 to 0, starts at t=0, duration 0.2s.
  - `ActionBarOffsetY(t) float64`: slides from +60 to 0, starts at t=0.1s, duration 0.2s.
  - `MinimapAlpha(t) float64`: fades from 0 to 1, starts at t=0.2s, duration 0.2s.
  - `WavePanelAlpha(t) float64`: fades from 0 to 1, starts at t=0.3s, duration 0.2s.
- Each Draw function gains an animation parameter:
  - `DrawTopBar(screen, data, offsetY float64)` — applies Y offset to all drawing.
  - `DrawActionBar(screen, data, offsetY float64)` — same.
  - `DrawMinimap(screen, vm, alpha float64)` — scales all alphas.
- All use `easeOutCubic` for snappy feel.
- stage.go: create `HUDAnimController` in `NewStageScene`, call `Update(gameDT)` each frame, pass offsets to Draw calls.

**Test cases:**
- At t=0: TopBar offset = -60, ActionBar = +60, Minimap alpha = 0.
- At t=0.2: TopBar settled, ActionBar near settled, Minimap starting.
- At t=0.8: all settled (offset=0, alpha=1).
- Easing produces smooth values.

---

## Task 7: Panel Open/Close Animations

**Purpose:** Build menu, choice panel, item panel animate in/out instead of instant toggle.

**Files:**
- Create: `internal/render/hud/panel_anim.go` (shared panel animation state)
- Modify: `internal/render/hud/build_menu.go` (apply animation)
- Modify: `internal/render/hud/choice_panel.go` (apply animation)
- Modify: `internal/render/hud/item_panel.go` (apply animation)
- Test: `tests/core/panel_anim_test.go`

**Design:**
- `PanelAnim` struct:
  ```go
  type PanelAnim struct {
      T        float64 // 0=closed, 1=open
      Target   float64 // 0 or 1
      Speed    float64 // transition speed (default: 1/0.15 = ~6.67)
  }
  ```
- `Update(dt float64)`: lerps T toward Target at Speed. Returns `Visible() bool` (T > 0.001).
- `Scale() float64`: `0.92 + 0.08 * easeOutCubic(T)`.
- `Alpha() float64`: `easeOutQuad(T)`.
- `Open()`: sets Target=1. `Close()`: sets Target=0.

**Integration per panel:**
- BuildMenu: `BuildMenuData` adds a `PanelAnim` pointer. In `DrawBuildMenu`, apply Scale/Alpha to the panel background and contents. In stage.go, call `PanelAnim.Open()` on entering `modeBuildMenu`, `PanelAnim.Close()` on exit. Don't set `Visible=false` until `PanelAnim.Visible()` returns false.
- ChoicePanel: `ChoicePanel.Show()` calls `anim.Open()`, `Close()` calls `anim.Close()`. Draw applies Scale/Alpha.
- ItemPanel: same pattern.

**Ability choice card stagger:** ChoicePanel draws N option cards. Add per-card delay: card[i] starts animation at `T - i*0.05`, creating a cascade fly-in effect.

**Test cases:**
- Open sets target=1, Close sets target=0.
- T lerps toward target over frames.
- Scale at T=0 is 0.92, at T=1 is 1.0.
- Alpha at T=0 is 0, at T=1 is 1.
- Visible returns false when T < 0.001.

---

## Task 8: Wave Breathing Rhythm

**Purpose:** Create tension-release rhythm with breather waves and act structure.

**Files:**
- Modify: `internal/core/enemy/spawner.go:207-213,284-289` (startWave + enemyCount)
- Modify: `internal/core/enemy/spawner.go:172-180` (wave completion interval)
- Modify: `internal/render/hud/wave_announce.go` (act complete banner)
- Test: `tests/core/wave_rhythm_test.go`

**Design:**
- `isBreatherWave(wave int) bool`: wave % 5 == 4 (i.e., wave 4, 9, 14, 19...). The wave before each boss wave is the breather.
- `isActBoundary(wave int) bool`: wave % 5 == 0 (boss wave marks end of act).
- Breather wave adjustments in `startWave()`:
  - Enemy count: `enemyCount() * 0.6` (60% of normal).
  - No wave buffs: skip `applyWaveBuffs()`.
  - Reward bonus: gold multiplier 1.5x (extra income for preparation).
- After act boundary (boss wave cleared):
  - `WaveTimer = WaveInterval + 5.0` (extra 5s gap for "Act N Complete" display).
- WaveAnnounce: new trigger type for act completion. `TriggerActComplete(actNum int)` — shows "Act N Complete" banner in gold, 2s hold.

**Test cases:**
- Wave 4, 9, 14 are breather waves.
- Wave 5, 10, 15 are act boundaries (boss).
- Breather wave enemy count is 60% of normal.
- Breather wave skips buffs.
- Post-act WaveTimer includes 5s extra.
- Act number = wave / 5.

---

## Task 9: Kill-Streak / Combo System

**Purpose:** Visible combo counter with escalating feedback.

**Files:**
- Modify: `internal/scene/stage.go:1982-2002,2043-2047` (enhance existing multi-kill)
- Modify: `internal/render/hud/` (add combo counter HUD element)
- Create: `internal/render/hud/combo_counter.go`
- Test: `tests/core/combo_test.go`

**Design:**
- Existing: `multiKillCount` + `multiKillTimer` (1.5s window). Currently only shows text at x5 and x10.
- Extend timer window: 2.0s (from 1.5s) for more forgiving combo maintenance.
- New `ComboCounterVM` struct: `Count int`, `Timer float64`, `MaxTimer float64`.
- `DrawComboCounter(screen, vm)`:
  - Position: right side of screen, vertically centered.
  - Shows "×N" with escalating style:
    - 3-4: white, size 14.
    - 5-9: yellow, size 16, slight glow.
    - 10-19: orange, size 18, stronger glow.
    - 20+: red, size 22, pulsing scale.
  - Timer bar below counter showing remaining combo window.
  - Fade out when timer expires (0.3s fade).
- Milestone feedback (in stage.go kill handler):
  - x3: first appearance, subtle.
  - x5: `SpawnText("×5 连杀!", yellow)` + light shake(1.0, 0.1).
  - x10: `SpawnText("×10 超级连杀!", orange)` + shake(2.0, 0.15).
  - x20: `SpawnText("×20 无双!", red)` + shake(3.0, 0.2) + bonus gold 150.
  - x50: `SpawnText("×50 传说!", gold)` + slow-mo(0.3, 0.1, 0.3, 0.3).
- Gold bonus at milestones: x10 = +50g, x20 = +150g, x50 = +500g.

**Test cases:**
- Kill increments combo count.
- Timer resets to 2.0s on each kill.
- Count resets to 0 when timer expires.
- Milestone thresholds trigger at correct counts.
- Gold bonus awarded at milestones.

---

## Task 10: Warden Multi-Frame Animation Scaffolding

**Purpose:** Code-side support for warden animation sprites (works with placeholder/single frame).

**Files:**
- Modify: `internal/render/draw_warden.go` (use Animator for warden sprites)
- Modify: `internal/render/sprite/` (verify Animator handles single-frame gracefully)
- Test: `tests/core/warden_anim_test.go`

**Design:**
- Memory note: "Animator 框架就绪，当前单帧兼容，多帧 PNG 放入即自动生效".
- Verify Animator.Update(dt) works with 1-frame sprites (no-op).
- In `DrawWarden`: replace static sprite lookup with Animator-driven frame selection.
- Warden animation states: `idle` (default loop), `attack` (triggered on fire, plays once then returns to idle).
- `WardenState` gains `AnimState string` field, toggled by combat callbacks.
- On `ShootTimer` set (attack): switch to "attack" animation. On animation complete: revert to "idle".

**Asset note:** Requires 20-40 PNG files. With single-frame, Animator shows static sprite (current behavior). Multi-frame PNGs drop in without code change.

**Test cases:**
- Animator with 1 frame: Update is no-op, Frame returns index 0.
- Animator with N frames: cycles through frames at configured FPS.
- Attack animation plays once and reverts to idle.
- AnimState transitions correctly on shoot event.

---

## Task 11: Map Micro-Animations

**Purpose:** Animated overlay layer for path flow and decoration movement.

**Files:**
- Create: `internal/render/draw_map_overlay.go` (animated overlay pass)
- Modify: `internal/scene/stage.go` (call overlay draw after DrawMap)
- Modify: `internal/render/draw_map.go` (export path data for overlay)
- Test: `tests/core/map_overlay_test.go`

**Design:**
- New `DrawMapOverlay(screen, gm, theme, animTime float64)` rendered every frame (not cached).
- **Path flow arrows**: draw small chevron markers along path segments, offset by `animTime * speed`. Markers wrap around path length. Semi-transparent (alpha 30-40), theme path color. ~10 markers per path.
- **Decoration sway**: read decoration positions from cached layout, apply `sin(animTime*1.5 + hash) * 1.5` pixel offset. Very subtle oscillation.
- **Theme-specific effects:**
  - Lava: cells adjacent to path pulse orange alpha `0.03 + 0.02*sin(animTime*2 + hash)`.
  - Ice: occasional shimmer sparkle (1 in 200 chance per frame per decoration).
  - Tech: blinking dots (square wave, on/off every 2s with hash offset).
  - Others: just sway (forest/desert/stone/dark/void).
- Performance: O(decorationCount) per frame, ~50-100 decorations, simple sin/cos math. Well within budget.

**Test cases:**
- Path flow marker positions advance with animTime.
- Markers wrap around path length.
- Decoration sway returns oscillating offset.
- Theme-specific effects only apply to matching themes.

---

## Implementation Order

```
Task 1: BGM Crossfade        (foundation for audio tasks)
Task 2: Audio Ducking         (extends Task 1 Manager changes)
Task 3: Ambient Audio         (extends Task 1/2 Manager changes)
Task 8: Wave Breathing        (independent spawner change)
Task 9: Kill-Streak           (extends P0 Task 9 multi-kill)
Task 4: Warden Entrance       (independent warden change)
Task 10: Warden Anim Scaffold (extends Task 4 warden rendering)
Task 5: Scene Transitions     (independent game.go change)
Task 6: HUD Entry Anims       (independent HUD change)
Task 7: Panel Animations      (independent HUD change)
Task 11: Map Micro-Anims      (independent render change)
```

Tasks 1-3 are sequential (audio manager changes). Tasks 5-7, 8, 11 are independent of each other.

## Verification

After all tasks:
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] `make run` — verify BGM plays (if assets exist), HUD slides in, panels animate
- [ ] `go run cmd/autoplay/main.go --scenario attack-style-coverage` — no anomalies
- [ ] Breather waves have fewer enemies (log check)
- [ ] Combo counter appears during multi-kills
