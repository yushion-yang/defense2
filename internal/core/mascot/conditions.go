package mascot

import "fmt"

// ConditionFunc evaluates game context and returns a trigger name, or "" if no trigger.
type ConditionFunc func(ctx *GameContext, state *ConditionState) string

// ConditionState tracks cooldowns and one-shot flags for condition evaluation.
type ConditionState struct {
	LastFireTime     map[string]float64 // trigger -> SessionSecs when last fired
	TriggeredOnce    map[string]bool    // one-shot milestone flags
	PrevWave         int                // for detecting wave changes
	PrevInStage      bool               // for detecting stage enter/exit
	LastGreetingBand int                // -1 = none, 0=morning, 1=afternoon, 2=evening, 3=latenight
	PrevInteractMode int                // for detecting UI mode changes
	PrevQuality      int                // for detecting quality downgrades (-1 = uninitialized)
	LowFPSFrames     int                // consecutive eval cycles with FPS < 30

	// Meta condition tracking
	PrevGold      int     // previous gold for spike detection
	PrevGoldInit  bool    // true after first gold reading
	PrevLives     int     // previous lives for anomaly detection
	PrevLivesInit bool    // true after first lives reading
	PrevKills     int     // previous kills (to correlate with gold changes)
	PauseAccum    float64 // accumulated pause time in seconds
	StageDefeats  int     // consecutive defeats in this session
	PrevVictory   bool    // previous Victory flag
	PrevDefeat    bool    // previous Defeat flag
}

// NewConditionState creates a zero-value condition state ready for use.
func NewConditionState() *ConditionState {
	return &ConditionState{
		LastFireTime:     make(map[string]float64),
		TriggeredOnce:    make(map[string]bool),
		LastGreetingBand: -1,
		PrevQuality:      -1,
	}
}

// cooldownOK checks if enough time has passed since last fire of this trigger.
func (cs *ConditionState) cooldownOK(trigger string, sessionSecs, cooldownSecs float64) bool {
	last, ok := cs.LastFireTime[trigger]
	if !ok {
		return true
	}
	return sessionSecs-last >= cooldownSecs
}

// markFired records that a trigger was just fired.
func (cs *ConditionState) markFired(trigger string, sessionSecs float64) {
	cs.LastFireTime[trigger] = sessionSecs
}

// DefaultConditions returns all built-in condition functions.
// Order determines priority: UI context first, then performance, diagnostics, meta, then original conditions.
func DefaultConditions() []ConditionFunc {
	return []ConditionFunc{
		// UI context (highest priority — respond to user actions immediately)
		condUITowerSelected(),
		condUIAbilityChoice(),
		condUIWardenSelect(),
		condUIBuildMenu(),
		condUIItemPanel(),
		// Performance
		condPerfFPSLow(),
		condPerfFPSGreat(),
		condPerfGCHeavy(),
		// Diagnostics
		condDiagQualityDowngrade(),
		condDiagEnemySwarmCritical(),
		// Meta (fourth-wall / anomaly detection)
		condMetaGoldSpike(),
		condMetaLivesUp(),
		condMetaPerfectClear(),
		condMetaSpeedrun(),
		condMetaRepeatedLoss(),
		condMetaLongPause(),
		condMetaNoTowers(),
		// Original conditions
		condBossIncoming(),
		condLowHealth(),
		condSessionDuration(),
		condTimeGreeting(),
		condWaveComplete(),
		condKillStreak(),
		condGoldShortage(),
		condEnemySwarm(),
		condProgressMilestone(),
		condIdleChatter(),
	}
}

// condBossIncoming fires when a new boss wave starts.
func condBossIncoming() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if !ctx.IsBossWave || !ctx.WaveActive {
			return ""
		}
		if ctx.Wave == state.PrevWave {
			return ""
		}
		return "boss_incoming"
	}
}

// condLowHealth fires when lives drop to 30% or below. 60s cooldown.
func condLowHealth() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if ctx.Lives <= 0 || ctx.MaxLives <= 0 {
			return ""
		}
		threshold := ctx.MaxLives * 30 / 100
		if ctx.Lives > threshold {
			return ""
		}
		if !state.cooldownOK("low_health", ctx.SessionSecs, 60) {
			return ""
		}
		return "low_health"
	}
}

// condSessionDuration fires at 30min, 60min, 90min thresholds. One-shot each.
func condSessionDuration() ConditionFunc {
	thresholds := []struct {
		secs    float64
		key     string
		trigger string
	}{
		{1800, "session_30m", "session_30min"},
		{3600, "session_60m", "session_60min"},
		{5400, "session_90m", "session_90min"},
	}
	return func(ctx *GameContext, state *ConditionState) string {
		for _, th := range thresholds {
			if state.TriggeredOnce[th.key] {
				continue
			}
			if ctx.SessionSecs >= th.secs {
				state.TriggeredOnce[th.key] = true
				return th.trigger
			}
		}
		return ""
	}
}

// condTimeGreeting fires once per time-of-day band change.
// Bands: morning(6-11)=0, afternoon(12-17)=1, evening(18-22)=2, latenight(23||0-5)=3.
func condTimeGreeting() ConditionFunc {
	bandTriggers := []string{"greet_morning", "greet_afternoon", "greet_evening", "greet_latenight"}
	return func(ctx *GameContext, state *ConditionState) string {
		band := hourBand(ctx.HourOfDay)
		if band == state.LastGreetingBand {
			return ""
		}
		state.LastGreetingBand = band
		return bandTriggers[band]
	}
}

// hourBand returns the time-of-day band for a given hour (0-23).
func hourBand(hour int) int {
	switch {
	case hour >= 6 && hour <= 11:
		return 0 // morning
	case hour >= 12 && hour <= 17:
		return 1 // afternoon
	case hour >= 18 && hour <= 22:
		return 2 // evening
	default:
		return 3 // late night (23, 0-5)
	}
}

// condWaveComplete fires when wave number increases and wave is no longer active.
func condWaveComplete() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if ctx.WaveActive {
			return ""
		}
		if ctx.Wave <= state.PrevWave {
			return ""
		}
		return "wave_complete"
	}
}

// condKillStreak fires when MultiKill >= 5. 30s cooldown.
func condKillStreak() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if ctx.MultiKill < 5 {
			return ""
		}
		if !state.cooldownOK("kill_streak", ctx.SessionSecs, 30) {
			return ""
		}
		return "kill_streak"
	}
}

// condGoldShortage fires when Gold < 20 and player has towers. 60s cooldown.
func condGoldShortage() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if ctx.Gold >= 20 || ctx.TowerCount <= 0 {
			return ""
		}
		if !state.cooldownOK("gold_shortage", ctx.SessionSecs, 60) {
			return ""
		}
		return "gold_shortage"
	}
}

// condEnemySwarm fires when EnemyCount > 15. 45s cooldown.
func condEnemySwarm() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if ctx.EnemyCount <= 15 {
			return ""
		}
		if !state.cooldownOK("enemy_swarm", ctx.SessionSecs, 45) {
			return ""
		}
		return "enemy_swarm"
	}
}

// condProgressMilestone fires for TotalWins at 1,5,10,25,50 and TotalKills at 100,500,1000,5000.
// One-shot each milestone.
func condProgressMilestone() ConditionFunc {
	winMilestones := []int{1, 5, 10, 25, 50}
	killMilestones := []int{100, 500, 1000, 5000}

	return func(ctx *GameContext, state *ConditionState) string {
		for _, m := range winMilestones {
			key := fmt.Sprintf("wins_%d", m)
			if state.TriggeredOnce[key] {
				continue
			}
			if ctx.TotalWins >= m {
				state.TriggeredOnce[key] = true
				return fmt.Sprintf("milestone_wins_%d", m)
			}
		}
		for _, m := range killMilestones {
			key := fmt.Sprintf("kills_%d", m)
			if state.TriggeredOnce[key] {
				continue
			}
			if ctx.TotalKills >= m {
				state.TriggeredOnce[key] = true
				return fmt.Sprintf("milestone_kills_%d", m)
			}
		}
		return ""
	}
}

// condIdleChatter fires as a fallback when no other condition triggered.
// Short cooldown ensures continuous idle dialog rotation with brief pauses.
func condIdleChatter() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !state.cooldownOK("idle_chatter", ctx.SessionSecs, 2) {
			return ""
		}
		return "idle_chatter"
	}
}

// --- Performance conditions ---

// condPerfFPSLow fires when FPS has been below 30 for at least one eval cycle (~5s). 120s cooldown.
func condPerfFPSLow() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage || ctx.FPS <= 0 {
			state.LowFPSFrames = 0
			return ""
		}
		if ctx.FPS < 30 {
			state.LowFPSFrames++
		} else {
			state.LowFPSFrames = 0
			return ""
		}
		if state.LowFPSFrames < 1 {
			return ""
		}
		if !state.cooldownOK("perf_fps_low", ctx.SessionSecs, 120) {
			return ""
		}
		return "perf_fps_low"
	}
}

// condPerfFPSGreat fires when FPS >= 58 and HeapMB < 100. 300s cooldown.
func condPerfFPSGreat() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if ctx.FPS < 58 || ctx.HeapMB <= 0 || ctx.HeapMB >= 100 {
			return ""
		}
		if !state.cooldownOK("perf_fps_great", ctx.SessionSecs, 300) {
			return ""
		}
		return "perf_fps_great"
	}
}

// condPerfGCHeavy fires when GCPauseUs > 5000. 120s cooldown.
func condPerfGCHeavy() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if ctx.GCPauseUs <= 5000 {
			return ""
		}
		if !state.cooldownOK("perf_gc_heavy", ctx.SessionSecs, 120) {
			return ""
		}
		return "perf_gc_heavy"
	}
}

// --- UI context conditions ---
// Note: PrevInteractMode is updated in UpdateContext after all conditions run.

// condUITowerSelected fires when InteractMode changes to 3 (modeTowerSel). 30s cooldown.
func condUITowerSelected() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if ctx.InteractMode != 3 || state.PrevInteractMode == 3 {
			return ""
		}
		if !state.cooldownOK("ui_tower_selected", ctx.SessionSecs, 30) {
			return ""
		}
		return "ui_tower_selected"
	}
}

// condUIAbilityChoice fires when InteractMode changes to 8 (modeUpgrade). No cooldown.
func condUIAbilityChoice() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if ctx.InteractMode != 8 || state.PrevInteractMode == 8 {
			return ""
		}
		return "ui_ability_choice"
	}
}

// condUIWardenSelect fires when InteractMode changes to 7 (modeWardenSelect). No cooldown.
func condUIWardenSelect() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if ctx.InteractMode != 7 || state.PrevInteractMode == 7 {
			return ""
		}
		return "ui_warden_select"
	}
}

// condUIBuildMenu fires when InteractMode changes to 1 (modeBuildMenu). 60s cooldown.
func condUIBuildMenu() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if ctx.InteractMode != 1 || state.PrevInteractMode == 1 {
			return ""
		}
		if !state.cooldownOK("ui_build_menu", ctx.SessionSecs, 60) {
			return ""
		}
		return "ui_build_menu"
	}
}

// condUIItemPanel fires when InteractMode changes to 9 (modeItemPanel). 60s cooldown.
func condUIItemPanel() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if ctx.InteractMode != 9 || state.PrevInteractMode == 9 {
			return ""
		}
		if !state.cooldownOK("ui_item_panel", ctx.SessionSecs, 60) {
			return ""
		}
		return "ui_item_panel"
	}
}

// --- Diagnostic conditions ---

// condDiagQualityDowngrade fires once when quality level increases (downgrade). One-shot.
func condDiagQualityDowngrade() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if state.PrevQuality < 0 {
			return ""
		}
		if state.TriggeredOnce["diag_quality_downgrade"] {
			return ""
		}
		if ctx.QualityLevel <= state.PrevQuality {
			return ""
		}
		state.TriggeredOnce["diag_quality_downgrade"] = true
		return "diag_quality_downgrade"
	}
}

// condDiagEnemySwarmCritical fires when EnemyCount > 30. 90s cooldown.
func condDiagEnemySwarmCritical() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if ctx.EnemyCount <= 30 {
			return ""
		}
		if !state.cooldownOK("diag_enemy_swarm_critical", ctx.SessionSecs, 90) {
			return ""
		}
		return "diag_enemy_swarm_critical"
	}
}

// --- Meta / fourth-wall conditions ---

// condMetaGoldSpike fires when gold increases by > 500 in one eval cycle
// without a corresponding kill increase. One-shot per session.
func condMetaGoldSpike() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage || !state.PrevGoldInit {
			return ""
		}
		if state.TriggeredOnce["meta_gold_spike"] {
			return ""
		}
		goldDelta := ctx.Gold - state.PrevGold
		killDelta := ctx.Kills - state.PrevKills
		// Suspicious: gold jumped by >500 with fewer than 5 kills in the same period.
		if goldDelta > 500 && killDelta < 5 {
			state.TriggeredOnce["meta_gold_spike"] = true
			return "meta_gold_spike"
		}
		return ""
	}
}

// condMetaLivesUp fires when lives increase (no healing mechanic exists). One-shot.
func condMetaLivesUp() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage || !state.PrevLivesInit {
			return ""
		}
		if state.TriggeredOnce["meta_lives_up"] {
			return ""
		}
		if ctx.Lives > state.PrevLives {
			state.TriggeredOnce["meta_lives_up"] = true
			return "meta_lives_up"
		}
		return ""
	}
}

// condMetaPerfectClear fires on victory with zero lives lost. One-shot per session.
func condMetaPerfectClear() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if state.TriggeredOnce["meta_perfect_clear"] {
			return ""
		}
		// Detect transition into victory state.
		if ctx.Victory && !state.PrevVictory && ctx.Lives == ctx.MaxLives {
			state.TriggeredOnce["meta_perfect_clear"] = true
			return "meta_perfect_clear"
		}
		return ""
	}
}

// condMetaSpeedrun fires on victory within 3 minutes. One-shot per session.
func condMetaSpeedrun() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if state.TriggeredOnce["meta_speedrun"] {
			return ""
		}
		if ctx.Victory && !state.PrevVictory && ctx.ElapsedSecs < 180 {
			state.TriggeredOnce["meta_speedrun"] = true
			return "meta_speedrun"
		}
		return ""
	}
}

// condMetaRepeatedLoss fires after 3+ consecutive defeats in this session. One-shot.
func condMetaRepeatedLoss() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if state.TriggeredOnce["meta_repeated_loss"] {
			return ""
		}
		// StageDefeats is incremented in UpdateContext when Defeat transitions true.
		if state.StageDefeats >= 3 {
			state.TriggeredOnce["meta_repeated_loss"] = true
			return "meta_repeated_loss"
		}
		return ""
	}
}

// condMetaLongPause fires when the game has been paused for > 120s. 120s cooldown.
func condMetaLongPause() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage || !ctx.Paused {
			return ""
		}
		if state.PauseAccum < 120 {
			return ""
		}
		if !state.cooldownOK("meta_long_pause", ctx.SessionSecs, 120) {
			return ""
		}
		return "meta_long_pause"
	}
}

// condMetaNoTowers fires when a wave is active but no towers have been built. One-shot.
func condMetaNoTowers() ConditionFunc {
	return func(ctx *GameContext, state *ConditionState) string {
		if !ctx.InStage {
			return ""
		}
		if state.TriggeredOnce["meta_no_towers"] {
			return ""
		}
		if ctx.WaveActive && ctx.TowerCount == 0 && ctx.Wave > 1 {
			state.TriggeredOnce["meta_no_towers"] = true
			return "meta_no_towers"
		}
		return ""
	}
}
