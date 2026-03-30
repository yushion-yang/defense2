package autoplay

import "testing"

func TestAnomalyDetector_GoldNegative(t *testing.T) {
	d := NewAnomalyDetector()
	state := &GameState{Tick: 1, Gold: 100, Lives: 20}
	d.Check(state, 10) // 初始化

	state.Tick = 2
	state.Gold = -5
	anomalies := d.Check(state, 10)

	found := false
	for _, a := range anomalies {
		if a.Type == "gold_negative" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected gold_negative anomaly")
	}
}

func TestAnomalyDetector_GoldSpike(t *testing.T) {
	d := NewAnomalyDetector()
	state := &GameState{Tick: 1, Gold: 100, Lives: 20}
	d.Check(state, 10)

	state.Tick = 2
	state.Gold = 700 // +600
	anomalies := d.Check(state, 10)

	found := false
	for _, a := range anomalies {
		if a.Type == "gold_spike" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected gold_spike anomaly")
	}
}

func TestAnomalyDetector_LivesDrop(t *testing.T) {
	d := NewAnomalyDetector()
	state := &GameState{Tick: 1, Gold: 100, Lives: 20}
	d.Check(state, 10)

	state.Tick = 2
	state.Lives = 10 // -10
	anomalies := d.Check(state, 10)

	found := false
	for _, a := range anomalies {
		if a.Type == "lives_drop" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected lives_drop anomaly")
	}
}

func TestAnomalyDetector_DeadEnemyWalking(t *testing.T) {
	d := NewAnomalyDetector()
	state := &GameState{
		Tick: 1, Gold: 100, Lives: 20,
		Enemies: []EnemyInfo{
			{ID: 1, HP: 0, Active: true, Dying: false, X: 100, Y: 100},
		},
	}
	d.Check(state, 10) // 初始化

	state.Tick = 2
	anomalies := d.Check(state, 10)

	found := false
	for _, a := range anomalies {
		if a.Type == "dead_enemy_walking" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected dead_enemy_walking anomaly")
	}
}

func TestAnomalyDetector_EnemyStuck(t *testing.T) {
	d := NewAnomalyDetector()

	// 模拟敌人连续帧不动
	found := false
	for tick := 1; tick <= stuckThreshold+10; tick++ {
		state := &GameState{
			Tick: tick, Gold: 100, Lives: 20,
			Enemies: []EnemyInfo{
				{ID: 1, HP: 100, Speed: 50, Active: true, Dying: false, X: 100, Y: 100},
			},
		}
		anomalies := d.Check(state, 10)
		for _, a := range anomalies {
			if a.Type == "enemy_stuck" {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Error("expected enemy_stuck anomaly within threshold+10 ticks")
	}
}

func TestAnomalyDetector_NoFalsePositives(t *testing.T) {
	d := NewAnomalyDetector()
	state := &GameState{Tick: 1, Gold: 100, Lives: 20}
	d.Check(state, 10)

	state.Tick = 2
	state.Gold = 120
	state.Lives = 19
	anomalies := d.Check(state, 10)

	if len(anomalies) != 0 {
		t.Errorf("expected no anomalies for normal state, got %d", len(anomalies))
	}
}

func TestAnomalySeverity_String(t *testing.T) {
	tests := []struct {
		s    AnomalySeverity
		want string
	}{
		{SeverityLow, "LOW"},
		{SeverityMedium, "MEDIUM"},
		{SeverityHigh, "HIGH"},
		{SeverityCritical, "CRITICAL"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("Severity(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}
