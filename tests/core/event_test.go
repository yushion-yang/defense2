package core_test

import (
	"testing"

	"defense2/internal/core/event"
)

func TestPoolPickTiered(t *testing.T) {
	pool := event.NewPool(event.DefaultAllyEvents())
	picks := pool.PickTiered(5)
	if len(picks) < 2 {
		t.Fatalf("应至少抽出 2 个事件，实际 %d", len(picks))
	}
	if len(picks) > 3 {
		t.Fatalf("最多 3 个事件，实际 %d", len(picks))
	}
}

func TestPoolPickByWaveFilter(t *testing.T) {
	events := []event.Event{
		{ID: "early", Tier: 1, Weight: 1, MinWave: 1, MaxWave: 5},
		{ID: "late", Tier: 1, Weight: 1, MinWave: 10, MaxWave: 20},
	}
	pool := event.NewPool(events)

	// 波次 3：只应抽到 early
	picks := pool.Pick(2, 3)
	for _, p := range picks {
		if p.ID == "late" {
			t.Fatal("波次 3 不应抽到 late 事件")
		}
	}

	// 波次 15：只应抽到 late
	picks = pool.Pick(2, 15)
	for _, p := range picks {
		if p.ID == "early" {
			t.Fatal("波次 15 不应抽到 early 事件")
		}
	}
}

func TestRewardWaves(t *testing.T) {
	waves := event.RewardWaves()
	if len(waves) == 0 {
		t.Fatal("奖励波次不应为空")
	}
	// 应包含常见检查点
	found4 := false
	for _, w := range waves {
		if w == 4 {
			found4 = true
		}
	}
	if !found4 {
		t.Fatal("奖励波次应包含第 4 波")
	}
}
