package gamemode

import "testing"

func TestAutoPlayMode_ID(t *testing.T) {
	m := NewAutoPlayMode()
	if m.ID() != "autoplay" {
		t.Errorf("expected 'autoplay', got %q", m.ID())
	}
}

func TestAutoPlayMode_Immortal(t *testing.T) {
	m := NewAutoPlayMode()
	ctx := &Context{Lives: 0, Gold: 0}
	if m.CheckDefeat(ctx) {
		t.Error("autoplay mode should never defeat")
	}
}

func TestAutoPlayMode_VictoryOnAllWaves(t *testing.T) {
	m := NewAutoPlayMode()
	ctx := &Context{Wave: 25, MaxWaves: 25, Spawning: false}
	if !m.CheckVictory(ctx) {
		t.Error("expected victory when all waves cleared")
	}
}

func TestAutoPlayMode_NoVictoryDuringSpawn(t *testing.T) {
	m := NewAutoPlayMode()
	ctx := &Context{Wave: 25, MaxWaves: 25, Spawning: true}
	if m.CheckVictory(ctx) {
		t.Error("should not victory while spawning")
	}
}

func TestAutoPlayMode_AutoStart(t *testing.T) {
	m := NewAutoPlayMode()
	if !m.ShouldAutoStart() {
		t.Error("autoplay should auto-start waves")
	}
}

func TestAutoPlayMode_EnableEvents(t *testing.T) {
	m := NewAutoPlayMode()
	if !m.EnableEvents() {
		t.Error("autoplay should enable events for coverage")
	}
}

func TestAutoPlayMode_ShortIntermission(t *testing.T) {
	m := NewAutoPlayMode()
	if m.IntermissionSecs() != 2 {
		t.Errorf("expected 2s intermission, got %.1f", m.IntermissionSecs())
	}
}

func TestAutoPlayMode_Registered(t *testing.T) {
	m := Get("autoplay")
	if m == nil {
		t.Error("autoplay mode not registered")
	}
}
