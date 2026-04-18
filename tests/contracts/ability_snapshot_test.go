// ability_snapshot_test.go — 能力行为快照测试。
//
// 对 29 种能力逐一调用 OnHit/OnTick，验证描述符引擎输出与旧系统行为一致。
// 标准测试参数：塔 Damage=10, Range=150, Strength=100；敌人 HP=100, MaxHP=100, Active=true。
// 概率型能力（crit/stunChance/stunDuration）运行 1000 次验证至少触发 1 次。
//
// 依赖：descriptor_contracts_test.go 的 init() 已调用 descriptor.InitDescriptorAbilities，
// config_rules_test.go 的 init() 已调用 config.SetDataFS + LoadBuffRules + LoadSpawnerConfig。
package contracts_test

import (
	"fmt"
	"testing"

	"defense2/internal/core/buff"
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// ════════════════════════════════════════════════════════════════
// 一、OnHit 能力快照测试
// ════════════════════════════════════════════════════════════════

// TestSnapshot_OnHit_SlowPower 验证 slowPower 返回正确的减速效果。
// abilities.json: base=0.20, param(duration)=1.0
func TestSnapshot_OnHit_SlowPower(t *testing.T) {
	ab := snapshotMustLookup(t, "slowPower")
	tw := testTower("slowPower")
	p := testProjectile()
	e := testEnemy()

	result := ab.OnHit(tw, p, e)
	snapshotRequireNonNilHR(t, result, "slowPower OnHit 返回 nil")
	if result.Slow == nil {
		t.Fatal("slowPower Slow 为 nil")
	}

	// 减速语义：JSON base=0.20 → CalcScale(100)=0.25 → Factor = 1 - 0.25 = 0.75（速度降至 75%）
	snapshotAssertInRange(t, "Slow.Factor", result.Slow.Factor, 0.50, 0.90)
	// param=1.0 → duration ≈ 1.0
	snapshotAssertInRange(t, "Slow.Duration", result.Slow.Duration, 0.5, 2.0)
}

// TestSnapshot_OnHit_SlowDuration 验证 slowDuration 的减速持续时间随强度缩放。
// abilities.json: base(duration)=0.5, potential=0.1, param(factor)=0.35
func TestSnapshot_OnHit_SlowDuration(t *testing.T) {
	ab := snapshotMustLookup(t, "slowDuration")
	tw := testTower("slowDuration")
	p := testProjectile()
	e := testEnemy()

	result := ab.OnHit(tw, p, e)
	snapshotRequireNonNilHR(t, result, "slowDuration OnHit 返回 nil")
	if result.Slow == nil {
		t.Fatal("slowDuration Slow 为 nil")
	}

	// param(factor)=0.35 → Factor = 1 - 0.35 = 0.65（速度降至 65%）
	snapshotAssertInRange(t, "Slow.Factor", result.Slow.Factor, 0.40, 0.80)
	snapshotAssertGt(t, "Slow.Duration", result.Slow.Duration, 0)
}

// TestSnapshot_OnHit_StunChance 验证 stunChance 的概率触发。
// abilities.json: base(chance)=0.1, param(duration)=0.5
func TestSnapshot_OnHit_StunChance(t *testing.T) {
	ab := snapshotMustLookup(t, "stunChance")
	tw := testTower("stunChance")
	p := testProjectile()
	e := testEnemy()

	hasStun, hasNil := false, false
	for i := 0; i < 1000; i++ {
		result := ab.OnHit(tw, p, e)
		if result != nil && result.Stun != nil {
			hasStun = true
			snapshotAssertInRange(t, "Stun.Duration", result.Stun.Duration, 0.1, 2.0)
		} else {
			hasNil = true
		}
		if hasStun && hasNil {
			break
		}
	}
	if !hasStun {
		t.Error("stunChance 1000 次从未触发眩晕")
	}
	if !hasNil {
		t.Error("stunChance 1000 次全部触发，不符合概率模型")
	}
}

// TestSnapshot_OnHit_StunDuration 验证 stunDuration 的概率触发和持续时间缩放。
// abilities.json: base(duration)=0.2, potential=0.1, param(chance)=0.25
func TestSnapshot_OnHit_StunDuration(t *testing.T) {
	ab := snapshotMustLookup(t, "stunDuration")
	tw := testTower("stunDuration")
	p := testProjectile()
	e := testEnemy()

	hasStun := false
	for i := 0; i < 1000; i++ {
		result := ab.OnHit(tw, p, e)
		if result != nil && result.Stun != nil {
			hasStun = true
			snapshotAssertGt(t, "Stun.Duration", result.Stun.Duration, 0)
			break
		}
	}
	if !hasStun {
		t.Error("stunDuration 1000 次从未触发眩晕")
	}
}

// TestSnapshot_OnHit_Crit 验证暴击的概率触发和额外伤害。
// abilities.json: base(chance)=0.1, param(multiplier)=2
func TestSnapshot_OnHit_Crit(t *testing.T) {
	ab := snapshotMustLookup(t, "crit")
	tw := testTower("crit")
	tw.Damage = 10
	p := testProjectile()
	e := testEnemy()

	hasCrit, hasNoCrit := false, false
	for i := 0; i < 1000; i++ {
		result := ab.OnHit(tw, p, e)
		if result != nil && result.IsCrit {
			hasCrit = true
			snapshotAssertGt(t, "BonusDamage", result.BonusDamage, 0)
		} else {
			hasNoCrit = true
		}
		if hasCrit && hasNoCrit {
			break
		}
	}
	if !hasCrit {
		t.Error("crit 1000 次从未触发暴击")
	}
}

// TestSnapshot_OnHit_FlatDamage 验证固伤返回 SeparateDamage。
// abilities.json: base=2, potential=5 → CalcScale(100)=7
func TestSnapshot_OnHit_FlatDamage(t *testing.T) {
	ab := snapshotMustLookup(t, "flatDamage")
	tw := testTower("flatDamage")
	p := testProjectile()
	e := testEnemy()

	result := ab.OnHit(tw, p, e)
	snapshotRequireNonNilHR(t, result, "flatDamage OnHit 返回 nil")
	snapshotAssertGt(t, "SeparateDamage", result.SeparateDamage, 0)
	snapshotAssertInRange(t, "SeparateDamage", result.SeparateDamage, 5.0, 10.0)
}

// TestSnapshot_OnHit_DistanceDamage 验证距离伤害随塔-敌距离缩放。
// abilities.json: base(bonusPerStep)=0.1, param(stepDist)=150
func TestSnapshot_OnHit_DistanceDamage(t *testing.T) {
	ab := snapshotMustLookup(t, "distanceDamage")
	tw := testTower("distanceDamage")
	tw.X = 0
	tw.Y = 0
	tw.Damage = 10
	p := testProjectile()

	// 敌人在距离 300 处（2 个步长）
	e := testEnemy()
	e.X = 300
	e.Y = 0

	result := ab.OnHit(tw, p, e)
	snapshotRequireNonNilHR(t, result, "distanceDamage OnHit 返回 nil")
	snapshotAssertGt(t, "BonusDamage", result.BonusDamage, 0)

	// 距离为 0 时：distanceStep 缩放不生效（跳过 dist/step 乘法），
	// 但 base ratio 伤害仍然存在（CalcScale(100)=0.15 × towerDmg=10 → 1.5）
	eClose := testEnemy()
	eClose.X = 0
	eClose.Y = 0
	resultClose := ab.OnHit(tw, p, eClose)
	// 距离为 0 时仍有 base ratio 加成，但应小于远距离
	if resultClose != nil && result.BonusDamage > 0 {
		if resultClose.BonusDamage >= result.BonusDamage {
			t.Errorf("distanceDamage 距离 0 时 BonusDamage=%.2f 应 < 距离 300 时=%.2f",
				resultClose.BonusDamage, result.BonusDamage)
		}
	}
}

// TestSnapshot_OnHit_ExecutionBonus 验证斩杀效果：低血触发、满血不触发。
// abilities.json: base(damageBonus)=0.2, param(hpThreshold)=0.5
func TestSnapshot_OnHit_ExecutionBonus(t *testing.T) {
	ab := snapshotMustLookup(t, "executionBonus")
	tw := testTower("executionBonus")
	tw.Damage = 10
	p := testProjectile()

	// 低血敌人（30/100 = 30% < 50% threshold）
	eLow := &enemy.Enemy{HP: 30, MaxHP: 100, Speed: 60, BaseSpeed: 60, Active: true}
	eLow.Buffs = buff.NewDefaultBuffList()
	result := ab.OnHit(tw, p, eLow)
	snapshotRequireNonNilHR(t, result, "executionBonus 低血返回 nil")
	snapshotAssertGt(t, "BonusDamage(低血)", result.BonusDamage, 0)

	// 满血敌人（100/100 = 100% > 50% threshold）
	eFull := testEnemy()
	resultFull := ab.OnHit(tw, p, eFull)
	if resultFull != nil && resultFull.BonusDamage > 0 {
		t.Errorf("executionBonus 满血敌人不应有额外伤害，got BonusDamage=%.2f", resultFull.BonusDamage)
	}
}

// TestSnapshot_OnHit_Splash 验证溅射返回 SplashEffect。
// abilities.json: base(ratio)=0.75, param(radius)=50
func TestSnapshot_OnHit_Splash(t *testing.T) {
	ab := snapshotMustLookup(t, "splash")
	tw := testTower("splash")
	p := testProjectile()
	e := testEnemy()

	result := ab.OnHit(tw, p, e)
	snapshotRequireNonNilHR(t, result, "splash OnHit 返回 nil")
	if result.Splash == nil {
		t.Fatal("splash Splash 为 nil")
	}

	snapshotAssertInRange(t, "Splash.Radius", result.Splash.Radius, 30, 80)
	snapshotAssertInRange(t, "Splash.Ratio", result.Splash.Ratio, 0.3, 1.0)
}

// TestSnapshot_OnHit_Bounce 验证弹射返回 BounceEffect。
// abilities.json: base(maxBounces)=1, param(damageDecay)=0.8, param2(bounceRange)=80
func TestSnapshot_OnHit_Bounce(t *testing.T) {
	ab := snapshotMustLookup(t, "bounce")
	tw := testTower("bounce")
	tw.Damage = 10
	tw.Range = 150
	p := testProjectile()
	e := testEnemy()

	result := ab.OnHit(tw, p, e)
	snapshotRequireNonNilHR(t, result, "bounce OnHit 返回 nil")
	if result.Bounce == nil {
		t.Fatal("bounce Bounce 为 nil")
	}

	if result.Bounce.MaxBounces < 1 {
		t.Errorf("MaxBounces=%d 应 >= 1", result.Bounce.MaxBounces)
	}
	snapshotAssertGt(t, "Range", result.Bounce.Range, 0)
	snapshotAssertInRange(t, "DamageRatio", result.Bounce.DamageRatio, 0.3, 1.0)
	snapshotAssertGt(t, "SrcDamage", result.Bounce.SrcDamage, 0)
}

// TestSnapshot_OnHit_Burn 验证灼烧 DoT。
// abilities.json: base(ratio)=0.10, param(duration)=2.0
func TestSnapshot_OnHit_Burn(t *testing.T) {
	ab := snapshotMustLookup(t, "burn")
	tw := testTower("burn")
	tw.Damage = 10
	p := testProjectile()
	e := testEnemy()

	result := ab.OnHit(tw, p, e)
	snapshotRequireNonNilHR(t, result, "burn OnHit 返回 nil")
	if result.Burn == nil {
		t.Fatal("burn Burn 为 nil")
	}

	snapshotAssertGt(t, "Burn.DPS", result.Burn.DPS, 0)
	snapshotAssertInRange(t, "Burn.Duration", result.Burn.Duration, 1.0, 4.0)
}

// TestSnapshot_OnHit_BleedDot 验证流血 DoT（非 Boss 触发，Boss 免疫）。
// abilities.json: base(hpPercent)=0.01, param(duration)=3.0
func TestSnapshot_OnHit_BleedDot(t *testing.T) {
	ab := snapshotMustLookup(t, "bleedDot")
	tw := testTower("bleedDot")
	p := testProjectile()

	// 普通敌人
	e := testEnemy()
	result := ab.OnHit(tw, p, e)
	snapshotRequireNonNilHR(t, result, "bleedDot OnHit(普通敌人) 返回 nil")
	if result.Bleed == nil {
		t.Fatal("bleedDot Bleed(普通敌人) 为 nil")
	}
	snapshotAssertGt(t, "Bleed.DPS", result.Bleed.DPS, 0)
	snapshotAssertInRange(t, "Bleed.Duration", result.Bleed.Duration, 1.0, 6.0)

	// Boss 敌人应免疫流血
	eBoss := testEnemy()
	eBoss.Boss = true
	resultBoss := ab.OnHit(tw, p, eBoss)
	if resultBoss != nil && resultBoss.Bleed != nil {
		t.Error("bleedDot Boss 敌人不应触发流血")
	}
}

// TestSnapshot_OnHit_Poison 验证中毒 DoT。
// abilities.json: base(dps)=3, potential=5, param(duration)=4.0
func TestSnapshot_OnHit_Poison(t *testing.T) {
	ab := snapshotMustLookup(t, "poison")
	tw := testTower("poison")
	p := testProjectile()
	e := testEnemy()

	result := ab.OnHit(tw, p, e)
	snapshotRequireNonNilHR(t, result, "poison OnHit 返回 nil")
	if result.Poison == nil {
		t.Fatal("poison Poison 为 nil")
	}

	snapshotAssertInRange(t, "Poison.DPS", result.Poison.DPS, 2, 20)
	snapshotAssertInRange(t, "Poison.Duration", result.Poison.Duration, 2.0, 6.0)
}

// TestSnapshot_OnHit_Weaken 验证易伤效果。
// abilities.json: base(amplify)=0.15, param(duration)=3.0
func TestSnapshot_OnHit_Weaken(t *testing.T) {
	ab := snapshotMustLookup(t, "weaken")
	tw := testTower("weaken")
	p := testProjectile()
	e := testEnemy()

	result := ab.OnHit(tw, p, e)
	snapshotRequireNonNilHR(t, result, "weaken OnHit 返回 nil")
	if result.Weaken == nil {
		t.Fatal("weaken Weaken 为 nil")
	}

	snapshotAssertInRange(t, "Weaken.Amplify", result.Weaken.Amplify, 0.05, 0.50)
	snapshotAssertInRange(t, "Weaken.Duration", result.Weaken.Duration, 1.0, 5.0)
}

// TestSnapshot_OnHit_Momentum 验证蓄势始终触发 BonusDamage。
// abilities.json: base(bonusRatio)=0.05, potential=0.05
func TestSnapshot_OnHit_Momentum(t *testing.T) {
	ab := snapshotMustLookup(t, "momentum")
	tw := testTower("momentum")
	tw.Damage = 10
	p := testProjectile()
	e := testEnemy()

	result := ab.OnHit(tw, p, e)
	snapshotRequireNonNilHR(t, result, "momentum OnHit 返回 nil")
	snapshotAssertGt(t, "BonusDamage", result.BonusDamage, 0)
}

// ════════════════════════════════════════════════════════════════
// 二、OnTick 能力快照测试（光环/区域/经济）
// ════════════════════════════════════════════════════════════════

// TestSnapshot_OnTick_DamageUpAura 验证伤害光环对附近塔施加 damage buff。
// abilities.json: base(bonus)=0.05, param(radius)=150
func TestSnapshot_OnTick_DamageUpAura(t *testing.T) {
	ab := snapshotMustLookup(t, "damageUpAura")
	ticker := snapshotMustTicker(t, ab, "damageUpAura")

	src := snapshotTowerAt("damageUpAura", 100, 100, 0)
	towerPool, target := snapshotBuildAuraPool(t, src, 200, 100, 1)
	enemyPool := enemy.NewPool(10)

	ctx := &tower.TickContext{Enemies: enemyPool, Towers: towerPool, DT: 1.0 / 60.0}
	ticker.OnTick(src, ctx)

	val := target.Buffs.SumByID(buff.IDAuraDamageAmp)
	snapshotAssertGt(t, "damageUpAura buff value", val, 0)
}

// TestSnapshot_OnTick_AttackSpeedAura 验证攻速光环对附近塔施加 speed buff。
func TestSnapshot_OnTick_AttackSpeedAura(t *testing.T) {
	ab := snapshotMustLookup(t, "attackSpeedAura")
	ticker := snapshotMustTicker(t, ab, "attackSpeedAura")

	src := snapshotTowerAt("attackSpeedAura", 100, 100, 0)
	towerPool, target := snapshotBuildAuraPool(t, src, 200, 100, 1)
	enemyPool := enemy.NewPool(10)

	ctx := &tower.TickContext{Enemies: enemyPool, Towers: towerPool, DT: 1.0 / 60.0}
	ticker.OnTick(src, ctx)

	val := target.Buffs.SumByID(buff.IDAuraPctSpeed)
	snapshotAssertGt(t, "attackSpeedAura buff value", val, 0)
}

// TestSnapshot_OnTick_RangeAura 验证射程光环对附近塔施加 range buff。
func TestSnapshot_OnTick_RangeAura(t *testing.T) {
	ab := snapshotMustLookup(t, "rangeAura")
	ticker := snapshotMustTicker(t, ab, "rangeAura")

	src := snapshotTowerAt("rangeAura", 100, 100, 0)
	towerPool, target := snapshotBuildAuraPool(t, src, 200, 100, 1)
	enemyPool := enemy.NewPool(10)

	ctx := &tower.TickContext{Enemies: enemyPool, Towers: towerPool, DT: 1.0 / 60.0}
	ticker.OnTick(src, ctx)

	val := target.Buffs.SumByID(buff.IDAuraFlatRange)
	snapshotAssertGt(t, "rangeAura buff value", val, 0)
}

// TestSnapshot_OnTick_CritAura 验证暴击光环对附近塔施加 crit buff。
func TestSnapshot_OnTick_CritAura(t *testing.T) {
	ab := snapshotMustLookup(t, "critAura")
	ticker := snapshotMustTicker(t, ab, "critAura")

	src := snapshotTowerAt("critAura", 100, 100, 0)
	towerPool, target := snapshotBuildAuraPool(t, src, 200, 100, 1)
	enemyPool := enemy.NewPool(10)

	ctx := &tower.TickContext{Enemies: enemyPool, Towers: towerPool, DT: 1.0 / 60.0}
	ticker.OnTick(src, ctx)

	val := target.Buffs.SumByID(buff.IDAuraCrit)
	snapshotAssertGt(t, "critAura buff value", val, 0)
}

// TestSnapshot_OnTick_SoloBoost 验证独行加成：无附近塔时自身获得 damage buff。
// abilities.json: base(bonus)=0.05, param(checkRadius)=120
func TestSnapshot_OnTick_SoloBoost(t *testing.T) {
	ab := snapshotMustLookup(t, "soloBoost")
	ticker := snapshotMustTicker(t, ab, "soloBoost")

	// 只放一个塔（自身），无其他塔
	src := snapshotTowerAt("soloBoost", 100, 100, 0)
	towerPool := snapshotBuildSoloPool(src)
	enemyPool := enemy.NewPool(10)

	ctx := &tower.TickContext{Enemies: enemyPool, Towers: towerPool, DT: 1.0 / 60.0}
	ticker.OnTick(src, ctx)

	val := src.Buffs.SumByID(buff.IDAuraDamageAmp)
	snapshotAssertGt(t, "soloBoost self buff value", val, 0)
}

// TestSnapshot_OnTick_SoloBoost_WithNeighbor 验证有附近塔时不触发。
func TestSnapshot_OnTick_SoloBoost_WithNeighbor(t *testing.T) {
	ab := snapshotMustLookup(t, "soloBoost")
	ticker := snapshotMustTicker(t, ab, "soloBoost")

	src := snapshotTowerAt("soloBoost", 100, 100, 0)
	// 邻居在距离 50 处 < checkRadius=120
	towerPool, _ := snapshotBuildAuraPool(t, src, 150, 100, 1)
	enemyPool := enemy.NewPool(10)

	ctx := &tower.TickContext{Enemies: enemyPool, Towers: towerPool, DT: 1.0 / 60.0}
	ticker.OnTick(src, ctx)

	val := src.Buffs.SumByID(buff.IDAuraDamageAmp)
	if val > 0 {
		t.Errorf("soloBoost 有附近塔时不应触发 self buff，got %.4f", val)
	}
}

// TestSnapshot_OnTick_GoldPassive 验证被动产金（需等冷却结束）。
// abilities.json: base(amount)=1, param(interval)=3
func TestSnapshot_OnTick_GoldPassive(t *testing.T) {
	ab := snapshotMustLookup(t, "goldPassive")
	ticker := snapshotMustTicker(t, ab, "goldPassive")

	src := snapshotTowerAt("goldPassive", 100, 100, 0)
	towerPool := snapshotBuildSoloPool(src)
	enemyPool := enemy.NewPool(10)

	// 用大 DT 跳过冷却（interval=3s，给 4s）
	ctx := &tower.TickContext{Enemies: enemyPool, Towers: towerPool, DT: 4.0}
	result := ticker.OnTick(src, ctx)
	if result == nil || result.GoldEarned <= 0 {
		t.Errorf("goldPassive 大 DT(4s) 应产金，got result=%v", result)
	}
}

// TestSnapshot_OnTick_GoldPassive_NoCooldown 验证冷却未满不产金。
func TestSnapshot_OnTick_GoldPassive_NoCooldown(t *testing.T) {
	ab := snapshotMustLookup(t, "goldPassive")
	ticker := snapshotMustTicker(t, ab, "goldPassive")

	src := snapshotTowerAt("goldPassive", 100, 100, 0)
	towerPool := snapshotBuildSoloPool(src)
	enemyPool := enemy.NewPool(10)

	ctx := &tower.TickContext{Enemies: enemyPool, Towers: towerPool, DT: 0.016}
	result := ticker.OnTick(src, ctx)
	if result != nil && result.GoldEarned > 0 {
		t.Errorf("goldPassive 冷却未满不应产金，got GoldEarned=%d", result.GoldEarned)
	}
}

// TestSnapshot_OnTick_SilenceZone 验证沉默区对射程内敌人设置 Silenced。
func TestSnapshot_OnTick_SilenceZone(t *testing.T) {
	ab := snapshotMustLookup(t, "silenceZone")
	ticker := snapshotMustTicker(t, ab, "silenceZone")

	src := snapshotTowerAt("silenceZone", 100, 100, 0)
	src.Range = 150

	enemyPool := enemy.NewPool(10)
	e := enemyPool.Spawn(120, 100, 100, 60, 1, "normal", nil)
	e.SpawnTimer = 0 // 跳过出生动画

	towerPool := snapshotBuildSoloPool(src)
	ctx := &tower.TickContext{Enemies: enemyPool, Towers: towerPool, DT: 1.0 / 60.0}
	ticker.OnTick(src, ctx)

	if !e.Silenced {
		t.Error("silenceZone 应将射程内敌人设为 Silenced=true")
	}
}

// TestSnapshot_OnTick_WeakenZone 验证脆弱区对射程内敌人施加 weaken debuff。
// abilities.json: base(amplify)=0.05
func TestSnapshot_OnTick_WeakenZone(t *testing.T) {
	ab := snapshotMustLookup(t, "weakenZone")
	ticker := snapshotMustTicker(t, ab, "weakenZone")

	src := snapshotTowerAt("weakenZone", 100, 100, 0)
	src.Range = 150

	enemyPool := enemy.NewPool(10)
	e := enemyPool.Spawn(120, 100, 100, 60, 1, "normal", nil)
	e.SpawnTimer = 0

	towerPool := snapshotBuildSoloPool(src)
	ctx := &tower.TickContext{Enemies: enemyPool, Towers: towerPool, DT: 1.0 / 60.0}
	ticker.OnTick(src, ctx)

	val := e.Buffs.SumByID(buff.IDWeaken)
	snapshotAssertGt(t, "weakenZone debuff value", val, 0)
}

// TestSnapshot_OnTick_PoisonZone 验证毒区对射程内敌人累积 ZoneDmgAccum。
// abilities.json: base(dps)=3
func TestSnapshot_OnTick_PoisonZone(t *testing.T) {
	ab := snapshotMustLookup(t, "poisonZone")
	ticker := snapshotMustTicker(t, ab, "poisonZone")

	src := snapshotTowerAt("poisonZone", 100, 100, 0)
	src.Range = 150

	enemyPool := enemy.NewPool(10)
	e := enemyPool.Spawn(120, 100, 100, 60, 1, "normal", nil)
	e.SpawnTimer = 0

	towerPool := snapshotBuildSoloPool(src)
	ctx := &tower.TickContext{Enemies: enemyPool, Towers: towerPool, DT: 1.0 / 60.0}
	ticker.OnTick(src, ctx)

	snapshotAssertGt(t, "poisonZone ZoneDmgAccum", e.ZoneDmgAccum, 0)
}

// TestSnapshot_OnTick_CurseZone 验证诅咒区对射程内敌人累积 ZoneDmgAccum。
// abilities.json: base(hpPercentPerSec)=0.01
func TestSnapshot_OnTick_CurseZone(t *testing.T) {
	ab := snapshotMustLookup(t, "curseZone")
	ticker := snapshotMustTicker(t, ab, "curseZone")

	src := snapshotTowerAt("curseZone", 100, 100, 0)
	src.Range = 150

	enemyPool := enemy.NewPool(10)
	e := enemyPool.Spawn(120, 100, 100, 60, 1, "normal", nil)
	e.SpawnTimer = 0

	towerPool := snapshotBuildSoloPool(src)
	ctx := &tower.TickContext{Enemies: enemyPool, Towers: towerPool, DT: 1.0 / 60.0}
	ticker.OnTick(src, ctx)

	snapshotAssertGt(t, "curseZone ZoneDmgAccum", e.ZoneDmgAccum, 0)
}

// ════════════════════════════════════════════════════════════════
// 三、攻击方式能力注册和安全检查
// ════════════════════════════════════════════════════════════════

// TestSnapshot_AttackStyles_RegisteredAndSafe 验证攻击方式能力已注册且 OnHit 不 panic。
func TestSnapshot_AttackStyles_RegisteredAndSafe(t *testing.T) {
	attackAbilities := []string{
		"scatter", "wideBeam", "spinAoe", "radial",
		"barrage", "multiTarget", "enhance",
	}

	tw := testTower("")
	tw.Damage = 10
	tw.Range = 150
	p := testProjectile()
	e := testEnemy()

	for _, name := range attackAbilities {
		t.Run(name, func(t *testing.T) {
			ab, ok := tower.Registry[name]
			if !ok {
				t.Fatalf("攻击方式能力 %q 未注册", name)
			}
			tw.Abilities = []string{name}
			// OnHit 不 panic 即通过
			_ = ab.OnHit(tw, p, e)
		})
	}
}

// ════════════════════════════════════════════════════════════════
// 四、全量注册完整性验证
// ════════════════════════════════════════════════════════════════

// TestSnapshot_AllAbilities_Registered 验证 abilities.json 中所有能力均已注册。
func TestSnapshot_AllAbilities_Registered(t *testing.T) {
	expected := []string{
		// attack
		"enhance", "scatter", "wideBeam", "spinAoe", "bounce", "splash",
		"multiTarget", "radial", "barrage",
		// cc
		"slowPower", "slowDuration", "stunChance", "stunDuration",
		// damage
		"crit", "distanceDamage", "executionBonus", "flatDamage", "momentum",
		// buff
		"damageUpAura", "attackSpeedAura", "rangeAura", "critAura",
		"soloBoost", "goldPassive",
		// dot
		"burn", "bleedDot", "poison", "weaken",
		// zone
		"poisonZone", "silenceZone", "curseZone", "weakenZone",
	}

	for _, name := range expected {
		t.Run(name, func(t *testing.T) {
			if _, ok := tower.Registry[name]; !ok {
				t.Errorf("能力 %q 未在 Registry 中注册", name)
			}
		})
	}
}

// TestSnapshot_TickAbilities_ImplementTicker 验证需要 OnTick 的能力实现了 Ticker 接口。
func TestSnapshot_TickAbilities_ImplementTicker(t *testing.T) {
	tickAbilities := []string{
		"damageUpAura", "attackSpeedAura", "rangeAura", "critAura",
		"soloBoost", "goldPassive",
		"poisonZone", "silenceZone", "curseZone", "weakenZone",
	}

	for _, name := range tickAbilities {
		t.Run(name, func(t *testing.T) {
			ab, ok := tower.Registry[name]
			if !ok {
				t.Fatalf("能力 %q 未注册", name)
			}
			if _, isTicker := ab.(tower.Ticker); !isTicker {
				t.Errorf("能力 %q 应实现 Ticker 接口", name)
			}
		})
	}
}

// TestSnapshot_HitOnlyAbilities_NotTicker 验证纯 OnHit 能力不实现 Ticker 接口。
func TestSnapshot_HitOnlyAbilities_NotTicker(t *testing.T) {
	hitOnlyAbilities := []string{
		"slowPower", "slowDuration", "stunChance", "stunDuration",
		"crit", "distanceDamage", "executionBonus", "flatDamage", "momentum",
		"burn", "bleedDot", "poison", "weaken",
		"splash", "bounce",
	}

	for _, name := range hitOnlyAbilities {
		t.Run(name, func(t *testing.T) {
			ab, ok := tower.Registry[name]
			if !ok {
				t.Skipf("能力 %q 未注册", name)
				return
			}
			if _, isTicker := ab.(tower.Ticker); isTicker {
				t.Errorf("纯 OnHit 能力 %q 不应实现 Ticker 接口（会导致 pipeline 白白每帧调用）", name)
			}
		})
	}
}

// ════════════════════════════════════════════════════════════════
// 辅助函数（snapshot 前缀避免与 ability_contracts_test.go 冲突）
// ════════════════════════════════════════════════════════════════

// snapshotMustLookup 从 Registry 查找能力，找不到则 Fatal。
func snapshotMustLookup(t *testing.T, name string) tower.Ability {
	t.Helper()
	ab, ok := tower.Registry[name]
	if !ok {
		t.Fatalf("能力 %q 未在 Registry 中注册", name)
	}
	return ab
}

// snapshotMustTicker 断言能力实现 Ticker 接口并返回。
func snapshotMustTicker(t *testing.T, ab tower.Ability, name string) tower.Ticker {
	t.Helper()
	ticker, ok := ab.(tower.Ticker)
	if !ok {
		t.Fatalf("能力 %q 未实现 Ticker 接口", name)
	}
	return ticker
}

// snapshotTowerAt 创建用于 OnTick 测试的塔（指定坐标和 row 索引）。
func snapshotTowerAt(abilityName string, x, y float64, row int) *tower.Tower {
	tw := testTower(abilityName)
	tw.X = x
	tw.Y = y
	tw.Row = row
	tw.Col = 0
	tw.InstanceKey = fmt.Sprintf("test_%d_0", row)
	tw.Active = true
	tw.Buffs = buff.NewDefaultBuffList()
	return tw
}

// snapshotBuildAuraPool 通过 tower.Pool.Place 创建含 src + 一个目标塔的池。
// src 放在 row=srcRow（通过 Place 注册到 grid），目标塔放在 row=targetRow。
// 返回池和目标塔的指针。
func snapshotBuildAuraPool(t *testing.T, src *tower.Tower, targetX, targetY float64, targetRow int) (*tower.Pool, *tower.Tower) {
	t.Helper()
	pool := tower.NewPool(8)

	// 用 FixedTiers 放置 src 塔到 pool，然后将 pool 中的实际塔覆盖为 src 的属性
	def := tower.TowerDef{
		Key: "test", Range: 150, Damage: 10, AttackSpeed: 1.0, Cost: 10,
		CfgBaseDamage: 10, PotentialDamage: 10,
		CfgBaseSpeed: 1.0, PotentialSpeed: 0.5,
		CfgBaseRange: 150, PotentialRange: 30,
		FixedTiers: true,
	}

	// 放置 src 塔
	placed := pool.Place(src.Row, src.Col, src.X, src.Y, def)
	if placed == nil {
		t.Fatal("无法放置 src 塔到 pool")
	}
	// 复制 src 的关键属性到 pool 中的塔
	placed.Abilities = src.Abilities
	placed.InstanceKey = src.InstanceKey
	placed.Buffs = src.Buffs
	placed.Range = src.Range

	// 更新 src 指向 pool 内的实际塔（OnTick 通过指针比较排除自身）
	*src = *placed

	// 放置目标塔
	target := pool.Place(targetRow, 1, targetX, targetY, def)
	if target == nil {
		t.Fatal("无法放置 target 塔到 pool")
	}
	target.InstanceKey = fmt.Sprintf("target_%d_1", targetRow)
	target.Buffs = buff.NewDefaultBuffList()

	return pool, target
}

// snapshotBuildSoloPool 构建只含一个塔的 pool（用于 soloBoost/goldPassive 测试）。
func snapshotBuildSoloPool(src *tower.Tower) *tower.Pool {
	pool := tower.NewPool(8)
	def := tower.TowerDef{
		Key: "test", Range: 150, Damage: 10, AttackSpeed: 1.0, Cost: 10,
		CfgBaseDamage: 10, PotentialDamage: 10,
		CfgBaseSpeed: 1.0, PotentialSpeed: 0.5,
		CfgBaseRange: 150, PotentialRange: 30,
		FixedTiers: true,
	}
	placed := pool.Place(src.Row, src.Col, src.X, src.Y, def)
	if placed != nil {
		placed.Abilities = src.Abilities
		placed.InstanceKey = src.InstanceKey
		placed.Buffs = src.Buffs
		placed.Range = src.Range
		// 把 pool 中塔的属性同步回 src（使得外部 src 指针与 pool 内容一致）
		*src = *placed
	}
	return pool
}

// snapshotRequireNonNilHR 断言 HitResult 非 nil。
func snapshotRequireNonNilHR(t *testing.T, hr *tower.HitResult, msg string) {
	t.Helper()
	if hr == nil {
		t.Fatal(msg)
	}
}

func snapshotAssertGt(t *testing.T, label string, v, threshold float64) {
	t.Helper()
	if v <= threshold {
		t.Errorf("%s=%.4f 应 > %.4f", label, v, threshold)
	}
}

func snapshotAssertInRange(t *testing.T, label string, v, lo, hi float64) {
	t.Helper()
	if v < lo || v > hi {
		t.Errorf("%s=%.4f 不在范围 [%.4f, %.4f]", label, v, lo, hi)
	}
}
