// balance_config_test.go — BalanceConfig 契约测试。
// 验证默认值、JSON 加载、道具完整性、公式合理性及数值范围。
package contracts_test

import (
	"testing"

	"defense2/internal/config"
)

// TestBalanceDefaultValues 验证未加载 JSON 时 GlobalBalance() 返回非空且含合理默认值。
func TestBalanceDefaultValues(t *testing.T) {
	// GlobalBalance() 在 globalBalance == nil 时返回 defaultBalance()。
	// 因为本包的 init（config_rules_test.go）只设置了 dataFS，
	// 没有调用 LoadBalance，所以此时拿到的就是默认值。
	bal := config.GlobalBalance()
	if bal == nil {
		t.Fatal("GlobalBalance() 返回 nil")
	}

	tests := []struct {
		name string
		got  float64
		want float64
	}{
		{"Spawner.HpBase", bal.Spawner.HpBase, 52},
		{"Combat.MinSpeedRatio", bal.Combat.MinSpeedRatio, 0.2},
		{"Economy.KillReward", bal.Economy.KillReward, 15},
		{"Chain.Distance", bal.Chain.Distance, 150},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("%s = %v, want %v", tc.name, tc.got, tc.want)
		}
	}
	if bal.Tower.WavesPerUnlock != 2 {
		t.Errorf("Tower.WavesPerUnlock = %d, want 2", bal.Tower.WavesPerUnlock)
	}
}

// TestBalanceLoadFromJSON 加载 balance.json 并验证值与 JSON 文件一致。
func TestBalanceLoadFromJSON(t *testing.T) {
	bal, err := config.LoadBalance()
	if err != nil {
		t.Fatalf("LoadBalance() 失败: %v", err)
	}
	if bal == nil {
		t.Fatal("LoadBalance() 返回 nil")
	}

	// 抽查 JSON 中的关键字段（与 config/balance.json 对照）
	checks := []struct {
		name string
		got  float64
		want float64
	}{
		{"Spawner.HpBase", bal.Spawner.HpBase, 52},
		{"Spawner.HpPerWave", bal.Spawner.HpPerWave, 21},
		{"Spawner.SpeedBase", bal.Spawner.SpeedBase, 58},
		{"Spawner.SpawnInterval", bal.Spawner.SpawnInterval, 0.6},
		{"Spawner.BossHpMultBase", bal.Spawner.BossHpMultBase, 8},
		{"Economy.KillReward", bal.Economy.KillReward, 15},
		{"Economy.SellRefundRatio", bal.Economy.SellRefundRatio, 0.7},
		{"Combat.MaxDamageAmplify", bal.Combat.MaxDamageAmplify, 0.5},
		{"Combat.MinSpeedRatio", bal.Combat.MinSpeedRatio, 0.2},
		{"Combat.CritMultiplier", bal.Combat.CritMultiplier, 2},
		{"Combat.DefaultProjectileSpeed", bal.Combat.DefaultProjectileSpeed, 300},
		{"Tower.AttackSpeedFloor", bal.Tower.AttackSpeedFloor, 0.1},
		{"Chain.Distance", bal.Chain.Distance, 150},
		{"Chain.StrengthPerTower", bal.Chain.StrengthPerTower, 10},
		{"Split.HpRatio", bal.Split.HpRatio, 0.3},
		{"DeathSpawn.HpRatio", bal.DeathSpawn.HpRatio, 0.2},
		{"Dying.NormalDuration", bal.Dying.NormalDuration, 0.3},
		{"Warden.InitialStrength", bal.Warden.InitialStrength, 100},
		{"Gameplay.StarRatingThreshold", bal.Gameplay.StarRatingThreshold, 0.8},
		{"Gameplay.MultiKillWindow", bal.Gameplay.MultiKillWindow, 1.5},
	}
	for _, tc := range checks {
		if tc.got != tc.want {
			t.Errorf("%s = %v, want %v", tc.name, tc.got, tc.want)
		}
	}

	// int 字段单独检查
	intChecks := []struct {
		name string
		got  int
		want int
	}{
		{"Spawner.EnemiesPerWave", bal.Spawner.EnemiesPerWave, 5},
		{"Spawner.BossEveryNWaves", bal.Spawner.BossEveryNWaves, 5},
		{"Tower.StrengthBuyCost", bal.Tower.StrengthBuyCost, 10},
		{"Tower.WavesPerUnlock", bal.Tower.WavesPerUnlock, 2},
		{"Tower.ChoicesPerUnlock", bal.Tower.ChoicesPerUnlock, 3},
		{"Combat.ScatterBasePellets", bal.Combat.ScatterBasePellets, 3},
		{"Combat.RadialBaseShots", bal.Combat.RadialBaseShots, 3},
		{"Gameplay.MultiKillAnnounce1", bal.Gameplay.MultiKillAnnounce1, 5},
		{"Gameplay.MultiKillAnnounce2", bal.Gameplay.MultiKillAnnounce2, 10},
	}
	for _, tc := range intChecks {
		if tc.got != tc.want {
			t.Errorf("%s = %d, want %d", tc.name, tc.got, tc.want)
		}
	}

	// 字符串字段
	if bal.DeathSpawn.DefaultArch != "normal" {
		t.Errorf("DeathSpawn.DefaultArch = %q, want %q", bal.DeathSpawn.DefaultArch, "normal")
	}
}

// TestBalanceItemsComplete 验证 Items 有 6 个条目且每个字段有效。
func TestBalanceItemsComplete(t *testing.T) {
	bal, err := config.LoadBalance()
	if err != nil {
		t.Fatalf("LoadBalance() 失败: %v", err)
	}

	if len(bal.Items) != 6 {
		t.Fatalf("Items 数量 = %d, want 6", len(bal.Items))
	}

	for i, item := range bal.Items {
		if item.Kind == "" {
			t.Errorf("Items[%d].Kind 为空", i)
		}
		if item.Label == "" {
			t.Errorf("Items[%d].Label 为空", i)
		}
		if item.Boost <= 0 {
			t.Errorf("Items[%d].Boost = %v, want > 0", i, item.Boost)
		}
	}
}

// TestBalanceSpawnerFormula 验证出怪公式产出合理值。
func TestBalanceSpawnerFormula(t *testing.T) {
	bal, err := config.LoadBalance()
	if err != nil {
		t.Fatalf("LoadBalance() 失败: %v", err)
	}

	// HP at wave 1 = HpBase + 1*HpPerWave
	hpWave1 := bal.Spawner.HpBase + 1*bal.Spawner.HpPerWave
	if hpWave1 <= 0 {
		t.Errorf("HP at wave 1 = %.1f, want > 0", hpWave1)
	}

	// Speed at wave 10 = SpeedBase + 10*SpeedPerWave
	speedWave10 := bal.Spawner.SpeedBase + 10*bal.Spawner.SpeedPerWave
	if speedWave10 <= 0 {
		t.Errorf("Speed at wave 10 = %.1f, want > 0", speedWave10)
	}

	// HP should grow with waves
	hpWave20 := bal.Spawner.HpBase + 20*bal.Spawner.HpPerWave
	if hpWave20 <= hpWave1 {
		t.Errorf("HP at wave 20 (%.1f) should > HP at wave 1 (%.1f)", hpWave20, hpWave1)
	}
}

// TestBalanceCombatRanges 验证战斗参数在合理范围内。
func TestBalanceCombatRanges(t *testing.T) {
	bal, err := config.LoadBalance()
	if err != nil {
		t.Fatalf("LoadBalance() 失败: %v", err)
	}

	c := bal.Combat

	if c.MinSpeedRatio <= 0 || c.MinSpeedRatio >= 1 {
		t.Errorf("MinSpeedRatio = %v, want in (0, 1)", c.MinSpeedRatio)
	}
	if c.MaxDamageAmplify <= 0 || c.MaxDamageAmplify >= 5 {
		t.Errorf("MaxDamageAmplify = %v, want in (0, 5)", c.MaxDamageAmplify)
	}
	if c.CritMultiplier < 1 {
		t.Errorf("CritMultiplier = %v, want >= 1", c.CritMultiplier)
	}
	if c.DotTickInterval <= 0 {
		t.Errorf("DotTickInterval = %v, want > 0", c.DotTickInterval)
	}
	if c.DefaultProjectileSpeed <= 0 {
		t.Errorf("DefaultProjectileSpeed = %v, want > 0", c.DefaultProjectileSpeed)
	}
	if c.DefaultProjectileRadius <= 0 {
		t.Errorf("DefaultProjectileRadius = %v, want > 0", c.DefaultProjectileRadius)
	}
}

// TestBalanceGameplayRanges 验证游戏性参数在合理范围内。
func TestBalanceGameplayRanges(t *testing.T) {
	bal, err := config.LoadBalance()
	if err != nil {
		t.Fatalf("LoadBalance() 失败: %v", err)
	}

	g := bal.Gameplay

	if g.StarRatingThreshold <= 0 || g.StarRatingThreshold >= 1 {
		t.Errorf("StarRatingThreshold = %v, want in (0, 1)", g.StarRatingThreshold)
	}
	if g.MultiKillWindow <= 0 {
		t.Errorf("MultiKillWindow = %v, want > 0", g.MultiKillWindow)
	}
	if g.MultiKillAnnounce1 >= g.MultiKillAnnounce2 {
		t.Errorf("MultiKillAnnounce1 (%d) should < MultiKillAnnounce2 (%d)",
			g.MultiKillAnnounce1, g.MultiKillAnnounce2)
	}
}
