# Experience Polish P3 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement 3 long-term content features — daily challenge mode, achievement expansion, and codex/encyclopedia system.

**Architecture:** Adds new game mode (daily challenge) with seed-based RNG, expands existing achievement system from 15→30+, creates new codex scene with entity database.

**Tech Stack:** Go 1.24+, Ebitengine v2.9.9

**Depends on:** P0 (kill feedback), P1 (combo system), P2 (unlock showcase, progress tracking)

---

## Task 1: Daily Challenge Mode

**Purpose:** One fixed-seed map + modifier per day, local leaderboard.

**Files:**
- Create: `internal/core/gamemode/daily.go` (daily challenge logic)
- Create: `internal/scene/daily_select.go` (daily challenge UI)
- Modify: `internal/scene/select.go` (add daily challenge card)
- Modify: `internal/core/persistence/progress.go` (daily scores)
- Modify: `internal/core/enemy/spawner.go` (seed-based RNG)
- Test: `tests/core/daily_challenge_test.go`

**Design:**

### 1.1 Seed Generation
- `DailySeed(date time.Time) int64`: hash of `date.Format("2006-01-02")` using FNV-64.
- Deterministic from seed: map selection, modifier, wave composition.
- `DailyChallenge` struct:
  ```go
  type DailyChallenge struct {
      Seed       int64
      Date       string
      MapID      string     // deterministic from seed
      Modifier   Modifier   // gameplay modifier
      MaxWaves   int        // fixed 20
  }
  ```

### 1.2 Modifiers
- `Modifier` struct: `ID string`, `Name string`, `Description string`, `Apply func(opts *StageOptions)`.
- Pool of ~10 modifiers:
  1. `"tower_limit_3"` — "Only 3 tower types" — randomly disables 2 of 5 tower types.
  2. `"double_speed"` — "Double speed enemies" — enemy speed ×2.
  3. `"no_items"` — "No items" — item drops disabled.
  4. `"boss_rush"` — "Boss every 3 waves" — BossEveryWave override to every 3rd.
  5. `"rich_start"` — "Start with 500 gold" — initial gold 500 (default 150).
  6. `"glass_cannon"` — "5 lives only" — starting lives reduced to 5.
  7. `"no_selling"` — "No selling towers" — sell button disabled.
  8. `"fast_waves"` — "Waves auto-start, 3s interval" — WaveInterval=3, ManualWave=false.
  9. `"random_towers"` — "Random tower placement" — tower type randomized on build.
  10. `"fog_of_war"` — "Fog of war" — enemy visibility range limited.
- Modifier selected: `modifiers[seed % len(modifiers)]`.
- Map selected: `maps[seed % len(maps)]` (from unlocked maps only? or all maps?). Decision: all maps — daily challenge is a level playing field.

### 1.3 Daily Select UI
- New scene accessible from select screen (add "Daily Challenge" card alongside existing modes).
- Shows: today's map preview, modifier name + description, personal best score, "Start" button.
- If already completed today: show score + "Play Again" (score still recorded, best kept).

### 1.4 Scoring & Persistence
- Add to Progress: `DailyScores map[string]int` (key = date string "2006-01-02", value = score).
- Score formula: `kills * 10 + wavesCleared * 50 + (won ? 1000 : 0) + goldRemaining`.
- Keep last 7 days of scores (auto-prune older).
- Display on daily select: last 7 days scores as a simple list.

### 1.5 Spawner Seed Integration
- Spawner gains `Seed int64` field. When set, `compositionForWave` and wave buffs use `rand.New(rand.NewSource(Seed + wave))` for deterministic composition.
- This ensures every player faces the exact same enemies on the same day.

**Test cases:**
- Same date produces same seed.
- Different dates produce different seeds.
- Seed deterministically selects map and modifier.
- Modifier application correctly changes StageOptions.
- Spawner with seed produces identical wave composition across runs.
- Daily score persists and best is kept.
- Old scores pruned after 7 days.

---

## Task 2: Achievement Expansion

**Purpose:** Expand from 15 to 30+ achievements with hidden achievements and richer unlock animation.

**Files:**
- Modify: `internal/core/achievement/achievement.go` (add achievements, add Hidden field)
- Modify: `internal/scene/stage.go` (add new achievement triggers)
- Create: `internal/scene/achievement_scene.go` (achievement gallery UI)
- Modify: `internal/scene/select.go` (add achievement button)
- Test: `tests/core/achievement_expanded_test.go`

**Design:**

### 2.1 New Achievements (15+ additions)

**Combat category:**
| ID | Name | Description | Tier | Hidden |
|----|------|-------------|------|--------|
| damage_1000 | Destroyer | Deal 1000 total damage in one game | Bronze | No |
| damage_10000 | Annihilator | Deal 10000 total damage in one game | Silver | No |
| overkill_5x | Overwhelming Force | Overkill an enemy by 5x their max HP | Silver | No |
| combo_30 | Combo Master | Achieve a 30 kill streak | Gold | No |
| combo_50 | Legendary Streak | Achieve a 50 kill streak | Diamond | Yes |
| all_damage_types | Elemental Master | Deal all 4 damage types in one game | Silver | Yes |

**Economy category:**
| ID | Name | Description | Tier | Hidden |
|----|------|-------------|------|--------|
| gold_2000 | Gold Hoarder | Hold 2000 gold at once | Gold | No |
| no_sell | Committed Builder | Win without selling any tower | Silver | No |
| min_towers | Minimalist | Win with 3 or fewer towers | Gold | Yes |

**Collection category:**
| ID | Name | Description | Tier | Hidden |
|----|------|-------------|------|--------|
| all_wardens_used | Warden Collector | Use all 5 wardens (across games) | Silver | No |
| all_maps_played | Explorer | Play all 8 maps | Bronze | No |
| all_maps_won | Conqueror | Win all 8 maps | Gold | No |

**Challenge category:**
| ID | Name | Description | Tier | Hidden |
|----|------|-------------|------|--------|
| daily_3_streak | Daily Warrior | Complete 3 daily challenges in a row | Silver | No |
| daily_7_streak | Daily Legend | Complete 7 daily challenges in a row | Gold | No |
| no_leak_extreme | Flawless Extreme | Zero leak on Extreme | Diamond | No |
| speedrun_5min | Blitz | Win in under 5 minutes | Diamond | Yes |

### 2.2 Hidden Achievements
- Add `Hidden bool` field to `Achievement` struct.
- In achievement gallery: hidden achievements show as "???" with locked icon until unlocked.
- Once unlocked, display normally with a "Hidden Achievement!" badge.

### 2.3 Achievement Gallery Scene
- New scene accessible from select screen (trophy icon button).
- Layout: scrollable grid of achievement cards (4 columns).
  - Unlocked: tier-colored border, icon, name, description.
  - Locked visible: gray border, locked icon, name, description.
  - Locked hidden: dark border, "???", "Complete a secret challenge".
- Top area: "Achievements: X/Y unlocked" progress bar.
- Categories as tab filters: All / Combat / Economy / Collection / Challenge.

### 2.4 New Tracker Fields
- Add to Tracker: `TotalDamageDealt float64`, `WardenTypesUsed map[string]bool`, `MapsPlayed map[string]bool`, `MapsWon map[string]bool`, `DailyStreak int`, `DamageTypesUsed map[string]bool`.
- Persist all collection fields (WardenTypesUsed, MapsPlayed, MapsWon).
- Session fields: TotalDamageDealt, DamageTypesUsed.

### 2.5 Unlock Animation
- Replace `hud.ShowToast` with P2 Task 4 `ShowAchievement` overlay (tier-colored badge fly-in, 2s).

**Test cases:**
- All 30+ achievements have unique IDs.
- New trigger conditions fire correctly.
- Hidden achievements not visible until unlocked.
- Collection achievements track across sessions.
- Achievement gallery shows correct unlock count.
- Daily streak increments on consecutive days.

---

## Task 3: Codex / Encyclopedia

**Purpose:** Collected database of all towers, enemies, and wardens.

**Files:**
- Create: `internal/core/codex/codex.go` (codex data + persistence)
- Create: `internal/scene/codex_scene.go` (codex UI)
- Create: `internal/scene/codex_detail.go` (detail view)
- Modify: `internal/scene/select.go` (add codex button)
- Modify: `internal/scene/stage.go` (register encounters)
- Test: `tests/core/codex_test.go`

**Design:**

### 3.1 Codex Data Model
```go
type CodexEntry struct {
    ID          string
    Category    string // "tower", "enemy", "warden"
    Discovered  bool
    UsageCount  int    // times built/encountered/used
    FirstSeen   string // date "2006-01-02"
}

type Codex struct {
    Entries map[string]*CodexEntry `json:"entries"`
}
```

### 3.2 Content Sources
- **Towers (5 entries)**: from `config/towers/towers.json`. Display: name, stats (damage, range, speed), attack style, cost, description, tier range, ability pool.
- **Enemies (~15 unique behavior entries)**: from `config/enemies/archetypes.json`. Display: name, HP range, speed, armor, behaviors, description.
  - Group by behavior type: healer, stealth, splitter, buffer, regenerator, tank (boss), standard.
  - Don't list all 75 sprite variants — group by archetype.
- **Wardens (5 entries)**: from `config/wardens/`. Display: name, type, mechanic description, strengths, synergies.

### 3.3 Discovery System
- Entries start as `Discovered: false`, shown as silhouettes with "???" name.
- Towers: discovered on first build.
- Enemies: discovered on first encounter (enemy spawned in player's game).
- Wardens: discovered on first selection.
- `UsageCount` incremented each time: tower built, enemy spawned, warden selected.

### 3.4 Codex Scene UI
- Accessible from select screen (book icon button).
- Three tabs: Towers / Enemies / Wardens.
- Grid layout: 4 columns, card per entry.
  - Discovered: colored icon + name + brief stat. Tap for detail.
  - Undiscovered: dark silhouette + "???" + "Encounter to discover".
- Top: "Discovered: X/Y" per category.

### 3.5 Detail View
- Full-screen overlay on card tap.
- Shows:
  - Large sprite/icon (centered).
  - Name + category badge.
  - Stats table (key-value pairs).
  - Description text (from config or hardcoded lore strings).
  - Usage stats: "Built X times" / "Encountered X times" / "Used X times".
  - First discovered date.
- Close button or back gesture.

### 3.6 Persistence
- Storage key: `"codex"`. Serialized as JSON.
- On stage start: pre-register all enemies that will spawn (mark as Discovered, increment UsageCount).
- On tower build: `codex.Discover("tower:" + key)`.
- On warden select: `codex.Discover("warden:" + key)`.

### 3.7 Lore Content
- Each entry has a `Lore string` field in codex config.
- Initial: short functional descriptions (1-2 sentences) derived from game mechanics.
- Can be expanded later with narrative content.
- Stored in `config/codex/lore.json` — separate from gameplay config.

**Test cases:**
- New codex has all entries undiscovered.
- Discover marks entry and records date.
- UsageCount increments on repeated use.
- Discovered entries show name and stats.
- Undiscovered entries show silhouette.
- Discovery count per category is correct.
- Persistence saves and loads correctly.

---

## Implementation Order

```
Task 1: Daily Challenge    (new mode, mostly independent)
Task 2: Achievements       (extends existing system)
Task 3: Codex              (new system, extends stage encounters)
```

All three tasks are independent of each other. Can be implemented in parallel sessions.

**Dependency on earlier tiers:**
- Task 1 needs P0 (time-scale for dramatic moments) and P1 (combo for scoring).
- Task 2 needs P0 (kill feedback tiers for overkill detection) and P2 (unlock showcase for animation).
- Task 3 needs no specific P0-P2 features, but benefits from P2 progress visualization patterns.

## Verification

After all tasks:
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] `make run` — verify daily challenge generates different content each day
- [ ] Same date → same daily challenge (deterministic seed)
- [ ] Achievement gallery displays all 30+ achievements correctly
- [ ] Hidden achievements show "???" until unlocked
- [ ] Codex discovers entries on first encounter
- [ ] Codex detail view shows correct stats
- [ ] `go run cmd/autoplay/main.go --sweep` — no anomalies from new game mode
