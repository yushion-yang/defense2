# AI AutoPlay System Design

Date: 2026-03-30

## Goal

Build an automated game-playing system that:
1. Systematically covers all game content and code paths
2. Collects structured JSON data + key-frame screenshots per session
3. Detects balance issues (overpowered/underpowered towers, enemies) and runtime anomalies (panic, stuck, state inconsistency)
4. Outputs artifacts for multimodal AI to analyze and discover problems

## Architecture

```
cmd/autoplay/main.go              # standalone entry, hidden window
internal/autoplay/
  ├── controller.go                # AutoPlayController: per-frame decision driver
  ├── strategy.go                  # Strategy interface
  ├── strategy_random.go           # random fuzzing
  ├── strategy_greedy.go           # heuristic greedy
  ├── strategy_focus.go            # single-tower extreme
  ├── strategy_scenario.go         # scripted scenarios (FSM/edge cases)
  ├── coverage.go                  # coverage matrix generator (pairwise + targeted)
  ├── recorder.go                  # JSON data collector
  ├── screenshotter.go             # key-frame PNG capture
  ├── anomaly.go                   # runtime anomaly detector
  └── report.go                    # summary report generator
internal/core/gamemode/
  └── autoplay.go                  # autoplay GameMode (immortal, auto-wave, 10x speed)
```

## Strategy Interface

```go
type Strategy interface {
    Name() string
    // Called every frame, returns 0~N actions
    Decide(state *GameState) []Action
}

type GameState struct {
    Tick         int
    Gold         int
    Lives        int
    Wave         int
    MaxWaves     int
    WaveActive   bool
    Enemies      []EnemyInfo      // live enemy snapshot
    Towers       []TowerInfo      // placed tower snapshot
    BuildCells   []Cell           // available build positions
    TowerDefs    []TowerDefInfo   // available tower types
}

type Action struct {
    Type     ActionType  // Build / Upgrade / Sell / StartWave / SelectWarden
    TowerKey string
    Cell     Cell
}
```

### Built-in Strategies (v1: 4 strategies)

| Strategy | Behavior | Purpose |
|----------|----------|---------|
| `RandomStrategy` | random tower, random position, random upgrade/sell timing | Fuzzing: wide coverage, find panics |
| `GreedyStrategy` | prefer path-corner positions, pick best cost-efficiency tower, save up for upgrades | Simulate normal player, test balance |
| `FocusStrategy` | build only one tower type, fill all slots then upgrade | Test single-tower limits, find OP/UP towers |
| `ScenarioStrategy` | scripted action sequences for FSM transitions and edge cases | Targeted code-path coverage |

## Screenshot Capture

### Trigger Points

| Event | Filename Pattern |
|-------|-----------------|
| Game start (before wave 1) | `{session}/start.png` |
| Every 5 waves cleared | `{session}/wave_{N}.png` |
| Boss wave spawn | `{session}/boss_wave_{N}.png` |
| Enemy leak (reaches endpoint) | `{session}/leak_wave_{N}_{tick}.png` |
| Victory / Defeat | `{session}/result.png` |
| Anomaly detected | `{session}/anomaly_{type}_{tick}.png` |
| Panic recovered | `{session}/panic_{tick}.png` |

### Implementation

Capture via `screen.WritePixels` after `Draw()` completes, encode to PNG.
Hidden window mode: Ebitengine renders normally but window is minimized.

## JSON Report Structure

```json
{
  "session_id": "2026-03-30_greedy_map01_normal_001",
  "strategy": "greedy",
  "map_id": "map_01",
  "difficulty": "normal",
  "warden": "prince",
  "result": "victory",
  "waves_survived": 25,
  "total_waves": 25,
  "final_gold": 340,
  "final_lives": 18,
  "total_kills": 487,
  "duration_ticks": 18200,
  "towers_built": [
    {"key": "archer", "count": 5, "total_cost": 500, "total_dps": 120.5}
  ],
  "wave_log": [
    {"wave": 1, "enemies": 8, "leaked": 0, "gold_earned": 40, "gold_spent": 60}
  ],
  "anomalies": [
    {"tick": 5200, "type": "enemy_stuck", "detail": "enemy at (340,120) not moving for 60 ticks", "screenshot": "anomaly_enemy_stuck_5200.png"}
  ],
  "screenshots": ["start.png", "wave_5.png", "result.png"],
  "coverage": {}
}
```

## Coverage Matrix

### Game Content Inventory

The system must cover the following content dimensions:

#### Towers (8 types)

| Key | Label | Attack Style | Abilities |
|-----|-------|-------------|-----------|
| `laser` | Laser | `laser` | `distanceDamage` |
| `freeze` | Freeze | `projectile` | `onHitSlow`, `multiTarget` |
| `electric` | Electric | `projectile` | `stun`, `bounce` |
| `hunter` | Hunter | `projectile` | `percentHpDamage` |
| `en-04` | Wide Beam | `wideBeam` | `burn` |
| `en-05` | Scatter | `scatter` | `bleedDot` |
| `en-08` | Charge | `charge` | `chargeShot` |
| `wl-02` | Spin AoE | `spin_aoe` | `percentHpMinor` |

Note: `aura_dot` attack style is registered but no tower uses it.

#### Enemy Archetypes (13 in config)

| Key | Special Property |
|-----|-----------------|
| `normal` | Balanced stats |
| `runner` | High speed (1.84x), low HP |
| `tank` | Very high HP (2.85x), slow |
| `armored` | High HP (1.4x), slow |
| `shielded` | Shield = 82% MaxHP |
| `swarm` | Fastest (2.08x), tiny HP (0.5x) |
| `stealth` | 3s stealth on spawn |
| `splitter` | Death splits into 2 children |
| `teleporter` | Teleports every 5s |
| `healer` | Heals allies in 105px range |
| `buffer` | Aura: +20% speed, +10 armor |
| `flying` | Direct flight to endpoint |
| `dummy` | Immobile, 10000x HP |

#### Boss Templates (7)

`bossPhase`, `bossTeleport`, `bossSpawnMinions`, `bossReflect`, `bossRotateWeakness`, `bossGoldSteal`, `bossAura`

#### Maps (8 campaign + 2 test)

`map_01` ~ `map_08`, `map_test`, `map_dummy`

#### Wardens (5)

`prince` (fire), `core` (mech), `chain` (link), `skystrike` (water), `envoy` (gold)

#### Game Modes (7)

`campaign`, `endless`, `timed`, `bossRush`, `challenge`, `test`, `skillTest`

#### Difficulties (4)

| ID | HP Scale | Speed Scale | Reward Scale | Start Gold |
|----|----------|------------|-------------|-----------|
| `easy` | 0.7 | 0.85 | 1.3 | 180 |
| `normal` | 1.0 | 1.0 | 1.0 | 120 |
| `hard` | 1.4 | 1.15 | 0.8 | 100 |
| `extreme` | 2.0 | 1.3 | 0.6 | 80 |

#### Attack Styles (7)

`projectile`, `laser`, `wideBeam`, `scatter`, `charge`, `spin_aoe`, `aura_dot`

#### Abilities (27 types)

**Combat (12):** `bounce`, `chargeShot`, `crit`, `deathMark`, `distanceDamage`, `executionBonus`, `flatDamage`, `multiTarget`, `percentHpDamage`, `percentHpMinor`, `splash`, `stackDamage`

**Control (5):** `bleedDot`, `buffPurge`, `burn`, `onHitSlow`, `stun`

**Aura (5):** `attackSpeedAura`, `critAura`, `damageUpAura`, `rangeAura`, `soloBoost`

**Zone (3):** `curseZone`, `poisonZone`, `silenceZone`

**Economy (2):** `goldPassive`, `goldOnKill`

#### Skills (9)

`chainLightning`, `nukeBomb`, `windBlade`, `channelLaser`, `missileBarrage`, `judgmentBeam`, `chainLightningBolts`, `judgmentRain`, `thunderSmite`

#### Buff Stack Modes (6)

`ModeStrongest`, `ModeAdditive`, `ModeMultiplicative`, `ModeOverride`, `ModeIndependent`, `ModeIndependentPerSource`

#### Buff Type Rules (21)

`slow`, `stun`, `knockup`, `root`, `silence`, `disarm`, `speedUp`, `damageUp`, `damageDown`, `fireRateUp`, `invincible`, `damageImmune`, `controlImmune`, `slowImmune`, `stunImmune`, `rootImmune`, `untargetable`, `shield`, `dot`, `tenacity`

#### Enemy Buff Templates (14)

`berserk`, `regen`, `healAura`, `speedAura`, `damageReduce`, `empBurst`, `siphonShield`, `blink`, `deathSplit`, `deathSlow`, `reflect`, `timewarp`, `revive`, `spawnMinions`

#### Damage Types (4)

| Type | Ignores Reduction | Ignores Shield | Ignores Invincible |
|------|-------------------|----------------|--------------------|
| `physical` | No | No | No |
| `magic` | No | No | No |
| `true` | Yes | No | No |
| `pure` | Yes | Yes | Yes |

#### Damage Pipeline Steps (8)

1. Immunity check (untargetable/invincible/damageImmune)
2. Boss %HP cap (5% MaxHP per hit)
3. Attacker damageUp buff multiplier
4. Target damageDown buff multiplier + damageCap
5. Shield absorption (multi-shield, expiry order)
6. HP deduction (min 1 damage)
7. Threshold triggers (HP ratio checks)
8. Death check (HP <= 0)

#### Choosable In-Game Events (7)

`bonus-gold`, `build-discount`, `kill-reward-up`, `output-surge`, `range-extend`, `speed-boost`, `durability`

Reward waves: 4, 8, 12, 16, 20

#### Interaction Modes (9)

`modeIdle`, `modeBuildMenu`, `modeBuildPlace`, `modeTowerSel`, `modeSpawnMenu`, `modeSpawnPlace`, `modeEvent`, `modePaused`, `modeWardenSelect`

#### Tower Firing Modes (3)

`balanced`, `rapid`, `sniper`

### Coverage Strategy

#### Dimension 1: Content Combination (Pairwise)

Full enumeration is impractical (8 towers x 10 maps x 5 wardens x 4 difficulties = 1600).
Use **pairwise coverage** — every pair of dimensions has each combination tested at least once.

| Dimension | Values |
|-----------|--------|
| Map | 8 (map_01 ~ map_08) |
| Difficulty | 4 (easy/normal/hard/extreme) |
| Warden | 5 (prince/core/chain/skystrike/envoy) |
| Strategy | 4 (random/greedy/focus/balanced) |

Pairwise algorithm generates ~**30 test cases** covering all 2-way combinations.

#### Dimension 2: Tower x Ability Coverage

`FocusStrategy` handles this:

```
For each tower (8) x each firing mode (3) = 24 sessions
  → Build only that tower type, upgrade to max
  → Record: DPS curve, ability trigger count, anomalies
```

**Cross-ability interactions** need multi-tower sessions:
- `bounce` + `multiTarget` (electric + freeze)
- `burn` + `bleedDot` (wideBeam + scatter)
- `stun` + `onHitSlow` (electric + freeze)
- Multiple aura towers adjacent (attackSpeed + damage + range)
- Zone overlap (curse + poison + silence)

#### Dimension 3: Enemy Archetype Coverage

Use `EnemyFilter` to control spawns per session:

| Test Target | EnemyFilter | Focus |
|------------|-------------|-------|
| Each of 13 base archetypes | per-type filter | movement, special behavior |
| Flying enemies | `flying-only` | pathing, tower targeting |
| Boss templates (7) | `boss-only` | each boss mechanic |
| Splitters | splitter-focused | child spawn correctness |
| Healers + Buffers | healer+buffer mix | aura stacking |
| Teleporters | teleporter-focused | post-teleport path/targeting |
| Stealth | stealth-focused | visibility/targeting toggle |

#### Dimension 4: Damage Pipeline Path Coverage

Construct specific conditions to trigger each of the 8 pipeline steps:

| Pipeline Step | Trigger Condition | Construction Method |
|--------------|-------------------|---------------------|
| Immunity check | invincible/damageImmune enemy | Boss with bossPhase template |
| Boss %HP cap | Boss + percentHpDamage tower | hunter tower vs boss |
| Attacker damageUp | damageUp buff source | envoy warden aura |
| Target damageDown | damageReduce template enemy | damageReduce buff template |
| DamageCap | enemy with damageCap | inject via config |
| Shield absorption | shielded archetype | shielded enemy sessions |
| Threshold triggers | HP ratio triggers | Boss phase transitions |
| Death check | normal kill | every session |

#### Dimension 5: Interaction Mode FSM Coverage

All 9 `interactMode` states and their legal transitions must be exercised:

```go
// Example scripted scenarios for ScenarioStrategy:

// Build flow: idle → buildMenu → buildPlace → idle
{"build-flow", [OpenBuildMenu, SelectTower("laser"), PlaceAt(3,5), Deselect]}

// Tower lifecycle: idle → towerSel → upgrade → sell → idle
{"tower-lifecycle", [ClickTower(3,5), Upgrade, Sell]}

// Pause/resume: idle → paused → idle (restore prePauseMode)
{"pause-resume", [Pause, Resume, SetSpeed(3)]}

// Event choice: idle → event → idle
{"event-choice", [WaitForEvent, ChooseOption(0)]}

// Warden select: idle → wardenSelect → idle
{"warden-select", [WaitForWardenPrompt, SelectWarden("prince")]}

// Build during pause: idle → paused → resume → buildMenu → buildPlace
{"build-after-pause", [Pause, Resume, OpenBuildMenu, SelectTower("freeze"), PlaceAt(2,4)]}

// Rapid mode switch: idle → buildMenu → cancel → towerSel → sell → buildMenu
{"rapid-switch", [OpenBuildMenu, Cancel, ClickTower(3,5), Sell, OpenBuildMenu]}
```

#### Dimension 6: Edge Cases & Stress Tests

| Scenario | Construction | Detection Target |
|----------|-------------|-----------------|
| 0 gold build attempt | start with 0 gold | insufficient gold protection |
| Fill all build cells | 99999 gold, build everywhere | full-tower state |
| Build then immediately sell | build + sell same frame | build/sell race condition |
| Rapid actions per frame | multiple actions in 1 frame | state machine robustness |
| 10x speed full game | turbo speed entire session | floating point precision at high speed |
| 0 lives continue | leak to 0 in test mode | lives=0 boundary |
| All skills on cooldown fire | activate every skill ASAP | concurrent skill execution |
| High particle density | dense combat | particle pool overflow |
| All auras stacking | 6+ aura towers adjacent | buff multiplication correctness |
| Max wave count | endless mode, 100+ waves | long session stability |

### Coverage Tracking

Each session records what was actually triggered:

```json
{
  "coverage": {
    "towers_used": ["laser", "freeze"],
    "abilities_triggered": ["distanceDamage", "onHitSlow", "stun"],
    "attack_styles_fired": ["laser", "projectile"],
    "enemy_archetypes_seen": ["normal", "runner", "tank", "flying"],
    "damage_pipeline_steps": ["immunity_skip", "shield_absorb", "hp_deduct", "death"],
    "buff_types_applied": ["slow", "stun", "dot"],
    "events_chosen": ["bonus-gold", "output-surge"],
    "interaction_modes_entered": ["idle", "buildMenu", "buildPlace", "towerSel", "paused"],
    "skills_activated": ["chainLightning", "nukeBomb"],
    "warden_abilities_used": ["fireball", "burnZone"]
  }
}
```

After all ~100 cases complete, `report.go` outputs a **coverage gap report**:

```
=== Coverage Report ===
Towers:          8/8   (100%)
Abilities:      23/27  (85%)  -- MISSING: silenceZone, soloBoost, goldOnKill, buffPurge
Attack Styles:   6/7   (86%)  -- MISSING: aura_dot (no tower uses it)
Enemy Types:    13/13  (100%)
Boss Templates:  5/7   (71%)  -- MISSING: bossRotateWeakness, bossGoldSteal
Pipeline Steps:  8/8   (100%)
Buff Types:     16/21  (76%)  -- MISSING: knockup, disarm, silence, rootImmune, stunImmune
Events:          7/7   (100%)
Interactions:    9/9   (100%)
Skills:          9/9   (100%)
```

### Test Plan Generation

```go
func GenerateTestPlan() []TestCase {
    var cases []TestCase

    // 1. Pairwise combination coverage (~30 cases)
    cases = append(cases, generatePairwise()...)

    // 2. Single-tower focus (8 towers x 3 modes = 24 cases)
    cases = append(cases, generateFocusTower()...)

    // 3. Enemy-specific (13 archetypes + 7 boss = 20 cases)
    cases = append(cases, generateEnemySpecific()...)

    // 4. Damage pipeline paths (~10 cases)
    cases = append(cases, generatePipelineCoverage()...)

    // 5. Interaction FSM (~10 cases)
    cases = append(cases, generateInteractionScenarios()...)

    // 6. Edge cases / stress (~8 cases)
    cases = append(cases, generateEdgeCases()...)

    return cases // ~100 cases total
}
```

## Anomaly Detection

The controller checks every frame for:

| Anomaly | Condition | Severity |
|---------|-----------|----------|
| Enemy stuck | position unchanged for 60 ticks, not stunned/dead | HIGH |
| Gold negative | gold < 0 | CRITICAL |
| Gold spike | gold increases >500 in single frame | HIGH |
| Lives drop | lives drops >5 in single frame | MEDIUM |
| FPS drop | Update() >50ms for 10 consecutive frames | MEDIUM |
| Panic recovered | recover() catches panic | CRITICAL |
| Infinite loop | single Update() >1s | CRITICAL |
| Tower orphan | tower exists at non-buildable cell | HIGH |
| Dead enemy walking | enemy with HP<=0 still active and not dying | CRITICAL |
| Projectile leak | projectile alive >10s with no target | LOW |

Each anomaly triggers:
1. Screenshot capture
2. JSON anomaly record with tick, type, detail, game state snapshot
3. Log to stderr

## Runtime

```bash
# Run 10 sessions, each strategy once, map_01 normal difficulty
go run cmd/autoplay/main.go \
  --runs 10 \
  --strategies random,greedy,focus \
  --map map_01 \
  --difficulty normal \
  --output ./autoplay-results/

# Full sweep: all maps x all difficulties x all wardens (pairwise)
go run cmd/autoplay/main.go --sweep --output ./autoplay-results/

# Single scenario for debugging
go run cmd/autoplay/main.go \
  --scenario build-flow \
  --map map_01 \
  --output ./autoplay-results/
```

Output directory structure:

```
autoplay-results/
  ├── 2026-03-30_greedy_map01_normal_001/
  │   ├── report.json
  │   ├── start.png
  │   ├── wave_5.png
  │   ├── boss_wave_10.png
  │   ├── result.png
  │   └── anomaly_enemy_stuck_5200.png
  ├── 2026-03-30_random_map02_hard_001/
  │   └── ...
  └── coverage_summary.json
```

## Integration with Existing Code

1. **StageScene exposes decision interface**: add `SetAutoPlayer(ap AutoPlayer)` method. At end of `updatePlaying()`, if AutoPlayer is set, call `ap.Decide()` and execute returned Actions.
2. **GameMode registration**: new `autoplay` mode based on `testMode` (immortal) + auto-wave.
3. **Screenshot hook**: `Game.Draw()` checks screenshot request queue after rendering, exports current screen to PNG.

## Out of Scope (YAGNI)

- No replay/playback system
- No ML training / reinforcement learning
- No network battle simulation
- No real-time visualization dashboard
- No LLM-driven strategy (v1 uses rule-based only; pluggable interface allows future addition)
