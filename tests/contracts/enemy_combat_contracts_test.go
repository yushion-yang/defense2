// enemy_combat_contracts_test.go — 敌人/战斗/战灵系统契约测试。
package contracts_test

import (
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// ═══════════════════════════════════════
// 敌人原型契约
// ═══════════════════════════════════════

func TestEnemyArchetype_RunnerFasterThanNormal(t *testing.T) {
	archs, err := config.LoadEnemyArchetypes()
	if err != nil {
		t.Fatal(err)
	}
	normal := archs["normal"]
	runner := archs["runner"]
	if normal == nil || runner == nil {
		t.Fatal("缺少 normal 或 runner 原型")
	}
	if runner.SpeedScale <= normal.SpeedScale {
		t.Errorf("runner speedScale=%.2f 应 > normal speedScale=%.2f", runner.SpeedScale, normal.SpeedScale)
	}
}

func TestEnemyArchetype_TankMoreHPThanNormal(t *testing.T) {
	archs, err := config.LoadEnemyArchetypes()
	if err != nil {
		t.Fatal(err)
	}
	normal := archs["normal"]
	tank := archs["tank"]
	if normal == nil || tank == nil {
		t.Fatal("缺少 normal 或 tank 原型")
	}
	if tank.HPScale <= normal.HPScale {
		t.Errorf("tank hpScale=%.2f 应 > normal hpScale=%.2f", tank.HPScale, normal.HPScale)
	}
}

func TestEnemyArchetype_SwarmSmallRadius(t *testing.T) {
	archs, err := config.LoadEnemyArchetypes()
	if err != nil {
		t.Fatal(err)
	}
	normal := archs["normal"]
	swarm := archs["swarm"]
	if normal == nil || swarm == nil {
		t.Fatal("缺少 normal 或 swarm 原型")
	}
	if swarm.Radius >= normal.Radius {
		t.Errorf("swarm radius=%.0f 应 < normal radius=%.0f", swarm.Radius, normal.Radius)
	}
}

func hasAbility(a *config.EnemyArchetype, abilType string) bool {
	for _, ref := range a.Abilities {
		if ref.Type == abilType {
			return true
		}
	}
	return false
}

func TestEnemyArchetype_SplitterHasDeathSplit(t *testing.T) {
	archs, err := config.LoadEnemyArchetypes()
	if err != nil {
		t.Fatal(err)
	}
	sp := archs["splitter"]
	if sp == nil {
		t.Fatal("缺少 splitter 原型")
	}
	if !hasAbility(sp, "deathSplit") {
		t.Error("splitter 应装配 deathSplit 能力")
	}
}

func TestEnemyArchetype_StealthHasAbility(t *testing.T) {
	archs, err := config.LoadEnemyArchetypes()
	if err != nil {
		t.Fatal(err)
	}
	st := archs["phantom"]
	if st == nil {
		t.Fatal("缺少 phantom 原型")
	}
	if !hasAbility(st, "evasion") {
		t.Error("phantom 应装配 evasion 能力")
	}
}

func TestEnemyArchetype_HealerHasAbility(t *testing.T) {
	archs, err := config.LoadEnemyArchetypes()
	if err != nil {
		t.Fatal(err)
	}
	h := archs["healer"]
	if h == nil {
		t.Fatal("缺少 healer 原型")
	}
	if !hasAbility(h, "healAura") {
		t.Error("healer 应装配 healAura 能力")
	}
}

func TestEnemyArchetype_TeleporterHasAbility(t *testing.T) {
	archs, err := config.LoadEnemyArchetypes()
	if err != nil {
		t.Fatal(err)
	}
	tp := archs["phaser"]
	if tp == nil {
		t.Fatal("缺少 phaser 原型")
	}
	if !hasAbility(tp, "phaseShift") {
		t.Error("phaser 应装配 phaseShift 能力")
	}
}

func TestEnemyArchetype_BufferHasAbility(t *testing.T) {
	archs, err := config.LoadEnemyArchetypes()
	if err != nil {
		t.Fatal(err)
	}
	b := archs["buffer"]
	if b == nil {
		t.Fatal("缺少 buffer 原型")
	}
	if !hasAbility(b, "speedAura") {
		t.Error("buffer 应装配 speedAura 能力")
	}
}

// TestEnemyArchetype_FlyingHasMovementType 已移除（飞行概念已删除）

// ═══════════════════════════════════════
// 战斗系统契约
// ═══════════════════════════════════════

func TestDamagePipeline_WeakenAmplifyHasCap(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Active: true, StatusEffects: enemy.StatusEffects{DamageAmplify: 0.8}} // 超过 cap
	r := combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 10,
	})
	// MaxDamageAmplify = 0.5，所以最多 +50%
	if r.FinalDamage > 16 { // 10 * 1.5 = 15, 保底约束后略有浮动
		t.Errorf("虚弱增伤应有上限，FinalDamage=%.1f 过高", r.FinalDamage)
	}
}

func TestDamagePipeline_BossPercentHPCap(t *testing.T) {
	e := &enemy.Enemy{HP: 1000, MaxHP: 1000, Active: true, Boss: true}
	r := combat.ProcessDamage(combat.DamageInput{
		Target:      e,
		RawDamage:   500,
		IsPercentHP: true,
	})
	// Boss %HP cap = 5% of MaxHP = 50
	if r.FinalDamage > 55 { // 允许小误差
		t.Errorf("Boss %%HP cap 应限制在 ~50，实际 FinalDamage=%.1f", r.FinalDamage)
	}
}

func TestDamagePipeline_SilenceDisablesDamageCap(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Active: true, StatusEffects: enemy.StatusEffects{Silenced: true}, AbilityFields: enemy.AbilityFields{DamageCap: 5}}
	r := combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 50,
	})
	// 沉默时 damageCap 失效，应造成完整伤害
	if r.FinalDamage < 40 {
		t.Errorf("沉默应禁用 damageCap，FinalDamage=%.1f 应接近 50", r.FinalDamage)
	}
}

func TestDamagePipeline_DamageCapWhenNotSilenced(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Active: true, AbilityFields: enemy.AbilityFields{DamageCap: 5}}
	r := combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 50,
	})
	if r.FinalDamage > 6 { // cap=5 + 保底容差
		t.Errorf("damageCap=5 应限制伤害，FinalDamage=%.1f", r.FinalDamage)
	}
}

// ═══════════════════════════════════════
// 攻击方式覆盖契约
// ═══════════════════════════════════════

func TestAllTowerAttackStylesHaveHandlers(t *testing.T) {
	// 所有活跃的攻击方式
	styles := []tower.AttackStyle{
		tower.StyleProjectile, tower.StyleWideBeam, tower.StyleScatter,
		tower.StyleSpinAoE, tower.StyleRadial,
	}
	for _, style := range styles {
		if h := combat.Get(style); h == nil {
			t.Errorf("攻击方式 %q 没有注册 handler", style)
		}
	}
}

// ═══════════════════════════════════════
// CC 系统契约
// ═══════════════════════════════════════

func TestCC_SlowRespectsMinSpeedRatio(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Speed: 60, BaseSpeed: 60, Active: true}
	combat.ApplySlow(e, 0.01, 5.0, "test") // 极端减速
	minSpeed := e.BaseSpeed * combat.MinSpeedRatio()
	if e.Speed < minSpeed {
		t.Errorf("Speed=%.1f < MinSpeed=%.1f (BaseSpeed*%.1f)", e.Speed, minSpeed, combat.MinSpeedRatio())
	}
}

func TestCC_StunRespectsImmunity(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Active: true, StatusEffects: enemy.StatusEffects{IsStunImmune: true}}
	ok := combat.ApplyStun(e, 5.0, "test")
	if ok {
		t.Error("IsStunImmune=true 时 ApplyStun 应返回 false")
	}
	if e.StunTimer > 0 {
		t.Error("免疫后 StunTimer 应为 0")
	}
}

func TestCC_TenacityReducesDuration(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Active: true, StatusEffects: enemy.StatusEffects{Tenacity: 0.5}}
	combat.ApplyStun(e, 2.0, "test")
	// 韧性 0.5 → 持续 = 2.0 * (1-0.5) = 1.0
	if e.StunTimer > 1.1 {
		t.Errorf("Tenacity=0.5 时 StunTimer=%.2f 应 ≈ 1.0", e.StunTimer)
	}
}

func TestCC_FullTenacityImmune(t *testing.T) {
	e := &enemy.Enemy{HP: 100, MaxHP: 100, Active: true, StatusEffects: enemy.StatusEffects{Tenacity: 1.0}}
	ok := combat.ApplyStun(e, 5.0, "test")
	if ok {
		t.Error("Tenacity=1.0 时应完全免疫")
	}
}

// Buff 模板契约已移除 — 旧 BuffTemplate 系统已被 abilities.json + 直接字段设置替代。
