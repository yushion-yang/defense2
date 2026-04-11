// sys_projectile.go — 弹射物移动 + 命中检测步骤。
package pipeline

import (
	"defense2/internal/core/enemy"
)

// SysProjectileMove 弹射物移动 + 光束衰减。
type SysProjectileMove struct{}

func (SysProjectileMove) Tick(ctx *TickCtx) bool {
	ctx.Projectiles.Tick(ctx.DT)
	ctx.Beams.Tick(ctx.DT)
	return false
}

// SysProjectileHit 弹射物命中检测（含能力触发）。
type SysProjectileHit struct{}

func (SysProjectileHit) Tick(ctx *TickCtx) bool {
	TickProjectileHits(ctx.Projectiles, ctx.Enemies, ctx.Towers,
		func(e *enemy.Enemy, damage float64, killed bool, attackStyle string, crit bool) {
			if ctx.CB.OnProjectileHit != nil {
				ctx.CB.OnProjectileHit(e, damage, killed, attackStyle, crit)
			}
		}, ctx.CB.OnCC)
	return false
}

// SysPostSafetyNet 后置安全网：清除本帧 HP<=0 但未 Kill 的敌人。
type SysPostSafetyNet struct{}

func (SysPostSafetyNet) Tick(ctx *TickCtx) bool {
	TickEnemyStatusEffects(ctx.Enemies, 0, nil) // dt=0 不触发 DoT，仅做 HP<=0 检查
	return false
}
