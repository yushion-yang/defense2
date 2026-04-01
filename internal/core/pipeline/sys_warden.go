// sys_warden.go — 战灵行为 + 战灵技能步骤。
package pipeline

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/warden"
)

// SysWardenTick 驱动战灵行为（移动/攻击/特殊能力）。
type SysWardenTick struct{}

func (SysWardenTick) Tick(ctx *TickCtx) bool {
	if !ctx.WardenReady || ctx.WardenUnit == nil {
		return false
	}
	ctx.WardenUnit.Tick(&warden.TickContext{
		Enemies:     ctx.Enemies,
		Towers:      ctx.Towers,
		Projectiles: ctx.Projectiles,
		DT:          ctx.DT,
		OnKill: func(e *enemy.Enemy) {
			if ctx.CB.OnWardenKill != nil {
				ctx.CB.OnWardenKill(e)
			}
		},
		OnFire: func() {
			if ctx.CB.OnWardenFire != nil {
				ctx.CB.OnWardenFire()
			}
		},
		OnSpecial: func() {
			if ctx.CB.OnWardenSpecial != nil {
				ctx.CB.OnWardenSpecial()
			}
		},
		OnDamage: func(x, y, dmg float64, crit bool) {
			if ctx.CB.OnWardenDamage != nil {
				ctx.CB.OnWardenDamage(x, y, dmg, crit)
			}
		},
	})
	return false
}

