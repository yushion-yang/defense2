# Experience Polish P2 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement 10 system-level experience features — tutorial redesign, unlock showcase, progress display, failure guidance, result screen upgrade, minimap enhancement, and battle report.

**Architecture:** Extends tutorial system with stage-based progression, adds new HUD overlay components, enriches result scene with highlights and tips, adds per-game stat collection for battle report.

**Tech Stack:** Go 1.24+, Ebitengine v2.9.9

**Depends on:** P0 (effects pipeline, particles), P1 (panel animations, HUD anim controller, combo system)

---

## Task 1: Settings — Hints Toggle

**Purpose:** Single on/off switch for all guidance features.

**Files:**
- Modify: `internal/scene/settings_persist.go:13-21` (add HintsEnabled to SettingsData)
- Modify: `internal/scene/settings.go` (add toggle button to UI)
- Modify: `internal/core/game/quality.go` or create `internal/core/game/settings.go` (global accessor)
- Test: `tests/core/settings_hints_test.go`

**Design:**
- Add `HintsEnabled bool` to `SettingsData` (default `true` for new players).
- Global accessor: `game.HintsEnabled() bool` reads from loaded settings.
- Settings UI: add a toggle button row below quality row — "Hints: ON / OFF".
- Auto-detection: in `ProgressManager`, if `TotalWins >= 3` and first load, suggest hints off (via toast on select screen, not forced).
- All tutorial/tip/guidance systems check `game.HintsEnabled()` before activating.

**Test cases:**
- Default SettingsData has HintsEnabled=true.
- SaveSettings persists HintsEnabled.
- LoadSettings reads HintsEnabled.
- game.HintsEnabled() returns correct value.

---

## Task 2: Layered Tutorial Redesign

**Purpose:** Drip-feed tutorial content based on player progression.

**Files:**
- Modify: `internal/core/tutorial/tutorial.go` (multi-stage tutorial)
- Modify: `internal/core/persistence/progress.go` (per-stage tutorial completion)
- Modify: `internal/scene/stage.go` (stage-aware tutorial init)
- Test: `tests/core/layered_tutorial_test.go`

**Design:**
- Replace single 8-step tutorial with 6 independent tutorial stages:
  ```go
  type TutorialStage struct {
      ID    string
      Steps []Step
      Condition func(progress *Progress) bool // when to activate
  }
  ```
- Stages:
  1. `"basics"` — build + start wave (current steps 1-4). Condition: never played.
  2. `"tower_types"` — tower selection + range. Condition: 2+ tower types unlocked.
  3. `"abilities"` — ability choice + quality tiers. Condition: first ability choice encounter.
  4. `"items"` — drag items. Condition: first item acquired in-game.
  5. `"wardens"` — warden selection + role. Condition: first warden unlocked.
  6. `"bosses"` — boss mechanics. Condition: first boss wave encounter.

- Persistence: `Progress.TutorialStages map[string]bool` (replaces single `TutorialDone bool`). Backward compat: if `TutorialDone=true`, mark `"basics"` as done.
- `Tutorial.Init(stages []TutorialStage, progress *Progress)` — finds first applicable uncompleted stage, loads its steps.
- Each stage is self-contained, 2-4 steps each. Skippable with "Skip" button.
- All stages check `game.HintsEnabled()` before activating.

**Integration:**
- stage.go `NewStageScene()`: build list of applicable stages, init tutorial with matching stage.
- Event triggers remain the same mechanism (Trigger/ClickAdvance), just mapped per-stage.

**Test cases:**
- Stage condition evaluation: basics triggers on first play.
- Stage skipping: skipped stages marked complete.
- Multiple stages: after basics done, tower_types activates when condition met.
- Backward compat: old TutorialDone=true → basics stage marked done.
- HintsEnabled=false suppresses all stages.

---

## Task 3: Contextual Tips System

**Purpose:** Just-in-time guidance at key moments, fires once per session.

**Files:**
- Create: `internal/core/tips/tips.go` (tip system)
- Create: `internal/render/hud/tip_toast.go` (info-style toast)
- Modify: `internal/scene/stage.go` (register tip triggers)
- Test: `tests/core/tips_test.go`

**Design:**
- `TipManager` struct: `firedThisSession map[string]bool`, `firedEver map[string]bool` (persisted).
- `TryShow(id string, message string) bool`: if not fired and HintsEnabled, fires once.
- Tips pool (~15 tips):
  - `"first_boss"`: "Boss 来了！血量很高，集中火力！" — triggered on first boss wave.
  - `"gold_overflow"`: "金币充裕，考虑多建几座塔！" — triggered when gold > 500 for 30s.
  - `"lives_low"`: "生命值告急！在路径瓶颈处补塔" — triggered when lives < 30%.
  - `"first_ability"`: highlight recommended option — triggered on first ability choice.
  - `"stealth_leak"`: "隐身敌人溜过去了！棱镜塔可以反隐" — triggered on stealth enemy leak.
  - `"no_towers_near_path"`: "路径附近没有塔，敌人会直接通过" — triggered when enemies leak with 0 tower in range.
  - etc.
- Display: extended Toast (3s duration) with info-blue background instead of gold.
- `TipToast` component: same rendering as Toast but with a light blue (#4488CC) pill background and info icon prefix.

**Integration:**
- stage.go: check tip conditions at relevant events (boss wave start, gold accumulation, lives change, leak, ability panel open).
- Persistence: `"tips"` key in Storage, stores `firedEver` map.

**Test cases:**
- Tip fires once per session.
- Tip fires once ever (persisted tips).
- HintsEnabled=false suppresses all tips.
- TryShow returns false on second call.
- Tip conditions evaluated correctly.

---

## Task 4: Unlock Showcase Overlay

**Purpose:** Dedicated presentation when new content is unlocked.

**Files:**
- Create: `internal/render/hud/unlock_showcase.go` (full-screen overlay)
- Modify: `internal/scene/stage.go` (trigger showcase on victory unlocks)
- Modify: `internal/scene/result.go` (show showcase before result)
- Test: `tests/core/unlock_showcase_test.go`

**Design:**
- `UnlockShowcase` struct with animation phases:
  1. Background dim (0.2s): darken screen to 40%.
  2. Card fly-in (0.4s): unlock card slides up from bottom with scale 0.8→1.0 + overshoot.
  3. Light burst (0.2s): radial particle confetti from card center.
  4. Content reveal (0.6s): card flips (Y-rotation simulation via scaleX 1→0→-1→0→1) revealing:
     - Front: "NEW UNLOCK!" header.
     - Back: item icon + name + brief description.
  5. Hold (1.0s): user can tap to dismiss early.
  6. Fade out (0.3s): card slides down, dim lifts.

- `ShowUnlocks(items []UnlockItem)`: queues multiple unlocks, shows one at a time.
- `UnlockItem`: `Type string` (map/tower/warden), `Key string`, `Name string`, `Description string`.

- For achievements: simpler animation. `ShowAchievement(name, tier string)`: floating badge from bottom with tier glow color (bronze/silver/gold/diamond), 2s auto-dismiss.

**Integration:**
- stage.go victory path: `RecordGameResult` returns list of newly unlocked items. If non-empty, show showcase before transitioning to ResultScene.
- Achievement unlock: replace `hud.ShowToast("成就解锁: ...")` with `ShowAchievement(name, tier)`.

**Test cases:**
- ShowUnlocks queues items.
- Animation phases advance on timer.
- Tap during hold phase dismisses.
- Multiple unlocks shown sequentially.
- Empty unlock list is no-op.

---

## Task 5: Progress Visualization

**Purpose:** Show completion progress on campaign select screen.

**Files:**
- Modify: `internal/scene/campaign_select.go` (add progress bar, star counts, next unlock)
- Modify: `internal/core/persistence/progress.go` (add star tracking)
- Test: `tests/core/progress_viz_test.go`

**Design:**
- **Star tracking**: Add `Stars map[string]int` to `Progress` (key = "modeID_mapID_diffID", value = 0-3). Updated in `RecordGameResult` alongside high scores.
- **Campaign select additions:**
  - Top area: "Campaign Progress: X/24 Stars" with progress bar (24 = 8 maps × 3 max stars).
  - Per-map card: show star indicators (filled/empty stars below map name). Color: gold for earned, gray for unearned.
  - Bottom area: "Next Unlock" teaser if applicable — shows locked icon + name + requirement text.
- Star count computed from `Stars` map filtered by current difficulty.
- `ProgressManager.TotalStars(modeID, diffID string) int` — sums all star values for the mode+difficulty.
- `ProgressManager.NextUnlock() (string, string)` — returns next lockable item name and requirement text based on current unlock state.

**Test cases:**
- Star recording: 3-star victory stores 3.
- Lower star doesn't override higher.
- TotalStars sums correctly.
- NextUnlock returns correct item when partially unlocked.
- NextUnlock returns empty when all unlocked.

---

## Task 6: Failure Guidance

**Purpose:** Data-driven tips on defeat screen.

**Files:**
- Create: `internal/core/tips/failure_tips.go` (failure analysis)
- Modify: `internal/scene/result.go` (show tip on defeat)
- Test: `tests/core/failure_tips_test.go`

**Design:**
- `AnalyzeDefeat(stats GameStats, mapConfig, towers, enemies) string` — heuristic tip generator.
- Heuristics (~20 rules, first match wins):
  - `GoldSpent < GoldEarned * 0.5` → "金币花得太少！尽量把金币转化为塔的战力"
  - `TowersBuilt <= 2` → "塔太少了，试试多建几座不同类型的塔"
  - Stealth enemies leaked (need new stat) → "隐身敌人溜过去了，棱镜塔可以反隐"
  - Boss killed 0 → "Boss 没有击杀，试试集中火力在 Boss 出现的位置"
  - `MaxKillStreak < 3` → "试试把塔建在路径拐弯处，提高命中率"
  - `ItemsUsed == 0` → "别忘了使用道具，道具能大幅强化塔"
  - Fast leak (lives dropped > 50% in first 5 waves) → "前期压力太大，开局优先建攻速快的塔"
  - etc.
- Result scene defeat state: show tip in a distinct info box below stats, blue tint, with lightbulb icon prefix.

**Integration:**
- result.go: on defeat, call `AnalyzeDefeat()` in `NewResultScene`, store result as `defeatTip string`.
- Add new draw section after stats phase: if defeat and defeatTip non-empty, draw tip box.

**Test cases:**
- Low gold spend triggers gold tip.
- Low tower count triggers build tip.
- Zero boss kills triggers boss tip.
- Victory returns empty tip (no advice needed).
- First matching heuristic wins (priority order).

---

## Task 7: Difficulty Labels

**Purpose:** Show difficulty rating on map cards.

**Files:**
- Modify: `internal/scene/campaign_select.go:278` (drawMapCards)
- Modify: `internal/config/loader.go:33` (LevelEntry already has Difficulty field)
- Modify: `config/level-list.json` (verify difficulty values set)
- Test: `tests/core/difficulty_label_test.go`

**Design:**
- `LevelEntry.Difficulty` already exists as a string field. Verify it's populated in `config/level-list.json` for all 8 maps.
- Map card rendering: add difficulty badge at bottom-right of card.
  - Color coding: "easy"=green, "normal"=white, "hard"=orange, "extreme"=red (reuse existing `diffColors` map from select.go).
  - Format: small text label + colored dot. E.g., `"● Easy"` in green, size 10.
- If difficulty not set, derive from map config: `waves <= 15` = easy, `<= 25` = normal, `<= 35` = hard, `> 35` = extreme.

**Test cases:**
- Each level has a difficulty value (config test).
- Difficulty label color matches expected mapping.
- Derived difficulty matches manual when both present.

---

## Task 8: Minimap Enhancement

**Purpose:** Boss marker with pulse, enemy flow direction.

**Files:**
- Modify: `internal/render/hud/minimap.go:40-87` (DrawMinimap)
- Modify: `internal/scene/stage.go` (pass animTime to minimap VM)
- Test: `tests/core/minimap_enhanced_test.go`

**Design:**
- `MinimapVM` additions: `AnimTime float64` (for boss pulse animation).
- Boss marker: radius 3.0 (current 2.0) + pulsing glow ring. `pulseR = 3.0 + 1.0*sin(animTime*4)`. Color: bright red with alpha oscillation.
- Enemy flow direction: draw small arrow at the "front" of the enemy cluster (the enemy closest to base). Arrow points along the path direction at that enemy's position.
  - Simplified: draw 3 chevron marks along the path at 25%, 50%, 75% positions, animated offset by `animTime * 0.1` to create flowing feel. Alpha 60.
- Wave progress indicator: thin line along the bottom of minimap showing `enemiesKilled / totalEnemies` for current wave. Green fill on gray background.

**Integration:**
- stage.go minimap VM builder (line ~2660): pass `animTime` to VM.
- Add `WaveProgress float64` to MinimapVM (0.0-1.0), computed from spawner state.

**Test cases:**
- Boss dot radius is larger than normal.
- Pulse radius oscillates with animTime.
- Wave progress correctly computed from spawner state.
- Flow chevron positions advance with animTime.

---

## Task 9: Result Screen Upgrade

**Purpose:** Star animation, match highlights, victory confetti.

**Files:**
- Modify: `internal/scene/result.go` (enhance star animation, add highlights section, add confetti)
- Modify: `internal/render/particle/emitter.go` (add EmitConfetti preset)
- Test: `tests/core/result_upgrade_test.go`

**Design:**
- **Star animation upgrade**: current stars pop with scale 0→1.2→1.0. Enhance:
  - Each star rotates 360° during pop-in.
  - Radial light rays: 8 thin lines radiating outward from star center, alpha fading over 0.3s.
  - Golden particle burst (4-6 particles) on each star land.
  - 3-star bonus: all three stars emit combined radial glow.

- **Match highlights** (new section between stats and buttons):
  - Auto-generated from `GameStats`:
    - "MVP Tower: [name] — [kills] kills" (BestTowerName, BestTowerKills)
    - "Max Kill Streak: ×[count]" (MaxKillStreak, only if >= 5)
    - "Perfect Waves: [count]" (need new stat, or derive from LeaksTotal)
    - "Efficiency: [goldSpent/goldEarned]%" (economy rating)
  - Display: 2-column grid, each item with icon + text, fade-in during stats phase.

- **Victory confetti**: `EmitConfetti(pool, screenW)` — 30-50 particles from top of screen, multi-color (red/blue/gold/green), slow gravity fall, slight horizontal scatter. Triggered at resultStarsPhase start for victory.
- **3-star extra**: additional gold confetti burst + celebratory SFX (if available).

**Test cases:**
- Star animation duration matches expected timing.
- Match highlights generated correctly from stats.
- Highlights only show when stats meet threshold (e.g., streak >= 5).
- Confetti only on victory, not defeat.
- 3-star gets extra confetti.

---

## Task 10: Battle Report

**Purpose:** Post-game analytics accessible from result screen.

**Files:**
- Create: `internal/core/stats/collector.go` (per-wave stat collection)
- Create: `internal/render/hud/battle_report.go` (report overlay)
- Modify: `internal/scene/stage.go` (collect per-wave snapshots)
- Modify: `internal/scene/result.go` (add "Details" button opening report)
- Test: `tests/core/battle_report_test.go`

**Design:**
- **Stat collector**: `WaveSnapshot` struct per wave:
  ```go
  type WaveSnapshot struct {
      Wave       int
      DamageSum  float64
      KillCount  int
      GoldNet    int     // earned - spent this wave
      LeakCount  int
      TowerCount int
  }
  ```
- `StatsCollector` struct: `Snapshots []WaveSnapshot`, `CurrentWave WaveSnapshot`.
- On wave clear event: finalize current snapshot, append to Snapshots, start new.
- Per-tower damage: track `map[string]float64` (towerKey → totalDamage). Use existing `Tower.Kills` for kill attribution; add `Tower.TotalDamage float64` field incremented in `apply_hit.go`.

- **Battle report overlay**:
  - Full-screen semi-transparent overlay with scrollable content.
  - Sections:
    1. **DPS Timeline**: horizontal bar chart, one bar per wave, height = damageSum (normalized).
    2. **Tower Contribution**: horizontal stacked bar showing damage % per tower type. Color-coded by tower key.
    3. **Gold Economy**: simple line showing gold balance over time (gold after each wave).
    4. **Enemies Leaked**: bar chart per wave.
  - All rendered with draw primitives (FilledRect for bars, Line for chart axes, Text for labels).
  - Close button or tap-outside to dismiss.

- **Result scene**: add "Details" button (third button) at bottom of result panel. On tap, shows BattleReport overlay.

**Test cases:**
- WaveSnapshot records correct values.
- Multiple wave snapshots accumulate.
- Tower damage attribution tracks per-tower.
- DPS timeline normalizes correctly.
- Empty game (0 waves) shows "No data".

---

## Implementation Order

```
Task 1: Hints Toggle          (foundation, all others depend on it)
Task 7: Difficulty Labels     (small, independent config change)
Task 5: Progress Visualization (extends persistence, independent)
Task 2: Layered Tutorial      (extends tutorial system)
Task 3: Contextual Tips       (new system, depends on Task 1)
Task 8: Minimap Enhancement   (independent HUD change)
Task 6: Failure Guidance      (independent, extends result scene)
Task 4: Unlock Showcase       (extends result/stage flow)
Task 9: Result Screen Upgrade (extends result scene)
Task 10: Battle Report        (new system, extends stage + result)
```

## Verification

After all tasks:
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] `make run` — verify tutorials fire at correct moments, tips appear, unlock showcase works
- [ ] Hints toggle OFF suppresses all guidance
- [ ] Progress bar shows correct star count
- [ ] Defeat screen shows relevant tip
- [ ] Battle report renders correctly with 10+ waves of data
- [ ] `go run cmd/autoplay/main.go --sweep` — no anomalies from new stats tracking
