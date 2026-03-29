// wind_blade.go — 风刃技能。
// 冷却后从持有者位置发射 10 个穿刺弹，等角度间隔呈放射状，逐个生成。
package skill

import (
	"defense2/internal/core/enemy"
	"math"
)

// ── 常量 ──

const (
	wbCooldown        = 8.0   // 冷却时间（秒）
	wbBladeCount      = 10    // 风刃数量
	wbStaggerInterval = 0.12  // 每个风刃生成间隔（秒）
	wbDamageMul       = 3.0   // 伤害倍率
	wbSpeed           = 400.0 // 风刃飞行速度（像素/秒）
	wbPierceRadius    = 12.0  // 穿刺碰撞半径（像素）
	wbDefaultRange    = 200.0 // 默认攻击范围
)

// windBlade 风刃技能实现。
type windBlade struct {
	timer        float64 // 冷却计时器
	ready        bool    // 是否就绪
	firing       bool    // 正在释放中
	staggerTimer float64 // 当前风刃生成计时
	bladeIndex   int     // 已生成风刃数
	baseDmg      float64 // 基础伤害
	ownerX       float64 // 释放时持有者 X（锁定）
	ownerY       float64 // 释放时持有者 Y（锁定）
}

func init() {
	Register("windBlade", func() CoreSkill {
		return &windBlade{}
	})
}

func (w *windBlade) Name() string { return "windBlade" }

func (w *windBlade) Init(_ interface{}) {
	w.timer = 0
	w.ready = false
	w.firing = false
	w.bladeIndex = 0
}

func (w *windBlade) Tick(owner interface{}, enemies []*enemy.Enemy, dt float64, ctx *SkillContext) bool {
	// 释放阶段：逐个生成风刃
	if w.firing {
		w.staggerTimer += dt
		for w.staggerTimer >= wbStaggerInterval && w.bladeIndex < wbBladeCount {
			w.staggerTimer -= wbStaggerInterval

			// 计算当前风刃角度（等角度间隔）
			angle := 2 * math.Pi * float64(w.bladeIndex) / float64(wbBladeCount)
			vx := math.Cos(angle) * wbSpeed
			vy := math.Sin(angle) * wbSpeed

			// 通过 SkillContext.Projectiles 发射（如果可用）
			if ctx != nil && ctx.Projectiles != nil {
				// 通过接口发射穿刺弹
				if spawner, ok := ctx.Projectiles.(bladeSpawner); ok {
					spawner.SpawnSkillProjectile(
						w.ownerX, w.ownerY,
						vx, vy,
						w.baseDmg*wbDamageMul,
						wbPierceRadius,
						true, // 穿刺
					)
				}
			}
			w.bladeIndex++
		}
		// 所有风刃生成完毕
		if w.bladeIndex >= wbBladeCount {
			w.firing = false
			w.bladeIndex = 0
		}
		return true // 释放期间压制普攻
	}

	// 冷却阶段
	w.timer += dt
	if w.timer >= wbCooldown {
		w.ready = true
	}

	if !w.ready {
		return false
	}

	// 尝试激活：范围内需要有敌人
	ox, oy, rng := getOwnerPosAndRange(owner, wbDefaultRange)
	if !hasEnemyInRange(enemies, ox, oy, rng) {
		return false
	}

	// 获取基础伤害并锁定发射位置
	w.baseDmg = getOwnerDamage(owner, 10.0)
	w.ownerX = ox
	w.ownerY = oy

	// 开始释放
	w.firing = true
	w.staggerTimer = 0
	w.bladeIndex = 0
	w.timer = 0
	w.ready = false
	return true
}

func (w *windBlade) ShouldSuppressFire(_ interface{}) bool { return w.firing }
func (w *windBlade) ShouldSuppressMove(_ interface{}) bool { return false }

func (w *windBlade) GetVFX() *SkillVFX {
	if !w.firing {
		return nil
	}
	return &SkillVFX{Type: "blades", Active: true, Points: [][2]float64{{w.ownerX, w.ownerY}}, Timer: 0.5}
}

func (w *windBlade) GetProgress(_ interface{}) (float64, bool) {
	if w.firing {
		return 1.0, false
	}
	ratio := w.timer / wbCooldown
	if ratio > 1 {
		ratio = 1
	}
	return ratio, w.ready
}

// bladeSpawner 风刃弹生成接口（由 projectile.Pool 实现）。
type bladeSpawner interface {
	SpawnSkillProjectile(x, y, vx, vy, damage, radius float64, pierce bool)
}
