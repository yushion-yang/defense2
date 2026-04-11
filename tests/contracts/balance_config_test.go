// balance_config_test.go — BalanceConfig 契约测试。
// 验证结构完整性、数值范围、自洽性。不硬编码配置值，配置文件是唯一真相源。
package contracts_test

import (
	"testing"

	"defense2/internal/config"
)

// TestBalanceFieldsNonZero 验证未加载 JSON 时 GlobalBalance() 返回非空且含合理默认值。
func TestBalanceFieldsNonZero(t *testing.T) {
	// GlobalBalance() 在 globalBalance == nil 时返回 defaultBalance()。
	// 因为本包的 init（config_rules_test.go）只设置了 dataFS，
	// 没有调用 LoadBalance，所以此时拿到的就是默认值。
	bal := config.GlobalBalance()
	if bal == nil {
		t.Fatal("GlobalBalance() 返回 nil")
	}

	checks := []struct {
		name  string
		check func() bool
		msg   string
	}{
		{"Combat.MinSpeedRatio", func() bool { return bal.Combat.MinSpeedRatio > 0 && bal.Combat.MinSpeedRatio < 1 }, "should be in (0, 1)"},
		{"Economy.KillReward", func() bool { return bal.Economy.KillReward > 0 }, "should be > 0"},
		{"Chain.Distance", func() bool { return bal.Chain.Distance > 0 }, "should be > 0"},
		{"Tower.WavesPerUnlock", func() bool { return bal.Tower.WavesPerUnlock > 0 }, "should be > 0"},
	}
	for _, tc := range checks {
		if !tc.check() {
			t.Errorf("%s: %s", tc.name, tc.msg)
		}
	}
}

// TestBalanceLoadedFieldsValid 加载 balance.json 并验证所有字段在合理范围内（不硬编码具体值）。
func TestBalanceLoadedFieldsValid(t *testing.T) {
	bal, err := config.LoadBalance()
	if err != nil {
		t.Fatalf("LoadBalance() 失败: %v", err)
	}
	if bal == nil {
		t.Fatal("LoadBalance() 返回 nil")
	}

	checks := []struct {
		name  string
		check func() bool
		msg   string
	}{
		// Economy
		{"Economy.KillReward", func() bool { return bal.Economy.KillReward > 0 }, "should be > 0"},
		{"Economy.SellRefundRatio", func() bool { return bal.Economy.SellRefundRatio > 0 && bal.Economy.SellRefundRatio < 1 }, "should be in (0, 1)"},

		// Combat
		{"Combat.CritMultiplier", func() bool { return bal.Combat.CritMultiplier > 1 }, "should be > 1"},
		{"Combat.MinSpeedRatio", func() bool { return bal.Combat.MinSpeedRatio > 0 && bal.Combat.MinSpeedRatio < 1 }, "should be in (0, 1)"},
		{"Combat.MaxDamageAmplify", func() bool { return bal.Combat.MaxDamageAmplify > 0 }, "should be > 0"},
		{"Combat.DefaultProjectileSpeed", func() bool { return bal.Combat.DefaultProjectileSpeed > 0 }, "should be > 0"},
		{"Combat.DotTickInterval", func() bool { return bal.Combat.DotTickInterval > 0 }, "should be > 0"},

		// Tower
		{"Tower.WavesPerUnlock", func() bool { return bal.Tower.WavesPerUnlock > 0 }, "should be > 0"},
		{"Tower.StrengthBuyCost", func() bool { return bal.Tower.StrengthBuyCost > 0 }, "should be > 0"},
		{"Tower.ChoicesPerUnlock", func() bool { return bal.Tower.ChoicesPerUnlock > 0 }, "should be > 0"},
		{"Tower.AttackSpeedFloor", func() bool { return bal.Tower.AttackSpeedFloor > 0 }, "should be > 0"},

		// Chain
		{"Chain.Distance", func() bool { return bal.Chain.Distance > 0 }, "should be > 0"},
		{"Chain.StrengthPerTower", func() bool { return bal.Chain.StrengthPerTower > 0 }, "should be > 0"},

		// Split
		{"Split.HpRatio", func() bool { return bal.Split.HpRatio > 0 && bal.Split.HpRatio < 1 }, "should be in (0, 1)"},

		// DeathSpawn
		{"DeathSpawn.HpRatio", func() bool { return bal.DeathSpawn.HpRatio > 0 && bal.DeathSpawn.HpRatio < 1 }, "should be in (0, 1)"},
		{"DeathSpawn.DefaultArch", func() bool { return bal.DeathSpawn.DefaultArch != "" }, "should not be empty"},

		// Dying
		{"Dying.NormalDuration", func() bool { return bal.Dying.NormalDuration > 0 }, "should be > 0"},

		// Warden
		{"Warden.InitialStrength", func() bool { return bal.Warden.InitialStrength > 0 }, "should be > 0"},

		// Gameplay
		{"Gameplay.StarRatingThreshold", func() bool {
			return bal.Gameplay.StarRatingThreshold > 0 && bal.Gameplay.StarRatingThreshold < 1
		}, "should be in (0, 1)"},
		{"Gameplay.MultiKillWindow", func() bool { return bal.Gameplay.MultiKillWindow > 0 }, "should be > 0"},
	}
	for _, tc := range checks {
		if !tc.check() {
			t.Errorf("%s: %s", tc.name, tc.msg)
		}
	}
}

// TestBalanceItemsComplete 验证 Items 至少有 3 个条目且每个字段有效。
func TestBalanceItemsComplete(t *testing.T) {
	bal, err := config.LoadBalance()
	if err != nil {
		t.Fatalf("LoadBalance() 失败: %v", err)
	}

	if len(bal.Items) < 3 {
		t.Fatalf("Items 数量 = %d, want >= 3", len(bal.Items))
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

// TestSpawnerConfigFormula 验证出怪公式产出合理值。
func TestSpawnerConfigFormula(t *testing.T) {
	sc := config.GlobalSpawnerConfig()

	// HP at wave 1 = HpBase + 1*HpPerWave
	hpWave1 := sc.Scaling.HpBase + 1*sc.Scaling.HpPerWave
	if hpWave1 <= 0 {
		t.Errorf("HP at wave 1 = %.1f, want > 0", hpWave1)
	}

	// Speed at wave 10 = SpeedBase + 10*SpeedPerWave
	speedWave10 := sc.Scaling.SpeedBase + 10*sc.Scaling.SpeedPerWave
	if speedWave10 <= 0 {
		t.Errorf("Speed at wave 10 = %.1f, want > 0", speedWave10)
	}

	// HP should grow with waves
	hpWave20 := sc.Scaling.HpBase + 20*sc.Scaling.HpPerWave
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
}
