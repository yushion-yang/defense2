// wave_economy_contracts_test.go — 波次与经济系统契约测试。
package contracts_test

import (
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/economy"
	"defense2/internal/core/enemy"
)

// ── 经济公式 ──

func TestKillGoldPositive(t *testing.T) {
	cfg := economy.DefaultConfig()
	if cfg.KillGold() <= 0 {
		t.Errorf("KillGold()=%d 应 > 0", cfg.KillGold())
	}
}

func TestWaveBonusIncreases(t *testing.T) {
	cfg := economy.DefaultConfig()
	prev := 0
	for wave := 1; wave <= 25; wave++ {
		bonus := cfg.WaveCompleteGold(wave)
		if bonus <= prev {
			t.Errorf("wave=%d: WaveCompleteGold=%d 应 > 前一波 %d", wave, bonus, prev)
		}
		prev = bonus
	}
}

func TestSellRefundLessThanCost(t *testing.T) {
	cfg := economy.DefaultConfig()
	costs := []int{50, 60, 70, 80, 100}
	for _, cost := range costs {
		refund := cfg.SellRefund(cost)
		if refund >= cost {
			t.Errorf("SellRefund(%d)=%d 应 < 原价", cost, refund)
		}
		if refund <= 0 {
			t.Errorf("SellRefund(%d)=%d 应 > 0", cost, refund)
		}
	}
}

// ── 难度缩放 ──

func TestDifficultyModesExist(t *testing.T) {
	modes, _, err := config.LoadDifficultyModes()
	if err != nil {
		t.Fatalf("加载难度配置失败: %v", err)
	}
	expected := []string{"easy", "normal", "hard", "extreme"}
	for _, id := range expected {
		if _, ok := modes[id]; !ok {
			t.Errorf("难度 %q 不存在", id)
		}
	}
}

func TestDifficultyHPScaleOrdering(t *testing.T) {
	modes, _, err := config.LoadDifficultyModes()
	if err != nil {
		t.Fatalf("加载难度配置失败: %v", err)
	}
	order := []string{"easy", "normal", "hard", "extreme"}
	prevHP := 0.0
	for _, id := range order {
		m := modes[id]
		if m.HPScale <= prevHP {
			t.Errorf("难度 %q HPScale=%.2f 应 > 前一级 %.2f", id, m.HPScale, prevHP)
		}
		prevHP = m.HPScale
	}
}

func TestDifficultyRewardScaleOrdering(t *testing.T) {
	modes, _, err := config.LoadDifficultyModes()
	if err != nil {
		t.Fatalf("加载难度配置失败: %v", err)
	}
	// 奖励应递减：easy 最高，extreme 最低
	order := []string{"easy", "normal", "hard", "extreme"}
	prevReward := 999.0
	for _, id := range order {
		m := modes[id]
		if m.RewardScale >= prevReward {
			t.Errorf("难度 %q RewardScale=%.2f 应 < 前一级 %.2f", id, m.RewardScale, prevReward)
		}
		prevReward = m.RewardScale
	}
}

func TestDifficultyStartGoldOrdering(t *testing.T) {
	modes, _, err := config.LoadDifficultyModes()
	if err != nil {
		t.Fatalf("加载难度配置失败: %v", err)
	}
	// 初始金币应递减
	order := []string{"easy", "normal", "hard", "extreme"}
	prevGold := 9999
	for _, id := range order {
		m := modes[id]
		if m.StartGold >= prevGold {
			t.Errorf("难度 %q StartGold=%d 应 < 前一级 %d", id, m.StartGold, prevGold)
		}
		prevGold = m.StartGold
	}
}

// ── 波次公式 ──

// TestSpawnerEnemyCountPositive 调用实际 spawner 代码验证每波出怪数 > 0。
func TestSpawnerEnemyCountPositive(t *testing.T) {
	s := enemy.NewSpawner(nil, 25)
	for wave := 1; wave <= 25; wave++ {
		count := s.EnemyCountForWave(wave)
		if count <= 0 {
			t.Errorf("wave=%d: enemyCount=%d 应 > 0", wave, count)
		}
	}
}

// TestSpawnerEnemyCountIncreases 验证每波出怪数递增。
func TestSpawnerEnemyCountIncreases(t *testing.T) {
	s := enemy.NewSpawner(nil, 25)
	prev := 0
	for wave := 1; wave <= 25; wave++ {
		count := s.EnemyCountForWave(wave)
		if count <= prev {
			t.Errorf("wave=%d: count=%d 应 > 前一波 %d", wave, count, prev)
		}
		prev = count
	}
}

// ── 塔费用 ──

func TestAtLeastOneTowerAffordablePerDifficulty(t *testing.T) {
	towers, err := config.LoadAllTowers()
	if err != nil {
		t.Fatalf("加载塔配置失败: %v", err)
	}
	modes, _, err := config.LoadDifficultyModes()
	if err != nil {
		t.Fatalf("加载难度配置失败: %v", err)
	}

	for id, diff := range modes {
		affordable := 0
		for _, tw := range towers {
			if tw.BuildCost <= diff.StartGold {
				affordable++
			}
		}
		if affordable == 0 {
			t.Errorf("难度 %q (初始金=%d): 没有任何塔能用初始金币建造", id, diff.StartGold)
		}
	}
}
