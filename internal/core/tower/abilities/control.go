// control.go — 控制类能力实现。
// 包含命中减速（onHitSlow）和流血（bleedDot）两种能力，通过 init() 自注册。
package abilities

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

func init() {
	tower.Register(&OnHitSlowAbility{})
	tower.Register(&BleedDotAbility{})
	tower.Register(&BurnAbility{})
	tower.Register(&StunAbility{})
	tower.Register(&RootAbility{})
	tower.Register(&RevealAbility{})
	tower.Register(&JudgmentMark{})
	tower.Register(&DeathMark{})
}

// OnHitSlowAbility 命中减速能力：命中时使敌人减至 60% 速度，持续 1 秒。
type OnHitSlowAbility struct{}

func (a *OnHitSlowAbility) Name() string { return "onHitSlow" }
func (a *OnHitSlowAbility) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return &tower.HitResult{
		Slow: &tower.SlowEffect{Factor: 0.6, Duration: 1.0},
	}
}

// BleedDotAbility 流血能力：命中后持续 3 秒，每秒 5 点伤害。
type BleedDotAbility struct{}

func (a *BleedDotAbility) Name() string { return "bleedDot" }
func (a *BleedDotAbility) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return &tower.HitResult{
		Bleed: &tower.BleedEffect{DPS: 5, Duration: 3},
	}
}

// BurnAbility 灼烧能力：命中后持续灼烧 2 秒。
type BurnAbility struct{}
func (a *BurnAbility) Name() string { return "burn" }
func (a *BurnAbility) OnHit(_ *tower.Tower, p *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return &tower.HitResult{
		Bleed: &tower.BleedEffect{DPS: p.Damage * 0.3, Duration: 2}, // 复用 Bleed 结构表示灼烧
	}
}

// StunAbility 眩晕能力：命中时短暂冻结敌人。
type StunAbility struct{}
func (a *StunAbility) Name() string { return "stun" }
func (a *StunAbility) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return &tower.HitResult{Stun: &tower.StunEffect{Duration: 0.3}}
}

// RootAbility 定身能力：命中时定住敌人 0.5 秒。
type RootAbility struct{}
func (a *RootAbility) Name() string { return "root" }
func (a *RootAbility) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return &tower.HitResult{Stun: &tower.StunEffect{Duration: 0.5}} // 复用 Stun 表示定身
}

// RevealAbility 揭示能力：命中隐身敌人时取消隐身。
type RevealAbility struct{}
func (a *RevealAbility) Name() string { return "reveal" }
func (a *RevealAbility) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil // 由伤害管线特殊处理
}

// JudgmentMark 审判标记：命中时标记目标，使其受到的后续伤害增加。
type JudgmentMark struct{}
func (a *JudgmentMark) Name() string { return "judgmentMark" }
func (a *JudgmentMark) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil // 由 buff 系统处理
}

// DeathMark 死亡标记：标记的敌人死亡时对周围造成爆炸伤害。
type DeathMark struct{}
func (a *DeathMark) Name() string { return "deathMark" }
func (a *DeathMark) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil // 由死亡管线处理
}
