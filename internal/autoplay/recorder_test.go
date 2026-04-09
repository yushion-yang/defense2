package autoplay

import (
	"encoding/json"
	"testing"
)

func TestRecorder_Finalize(t *testing.T) {
	r := NewRecorder("test-001", "random", "map_01", "normal", "prince")

	state := &GameState{
		Tick:     1000,
		Gold:     150,
		Lives:    18,
		Wave:     10,
		MaxWaves: 25,
		Victory:  false,
		Towers: []TowerInfo{
			{Key: "basic", Row: 1, Col: 2, Damage: 15, Cost: 60},
			{Key: "basic", Row: 2, Col: 3, Damage: 15, Cost: 60},
			{Key: "freeze", Row: 3, Col: 4, Damage: 8, Cost: 40},
		},
		Enemies: []EnemyInfo{
			{ID: 1, Active: true, Archetype: "normal"},
			{ID: 2, Active: true, Archetype: "tank"},
		},
	}

	// 模拟几帧
	for i := 0; i < 10; i++ {
		r.OnTick(state, 1.0/60.0)
	}

	record := r.Finalize(state, nil)

	if record.SessionID != "test-001" {
		t.Errorf("expected session ID 'test-001', got %q", record.SessionID)
	}
	if record.Strategy != "random" {
		t.Errorf("expected strategy 'random', got %q", record.Strategy)
	}
	if record.WavesSurvived != 10 {
		t.Errorf("expected waves 10, got %d", record.WavesSurvived)
	}

	// 验证 JSON 可序列化
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if len(data) == 0 {
		t.Error("empty JSON output")
	}
}

func TestRecorder_CoverageTracking(t *testing.T) {
	r := NewRecorder("test-002", "greedy", "map_01", "normal", "prince")

	state := &GameState{
		Tick: 100, Gold: 200, Lives: 20,
		Towers: []TowerInfo{
			{Key: "basic"},
			{Key: "freeze"},
		},
		Enemies: []EnemyInfo{
			{Active: true, Archetype: "normal"},
			{Active: true, Archetype: "runner"},
			{Active: true, Archetype: "tank"},
		},
	}

	r.OnTick(state, 1.0/60.0)

	record := r.Finalize(state, nil)

	// 应追踪到使用的塔
	if len(record.Coverage.TowersUsed) < 2 {
		t.Errorf("expected at least 2 towers tracked, got %d", len(record.Coverage.TowersUsed))
	}

	// 应追踪到敌人原型
	if len(record.Coverage.EnemyArchetypesSeen) < 3 {
		t.Errorf("expected at least 3 archetypes, got %d", len(record.Coverage.EnemyArchetypesSeen))
	}
}

func TestRecorder_DPSTracking(t *testing.T) {
	r := NewRecorder("test-003", "random", "map_01", "normal", "prince")

	// 模拟伤害
	r.RecordDamage(100)
	r.RecordDamage(200)

	// 模拟 1 秒（60 帧）
	for i := 0; i < 60; i++ {
		r.TickDPS(1.0 / 60.0)
	}

	if len(r.dpsSnapshots) == 0 {
		t.Error("expected at least 1 DPS snapshot after 1 second")
	}
	if r.peakDPS <= 0 {
		t.Error("expected positive peak DPS after recording damage")
	}
}
