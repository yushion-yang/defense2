// passive_adapter.go — 被动技能到 CoreSkill 注册表的桥接适配器。
// 4 种被动充能技能，充满自动释放，各有不同效果。
package skill

import (
	"math"
	"math/rand"

	"defense2/internal/core/enemy"
)

func init() {
	Register("missileBarrage", func() CoreSkill {
		return newPassiveAdapter("missile-barrage")
	})
	Register("judgmentBeam", func() CoreSkill {
		return newPassiveAdapter("judgment-beam")
	})
	Register("chainLightningBolts", func() CoreSkill {
		return newPassiveAdapter("chain-lightning")
	})
	Register("judgmentRain", func() CoreSkill {
		return newPassiveAdapter("judgment-rain")
	})
}

// passiveAdapter 将 PassiveSkillState 包装为 CoreSkill。
type passiveAdapter struct {
	typeName string
	passive  *PassiveSkillState

	// 释放效果状态
	releasing    bool           // 正在释放
	releaseTimer float64        // 释放持续时间
	releaseDur   float64        // 释放总时长
	releaseHits  int            // 已命中次数
	releaseMax   int            // 最大命中次数
	hitInterval  float64        // 命中间隔
	hitTimer     float64        // 命中计时
	vfxPoints    [][2]float64   // VFX 点位
	vfxType      string         // VFX 类型
	targets      []*enemy.Enemy // 锁定的目标
}

func newPassiveAdapter(typeName string) *passiveAdapter {
	def := GetPassiveDefaults(typeName)
	return &passiveAdapter{
		typeName: typeName,
		passive:  NewPassiveSkill(def),
	}
}

func (a *passiveAdapter) Name() string { return a.typeName }

func (a *passiveAdapter) Init(_ interface{}) {
	a.passive.Charge = 0
	a.passive.Ready = false
	a.releasing = false
}

func (a *passiveAdapter) Tick(owner interface{}, enemies []*enemy.Enemy, dt float64, ctx *SkillContext) bool {
	// 释放阶段
	if a.releasing {
		a.tickRelease(owner, enemies, dt, ctx)
		if !a.releasing {
			return false // 释放完毕
		}
		return true // 释放中压制普攻
	}

	// 充能阶段
	a.passive.Tick(dt)

	// 充满自动释放
	if a.passive.Ready {
		a.startRelease(owner, enemies, ctx)
		a.passive.TryRelease()
		return true
	}

	return false
}

// startRelease 根据类型启动释放效果。
func (a *passiveAdapter) startRelease(owner interface{}, enemies []*enemy.Enemy, _ *SkillContext) {
	a.releasing = true
	ox, oy, rng := getOwnerPosAndRange(owner, 200)
	baseDmg := getOwnerDamage(owner, 10)

	switch a.typeName {
	case "missile-barrage":
		// 导弹齐射：对范围内随机 6 个目标各造成 5x 伤害
		a.vfxType = "barrage"
		a.releaseDur = 0.8
		a.releaseMax = 6
		a.hitInterval = 0.12
		a.targets = pickRandomTargets(enemies, ox, oy, rng, 6)
		a.releaseHits = 0
		a.hitTimer = 0
		a.vfxPoints = [][2]float64{{ox, oy}}
		_ = baseDmg // 伤害在 tickRelease 中算

	case "judgment-beam":
		// 审判光束：对范围内所有敌人造成 8x 伤害
		a.vfxType = "beam_burst"
		a.releaseDur = 0.5
		a.releaseMax = 1
		a.hitInterval = 0
		a.releaseHits = 0
		a.hitTimer = 0
		a.vfxPoints = [][2]float64{{ox, oy}}

	case "chain-lightning":
		// 闪电风暴：对范围内所有敌人造成 4x 伤害 + 链式
		a.vfxType = "lightning_storm"
		a.releaseDur = 0.6
		a.releaseMax = 1
		a.hitInterval = 0
		a.releaseHits = 0
		a.hitTimer = 0
		a.vfxPoints = [][2]float64{{ox, oy}}

	case "judgment-rain":
		// 审判之雨：对范围内 8 个目标各造成 3x 伤害（从天降下）
		a.vfxType = "rain"
		a.releaseDur = 1.0
		a.releaseMax = 8
		a.hitInterval = 0.1
		a.targets = pickRandomTargets(enemies, ox, oy, rng, 8)
		a.releaseHits = 0
		a.hitTimer = 0
		a.vfxPoints = [][2]float64{{ox, oy}}
	}

	a.releaseTimer = a.releaseDur
}

// tickRelease 每帧处理释放效果。
func (a *passiveAdapter) tickRelease(owner interface{}, enemies []*enemy.Enemy, dt float64, ctx *SkillContext) {
	a.releaseTimer -= dt
	ox, oy, rng := getOwnerPosAndRange(owner, 200)
	baseDmg := getOwnerDamage(owner, 10)

	switch a.typeName {
	case "missile-barrage":
		a.hitTimer += dt
		for a.hitTimer >= a.hitInterval && a.releaseHits < a.releaseMax {
			a.hitTimer -= a.hitInterval
			if a.releaseHits < len(a.targets) {
				t := a.targets[a.releaseHits]
				if t.Active && t.HP > 0 {
					dmg := baseDmg * 5
					applySkillDamage(t, dmg, ctx)
					a.vfxPoints = append(a.vfxPoints, [2]float64{t.X, t.Y})
				}
			}
			a.releaseHits++
		}

	case "judgment-beam":
		if a.releaseHits == 0 {
			a.releaseHits = 1
			dmg := baseDmg * 8
			for _, e := range enemies {
				if !e.Active || e.HP <= 0 {
					continue
				}
				if math.Hypot(e.X-ox, e.Y-oy) <= rng {
					applySkillDamage(e, dmg, ctx)
					a.vfxPoints = append(a.vfxPoints, [2]float64{e.X, e.Y})
				}
			}
		}

	case "chain-lightning":
		if a.releaseHits == 0 {
			a.releaseHits = 1
			dmg := baseDmg * 4
			for _, e := range enemies {
				if !e.Active || e.HP <= 0 {
					continue
				}
				if math.Hypot(e.X-ox, e.Y-oy) <= rng {
					applySkillDamage(e, dmg, ctx)
					a.vfxPoints = append(a.vfxPoints, [2]float64{e.X, e.Y})
				}
			}
		}

	case "judgment-rain":
		a.hitTimer += dt
		for a.hitTimer >= a.hitInterval && a.releaseHits < a.releaseMax {
			a.hitTimer -= a.hitInterval
			if a.releaseHits < len(a.targets) {
				t := a.targets[a.releaseHits]
				if t.Active && t.HP > 0 {
					dmg := baseDmg * 3
					applySkillDamage(t, dmg, ctx)
					a.vfxPoints = append(a.vfxPoints, [2]float64{t.X, t.Y})
				}
			}
			a.releaseHits++
		}
	}

	if a.releaseTimer <= 0 {
		a.releasing = false
		a.vfxPoints = nil
		a.targets = nil
	}
}

func (a *passiveAdapter) ShouldSuppressFire(_ interface{}) bool { return a.releasing }
func (a *passiveAdapter) ShouldSuppressMove(_ interface{}) bool { return false }

func (a *passiveAdapter) GetProgress(_ interface{}) (float64, bool) {
	if a.releasing {
		return 1.0, false
	}
	return a.passive.Progress(), a.passive.Ready
}

func (a *passiveAdapter) GetVFX() *SkillVFX {
	if !a.releasing || len(a.vfxPoints) == 0 {
		return nil
	}
	vfxType := "explosion" // 默认爆炸
	switch a.vfxType {
	case "barrage":
		vfxType = "barrage"
	case "beam_burst":
		vfxType = "beam_burst"
	case "lightning_storm":
		vfxType = "lightning"
	case "rain":
		vfxType = "rain"
	}
	return &SkillVFX{
		Type:   vfxType,
		Active: true,
		Points: a.vfxPoints,
		Timer:  a.releaseTimer,
	}
}

// applySkillDamage 被动技能伤害统一入口。
func applySkillDamage(e *enemy.Enemy, dmg float64, ctx *SkillContext) {
	e.HP -= dmg
	e.HitFlash = 0.12
	killed := e.HP <= 0
	if killed {
		e.Active = false
	}
	if ctx != nil && ctx.OnHit != nil {
		ctx.OnHit(e, dmg, killed)
	}
}

// pickRandomTargets 从范围内随机选 n 个存活敌人。
func pickRandomTargets(enemies []*enemy.Enemy, cx, cy, rng float64, n int) []*enemy.Enemy {
	var inRange []*enemy.Enemy
	for _, e := range enemies {
		if !e.Active || e.HP <= 0 {
			continue
		}
		if math.Hypot(e.X-cx, e.Y-cy) <= rng {
			inRange = append(inRange, e)
		}
	}
	if len(inRange) <= n {
		return inRange
	}
	perm := rand.Perm(len(inRange))
	result := make([]*enemy.Enemy, n)
	for i := 0; i < n; i++ {
		result[i] = inRange[perm[i]]
	}
	return result
}
