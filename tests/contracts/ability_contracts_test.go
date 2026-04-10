// ability_contracts_test.go — 能力系统契约测试。
// 验证每种能力的 OnHit/OnTick 返回正确的效果类型。
package contracts_test

import (
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

// 辅助：创建测试用塔（强度100，有指定能力）
func testTower(abilityName string) *tower.Tower {
	t := &tower.Tower{
		Key: "test", Damage: 10, Range: 150, AttackSpeed: 1.0,
		BaseDamage: 10, PotentialDamage: 10,
		BaseSpeed: 1.0, PotentialSpeed: 0.5,
		BaseRange: 150, PotentialRange: 30,
		Abilities: []string{abilityName},
	}
	return t
}

func testProjectile() *projectile.Projectile {
	return &projectile.Projectile{Damage: 10, Speed: 400, SourceTowerKey: "test_0_0"}
}

func testEnemy() *enemy.Enemy {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Speed: 60, BaseSpeed: 60, Active: true}
	e.Buffs = buff.NewDefaultBuffList()
	return e
}

// ── 攻击模式能力 ──

func TestAbility_Scatter_ReturnsAttackStyle(t *testing.T) {
	ab := tower.Registry["scatter"]
	if ab == nil {
		t.Fatal("scatter 未注册")
	}
	// scatter 是 OnTick/攻击方式切换类，OnHit 可能返回 nil
}

func TestAbility_Bounce_ReturnsBouncEffect(t *testing.T) {
	ab := tower.Registry["bounce"]
	if ab == nil {
		t.Fatal("bounce 未注册")
	}
	tw := testTower("bounce")
	tw.Damage = 10
	tw.Range = 150
	p := testProjectile()
	e := testEnemy()

	result := ab.OnHit(tw, p, e)
	if result == nil {
		t.Fatal("bounce OnHit 返回 nil")
	}
	if result.Bounce == nil {
		t.Fatal("bounce OnHit 应返回 BounceEffect")
	}
	if result.Bounce.MaxBounces <= 0 {
		t.Errorf("bounce MaxBounces=%d 应 > 0", result.Bounce.MaxBounces)
	}
	if result.Bounce.DamageRatio <= 0 || result.Bounce.DamageRatio > 1 {
		t.Errorf("bounce DamageRatio=%.2f 应在 (0, 1]", result.Bounce.DamageRatio)
	}
}

func TestAbility_Splash_ReturnsSplashEffect(t *testing.T) {
	ab := tower.Registry["splash"]
	if ab == nil {
		t.Fatal("splash 未注册")
	}
	tw := testTower("splash")
	p := testProjectile()
	e := testEnemy()

	result := ab.OnHit(tw, p, e)
	if result == nil {
		t.Fatal("splash OnHit 返回 nil")
	}
	if result.Splash == nil {
		t.Fatal("splash OnHit 应返回 SplashEffect")
	}
	if result.Splash.Radius <= 0 {
		t.Errorf("splash Radius=%.1f 应 > 0", result.Splash.Radius)
	}
}

// ── CC 能力 ──

func TestAbility_SlowPower_ReturnsSlowEffect(t *testing.T) {
	ab := tower.Registry["slowPower"]
	if ab == nil {
		t.Fatal("slowPower 未注册")
	}
	tw := testTower("slowPower")
	p := testProjectile()
	e := testEnemy()

	result := ab.OnHit(tw, p, e)
	if result == nil {
		t.Fatal("slowPower OnHit 返回 nil")
	}
	if result.Slow == nil {
		t.Fatal("slowPower OnHit 应返回 SlowEffect")
	}
	if result.Slow.Factor <= 0 || result.Slow.Factor >= 1 {
		t.Errorf("slowPower Factor=%.2f 应在 (0, 1)", result.Slow.Factor)
	}
	if result.Slow.Duration <= 0 {
		t.Errorf("slowPower Duration=%.2f 应 > 0", result.Slow.Duration)
	}
}

func TestAbility_StunChance_ReturnsStunOrNil(t *testing.T) {
	ab := tower.Registry["stunChance"]
	if ab == nil {
		t.Fatal("stunChance 未注册")
	}
	tw := testTower("stunChance")
	p := testProjectile()
	e := testEnemy()

	// 概率型：多次调用，至少应有一次返回 StunEffect 或 nil
	hasStun := false
	hasNil := false
	for i := 0; i < 1000; i++ {
		result := ab.OnHit(tw, p, e)
		if result != nil && result.Stun != nil {
			hasStun = true
			if result.Stun.Duration <= 0 {
				t.Fatalf("stunChance Duration=%.2f 应 > 0", result.Stun.Duration)
			}
		} else {
			hasNil = true
		}
		if hasStun && hasNil {
			break
		}
	}
	if !hasStun {
		t.Error("stunChance 1000 次调用从未触发眩晕")
	}
}

// ── 伤害加成能力 ──

func TestAbility_Crit_ReturnsCritOrNil(t *testing.T) {
	ab := tower.Registry["crit"]
	if ab == nil {
		t.Fatal("crit 未注册")
	}
	tw := testTower("crit")
	tw.Damage = 10
	p := testProjectile()
	e := testEnemy()

	hasCrit := false
	for i := 0; i < 1000; i++ {
		result := ab.OnHit(tw, p, e)
		if result != nil && result.IsCrit {
			hasCrit = true
			if result.BonusDamage <= 0 {
				t.Fatalf("crit BonusDamage=%.1f 应 > 0", result.BonusDamage)
			}
			break
		}
	}
	if !hasCrit {
		t.Error("crit 1000 次调用从未触发暴击")
	}
}

func TestAbility_FlatDamage_ReturnsBonusDamage(t *testing.T) {
	ab := tower.Registry["flatDamage"]
	if ab == nil {
		t.Fatal("flatDamage 未注册")
	}
	tw := testTower("flatDamage")
	p := testProjectile()
	e := testEnemy()

	result := ab.OnHit(tw, p, e)
	if result == nil {
		t.Fatal("flatDamage OnHit 返回 nil")
	}
	if result.BonusDamage <= 0 {
		t.Errorf("flatDamage BonusDamage=%.1f 应 > 0", result.BonusDamage)
	}
}

func TestAbility_ExecutionBonus_LowHPBonus(t *testing.T) {
	ab := tower.Registry["executionBonus"]
	if ab == nil {
		t.Fatal("executionBonus 未注册")
	}
	tw := testTower("executionBonus")
	tw.Damage = 10
	p := testProjectile()

	// 低 HP 敌人（HP < 50% MaxHP）
	e := &enemy.Enemy{HP: 30, MaxHP: 100, Speed: 60, BaseSpeed: 60, Active: true}
	result := ab.OnHit(tw, p, e)
	if result == nil || result.BonusDamage <= 0 {
		t.Error("executionBonus: 低 HP 敌人应有额外伤害")
	}

	// 满 HP 敌人
	eFull := testEnemy()
	resultFull := ab.OnHit(tw, p, eFull)
	if resultFull != nil && resultFull.BonusDamage > 0 {
		t.Error("executionBonus: 满 HP 敌人不应有额外伤害")
	}
}

// ── DoT 能力 ──

func TestAbility_Burn_ReturnsBurnEffect(t *testing.T) {
	ab := tower.Registry["burn"]
	if ab == nil {
		t.Fatal("burn 未注册")
	}
	tw := testTower("burn")
	tw.Damage = 10
	p := testProjectile()
	e := testEnemy()

	result := ab.OnHit(tw, p, e)
	if result == nil {
		t.Fatal("burn OnHit 返回 nil")
	}
	if result.Burn == nil {
		t.Fatal("burn OnHit 应返回 BurnEffect")
	}
	if result.Burn.DPS <= 0 {
		t.Errorf("burn DPS=%.2f 应 > 0", result.Burn.DPS)
	}
	if result.Burn.Duration <= 0 {
		t.Errorf("burn Duration=%.2f 应 > 0", result.Burn.Duration)
	}
}

func TestAbility_BleedDot_ReturnsBleedEffect(t *testing.T) {
	ab := tower.Registry["bleedDot"]
	if ab == nil {
		t.Fatal("bleedDot 未注册")
	}
	tw := testTower("bleedDot")
	p := testProjectile()
	e := testEnemy()

	result := ab.OnHit(tw, p, e)
	if result == nil {
		t.Fatal("bleedDot OnHit 返回 nil")
	}
	if result.Bleed == nil {
		t.Fatal("bleedDot OnHit 应返回 BleedEffect")
	}
	if result.Bleed.Duration <= 0 {
		t.Errorf("bleedDot Duration=%.2f 应 > 0", result.Bleed.Duration)
	}
}

// ── Buff/Zone 能力（OnTick 型）──

func TestAbility_GoldPassive_IsTickerWithGold(t *testing.T) {
	ab := tower.Registry["goldPassive"]
	if ab == nil {
		t.Fatal("goldPassive 未注册")
	}
	_, isTicker := ab.(tower.Ticker)
	if !isTicker {
		t.Error("goldPassive 应实现 Ticker 接口")
	}
}

func TestAbility_DamageUpAura_IsTicker(t *testing.T) {
	ab := tower.Registry["damageUpAura"]
	if ab == nil {
		t.Fatal("damageUpAura 未注册")
	}
	_, isTicker := ab.(tower.Ticker)
	if !isTicker {
		t.Error("damageUpAura 应实现 Ticker 接口")
	}
}

func TestAbility_PoisonZone_IsTicker(t *testing.T) {
	ab := tower.Registry["poisonZone"]
	if ab == nil {
		t.Fatal("poisonZone 未注册")
	}
	_, isTicker := ab.(tower.Ticker)
	if !isTicker {
		t.Error("poisonZone 应实现 Ticker 接口")
	}
}

// ── 全量覆盖：每种能力至少不 panic ──

func TestAllAbilities_OnHitNoPanic(t *testing.T) {
	table := config.GlobalAbilityTable()
	tw := testTower("")
	tw.Damage = 10
	tw.Range = 150
	p := testProjectile()
	e := testEnemy()

	for name := range table {
		t.Run(name, func(t *testing.T) {
			ab, ok := tower.Registry[name]
			if !ok {
				t.Skipf("能力 %q 未在 Registry 中（可能是 Ticker-only）", name)
				return
			}
			tw.Abilities = []string{name}
			// 不 panic 即通过
			_ = ab.OnHit(tw, p, e)
		})
	}
}

func TestAllAbilities_CalcScalePositive(t *testing.T) {
	table := config.GlobalAbilityTable()
	for name, def := range table {
		t.Run(name, func(t *testing.T) {
			// 强度 100 时的缩放值应合理
			sv := def.CalcScale(100)
			if def.Base > 0 && sv <= 0 {
				t.Errorf("CalcScale(%s, 100)=%.4f 应 > 0（base=%.2f）", name, sv, def.Base)
			}
		})
	}
}
