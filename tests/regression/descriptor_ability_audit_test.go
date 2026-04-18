// descriptor_ability_audit_test.go — 回归测试：描述符能力审计发现的 4 个 bug。
//
// Bug 1: poisonZone/curseZone 的 onTick EffTypeDot/EffTypeDamage 被丢弃
// Bug 2: bleedDot 缺少 Boss 免疫（描述符缺 notBoss 条件 + IsBoss 未传播）
// Bug 3: distanceDamage 丢失距离缩放（描述符用简单 ratio，不考虑距离）
package regression_test

import (
	"testing"

	"defense2/internal/core/buff"
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

// ── poisonZone / curseZone ──

// TestRegression_PoisonZoneAppliesDot 验证 poisonZone 的 onTick 对附近敌人施加毒效果。
func TestRegression_PoisonZoneAppliesDot(t *testing.T) {
	ab, ok := tower.Registry["poisonZone"]
	if !ok {
		t.Fatal("poisonZone not registered")
	}
	ticker, ok := ab.(tower.Ticker)
	if !ok {
		t.Fatal("poisonZone should implement Ticker")
	}

	tw := &tower.Tower{Damage: 50, Range: 200, X: 100, Y: 100}
	tw.Abilities = []string{"poisonZone"}

	pool := enemy.NewPool(10)
	e := pool.Spawn(120, 120, 500, 60, 0, "normal", nil)
	e.SpawnTimer = 0 // 跳过出生动画，使 QueryRadius 能发现该敌人

	ctx := &tower.TickContext{
		DT:      1.0 / 60.0,
		Enemies: pool,
	}

	for i := 0; i < 10; i++ {
		ticker.OnTick(tw, ctx)
	}

	// poisonZone 应通过 ZoneDmgAccum 每帧累积伤害
	if e.ZoneDmgAccum <= 0 {
		t.Errorf("poisonZone ZoneDmgAccum = %v, want > 0", e.ZoneDmgAccum)
	}
}

// TestRegression_CurseZoneAppliesDamage 验证 curseZone 的 onTick 对附近敌人累积伤害。
func TestRegression_CurseZoneAppliesDamage(t *testing.T) {
	ab, ok := tower.Registry["curseZone"]
	if !ok {
		t.Fatal("curseZone not registered")
	}
	ticker, ok := ab.(tower.Ticker)
	if !ok {
		t.Fatal("curseZone should implement Ticker")
	}

	tw := &tower.Tower{Damage: 50, Range: 200, X: 100, Y: 100}
	tw.Abilities = []string{"curseZone"}

	pool := enemy.NewPool(10)
	e := pool.Spawn(120, 120, 1000, 60, 0, "normal", nil)
	e.SpawnTimer = 0

	ctx := &tower.TickContext{
		DT:      1.0 / 60.0,
		Enemies: pool,
	}

	for i := 0; i < 10; i++ {
		ticker.OnTick(tw, ctx)
	}

	if e.ZoneDmgAccum <= 0 {
		t.Errorf("curseZone ZoneDmgAccum = %v, want > 0", e.ZoneDmgAccum)
	}
}

// ── bleedDot Boss 免疫 ──

// TestRegression_BleedDotBossImmune 验证 bleedDot 对 Boss 不生效。
func TestRegression_BleedDotBossImmune(t *testing.T) {
	ab, ok := tower.Registry["bleedDot"]
	if !ok {
		t.Fatal("bleedDot not registered")
	}

	tw := &tower.Tower{Damage: 100, Range: 200}
	tw.Abilities = []string{"bleedDot"}
	p := &projectile.Projectile{Damage: 100}

	boss := &enemy.Enemy{HP: 10000, MaxHP: 10000, X: 100, Y: 100}
	boss.Active = true
	boss.Boss = true
	boss.Buffs = buff.NewDefaultBuffList()

	hr := ab.OnHit(tw, p, boss)
	if hr != nil && hr.Bleed != nil {
		t.Error("bleedDot should not apply to Boss (Boss免疫)")
	}
}

// ── distanceDamage 距离缩放 ──

// TestRegression_DistanceDamageScalesWithDistance 验证远距离伤害 > 近距离。
func TestRegression_DistanceDamageScalesWithDistance(t *testing.T) {
	ab, ok := tower.Registry["distanceDamage"]
	if !ok {
		t.Fatal("distanceDamage not registered")
	}

	tw := &tower.Tower{Damage: 100, Range: 500, X: 0, Y: 0}
	tw.Abilities = []string{"distanceDamage"}
	p := &projectile.Projectile{Damage: 100}

	nearEnemy := &enemy.Enemy{HP: 1000, MaxHP: 1000, X: 50, Y: 0}
	nearEnemy.Active = true
	nearEnemy.Buffs = buff.NewDefaultBuffList()

	farEnemy := &enemy.Enemy{HP: 1000, MaxHP: 1000, X: 300, Y: 0}
	farEnemy.Active = true
	farEnemy.Buffs = buff.NewDefaultBuffList()

	hrNear := ab.OnHit(tw, p, nearEnemy)
	hrFar := ab.OnHit(tw, p, farEnemy)

	if hrNear == nil || hrFar == nil {
		t.Fatal("distanceDamage OnHit returned nil")
	}
	nearDmg := hrNear.BonusDamage + hrNear.SeparateDamage
	farDmg := hrFar.BonusDamage + hrFar.SeparateDamage

	if farDmg <= nearDmg {
		t.Errorf("远距离伤害(%v) 应 > 近距离伤害(%v)，距离缩放失效", farDmg, nearDmg)
	}
}
