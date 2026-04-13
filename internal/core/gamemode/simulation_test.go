package gamemode

import "testing"

func TestSimulationMode_ID(t *testing.T) {
	m := NewSimulationMode()
	if m.ID() != "simulation" {
		t.Errorf("expected 'simulation', got %q", m.ID())
	}
}

func TestSimulationMode_Mortal(t *testing.T) {
	m := NewSimulationMode()
	ctx := &Context{Lives: 0}
	if !m.CheckDefeat(ctx) {
		t.Error("simulation mode should defeat when lives=0")
	}
}

func TestSimulationMode_AliveWithLives(t *testing.T) {
	m := NewSimulationMode()
	ctx := &Context{Lives: 5}
	if m.CheckDefeat(ctx) {
		t.Error("simulation mode should not defeat when lives>0")
	}
}

func TestSimulationMode_NoAutoStart(t *testing.T) {
	m := NewSimulationMode()
	if m.ShouldAutoStart() {
		t.Error("simulation should not auto-start waves")
	}
}

func TestSimulationMode_Intermission(t *testing.T) {
	m := NewSimulationMode()
	if m.IntermissionSecs() != 3 {
		t.Errorf("expected 3s intermission, got %.1f", m.IntermissionSecs())
	}
}

func TestSimulationMode_Victory(t *testing.T) {
	m := NewSimulationMode()
	ctx := &Context{Wave: 15, MaxWaves: 15, Spawning: false}
	if !m.CheckVictory(ctx) {
		t.Error("expected victory when all waves cleared")
	}
}

func TestSimulationMode_NoVictoryDuringSpawn(t *testing.T) {
	m := NewSimulationMode()
	ctx := &Context{Wave: 15, MaxWaves: 15, Spawning: true}
	if m.CheckVictory(ctx) {
		t.Error("should not victory while spawning")
	}
}

func TestSimulationMode_Registered(t *testing.T) {
	m := Get("simulation")
	if m == nil {
		t.Error("simulation mode not registered")
	}
}
