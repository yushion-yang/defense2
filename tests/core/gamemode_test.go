package core_test

import (
	"testing"

	"defense2/internal/core/gamemode"
)

// helper: 创建带固定值的 Context
func testCtx(wave, maxWaves, lives, kills, leaked int) *gamemode.Context {
	mw := maxWaves
	lv := lives
	g := 500
	return &gamemode.Context{
		Wave:        wave,
		MaxWaves:    maxWaves,
		Lives:       lives,
		Gold:        g,
		Kills:       kills,
		Leaked:      leaked,
		Spawning:    wave < maxWaves,
		SetMaxWaves: func(v int) { mw = v; _ = mw },
		SetLives:    func(v int) { lv = v; _ = lv },
		SetGold:     func(v int) { g = v; _ = g },
		AddGold:     func(v int) { g += v },
	}
}

// ── Campaign (casual) ──

func TestCampaignVictory(t *testing.T) {
	m := gamemode.GetOrDefault("casual")
	// wave < max → not victory
	ctx := testCtx(5, 12, 20, 10, 0)
	if m.CheckVictory(ctx) {
		t.Error("should not win when wave < maxWaves")
	}
	// wave >= max, spawning done → victory
	ctx = testCtx(12, 12, 20, 50, 0)
	ctx.Spawning = false
	if !m.CheckVictory(ctx) {
		t.Error("should win when wave >= maxWaves and not spawning")
	}
}

func TestCampaignDefeat(t *testing.T) {
	m := gamemode.GetOrDefault("casual")
	ctx := testCtx(5, 12, 0, 10, 5)
	if !m.CheckDefeat(ctx) {
		t.Error("should lose when lives <= 0")
	}
	ctx.Lives = 1
	if m.CheckDefeat(ctx) {
		t.Error("should not lose when lives > 0")
	}
}

func TestCampaignScore(t *testing.T) {
	m := gamemode.GetOrDefault("casual")
	// score = waves*100 + kills*10 - leaked*50
	ctx := testCtx(10, 12, 5, 30, 2)
	score := m.GetScore(ctx)
	expected := 10*100 + 30*10 - 2*50
	if score != expected {
		t.Errorf("score = %d, want %d", score, expected)
	}
}

func TestCampaignWaveClearBonus(t *testing.T) {
	m := gamemode.GetOrDefault("casual")
	ctx := testCtx(5, 12, 20, 0, 0)
	result := m.OnWaveCleared(5, ctx)
	expectedBonus := 12 + 5*4  // = 32 (economy.json campaign.waveBonus)
	expectedPerfect := 2 + 5*2 // = 12 (economy.json campaign.perfectBonus base=2)
	if result.BonusGold != expectedBonus {
		t.Errorf("bonus = %d, want %d", result.BonusGold, expectedBonus)
	}
	if result.PerfectBonus != expectedPerfect {
		t.Errorf("perfect = %d, want %d", result.PerfectBonus, expectedPerfect)
	}
}

// ── Endless ──

func TestEndlessNeverWins(t *testing.T) {
	m := gamemode.GetOrDefault("endless")
	ctx := testCtx(9999, 9999, 20, 1000, 0)
	ctx.Spawning = false
	if m.CheckVictory(ctx) {
		t.Error("endless should never win")
	}
}

func TestEndlessScore(t *testing.T) {
	m := gamemode.GetOrDefault("endless")
	ctx := testCtx(50, 9999, 5, 200, 10)
	score := m.GetScore(ctx)
	expected := 50*100 + 200*10
	if score != expected {
		t.Errorf("score = %d, want %d", score, expected)
	}
}

func TestEndlessOnInit(t *testing.T) {
	m := gamemode.GetOrDefault("endless")
	var maxWaves int
	ctx := &gamemode.Context{
		SetMaxWaves: func(v int) { maxWaves = v },
	}
	m.OnInit(ctx)
	if maxWaves != 9999 {
		t.Errorf("maxWaves = %d, want 9999", maxWaves)
	}
}

// ── Timed ──

func TestTimedVictory(t *testing.T) {
	m := gamemode.GetOrDefault("timed")
	ctx := &gamemode.Context{
		Lives:       20,
		SetMaxWaves: func(int) {},
	}
	m.OnInit(ctx)

	// Countdown 300 seconds
	for i := 0; i < 300; i++ {
		m.OnTick(1.0, ctx)
	}
	if !m.CheckVictory(ctx) {
		t.Error("timed: should win when time runs out and lives > 0")
	}
}

func TestTimedDefeat(t *testing.T) {
	m := gamemode.GetOrDefault("timed")
	ctx := &gamemode.Context{
		Lives:       0,
		SetMaxWaves: func(int) {},
	}
	m.OnInit(ctx)
	if !m.CheckDefeat(ctx) {
		t.Error("timed: should lose when lives <= 0")
	}
}

func TestTimedScore(t *testing.T) {
	m := gamemode.GetOrDefault("timed")
	ctx := testCtx(10, 9999, 15, 100, 5)
	// score = kills*5 + lives*50
	expected := 100*5 + 15*50
	if m.GetScore(ctx) != expected {
		t.Errorf("score = %d, want %d", m.GetScore(ctx), expected)
	}
}

// ── BossRush ──

func TestBossRushVictory(t *testing.T) {
	m := gamemode.GetOrDefault("bossRush")
	ctx := &gamemode.Context{
		SetMaxWaves: func(int) {},
	}
	m.OnInit(ctx)

	// Kill 5 bosses
	for i := 0; i < 5; i++ {
		m.OnEnemyKilled(true, ctx)
	}
	if !m.CheckVictory(ctx) {
		t.Error("boss rush: should win after killing 5 bosses")
	}
}

func TestBossRushNonBossKillsIgnored(t *testing.T) {
	m := gamemode.GetOrDefault("bossRush")
	ctx := &gamemode.Context{
		SetMaxWaves: func(int) {},
	}
	m.OnInit(ctx)

	// Kill 100 non-boss enemies
	for i := 0; i < 100; i++ {
		m.OnEnemyKilled(false, ctx)
	}
	if m.CheckVictory(ctx) {
		t.Error("boss rush: should not win from non-boss kills")
	}
}

func TestBossRushScore(t *testing.T) {
	m := gamemode.GetOrDefault("bossRush")
	// Fast game (100s elapsed) with 10 kills:
	// score = max(0, 10000-100*10) + 10*20 = 9000 + 200 = 9200
	ctx := testCtx(5, 5, 20, 10, 0)
	ctx.ElapsedTime = 100
	score := m.GetScore(ctx)
	if score != 9200 {
		t.Errorf("score = %d, want 9200", score)
	}
}

// ── Challenge ──

func TestChallengeScore(t *testing.T) {
	m := gamemode.GetOrDefault("challenge")
	// Challenge 使用与 casual 相同的分数公式
	ctx := testCtx(12, 12, 10, 40, 2)
	score := m.GetScore(ctx)
	// score = 12*100 + 40*10 + (-50)*2 = 1200 + 400 - 100 = 1500
	expected := 12*100 + 40*10 + (-50)*2
	if score != expected {
		t.Errorf("score = %d, want %d", score, expected)
	}
}

// ── TestMode ──

func TestTestModeNoDefeat(t *testing.T) {
	m := gamemode.GetOrDefault("test")
	ctx := testCtx(5, 10, 0, 0, 100)
	if m.CheckDefeat(ctx) {
		t.Error("test mode should never defeat")
	}
}

func TestTestModeVictoryWithWaves(t *testing.T) {
	m := gamemode.GetOrDefault("test")
	ctx := testCtx(10, 10, 20, 50, 0)
	ctx.Spawning = false
	if !m.CheckVictory(ctx) {
		t.Error("test mode: should win when waves completed")
	}
}

func TestTestModeNoVictoryWithZeroWaves(t *testing.T) {
	m := gamemode.GetOrDefault("test")
	ctx := testCtx(0, 0, 20, 0, 0)
	if m.CheckVictory(ctx) {
		t.Error("test mode: should not win with maxWaves=0")
	}
}

// ── Session ──

func TestSessionStats(t *testing.T) {
	m := gamemode.GetOrDefault("casual")
	s := gamemode.NewSession(m)

	ctx := testCtx(5, 12, 20, 0, 0)

	s.OnEnemyKilled(false, ctx)
	s.OnEnemyKilled(true, ctx)
	s.OnEnemyKilled(false, ctx)
	s.OnEnemyLeaked(ctx)

	if s.Stats.Kills != 3 {
		t.Errorf("kills = %d, want 3", s.Stats.Kills)
	}
	if s.Stats.BossKills != 1 {
		t.Errorf("bossKills = %d, want 1", s.Stats.BossKills)
	}
	if s.Stats.Leaked != 1 {
		t.Errorf("leaked = %d, want 1", s.Stats.Leaked)
	}

	s.OnTowerBuilt()
	s.OnTowerBuilt()
	if s.Stats.TowersBuilt != 2 {
		t.Errorf("towersBuilt = %d, want 2", s.Stats.TowersBuilt)
	}
}

func TestSessionVictoryDetection(t *testing.T) {
	m := gamemode.GetOrDefault("casual")
	s := gamemode.NewSession(m)

	ctx := testCtx(12, 12, 20, 50, 0)
	ctx.Spawning = false

	ended := s.TickEndConditions(ctx)
	if !ended {
		t.Error("should detect end")
	}
	if s.Status != gamemode.StatusVictory {
		t.Errorf("status = %d, want StatusVictory(%d)", s.Status, gamemode.StatusVictory)
	}
}

func TestSessionDefeatDetection(t *testing.T) {
	m := gamemode.GetOrDefault("casual")
	s := gamemode.NewSession(m)

	ctx := testCtx(5, 12, 0, 10, 5)
	ended := s.TickEndConditions(ctx)
	if !ended {
		t.Error("should detect end")
	}
	if s.Status != gamemode.StatusDefeat {
		t.Errorf("status = %d, want StatusDefeat(%d)", s.Status, gamemode.StatusDefeat)
	}
}

func TestSessionWaveClear(t *testing.T) {
	m := gamemode.GetOrDefault("casual")
	s := gamemode.NewSession(m)

	ctx := testCtx(3, 12, 20, 10, 0)
	result := s.OnWaveCleared(3, ctx)

	if s.Stats.WavesCleared != 1 {
		t.Errorf("wavesCleared = %d, want 1", s.Stats.WavesCleared)
	}
	if result.BonusGold <= 0 {
		t.Error("should have bonus gold")
	}
	// Perfect bonus: no leaks during this wave
	if result.PerfectBonus <= 0 {
		t.Error("should have perfect bonus (no leaks this wave)")
	}
}

func TestSessionElapsedTime(t *testing.T) {
	m := gamemode.GetOrDefault("casual")
	s := gamemode.NewSession(m)

	ctx := testCtx(1, 12, 20, 0, 0)
	s.Tick(0.5, ctx)
	s.Tick(0.5, ctx)
	s.Tick(1.0, ctx)

	if s.ElapsedTime != 2.0 {
		t.Errorf("elapsed = %f, want 2.0", s.ElapsedTime)
	}
}

// ── Difficulty ──

func TestDefaultDifficulty(t *testing.T) {
	d := gamemode.DefaultDifficulty()
	if d.HPScale != 1.0 || d.SpeedScale != 1.0 || d.RewardScale != 1.0 || d.StartGold != 120 {
		t.Errorf("unexpected default: %+v", d)
	}
}

func TestLoadDifficultyFallback(t *testing.T) {
	// Invalid ID should fallback to normal
	d := gamemode.LoadDifficulty("nonexistent")
	if d.HPScale != 1.0 {
		t.Errorf("fallback HPScale = %f, want 1.0", d.HPScale)
	}
}

// ── Registry ──

func TestModeRegistry(t *testing.T) {
	modes := gamemode.List()
	expected := map[string]bool{
		"casual":     true,
		"classic":    true,
		"coop":       true,
		"endless":    true,
		"timed":      true,
		"bossRush":   true,
		"challenge":  true,
		"test":       true,
		"autoplay":   true,
		"simulation": true,
	}
	for _, id := range modes {
		if !expected[id] {
			t.Errorf("unexpected mode: %s", id)
		}
		delete(expected, id)
	}
	for id := range expected {
		t.Errorf("missing mode: %s", id)
	}
}

func TestGetOrDefault(t *testing.T) {
	m := gamemode.GetOrDefault("nonexistent")
	if m.ID() != "casual" {
		t.Errorf("fallback ID = %s, want casual", m.ID())
	}
}

// ── HUD Config ──

func TestTimedHUDConfig(t *testing.T) {
	m := gamemode.GetOrDefault("timed")
	ctx := &gamemode.Context{SetMaxWaves: func(int) {}}
	m.OnInit(ctx)
	hud := m.GetHUDConfig(ctx)
	if !hud.ShowTimer {
		t.Error("timed mode should show timer")
	}
	if hud.TimerSeconds < 299 { // ~300 after init
		t.Errorf("timer = %f, want ~300", hud.TimerSeconds)
	}
}

func TestBossRushHUDConfig(t *testing.T) {
	m := gamemode.GetOrDefault("bossRush")
	ctx := &gamemode.Context{SetMaxWaves: func(int) {}}
	m.OnInit(ctx)
	hud := m.GetHUDConfig(ctx)
	if !hud.ShowBossCount {
		t.Error("boss rush should show boss count")
	}
	if hud.TotalBosses != 5 {
		t.Errorf("total = %d, want 5", hud.TotalBosses)
	}
}
