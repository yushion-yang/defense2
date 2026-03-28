// zone.go — 区域类能力实现。
// 区域能力通过 Ticker 接口持续对范围内敌人施加效果（毒圈、沉默圈等）。
// 通过 init() 自注册到全局注册表。
package abilities

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

func init() {
	tower.Register(&PoisonZone{})
	tower.Register(&SilenceZone{})
	tower.Register(&CurseZone{})
	tower.Register(&BuffPurge{})
	tower.Register(&ChannelLaser{})
}

// PoisonZone 毒圈：范围内敌人持续中毒，3 DPS。
type PoisonZone struct{}

func (a *PoisonZone) Name() string { return "poisonZone" }
func (a *PoisonZone) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}
func (a *PoisonZone) OnTick(t *tower.Tower, ctx *tower.TickContext) *tower.TickResult {
	ctx.Enemies.Each(func(e *enemy.Enemy) {
		if distToEnemy(t, e) <= t.Range {
			e.HP -= 3.0 * ctx.DT
		}
	})
	return nil
}

// SilenceZone 沉默圈：范围内敌人移动速度 -20%。
type SilenceZone struct{}

func (a *SilenceZone) Name() string { return "silenceZone" }
func (a *SilenceZone) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}
func (a *SilenceZone) OnTick(t *tower.Tower, ctx *tower.TickContext) *tower.TickResult {
	ctx.Enemies.Each(func(e *enemy.Enemy) {
		if distToEnemy(t, e) <= t.Range {
			// 施加短减速，每帧刷新
			e.SlowTimer = 0.2
			e.SlowFactor = 0.8
			e.Speed = e.BaseSpeed * 0.8
		}
	})
	return nil
}

// CurseZone 诅咒圈：范围内敌人每秒受 15% 额外伤害（以 MaxHP 的百分比扣血）。
type CurseZone struct{}

func (a *CurseZone) Name() string { return "curseZone" }
func (a *CurseZone) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}
func (a *CurseZone) OnTick(t *tower.Tower, ctx *tower.TickContext) *tower.TickResult {
	ctx.Enemies.Each(func(e *enemy.Enemy) {
		if distToEnemy(t, e) <= t.Range {
			// 每秒额外造成最大血量 1.5% 伤害（简化的 15% 增伤）
			e.HP -= e.MaxHP * 0.015 * ctx.DT
		}
	})
	return nil
}

// BuffPurge 净化：范围内敌人护盾每秒削减 5 点。
type BuffPurge struct{}

func (a *BuffPurge) Name() string { return "buffPurge" }
func (a *BuffPurge) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}
func (a *BuffPurge) OnTick(t *tower.Tower, ctx *tower.TickContext) *tower.TickResult {
	ctx.Enemies.Each(func(e *enemy.Enemy) {
		if distToEnemy(t, e) <= t.Range && e.ShieldHP > 0 {
			e.ShieldHP -= 5.0 * ctx.DT
			if e.ShieldHP < 0 {
				e.ShieldHP = 0
			}
		}
	})
	return nil
}

// ChannelLaser 引导激光：持续对射程内最近敌人造成 10 DPS。
type ChannelLaser struct{}

func (a *ChannelLaser) Name() string { return "channelLaser" }
func (a *ChannelLaser) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}
func (a *ChannelLaser) OnTick(t *tower.Tower, ctx *tower.TickContext) *tower.TickResult {
	var nearest *enemy.Enemy
	bestDist := math.MaxFloat64
	ctx.Enemies.Each(func(e *enemy.Enemy) {
		d := distToEnemy(t, e)
		if d <= t.Range && d < bestDist {
			bestDist = d
			nearest = e
		}
	})
	if nearest != nil {
		nearest.HP -= 10.0 * ctx.DT
	}
	return nil
}

// distToEnemy 计算塔到敌人的像素距离。
func distToEnemy(t *tower.Tower, e *enemy.Enemy) float64 {
	return math.Hypot(t.X-e.X, t.Y-e.Y)
}
