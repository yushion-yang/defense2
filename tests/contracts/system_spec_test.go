// system_spec_test.go — 系统规格契约测试。
// 验证 config/systems/*.json 规格与代码行为一致。
package contracts

import (
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/buff"
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
	// 自洽验证：Calc(wave) == Base + PerWave * wave
	expectedWave := c.WaveBonus.Base + c.WaveBonus.PerWave*5
	if got := c.WaveBonus.Calc(5); got != expectedWave {
		t.Errorf("campaign wave 5 bonus = %d, want %d (base=%d + perWave=%d * 5)", got, expectedWave, c.WaveBonus.Base, c.WaveBonus.PerWave)
	}
	expectedPerfect := c.PerfectBonus.Base + c.PerfectBonus.PerWave*5
	if got := c.PerfectBonus.Calc(5); got != expectedPerfect {
		t.Errorf("campaign wave 5 perfect = %d, want %d", got, expectedPerfect)
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

// TestEconomySpec_AllModesBonusPositive 验证所有模式的波次奖金和完美奖金为正。
func TestEconomySpec_AllModesBonusPositive(t *testing.T) {
	spec := config.GlobalEconomySpec()
	for name, m := range spec.Modes {
		t.Run(name, func(t *testing.T) {
			if got := m.WaveBonus.Calc(1); got <= 0 {
				t.Errorf("WaveBonus.Calc(1) = %d, want > 0", got)
			}
			if got := m.PerfectBonus.Calc(1); got <= 0 {
				t.Errorf("PerfectBonus.Calc(1) = %d, want > 0", got)
			}
		})
	}
}

// TestEconomySpec_AllModesSelfConsistentFormula 验证 Calc(w) == Base + PerWave * w。
func TestEconomySpec_AllModesSelfConsistentFormula(t *testing.T) {
	spec := config.GlobalEconomySpec()
	waves := []int{0, 1, 5, 10, 20}
	for name, m := range spec.Modes {
		t.Run(name, func(t *testing.T) {
			for _, w := range waves {
				expected := m.WaveBonus.Base + m.WaveBonus.PerWave*w
				if got := m.WaveBonus.Calc(w); got != expected {
					t.Errorf("wave %d: WaveBonus.Calc=%d, want %d", w, got, expected)
				}
			}
		})
	}
}

// TestEconomySpec_BossRushHasFields 验证 bossRush 模式有特有字段。
func TestEconomySpec_BossRushHasFields(t *testing.T) {
	spec := config.GlobalEconomySpec()
	br, ok := spec.Modes["bossRush"]
	if !ok {
		t.Fatal("缺少 bossRush 模式")
	}
	if br.TotalBosses <= 0 {
		t.Errorf("bossRush.TotalBosses = %d, want > 0", br.TotalBosses)
	}
	if br.IntermissionSecs <= 0 {
		t.Errorf("bossRush.IntermissionSecs = %.1f, want > 0", br.IntermissionSecs)
	}
}

// TestEconomySpec_TimedHasTargetSeconds 验证 timed 模式有目标时长。
func TestEconomySpec_TimedHasTargetSeconds(t *testing.T) {
	spec := config.GlobalEconomySpec()
	tm, ok := spec.Modes["timed"]
	if !ok {
		t.Fatal("缺少 timed 模式")
	}
	if tm.TargetSeconds <= 0 {
		t.Errorf("timed.TargetSeconds = %.1f, want > 0", tm.TargetSeconds)
	}
}

// ═══════════════════════════════════════
// 伤害管线规格
// ═══════════════════════════════════════

func TestDamagePipelineSpec_BossPercentCap(t *testing.T) {
	sc := config.GlobalSpawnerConfig()
	if sc.Boss.PercentHpCap <= 0 || sc.Boss.PercentHpCap > 0.2 {
		t.Errorf("boss.percentHpCap = %f, want (0, 0.2]", sc.Boss.PercentHpCap)
	}
}

// ═══════════════════════════════════════
// Boss 机制规格
// ═══════════════════════════════════════

func TestBossSpec_EntranceDelay(t *testing.T) {
	sc := config.GlobalSpawnerConfig()
	if sc.Boss.EntranceDelay <= 0 {
		t.Error("bossEntranceDelay should > 0")
	}
}

func TestBossSpec_HpMultBase(t *testing.T) {
	sc := config.GlobalSpawnerConfig()
	if sc.Boss.HpMultBase <= 0 {
		t.Error("bossHpMultBase should > 0")
	}
}

func TestBossSpec_EveryNWaves(t *testing.T) {
	sc := config.GlobalSpawnerConfig()
	if sc.Boss.EveryNWaves <= 0 {
		t.Error("bossEveryNWaves should > 0")
	}
}

func TestBossSpec_RadiusScale(t *testing.T) {
	sc := config.GlobalSpawnerConfig()
	if sc.Boss.RadiusScale <= 1 {
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
	sc := config.GlobalSpawnerConfig()
	tiers := sc.WaveBuffs.Tiers
	if len(tiers) == 0 {
		t.Fatal("WaveBuffs.Tiers is empty")
	}
	// minWaves 应递增
	for i := 1; i < len(tiers); i++ {
		if tiers[i].MinWave <= tiers[i-1].MinWave {
			t.Errorf("Tiers[%d].MinWave=%d should > Tiers[%d].MinWave=%d",
				i, tiers[i].MinWave, i-1, tiers[i-1].MinWave)
		}
	}
}

func TestWaveSpec_HpPerWavePositive(t *testing.T) {
	sc := config.GlobalSpawnerConfig()
	if sc.Scaling.HpPerWave <= 0 {
		t.Error("hpPerWave should > 0")
	}
	if sc.Scaling.SpeedPerWave <= 0 {
		t.Error("speedPerWave should > 0")
	}
}

// ═══════════════════════════════════════
// Buff 堆叠规格
// ═══════════════════════════════════════

func TestBuffStackSpec_DamageUpAdditive(t *testing.T) {
	// 已在 full_coverage_contracts_test.go TestBuffDamageUpUsesAdditive 中覆盖
	// 此处验证 cap > 0
	rules := buff.GlobalRules()
	r, ok := rules["damageUp"]
	if !ok {
		t.Fatal("缺少 damageUp 规则")
	}
	if r.Cap <= 0 {
		t.Error("damageUp cap should > 0")
	}
}

func TestBuffStackSpec_SlowCap(t *testing.T) {
	rules := buff.GlobalRules()
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

func TestTowerRandomizeSpec_TierNames(t *testing.T) {
	want := []string{"S", "B", "D"}
	got := config.TierNames
	if len(got) != len(want) {
		t.Fatalf("TierNames len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("TierNames[%d] = %q, want %q", i, got[i], want[i])
		}
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

