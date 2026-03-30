# AutoPlay Agent Collaboration System

Date: 2026-03-30
Status: Design
Depends on: `docs/plans/2026-03-30-autoplay-design.md` (autoplay core)

## Goal

Build a **fully autonomous** game optimization loop powered by Claude Code cron scheduling:

1. **Runner Agent** — executes automated game sessions, collects JSON reports + screenshots
2. **Analyst Agent** — multimodal analysis of results, identifies balance issues and bugs
3. **Fixer Agent** — modifies config/code to fix issues, validates with tests
4. **Orchestrator** — cron-driven loop that coordinates the three agents in 5-10min cycles

The system runs unattended, iterating until convergence (no critical issues found) or reaching a configurable iteration cap.

## Architecture Overview

```
                    ┌──────────────────────────────────┐
                    │        Orchestrator (cron)        │
                    │  5-10min interval, max N rounds   │
                    └──────┬───────┬───────┬───────────┘
                           │       │       │
                    ┌──────▼──┐ ┌──▼────┐ ┌▼────────┐
                    │ Runner  │ │Analyst│ │ Fixer   │
                    │ Agent   │ │Agent  │ │ Agent   │
                    └────┬────┘ └───┬───┘ └────┬────┘
                         │         │           │
              ┌──────────▼─────────▼───────────▼──────────┐
              │           autoplay-results/                │
              │  round-001/  round-002/  ...  summary.md  │
              └───────────────────────────────────────────┘
```

### Data Flow Per Round

```
Runner Agent
  │  go run cmd/autoplay/main.go --sweep --output autoplay-results/round-{N}/
  │  produces: report.json + screenshots per session
  ▼
Analyst Agent
  │  reads: all round-{N}/*.json + *.png (multimodal)
  │  produces: autoplay-results/round-{N}/analysis.md
  │            autoplay-results/round-{N}/issues.json
  ▼
Fixer Agent
  │  reads: issues.json
  │  modifies: config/*.json and/or internal/**/*.go
  │  runs: make test (gate)
  │  produces: git commit on claude/autoplay-fix branch
  ▼
Orchestrator
  │  checks: analysis.md for convergence
  │  if not converged && round < max: trigger next round
  │  else: produce final summary.md
  ▼
DONE
```

## Agent Definitions

### 1. Runner Agent

**Trigger**: Orchestrator cron fires
**Type**: `general-purpose` subagent
**Isolation**: worktree (runs autoplay binary, no code conflict)

**Actions**:
1. `go build cmd/autoplay/main.go` — build autoplay binary
2. Execute with configurable strategy mix:
   - Round 1: `--sweep` (full pairwise ~30 cases) — establish baseline
   - Round 2+: `--targeted` (re-run failing scenarios from previous analysis)
3. Output to `autoplay-results/round-{N}/`

**Session Selection Per Round**:

| Round | Strategy | Sessions | Purpose |
|-------|----------|----------|---------|
| 1 | Full sweep | ~30 (pairwise) | Baseline coverage |
| 2+ | Targeted rerun | 5-15 (from issues.json) | Verify fixes |
| Final | Validation sweep | ~30 (pairwise) | Confirm no regressions |

**Timeout**: 10 min per round (turbo 10x speed, ~30 sessions complete in ~5 min)

### 2. Analyst Agent

**Trigger**: Runner Agent completes
**Type**: `general-purpose` subagent (multimodal capable)
**Input**: `autoplay-results/round-{N}/` (JSON + PNG)

**Analysis Dimensions**:

#### A. Balance Analysis

For each tower:
- **Win rate** across maps/difficulties (< 30% → underpowered, > 90% on hard → overpowered)
- **Pick efficiency**: gold spent vs kills contributed
- **DPS curve**: damage output per wave progression

For each enemy archetype:
- **Leak rate**: how often this type reaches endpoint
- **Time alive**: average survival time (too short → too easy, too long → annoying)
- **Special mechanic trigger count**: did the special behavior actually activate?

#### B. Anomaly Triage

From `anomalies[]` in each report.json:
- Group by type (enemy_stuck, gold_negative, etc.)
- Deduplicate (same anomaly across sessions = 1 issue)
- Assign severity: CRITICAL > HIGH > MEDIUM > LOW
- Attach screenshot evidence

#### C. Coverage Gap Detection

From `coverage` field aggregated across all sessions:
- Flag untested content (abilities never triggered, enemy types never spawned)
- Suggest additional test scenarios to fill gaps

#### D. Visual Inspection (Multimodal)

Read screenshots for:
- UI rendering glitches (overlapping text, misaligned HUD)
- Visual effect anomalies (particles stuck, effects not clearing)
- Layout issues on different maps

**Output**: `analysis.md` (human-readable) + `issues.json` (machine-readable)

#### issues.json Schema

```json
{
  "round": 1,
  "timestamp": "2026-03-30T14:30:00Z",
  "converged": false,
  "critical_count": 2,
  "issues": [
    {
      "id": "BAL-001",
      "category": "balance",
      "severity": "HIGH",
      "title": "Laser tower overpowered on map_03",
      "detail": "Laser tower solo (FocusStrategy) wins map_03 extreme with 19/20 lives. DPS scales too aggressively with upgrades.",
      "evidence": {
        "sessions": ["round-001/focus_laser_map03_extreme_001"],
        "screenshots": ["result.png"],
        "metrics": {
          "win_rate": 1.0,
          "avg_lives_remaining": 19,
          "total_dps_wave_12": 450.5
        }
      },
      "suggested_fix": {
        "type": "config",
        "file": "config/towers/towers.json",
        "path": "$.laser.levels[*].damage",
        "action": "reduce_by_percent",
        "value": 15,
        "reasoning": "Laser level scaling too steep; 15% reduction brings win rate closer to 70% target on extreme"
      }
    },
    {
      "id": "BUG-001",
      "category": "anomaly",
      "severity": "CRITICAL",
      "title": "Enemy stuck at path corner (340, 120)",
      "detail": "3 sessions reported enemy_stuck at same coordinate. Likely pathfinding rounding issue at tight corner.",
      "evidence": {
        "sessions": ["round-001/greedy_map01_normal_001", "round-001/random_map01_hard_002"],
        "screenshots": ["anomaly_enemy_stuck_5200.png"],
        "occurrences": 3
      },
      "suggested_fix": {
        "type": "code",
        "file": "internal/core/enemy/movement.go",
        "description": "Add waypoint snap tolerance of 2px at path corners to prevent floating-point stuck"
      }
    }
  ],
  "rerun_scenarios": [
    {
      "strategy": "focus",
      "tower": "laser",
      "map": "map_03",
      "difficulty": "extreme",
      "reason": "Verify BAL-001 fix"
    }
  ]
}
```

### 3. Fixer Agent

**Trigger**: Analyst Agent completes with `converged: false`
**Type**: `general-purpose` subagent
**Branch**: `claude/autoplay-fix` (created on first run)

**Fix Pipeline**:

```
For each issue in issues.json (sorted by severity DESC):
  1. Read suggested_fix
  2. If type == "config":
       - Modify JSON file at specified path
       - Validate JSON syntax
  3. If type == "code":
       - Read target file
       - Implement fix (minimal, surgical)
       - Run `make lint` on modified files
  4. Run `make test`
       - If PASS: git add + commit with message "fix(autoplay): {issue.title} #{issue.id}"
       - If FAIL: revert changes, log skip reason, continue to next issue
  5. Record fix result in autoplay-results/round-{N}/fixes.json
```

**Safety Rails**:

| Rule | Implementation |
|------|---------------|
| No destructive git ops | Never `reset --hard`, `checkout --`, `clean` |
| Test gate | `make test` must pass before any commit |
| Scope limit | Only modify files referenced in issues.json |
| Change cap | Max 5 files modified per round |
| Diff cap | Max 50 lines changed per file |
| Revert on failure | `git checkout -- <file>` only for files Fixer just modified |
| Branch isolation | All work on `claude/autoplay-fix`, never touch current branch |

**fixes.json Schema**:

```json
{
  "round": 1,
  "fixes_attempted": 3,
  "fixes_applied": 2,
  "fixes_skipped": 1,
  "details": [
    {
      "issue_id": "BAL-001",
      "status": "applied",
      "files_modified": ["config/towers/towers.json"],
      "diff_lines": 8,
      "commit": "abc1234"
    },
    {
      "issue_id": "BUG-001",
      "status": "skipped",
      "reason": "make test failed: TestEnemyMovement/corner_snap assertion mismatch",
      "files_attempted": ["internal/core/enemy/movement.go"]
    }
  ]
}
```

### 4. Orchestrator (Cron Controller)

**Implementation**: Claude Code `CronCreate` with 7-min interval

**State Machine**:

```
         ┌─────────┐
         │  IDLE    │ ◄── initial state / manual trigger
         └────┬────┘
              │ /autoplay-start
              ▼
         ┌─────────┐
         │  RUN     │ ── spawn Runner Agent
         └────┬────┘
              │ Runner done
              ▼
         ┌─────────┐
         │ ANALYZE  │ ── spawn Analyst Agent
         └────┬────┘
              │ Analyst done
              ▼
         ┌─────────┐     converged || round >= max
         │  CHECK   │ ──────────────────────────────┐
         └────┬────┘                                │
              │ not converged                        ▼
              ▼                                ┌─────────┐
         ┌─────────┐                           │  DONE    │
         │  FIX     │ ── spawn Fixer Agent     └─────────┘
         └────┬────┘         │ produce summary.md
              │ Fixer done   │
              ▼              │
         ┌─────────┐        │
         │  RUN     │ ◄─────┘ (next round)
         └─────────┘
```

**Convergence Criteria**:
- `issues.json.critical_count == 0` AND `issues.json.issues.length <= 2` (only LOW remaining)
- OR `round >= max_rounds` (default: 5)

**State Persistence**: `autoplay-results/state.json`

```json
{
  "current_round": 2,
  "max_rounds": 5,
  "phase": "ANALYZE",
  "started_at": "2026-03-30T14:00:00Z",
  "rounds": [
    {
      "round": 1,
      "runner_status": "done",
      "analyst_status": "done",
      "fixer_status": "done",
      "issues_found": 5,
      "issues_fixed": 3,
      "critical_remaining": 1
    }
  ]
}
```

## Cron Schedule Design

```
Cron Job 1: "autoplay-tick" (every 7 minutes)
  - Read state.json
  - If phase == IDLE: do nothing (waiting for manual trigger)
  - If phase == RUN: check if Runner Agent done → if yes, transition to ANALYZE
  - If phase == ANALYZE: check if Analyst Agent done → if yes, check convergence
  - If phase == FIX: check if Fixer Agent done → if yes, transition to RUN (next round)
  - If phase == DONE: cancel cron, output summary

Cron Job 2: "autoplay-watchdog" (every 15 minutes)
  - Check for stuck agents (no progress for >15min)
  - If stuck: log warning, force-advance to next phase
```

**Typical Timeline** (5 rounds):

| Time | Phase | Duration |
|------|-------|----------|
| T+0 | Round 1: Runner (30 sessions @ 10x) | ~5 min |
| T+7 | Round 1: Analyst (read 30 JSONs + screenshots) | ~3 min |
| T+14 | Round 1: Fixer (apply 3-5 fixes) | ~4 min |
| T+21 | Round 2: Runner (targeted 10 sessions) | ~3 min |
| T+28 | Round 2: Analyst | ~2 min |
| T+35 | Round 2: Fixer | ~3 min |
| ... | ... | ... |
| T+~50 | Round 3-5 or convergence | |
| T+~60 | Final summary | |

Total: **~60 minutes** for 5 rounds of autonomous optimization.

## Directory Structure

```
autoplay-results/
├── state.json                          # orchestrator state
├── round-001/
│   ├── greedy_map01_normal_001/
│   │   ├── report.json                 # per-session data
│   │   ├── start.png
│   │   ├── wave_5.png
│   │   └── result.png
│   ├── random_map02_hard_001/
│   │   └── ...
│   ├── coverage_summary.json           # aggregated coverage
│   ├── analysis.md                     # analyst report (human-readable)
│   ├── issues.json                     # analyst findings (machine-readable)
│   └── fixes.json                      # fixer results
├── round-002/
│   └── ...
└── summary.md                          # final convergence report
```

## Final Summary Report

`summary.md` generated at completion, covering all rounds:

```markdown
# AutoPlay Optimization Summary

## Run Config
- Rounds: 3 (converged at round 3)
- Total sessions: 70 (30 + 25 + 15)
- Total time: 42 minutes
- Branch: claude/autoplay-fix (4 commits)

## Issues Found & Fixed
| ID | Category | Severity | Status | Fix |
|----|----------|----------|--------|-----|
| BAL-001 | balance | HIGH | FIXED | Laser damage -15% |
| BAL-002 | balance | MEDIUM | FIXED | Swarm HP +20% |
| BUG-001 | anomaly | CRITICAL | FIXED | Corner snap tolerance |
| BUG-002 | anomaly | LOW | OPEN | Projectile leak (cosmetic) |

## Coverage
- Towers: 8/8 (100%)
- Enemy archetypes: 13/13 (100%)
- Abilities: 25/27 (93%) — missing: aura_dot (no tower), silenceZone
- Pipeline steps: 8/8 (100%)

## Balance Changes Applied
| Target | Field | Before | After | Rationale |
|--------|-------|--------|-------|-----------|
| laser.levels[1].damage | damage | 45 | 38 | OP on map_03 extreme |
| swarm.hpScale | hpScale | 0.5 | 0.6 | Too easy to kill |

## Recommendations (not auto-fixed)
1. Consider adding a tower that uses `aura_dot` attack style
2. `silenceZone` ability never triggers — check if any tower has it
3. Map_07 has unusually low win rate across all strategies — map design issue?
```

## Manual Trigger

Start the loop from Claude Code session:

```
User: /autoplay-start
→ Creates claude/autoplay-fix branch
→ Sets state.json phase=RUN round=1
→ Creates cron job (7min interval)
→ Spawns first Runner Agent
```

Stop early:

```
User: /autoplay-stop
→ Cancels cron jobs
→ Generates summary.md from whatever rounds completed
→ Reports status
```

Check progress:

```
User: /autoplay-status
→ Reads state.json
→ Shows current round, phase, issues found/fixed
```

## Implementation Dependencies

### Required First (from autoplay-design.md)
1. `cmd/autoplay/main.go` — standalone autoplay binary
2. `internal/autoplay/` — controller, strategies, recorder, screenshotter
3. `internal/core/gamemode/autoplay.go` — autoplay game mode

### This Design Adds
1. `autoplay-results/` directory convention
2. Three agent prompt templates (runner/analyst/fixer)
3. Orchestrator cron logic (in Claude Code session)
4. `issues.json` / `fixes.json` / `state.json` schemas
5. `/autoplay-start` / `/autoplay-stop` / `/autoplay-status` commands (Claude Code skills)

### Agent Prompts (stored as Claude Code skills or inline)

**Runner Agent Prompt Template**:
```
Build and run the autoplay binary for round {N}.
Command: go run cmd/autoplay/main.go {args} --output autoplay-results/round-{N}/
Wait for completion. Report session count and any build errors.
If --targeted: use rerun_scenarios from round-{N-1}/issues.json.
```

**Analyst Agent Prompt Template**:
```
Analyze autoplay results in autoplay-results/round-{N}/.
Read all report.json files and examine screenshots.
Produce analysis.md and issues.json following the schema defined in
docs/autoplay-agent-collaboration.md.
Focus on: balance outliers, recurring anomalies, coverage gaps, visual glitches.
Compare with previous rounds if available.
```

**Fixer Agent Prompt Template**:
```
Fix issues listed in autoplay-results/round-{N}/issues.json.
Work on branch claude/autoplay-fix.
For each issue (sorted by severity):
  - Apply suggested_fix
  - Run make test
  - Commit if pass, revert if fail
Respect safety rails: max 5 files, max 50 lines per file.
Output fixes.json with results.
```

## Configuration

Stored in `autoplay-results/config.json` (created at /autoplay-start):

```json
{
  "max_rounds": 5,
  "cron_interval_min": 7,
  "runner": {
    "round1_strategy": "sweep",
    "subsequent_strategy": "targeted",
    "final_strategy": "sweep",
    "session_timeout_min": 10,
    "turbo_speed": 10
  },
  "analyst": {
    "balance_win_rate_low": 0.3,
    "balance_win_rate_high": 0.9,
    "anomaly_dedup_threshold": 2
  },
  "fixer": {
    "max_files_per_round": 5,
    "max_lines_per_file": 50,
    "test_command": "make test",
    "lint_command": "make lint",
    "auto_commit": true,
    "branch": "claude/autoplay-fix"
  },
  "convergence": {
    "max_critical": 0,
    "max_total_issues": 2
  }
}
```

## Out of Scope (YAGNI)

- No Web UI dashboard for monitoring
- No Slack/webhook notifications
- No parallel multi-branch fixes
- No ML-based strategy optimization
- No automated PR creation (human reviews branch manually)
- No cross-repository changes (this project only)
