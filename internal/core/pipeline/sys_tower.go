// sys_tower.go — 塔相关步骤：动画、能力、技能、战斗。
package pipeline

import (
	"defense2/internal/core/tower"
)

// SysTowerAnim tick 塔建造/出售动画。
type SysTowerAnim struct {
	// OnSellComplete 塔出售动画完成时的回调（用于 invalidate map cache 等）。
	OnSellComplete func()
}

func (s SysTowerAnim) Tick(ctx *TickCtx) bool {
	ctx.Towers.Each(func(t *tower.Tower) {
		if t.BuildAnim > 0 {
			t.BuildAnim -= ctx.DT
			if t.BuildAnim < 0 {
				t.BuildAnim = 0
			}
		}
		if t.SellAnim > 0 {
			t.SellAnim -= ctx.DT
			if t.SellAnim <= 0 {
				ctx.Towers.Remove(t)
				if s.OnSellComplete != nil {
					s.OnSellComplete()
				}
			}
		}
	})
	return false
}

// SysTowerAbilities tick 所有塔的 Ticker 能力，收集经济产出。
type SysTowerAbilities struct{}

func (SysTowerAbilities) Tick(ctx *TickCtx) bool {
	goldEarned := TickTowerAbilities(ctx.Towers, ctx.Enemies, ctx.DT)
	*ctx.Gold += goldEarned
	return false
}

// SysTowerCombat 塔索敌射击。
type SysTowerCombat struct{}

func (SysTowerCombat) Tick(ctx *TickCtx) bool {
	TickTowerCombat(ctx.Towers, ctx.Enemies, ctx.Projectiles, ctx.Beams, ctx.DT,
		ctx.CB.OnTowerFire, ctx.CB.OnTowerDirectHit, ctx.CB.OnCC, ctx.CB.OnSplashVFX)
	return false
}
