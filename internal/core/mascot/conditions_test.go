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
	if got := cond(ctx, state); got != "bossIncoming" {
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
	if got := cond(ctx, state); got != "lowHealth" {
		t.Fatalf("expected \"lowHealth\" at 30%% threshold, got %q", got)
	}

	// Mark as fired.
	state.markFired("lowHealth", 100)

	// Same session time: cooldown blocks.
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected cooldown to block, got %q", got)
	}

	// After cooldown (60s): should trigger again.
	ctx.SessionSecs = 161
	if got := cond(ctx, state); got != "lowHealth" {
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
	if got := cond(ctx, state); got != "sessionDuration" {
		t.Fatalf("expected \"sessionDuration\" at 30min, got %q", got)
	}

	// Mark as fired.
	state.TriggeredOnce["session_30m"] = true

	// 30 min again: one-shot, should not fire.
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no re-trigger for 30min, got %q", got)
	}

	// At 60 min: trigger.
	ctx.SessionSecs = 3600
	if got := cond(ctx, state); got != "sessionDuration" {
		t.Fatalf("expected \"sessionDuration\" at 60min, got %q", got)
	}
}

func TestTimeGreeting(t *testing.T) {
	cond := condTimeGreeting()
	state := NewConditionState()

	// Morning (hour=8, band=0). LastGreetingBand starts at -1.
	ctx := &GameContext{HourOfDay: 8}
	if got := cond(ctx, state); got != "timeGreeting" {
		t.Fatalf("expected \"timeGreeting\" for morning, got %q", got)
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
	if got := cond(ctx, state); got != "timeGreeting" {
		t.Fatalf("expected \"timeGreeting\" for afternoon, got %q", got)
	}
	if state.LastGreetingBand != 1 {
		t.Fatalf("expected LastGreetingBand=1, got %d", state.LastGreetingBand)
	}

	// Evening (hour=20, band=2).
	ctx.HourOfDay = 20
	if got := cond(ctx, state); got != "timeGreeting" {
		t.Fatalf("expected \"timeGreeting\" for evening, got %q", got)
	}

	// Late night (hour=2, band=3).
	ctx.HourOfDay = 2
	if got := cond(ctx, state); got != "timeGreeting" {
		t.Fatalf("expected \"timeGreeting\" for late night, got %q", got)
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
	if got := cond(ctx, state); got != "waveComplete" {
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
	if got := cond(ctx, state); got != "killStreak" {
		t.Fatalf("expected \"killStreak\", got %q", got)
	}

	// Cooldown blocks.
	state.markFired("killStreak", 100)
	ctx.MultiKill = 10
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected cooldown to block, got %q", got)
	}

	// After cooldown.
	ctx.SessionSecs = 131
	if got := cond(ctx, state); got != "killStreak" {
		t.Fatalf("expected \"killStreak\" after cooldown, got %q", got)
	}
}

func TestGoldShortage(t *testing.T) {
	cond := condGoldShortage()
	state := NewConditionState()

	// Gold < 20, has towers: trigger.
	ctx := &GameContext{InStage: true, SessionSecs: 100, StageSnapshot: StageSnapshot{Gold: 15, TowerCount: 3}}
	if got := cond(ctx, state); got != "goldShortage" {
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
	if got := cond(ctx, state); got != "enemySwarm" {
		t.Fatalf("expected \"enemySwarm\", got %q", got)
	}
}

func TestProgressMilestone(t *testing.T) {
	cond := condProgressMilestone()
	state := NewConditionState()

	// First win.
	ctx := &GameContext{TotalWins: 1, TotalKills: 50}
	if got := cond(ctx, state); got != "progressMilestone" {
		t.Fatalf("expected \"progressMilestone\" for first win, got %q", got)
	}
	// Mark.
	state.TriggeredOnce["wins_1"] = true

	// Same: no trigger.
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no re-trigger for wins_1, got %q", got)
	}

	// 100 kills.
	ctx.TotalKills = 100
	if got := cond(ctx, state); got != "progressMilestone" {
		t.Fatalf("expected \"progressMilestone\" for 100 kills, got %q", got)
	}
	state.TriggeredOnce["kills_100"] = true

	// 5 wins.
	ctx.TotalWins = 5
	if got := cond(ctx, state); got != "progressMilestone" {
		t.Fatalf("expected \"progressMilestone\" for 5 wins, got %q", got)
	}
}

func TestIdleChatter(t *testing.T) {
	cond := condIdleChatter()
	state := NewConditionState()

	// No previous fire: any session time >= 30s should trigger.
	ctx := &GameContext{SessionSecs: 31}
	if got := cond(ctx, state); got != "idleChatter" {
		t.Fatalf("expected \"idleChatter\" after 31s with no fires, got %q", got)
	}

	// Mark fired.
	state.markFired("idleChatter", 31)

	// Within cooldown: no trigger.
	ctx.SessionSecs = 50
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected cooldown to block idle chatter, got %q", got)
	}

	// After cooldown: trigger.
	ctx.SessionSecs = 62
	if got := cond(ctx, state); got != "idleChatter" {
		t.Fatalf("expected \"idleChatter\" after cooldown, got %q", got)
	}

	// With other recent fires: should not trigger if something fired recently.
	state.markFired("lowHealth", 70)
	state.markFired("idleChatter", 62) // reset idle
	ctx.SessionSecs = 80
	if got := cond(ctx, state); got != "" {
		t.Fatalf("expected no idle chatter when other trigger fired recently, got %q", got)
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
	if len(funcs) != 10 {
		t.Fatalf("expected 10 default conditions, got %d", len(funcs))
	}
}
