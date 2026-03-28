// zone.go — 区域类能力实现。
// 区域能力通过 tick 持续对范围内敌人施加效果（毒圈、沉默圈等）。
// 此处注册为命中无效果的占位，实际 tick 逻辑由区域子系统处理。
package abilities

import (
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

// PoisonZone 毒圈：范围内敌人持续中毒。
type PoisonZone struct{}
func (a *PoisonZone) Name() string { return "poisonZone" }
func (a *PoisonZone) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}

// SilenceZone 沉默圈：范围内敌人能力被压制。
type SilenceZone struct{}
func (a *SilenceZone) Name() string { return "silenceZone" }
func (a *SilenceZone) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}

// CurseZone 诅咒圈：范围内敌人受到的伤害增加。
type CurseZone struct{}
func (a *CurseZone) Name() string { return "curseZone" }
func (a *CurseZone) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}

// BuffPurge 净化：命中时移除敌人身上的增益效果。
type BuffPurge struct{}
func (a *BuffPurge) Name() string { return "buffPurge" }
func (a *BuffPurge) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}

// ChannelLaser 引导激光：持续对单一目标造成递增伤害。
type ChannelLaser struct{}
func (a *ChannelLaser) Name() string { return "channelLaser" }
func (a *ChannelLaser) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}
