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

func TestAnomalyDetector_WardenRangeLimited(t *testing.T) {
	d := NewAnomalyDetector()

	// 模拟：地图宽 2400（两屏），但战灵只在 X=200~500 活动
	for tick := 1; tick <= 2000; tick++ {
		x := 200.0 + float64(tick%300) // 200~500 之间
		state := &GameState{
			Tick: tick, Gold: 100, Lives: 20,
			WardenReady: true,
			WardenX:     x,
			WardenY:     270,
			MapPixelW:   2400,
			MapPixelH:   540,
		}
		anomalies := d.Check(state, 10)

		// 第 1800 帧之后应检测到
		if tick > 1800 {
			for _, a := range anomalies {
				if a.Type == "warden_range_limited_x" {
					return // 检测成功
				}
			}
		}
	}
	t.Error("expected warden_range_limited_x anomaly for large map with confined warden")
}

func TestAnomalyDetector_WardenRangeOK(t *testing.T) {
	d := NewAnomalyDetector()

	// 模拟：地图宽 2400，战灵在 100~2100 之间巡逻
	// 700帧内从 100 线性移到 2100，跨度 2000 > 960 阈值
	for tick := 1; tick <= 700; tick++ {
		x := 100.0 + 2000.0*float64(tick)/700.0 // 100 → 2100
		state := &GameState{
			Tick: tick, Gold: 100, Lives: 20,
			WardenReady: true,
			WardenX:     x,
			WardenY:     270,
			MapPixelW:   2400,
			MapPixelH:   540,
		}
		anomalies := d.Check(state, 10)
		for _, a := range anomalies {
			if a.Type == "warden_range_limited_x" {
				t.Error("should not flag warden range when it covers the map")
				return
			}
		}
	}
}

func TestAnomalyDetector_SmallMapNoCheck(t *testing.T) {
	d := NewAnomalyDetector()

	// 地图只有一屏大小(1200x540)，不应触发范围检查
	for tick := 1; tick <= 700; tick++ {
		state := &GameState{
			Tick: tick, Gold: 100, Lives: 20,
			WardenReady: true,
			WardenX:     300,
			WardenY:     200,
			MapPixelW:   1200,
			MapPixelH:   540,
		}
		anomalies := d.Check(state, 10)
		for _, a := range anomalies {
			if a.Type == "warden_range_limited_x" || a.Type == "warden_range_limited_y" {
				t.Error("should not check warden range on single-screen maps")
				return
			}
		}
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
