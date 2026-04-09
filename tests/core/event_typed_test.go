package core_test

import (
	"defense2/internal/core/event"
	"testing"
)

func TestOnTyped_BasicDispatch(t *testing.T) {
	bus := event.NewBus()
	var got event.TowerBuiltPayload
	event.OnTyped(bus, event.EvtTowerBuilt, func(p event.TowerBuiltPayload) {
		got = p
	})
	bus.Emit(event.EvtTowerBuilt, event.TowerBuiltPayload{TowerKey: "basic", Cost: 80})
	if got.TowerKey != "basic" || got.Cost != 80 {
		t.Fatalf("expected {laser,80}, got {%s,%d}", got.TowerKey, got.Cost)
	}
}

func TestOnTyped_WrongTypeSkipped(t *testing.T) {
	bus := event.NewBus()
	called := false
	event.OnTyped(bus, event.EvtTowerBuilt, func(_ event.TowerBuiltPayload) {
		called = true
	})
	// Emit wrong payload type — should be silently skipped
	bus.Emit(event.EvtTowerBuilt, event.EnemyKilledPayload{IsBoss: true})
	if called {
		t.Fatal("OnTyped should skip mismatched payload type")
	}
}

func TestOnTyped_Cancel(t *testing.T) {
	bus := event.NewBus()
	count := 0
	cancel := event.OnTyped(bus, event.EvtEnemyKilled, func(_ event.EnemyKilledPayload) {
		count++
	})
	bus.Emit(event.EvtEnemyKilled, event.EnemyKilledPayload{})
	if count != 1 {
		t.Fatalf("expected 1 call, got %d", count)
	}
	cancel()
	bus.Emit(event.EvtEnemyKilled, event.EnemyKilledPayload{})
	if count != 1 {
		t.Fatalf("expected 1 call after cancel, got %d", count)
	}
}

func TestOnTyped_MultipleSubscribers(t *testing.T) {
	bus := event.NewBus()
	var order []string
	event.OnTyped(bus, event.EvtWaveStarted, func(p event.WaveStartedPayload) {
		order = append(order, "a")
	})
	event.OnTyped(bus, event.EvtWaveStarted, func(p event.WaveStartedPayload) {
		order = append(order, "b")
	})
	bus.Emit(event.EvtWaveStarted, event.WaveStartedPayload{Wave: 1, IsBoss: false})
	if len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Fatalf("expected [a,b], got %v", order)
	}
}

func TestOnTyped_AllPayloadTypes(t *testing.T) {
	bus := event.NewBus()

	// Verify each payload type works
	tests := []struct {
		name string
		evt  string
		emit func()
	}{
		{"TowerBuilt", event.EvtTowerBuilt, func() {
			bus.Emit(event.EvtTowerBuilt, event.TowerBuiltPayload{TowerKey: "x", Cost: 1})
		}},
		{"TowerUpgraded", event.EvtTowerUpgraded, func() {
			bus.Emit(event.EvtTowerUpgraded, event.TowerUpgradedPayload{TowerKey: "x", Spent: 1})
		}},
		{"TowerSold", event.EvtTowerSold, func() {
			bus.Emit(event.EvtTowerSold, event.TowerSoldPayload{TowerKey: "x", Refund: 1})
		}},
		{"EnemyKilled", event.EvtEnemyKilled, func() {
			bus.Emit(event.EvtEnemyKilled, event.EnemyKilledPayload{IsBoss: true, KillerID: "p", GoldValue: 5})
		}},
		{"EnemyLeaked", event.EvtEnemyLeaked, func() {
			bus.Emit(event.EvtEnemyLeaked, event.EnemyLeakedPayload{})
		}},
		{"WaveStarted", event.EvtWaveStarted, func() {
			bus.Emit(event.EvtWaveStarted, event.WaveStartedPayload{Wave: 3, IsBoss: true})
		}},
		{"WaveCleared", event.EvtWaveCleared, func() {
			bus.Emit(event.EvtWaveCleared, event.WaveClearedPayload{Wave: 3, Perfect: true})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			cancel := bus.On(tt.evt, func(_ ...interface{}) { called = true })
			tt.emit()
			cancel()
			if !called {
				t.Fatalf("%s emit did not trigger subscriber", tt.name)
			}
		})
	}
}

func TestBusClear_StopsAllSubscriptions(t *testing.T) {
	bus := event.NewBus()
	count := 0
	event.OnTyped(bus, event.EvtTowerBuilt, func(_ event.TowerBuiltPayload) { count++ })
	event.OnTyped(bus, event.EvtEnemyKilled, func(_ event.EnemyKilledPayload) { count++ })
	bus.Clear()
	bus.Emit(event.EvtTowerBuilt, event.TowerBuiltPayload{})
	bus.Emit(event.EvtEnemyKilled, event.EnemyKilledPayload{})
	if count != 0 {
		t.Fatalf("expected 0 calls after Clear, got %d", count)
	}
}

func TestEnemyKilledPayload_Fields(t *testing.T) {
	bus := event.NewBus()
	var got event.EnemyKilledPayload
	event.OnTyped(bus, event.EvtEnemyKilled, func(p event.EnemyKilledPayload) {
		got = p
	})
	bus.Emit(event.EvtEnemyKilled, event.EnemyKilledPayload{
		IsBoss: true, KillerID: "warden", GoldValue: 15,
	})
	if !got.IsBoss || got.KillerID != "warden" || got.GoldValue != 15 {
		t.Fatalf("payload mismatch: %+v", got)
	}
}
