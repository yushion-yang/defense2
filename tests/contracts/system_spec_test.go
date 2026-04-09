// system_spec_test.go — 系统规格契约测试。
// 验证 config/systems/*.json 规格与代码行为一致。
package contracts

import (
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/tower"
)

// ═══════════════════════════════════════
// 经济系统规格
// ═══════════════════════════════════════

func TestEconomySpec_ModesExist(t *testing.T) {
	spec := config.GlobalEconomySpec()
	if spec == nil || spec.Modes == nil {
		t.Fatal("经济规格加载失败")
	}
	required := []string{"campaign", "endless", "timed", "bossRush"}
	for _, m := range required {
		if _, ok := spec.Modes[m]; !ok {
			t.Errorf("缺少模式 %q 的经济配置", m)
		}
	}
}

func TestEconomySpec_CampaignBonus(t *testing.T) {
	spec := config.GlobalEconomySpec()
	c := spec.Modes["campaign"]
	// wave 5: 12 + 5*4 = 32
	if got := c.WaveBonus.Calc(5); got != 32 {
		t.Errorf("campaign wave 5 bonus = %d, want 32", got)
	}
	// perfect: 8 + 5*2 = 18
	if got := c.PerfectBonus.Calc(5); got != 18 {
		t.Errorf("campaign wave 5 perfect = %d, want 18", got)
	}
}

func TestEconomySpec_EndlessHigherThanCampaign(t *testing.T) {
	spec := config.GlobalEconomySpec()
	c := spec.Modes["campaign"]
	e := spec.Modes["endless"]
	for _, wave := range []int{5, 10, 15, 20} {
		if e.WaveBonus.Calc(wave) <= c.WaveBonus.Calc(wave) {
			t.Errorf("wave %d: endless bonus (%d) should > campaign (%d)",
				wave, e.WaveBonus.Calc(wave), c.WaveBonus.Calc(wave))
		}
	}
}

// ═══════════════════════════════════════
// 伤害管线规格
// ═══════════════════════════════════════

func TestDamagePipelineSpec_BossPercentCap(t *testing.T) {
	bal := config.GlobalBalance()
	if bal.Combat.BossPercentHpCap <= 0 || bal.Combat.BossPercentHpCap > 0.2 {
		t.Errorf("bossPercentHpCap = %f, want (0, 0.2]", bal.Combat.BossPercentHpCap)
	}
}

func TestDamagePipelineSpec_DamageDownFloor(t *testing.T) {
	bal := config.GlobalBalance()
	if bal.Combat.DamageDownFloor <= 0 || bal.Combat.DamageDownFloor > 0.5 {
		t.Errorf("damageDownFloor = %f, want (0, 0.5]", bal.Combat.DamageDownFloor)
	}
}

func TestDamagePipelineSpec_MaxDamageAmplify(t *testing.T) {
	bal := config.GlobalBalance()
	if bal.Combat.MaxDamageAmplify <= 0 {
		t.Error("maxDamageAmplify should > 0")
	}
}

// ═══════════════════════════════════════
// Boss 机制规格
// ═══════════════════════════════════════

func TestBossSpec_EntranceDelay(t *testing.T) {
	bal := config.GlobalBalance()
	if bal.Spawner.BossEntranceDelay <= 0 {
		t.Error("bossEntranceDelay should > 0")
	}
}

func TestBossSpec_HpMultBase(t *testing.T) {
	bal := config.GlobalBalance()
	if bal.Spawner.BossHpMultBase <= 0 {
		t.Error("bossHpMultBase should > 0")
	}
}

func TestBossSpec_EveryNWaves(t *testing.T) {
	bal := config.GlobalBalance()
	if bal.Spawner.BossEveryNWaves <= 0 {
		t.Error("bossEveryNWaves should > 0")
	}
}

func TestBossSpec_RadiusScale(t *testing.T) {
	bal := config.GlobalBalance()
	if bal.Spawner.BossRadiusScale <= 1 {
		t.Error("bossRadiusScale should > 1")
	}
}

// ═══════════════════════════════════════
// CC 系统规格
// ═══════════════════════════════════════

func TestCCSpec_MinSpeedRatio(t *testing.T) {
	bal := config.GlobalBalance()
	if bal.Combat.MinSpeedRatio <= 0 || bal.Combat.MinSpeedRatio >= 1 {
		t.Errorf("minSpeedRatio = %f, want (0, 1)", bal.Combat.MinSpeedRatio)
	}
}

// ═══════════════════════════════════════
// 属性管线规格
// ═══════════════════════════════════════

func TestAttributeSpec_AttackSpeedFloor(t *testing.T) {
	bal := config.GlobalBalance()
	if bal.Tower.AttackSpeedFloor <= 0 {
		t.Error("attackSpeedFloor should > 0")
	}
}

func TestAttributeSpec_StrengthBuyCost(t *testing.T) {
	bal := config.GlobalBalance()
	if bal.Tower.StrengthBuyCost <= 0 {
		t.Error("strengthBuyCost should > 0")
	}
}

// ═══════════════════════════════════════
// 波次出怪规格
// ═══════════════════════════════════════

func TestWaveSpec_BuffTiersConsistent(t *testing.T) {
	bal := config.GlobalBalance()
	if len(bal.Spawner.BuffMinWaves) != len(bal.Spawner.BuffMaxBuffs) {
		t.Errorf("BuffMinWaves len=%d != BuffMaxBuffs len=%d",
			len(bal.Spawner.BuffMinWaves), len(bal.Spawner.BuffMaxBuffs))
	}
	// minWaves 应递增
	for i := 1; i < len(bal.Spawner.BuffMinWaves); i++ {
		if bal.Spawner.BuffMinWaves[i] <= bal.Spawner.BuffMinWaves[i-1] {
			t.Errorf("BuffMinWaves[%d]=%d should > BuffMinWaves[%d]=%d",
				i, bal.Spawner.BuffMinWaves[i], i-1, bal.Spawner.BuffMinWaves[i-1])
		}
	}
}

func TestWaveSpec_HpPerWavePositive(t *testing.T) {
	bal := config.GlobalBalance()
	if bal.Spawner.HpPerWave <= 0 {
		t.Error("hpPerWave should > 0")
	}
	if bal.Spawner.SpeedPerWave <= 0 {
		t.Error("speedPerWave should > 0")
	}
}

// ═══════════════════════════════════════
// Buff 堆叠规格
// ═══════════════════════════════════════

func TestBuffStackSpec_DamageUpAdditive(t *testing.T) {
	// 已在 full_coverage_contracts_test.go TestBuffDamageUpUsesAdditive 中覆盖
	// 此处验证 cap > 0
	rules := buff.DefaultStackRules
	r, ok := rules["damageUp"]
	if !ok {
		t.Fatal("缺少 damageUp 规则")
	}
	if r.Cap <= 0 {
		t.Error("damageUp cap should > 0")
	}
}

func TestBuffStackSpec_SlowCap(t *testing.T) {
	rules := buff.DefaultStackRules
	r, ok := rules["slow"]
	if !ok {
		t.Fatal("缺少 slow 规则")
	}
	if r.Cap <= 0 || r.Cap > 1 {
		t.Errorf("slow cap = %f, want (0, 1]", r.Cap)
	}
}

// ═══════════════════════════════════════
// P2: 塔随机化规格
// ═══════════════════════════════════════

func TestTowerRandomizeSpec_TierBudget(t *testing.T) {
	// TierBudget 定义在 randomize.go，S=4+A=3 > 6 → 不可能 S+A
	// 此处仅验证 JSON 和代码的值一致
	if tower.TierBudget != 6 {
		t.Errorf("TierBudget = %d, want 6", tower.TierBudget)
	}
}

// ═══════════════════════════════════════
// P2: 弹射物默认值
// ═══════════════════════════════════════

func TestProjectileSpec_DefaultsPositive(t *testing.T) {
	bal := config.GlobalBalance().Combat
	if bal.DefaultProjectileSpeed <= 0 {
		t.Error("defaultProjectileSpeed should > 0")
	}
	if bal.DefaultProjectileRadius <= 0 {
		t.Error("defaultProjectileRadius should > 0")
	}
}

