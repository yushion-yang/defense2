// sys_warden.go — 战灵行为 + 战灵技能步骤。
package pipeline

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/skill"
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

// SysWardenSkill 驱动战灵技能。
type SysWardenSkill struct{}

func (SysWardenSkill) Tick(ctx *TickCtx) bool {
	if !ctx.WardenReady || ctx.WardenUnit == nil || ctx.WardenUnit.Skill == nil {
		return false
	}
	base := ctx.WardenUnit.BaseState()
	if base == nil {
		return false
	}
	var enemySlice []*enemy.Enemy
	ctx.Enemies.Each(func(e *enemy.Enemy) {
		if !e.IsDying() {
			enemySlice = append(enemySlice, e)
		}
	})
	skill.TickEntitySkill(ctx.WardenUnit.Skill, base, enemySlice, ctx.DT, ctx.BuildSkillCtx())
	return false
}
