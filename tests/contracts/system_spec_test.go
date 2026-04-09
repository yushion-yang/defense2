// system_spec_test.go — 系统规格契约测试。
// 验证 config/systems/*.json 规格与代码行为一致。
package contracts

import (
	"testing"

	"defense2/internal/config"
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
