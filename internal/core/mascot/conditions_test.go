package mascot

import (
	"testing"
)

func TestBossIncoming(t *testing.T) {
	cond := condBossIncoming()
	state := NewConditionState()

	// No trigger when not in stage.
	ctx := &GameContext{InStage: false, StageSnapshot: StageSnapshot{IsBossWave: true, WaveActive: true, Wave: 5}}
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger outside stage, got %q", got)
	}

	// No trigger when not boss wave.
	ctx = &GameContext{InStage: true, StageSnapshot: StageSnapshot{IsBossWave: false, WaveActive: true, Wave: 5}}
	state.PrevWave = 4
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger when not boss wave, got %q", got)
	}

	// No trigger when wave hasn't changed.
	ctx = &GameContext{InStage: true, StageSnapshot: StageSnapshot{IsBossWave: true, WaveActive: true, Wave: 5}}
	state.PrevWave = 5
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger when wave unchanged, got %q", got)
	}

	// Trigger when boss wave, active, and wave changed.
	state.PrevWave = 4
	if got := cond(ctx, state); got != "boss_incoming" {
		t.Fatalf("expected \"bossIncoming\", got %q", got)
	}
}

func TestLowHealth(t *testing.T) {
	cond := condLowHealth()
	state := NewConditionState()

	// 30% of 20 = 6. Lives=6 should trigger.
	ctx := &GameContext{
		InStage:     true,
		SessionSecs: 100,
		StageSnapshot: StageSnapshot{
			Lives:    6,
			MaxLives: 20,
		},
	}
	if got := cond(ctx, state); got != "low_health" {
		t.Fatalf("expected \"lowHealth\" at 30%% threshold, got %q", got)
	}

	// Mark as fired.
	state.markFired("low_health", 100)

	// Same session time: cooldown blocks.
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected cooldown to block, got %q", got)
	}

	// After cooldown (60s): should trigger again.
	ctx.SessionSecs = 161
	if got := cond(ctx, state); got != "low_health" {
		t.Fatalf("expected \"lowHealth\" after cooldown, got %q", got)
	}

	// Above threshold: no trigger.
	ctx.Lives = 7
	ctx.SessionSecs = 300
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger above threshold, got %q", got)
	}

	// Lives=0 should not trigger (game over).
	ctx.Lives = 0
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger at 0 lives, got %q", got)
	}
}

func TestSessionDuration(t *testing.T) {
	cond := condSessionDuration()
	state := NewConditionState()

	// Before 30 min: no trigger.
	ctx := &GameContext{SessionSecs: 1799}
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger before 30min, got %q", got)
	}

	// At 30 min: trigger.
	ctx.SessionSecs = 1800
	if got := cond(ctx, state); got != "session_30min" {
		t.Fatalf("expected \"session_30min\" at 30min, got %q", got)
	}

	// Mark as fired.
	state.TriggeredOnce["session_30m"] = true

	// 30 min again: one-shot, should not fire.
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no re-trigger for 30min, got %q", got)
	}

	// At 60 min: trigger.
	ctx.SessionSecs = 3600
	if got := cond(ctx, state); got != "session_60min" {
		t.Fatalf("expected \"session_60min\" at 60min, got %q", got)
	}
}

func TestTimeGreeting(t *testing.T) {
	cond := condTimeGreeting()
	state := NewConditionState()

	// Morning (hour=8, band=0). LastGreetingBand starts at -1.
	ctx := &GameContext{HourOfDay: 8}
	if got := cond(ctx, state); got != "greet_morning" {
		t.Fatalf("expected \"greet_morning\" for morning, got %q", got)
	}
	// State should update.
	if state.LastGreetingBand != 0 {
		t.Fatalf("expected LastGreetingBand=0, got %d", state.LastGreetingBand)
	}

	// Same band: no trigger.
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger for same band, got %q", got)
	}

	// Afternoon (hour=14, band=1).
	ctx.HourOfDay = 14
	if got := cond(ctx, state); got != "greet_afternoon" {
		t.Fatalf("expected \"greet_afternoon\" for afternoon, got %q", got)
	}
	if state.LastGreetingBand != 1 {
		t.Fatalf("expected LastGreetingBand=1, got %d", state.LastGreetingBand)
	}

	// Evening (hour=20, band=2).
	ctx.HourOfDay = 20
	if got := cond(ctx, state); got != "greet_evening" {
		t.Fatalf("expected \"greet_evening\" for evening, got %q", got)
	}

	// Late night (hour=2, band=3).
	ctx.HourOfDay = 2
	if got := cond(ctx, state); got != "greet_latenight" {
		t.Fatalf("expected \"greet_latenight\" for late night, got %q", got)
	}
	if state.LastGreetingBand != 3 {
		t.Fatalf("expected LastGreetingBand=3, got %d", state.LastGreetingBand)
	}
}

func TestWaveComplete(t *testing.T) {
	cond := condWaveComplete()
	state := NewConditionState()
	state.PrevWave = 3

	// Wave increased and not active: trigger.
	ctx := &GameContext{InStage: true, StageSnapshot: StageSnapshot{Wave: 4, WaveActive: false}}
	if got := cond(ctx, state); got != "wave_complete" {
		t.Fatalf("expected \"waveComplete\", got %q", got)
	}

	// No trigger if wave still active.
	ctx.WaveActive = true
	state.PrevWave = 4
	ctx.Wave = 5
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger while wave active, got %q", got)
	}

	// No trigger if wave didn't change.
	ctx.WaveActive = false
	ctx.Wave = 5
	state.PrevWave = 5
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger when wave unchanged, got %q", got)
	}
}

func TestKillStreak(t *testing.T) {
	cond := condKillStreak()
	state := NewConditionState()

	// MultiKill < 5: no trigger.
	ctx := &GameContext{InStage: true, SessionSecs: 100, StageSnapshot: StageSnapshot{MultiKill: 4}}
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger for MultiKill=4, got %q", got)
	}

	// MultiKill >= 5: trigger.
	ctx.MultiKill = 5
	if got := cond(ctx, state); got != "kill_streak" {
		t.Fatalf("expected \"killStreak\", got %q", got)
	}

	// Cooldown blocks.
	state.markFired("kill_streak", 100)
	ctx.MultiKill = 10
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected cooldown to block, got %q", got)
	}

	// After cooldown.
	ctx.SessionSecs = 131
	if got := cond(ctx, state); got != "kill_streak" {
		t.Fatalf("expected \"killStreak\" after cooldown, got %q", got)
	}
}

func TestGoldShortage(t *testing.T) {
	cond := condGoldShortage()
	state := NewConditionState()

	// Gold < 20, has towers: trigger.
	ctx := &GameContext{InStage: true, SessionSecs: 100, StageSnapshot: StageSnapshot{Gold: 15, TowerCount: 3}}
	if got := cond(ctx, state); got != "gold_shortage" {
		t.Fatalf("expected \"goldShortage\", got %q", got)
	}

	// Gold >= 20: no trigger.
	ctx.Gold = 20
	ctx.SessionSecs = 200
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger with enough gold, got %q", got)
	}

	// No towers: no trigger.
	ctx.Gold = 5
	ctx.TowerCount = 0
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger without towers, got %q", got)
	}
}

func TestEnemySwarm(t *testing.T) {
	cond := condEnemySwarm()
	state := NewConditionState()

	// EnemyCount <= 15: no trigger.
	ctx := &GameContext{InStage: true, SessionSecs: 100, StageSnapshot: StageSnapshot{EnemyCount: 15}}
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger for 15 enemies, got %q", got)
	}

	// EnemyCount > 15: trigger.
	ctx.EnemyCount = 16
	if got := cond(ctx, state); got != "enemy_swarm" {
		t.Fatalf("expected \"enemySwarm\", got %q", got)
	}
}

func TestProgressMilestone(t *testing.T) {
	cond := condProgressMilestone()
	state := NewConditionState()

	// First win.
	ctx := &GameContext{TotalWins: 1, TotalKills: 50}
	if got := cond(ctx, state); got != "milestone_wins_1" {
		t.Fatalf("expected \"milestone_wins_1\" for first win, got %q", got)
	}
	// Mark.
	state.TriggeredOnce["wins_1"] = true

	// Same: no trigger.
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no re-trigger for wins_1, got %q", got)
	}

	// 100 kills.
	ctx.TotalKills = 100
	if got := cond(ctx, state); got != "milestone_kills_100" {
		t.Fatalf("expected \"milestone_kills_100\" for 100 kills, got %q", got)
	}
	state.TriggeredOnce["kills_100"] = true

	// 5 wins.
	ctx.TotalWins = 5
	if got := cond(ctx, state); got != "milestone_wins_5" {
		t.Fatalf("expected \"milestone_wins_5\" for 5 wins, got %q", got)
	}
}

func TestIdleChatter(t *testing.T) {
	cond := condIdleChatter()
	state := NewConditionState()

	// First call: should trigger immediately.
	ctx := &GameContext{SessionSecs: 1}
	if got := cond(ctx, state); got != "idle_chatter" {
		t.Fatalf("expected \"idle_chatter\" on first call, got %q", got)
	}

	// Mark fired.
	state.markFired("idle_chatter", 1)

	// Within 2s cooldown: no trigger.
	ctx.SessionSecs = 2
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected cooldown to block idle chatter, got %q", got)
	}

	// After 2s cooldown: trigger again.
	ctx.SessionSecs = 4
	if got := cond(ctx, state); got != "idle_chatter" {
		t.Fatalf("expected \"idle_chatter\" after cooldown, got %q", got)
	}
}

func TestCooldowns(t *testing.T) {
	state := NewConditionState()

	// No previous fire: cooldown OK.
	if !state.cooldownOK("test", 10, 5) {
		t.Fatal("expected cooldown OK with no previous fire")
	}

	// Fire at time 10.
	state.markFired("test", 10)

	// At time 14 (4s later, cooldown 5s): not OK.
	if state.cooldownOK("test", 14, 5) {
		t.Fatal("expected cooldown NOT OK within window")
	}

	// At time 15 (5s later): OK.
	if !state.cooldownOK("test", 15, 5) {
		t.Fatal("expected cooldown OK at exact boundary")
	}

	// At time 20 (10s later): OK.
	if !state.cooldownOK("test", 20, 5) {
		t.Fatal("expected cooldown OK well past window")
	}
}

func TestDefaultConditionsCount(t *testing.T) {
	funcs := DefaultConditions()
	if len(funcs) != 27 {
		t.Fatalf("expected 27 default conditions, got %d", len(funcs))
	}
}

// --- Performance condition tests ---

func TestPerfFPSLow(t *testing.T) {
	cond := condPerfFPSLow()
	state := NewConditionState()

	// FPS=0 (not yet measured): no trigger.
	ctx := &GameContext{InStage: true, SessionSecs: 100, StageSnapshot: StageSnapshot{FPS: 0}}
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger when FPS=0, got %q", got)
	}

	// FPS >= 30: no trigger, counter resets.
	ctx = &GameContext{InStage: true, SessionSecs: 100, StageSnapshot: StageSnapshot{FPS: 30}}
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger when FPS=30, got %q", got)
	}
	if state.LowFPSFrames != 0 {
		t.Fatalf("expected LowFPSFrames=0 after normal FPS, got %d", state.LowFPSFrames)
	}

	// FPS < 30: first eval increments counter and fires (>= 1 cycle).
	ctx = &GameContext{InStage: true, SessionSecs: 100, StageSnapshot: StageSnapshot{FPS: 25}}
	if got := cond(ctx, state); got != "perf_fps_low" {
		t.Fatalf("expected \"perf_fps_low\" after 1 low cycle, got %q", got)
	}

	// Mark fired, verify cooldown blocks.
	state.markFired("perf_fps_low", 100)
	ctx = &GameContext{InStage: true, SessionSecs: 110, StageSnapshot: StageSnapshot{FPS: 20}}
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected cooldown to block, got %q", got)
	}

	// After cooldown (120s).
	ctx = &GameContext{InStage: true, SessionSecs: 221, StageSnapshot: StageSnapshot{FPS: 20}}
	state.LowFPSFrames = 0 // reset to test fresh
	if got := cond(ctx, state); got != "perf_fps_low" {
		t.Fatalf("expected \"perf_fps_low\" after cooldown, got %q", got)
	}

	// Not in stage: no trigger.
	ctx = &GameContext{InStage: false, StageSnapshot: StageSnapshot{FPS: 10}}
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger outside stage, got %q", got)
	}
}

func TestPerfFPSGreat(t *testing.T) {
	cond := condPerfFPSGreat()
	state := NewConditionState()

	// FPS=60, HeapMB=50: trigger.
	ctx := &GameContext{InStage: true, SessionSecs: 100, StageSnapshot: StageSnapshot{FPS: 60, HeapMB: 50}}
	if got := cond(ctx, state); got != "perf_fps_great" {
		t.Fatalf("expected \"perf_fps_great\", got %q", got)
	}

	// FPS < 58: no trigger.
	ctx = &GameContext{InStage: true, SessionSecs: 500, StageSnapshot: StageSnapshot{FPS: 57, HeapMB: 50}}
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger for FPS=57, got %q", got)
	}

	// HeapMB >= 100: no trigger.
	ctx = &GameContext{InStage: true, SessionSecs: 500, StageSnapshot: StageSnapshot{FPS: 60, HeapMB: 100}}
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger for HeapMB=100, got %q", got)
	}

	// HeapMB <= 0: no trigger.
	ctx = &GameContext{InStage: true, SessionSecs: 500, StageSnapshot: StageSnapshot{FPS: 60, HeapMB: 0}}
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger for HeapMB=0, got %q", got)
	}
}

func TestPerfGCHeavy(t *testing.T) {
	cond := condPerfGCHeavy()
	state := NewConditionState()

	// GCPauseUs > 5000: trigger.
	ctx := &GameContext{InStage: true, SessionSecs: 100, StageSnapshot: StageSnapshot{GCPauseUs: 5001}}
	if got := cond(ctx, state); got != "perf_gc_heavy" {
		t.Fatalf("expected \"perf_gc_heavy\", got %q", got)
	}

	// GCPauseUs <= 5000: no trigger.
	ctx = &GameContext{InStage: true, SessionSecs: 300, StageSnapshot: StageSnapshot{GCPauseUs: 5000}}
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger for GCPauseUs=5000, got %q", got)
	}

	// Not in stage: no trigger.
	ctx = &GameContext{InStage: false, StageSnapshot: StageSnapshot{GCPauseUs: 10000}}
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger outside stage, got %q", got)
	}
}

// --- UI context condition tests ---

func TestUITowerSelected(t *testing.T) {
	cond := condUITowerSelected()
	state := NewConditionState()

	// Mode changes from 0 to 3: trigger.
	ctx := &GameContext{InStage: true, SessionSecs: 100, StageSnapshot: StageSnapshot{InteractMode: 3}}
	state.PrevInteractMode = 0
	if got := cond(ctx, state); got != "ui_tower_selected" {
		t.Fatalf("expected \"ui_tower_selected\", got %q", got)
	}

	// Same mode (3 to 3): no trigger.
	state.PrevInteractMode = 3
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger when mode unchanged, got %q", got)
	}

	// Cooldown blocks.
	state.PrevInteractMode = 0
	state.markFired("ui_tower_selected", 100)
	ctx.SessionSecs = 110
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected cooldown to block, got %q", got)
	}

	// After cooldown.
	ctx.SessionSecs = 131
	if got := cond(ctx, state); got != "ui_tower_selected" {
		t.Fatalf("expected \"ui_tower_selected\" after cooldown, got %q", got)
	}
}

func TestUIAbilityChoice(t *testing.T) {
	cond := condUIAbilityChoice()
	state := NewConditionState()

	// Mode changes from 0 to 8: trigger.
	ctx := &GameContext{InStage: true, SessionSecs: 100, StageSnapshot: StageSnapshot{InteractMode: 8}}
	state.PrevInteractMode = 0
	if got := cond(ctx, state); got != "ui_ability_choice" {
		t.Fatalf("expected \"ui_ability_choice\", got %q", got)
	}

	// Same mode: no trigger.
	state.PrevInteractMode = 8
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger when mode unchanged, got %q", got)
	}

	// No cooldown: should fire again on next mode change.
	state.PrevInteractMode = 0
	ctx.SessionSecs = 101
	if got := cond(ctx, state); got != "ui_ability_choice" {
		t.Fatalf("expected \"ui_ability_choice\" (no cooldown), got %q", got)
	}
}

func TestUIWardenSelect(t *testing.T) {
	cond := condUIWardenSelect()
	state := NewConditionState()

	// Mode changes from 0 to 7: trigger.
	ctx := &GameContext{InStage: true, SessionSecs: 100, StageSnapshot: StageSnapshot{InteractMode: 7}}
	state.PrevInteractMode = 0
	if got := cond(ctx, state); got != "ui_warden_select" {
		t.Fatalf("expected \"ui_warden_select\", got %q", got)
	}

	// Same mode: no trigger.
	state.PrevInteractMode = 7
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger when mode unchanged, got %q", got)
	}
}

func TestUIBuildMenu(t *testing.T) {
	cond := condUIBuildMenu()
	state := NewConditionState()

	// Mode changes from 0 to 1: trigger.
	ctx := &GameContext{InStage: true, SessionSecs: 100, StageSnapshot: StageSnapshot{InteractMode: 1}}
	state.PrevInteractMode = 0
	if got := cond(ctx, state); got != "ui_build_menu" {
		t.Fatalf("expected \"ui_build_menu\", got %q", got)
	}

	// Cooldown blocks re-fire.
	state.markFired("ui_build_menu", 100)
	state.PrevInteractMode = 0
	ctx.SessionSecs = 110
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected cooldown to block, got %q", got)
	}

	// After cooldown.
	ctx.SessionSecs = 161
	if got := cond(ctx, state); got != "ui_build_menu" {
		t.Fatalf("expected \"ui_build_menu\" after cooldown, got %q", got)
	}
}

func TestUIItemPanel(t *testing.T) {
	cond := condUIItemPanel()
	state := NewConditionState()

	// Mode changes from 0 to 9: trigger.
	ctx := &GameContext{InStage: true, SessionSecs: 100, StageSnapshot: StageSnapshot{InteractMode: 9}}
	state.PrevInteractMode = 0
	if got := cond(ctx, state); got != "ui_item_panel" {
		t.Fatalf("expected \"ui_item_panel\", got %q", got)
	}

	// Same mode: no trigger.
	state.PrevInteractMode = 9
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger when mode unchanged, got %q", got)
	}
}

// --- Diagnostic condition tests ---

func TestDiagQualityDowngrade(t *testing.T) {
	cond := condDiagQualityDowngrade()
	state := NewConditionState()

	// PrevQuality=-1 (uninitialized): no trigger.
	ctx := &GameContext{StageSnapshot: StageSnapshot{QualityLevel: 1}}
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger when PrevQuality uninitialized, got %q", got)
	}

	// Quality 0 -> 1 (downgrade): trigger.
	state.PrevQuality = 0
	if got := cond(ctx, state); got != "diag_quality_downgrade" {
		t.Fatalf("expected \"diag_quality_downgrade\", got %q", got)
	}

	// One-shot: should not fire again.
	state.PrevQuality = 0
	ctx.QualityLevel = 2
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no re-trigger (one-shot), got %q", got)
	}

	// Quality stays same or improves: no trigger.
	state2 := NewConditionState()
	state2.PrevQuality = 1
	ctx2 := &GameContext{StageSnapshot: StageSnapshot{QualityLevel: 1}}
	if got := cond(ctx2, state2); got != "" {
		t.Fatalf("expected no trigger when quality unchanged, got %q", got)
	}
	ctx2.QualityLevel = 0
	if got := cond(ctx2, state2); got != "" {
		t.Fatalf("expected no trigger when quality improved, got %q", got)
	}
}

func TestDiagEnemySwarmCritical(t *testing.T) {
	cond := condDiagEnemySwarmCritical()
	state := NewConditionState()

	// EnemyCount <= 30: no trigger.
	ctx := &GameContext{InStage: true, SessionSecs: 100, StageSnapshot: StageSnapshot{EnemyCount: 30}}
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger for 30 enemies, got %q", got)
	}

	// EnemyCount > 30: trigger.
	ctx.EnemyCount = 31
	if got := cond(ctx, state); got != "diag_enemy_swarm_critical" {
		t.Fatalf("expected \"diag_enemy_swarm_critical\", got %q", got)
	}

	// Cooldown blocks.
	state.markFired("diag_enemy_swarm_critical", 100)
	ctx.SessionSecs = 110
	ctx.EnemyCount = 40
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected cooldown to block, got %q", got)
	}

	// After cooldown (90s).
	ctx.SessionSecs = 191
	if got := cond(ctx, state); got != "diag_enemy_swarm_critical" {
		t.Fatalf("expected \"diag_enemy_swarm_critical\" after cooldown, got %q", got)
	}

	// Not in stage: no trigger.
	ctx = &GameContext{InStage: false, StageSnapshot: StageSnapshot{EnemyCount: 50}}
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no trigger outside stage, got %q", got)
	}
}
