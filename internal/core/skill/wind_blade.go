// wind_blade.go — 风刃技能。
// CD 8s → 在施法者周围逐片生成 10 片穿透风刃，各自飞向范围内随机目标。
package skill

import (
	"math"
	"math/rand"

	"defense2/internal/core/enemy"
)

// ── 常量 ──

const (
	wbCooldown        = 5.0   // 冷却时间（秒）
	wbBladeCount      = 10    // 风刃数量
	wbStaggerInterval = 0.12  // 逐片生成间隔（秒）
	wbDamageMul       = 3.0   // 每片伤害倍率
	wbSpeed           = 400.0 // 风刃飞行速度
	wbPierceRadius    = 8.0   // 穿刺判定半径
	wbDefaultRange    = 200.0 // 默认攻击范围
	wbSpawnOffset     = 20.0  // 生成偏移半径（围绕施法者）
)

// windBlade 风刃技能实现。
type windBlade struct {
	skillBase
	staggerTimer float64        // 当前风刃生成计时
	bladeIndex   int            // 已生成风刃数
	enemies      []*enemy.Enemy // 缓存敌人列表（每帧更新）
}

func init() {
	Register("windBlade", func() CoreSkill {
		return &windBlade{}
	})
}

func (w *windBlade) Name() string { return "windBlade" }

func (w *windBlade) Init(_ interface{}) {
	w.Timer = 0
	w.Ready = false
	w.Firing = false
	w.bladeIndex = 0
}

func (w *windBlade) Tick(owner interface{}, enemies []*enemy.Enemy, dt float64, ctx *SkillContext) bool {
	if w.Firing {
		w.enemies = enemies // 缓存当前帧敌人
		w.staggerTimer += dt
		for w.staggerTimer >= wbStaggerInterval && w.bladeIndex < wbBladeCount {
			w.staggerTimer -= wbStaggerInterval
			w.spawnBlade(ctx)
			w.bladeIndex++
		}
		if w.bladeIndex >= wbBladeCount {
			w.endFiring()
			w.bladeIndex = 0
			w.enemies = nil
		}
		return true
	}
	w.tickCD(dt, wbCooldown)
	if !w.tryActivate(owner, enemies, wbDefaultRange) {
		return false
	}
	w.enemies = enemies
	w.staggerTimer = 0
	w.bladeIndex = 0
	notifyActivate("windBlade", ctx)
	return true
}

// spawnBlade 生成一片风刃，飞向范围内随机敌人。
func (w *windBlade) spawnBlade(ctx *SkillContext) {
	if ctx == nil || ctx.Projectiles == nil {
		return
	}
	spawner, ok := ctx.Projectiles.(bladeSpawner)
	if !ok {
		return
	}

	// 选择范围内随机目标
	targets := pickRandomTargets(w.enemies, w.OX, w.OY, w.Rng, 1)
	if len(targets) == 0 {
		return
	}
	t := targets[0]

	// 生成位置：施法者周围随机偏移
	angle := rand.Float64() * 2 * math.Pi
	sx := w.OX + math.Cos(angle)*wbSpawnOffset
	sy := w.OY + math.Sin(angle)*wbSpawnOffset

	// 计算飞向目标的速度向量
	dx := t.X - sx
	dy := t.Y - sy
	dist := math.Hypot(dx, dy)
	if dist < 1 {
		dist = 1
	}
	vx := dx / dist * wbSpeed
	vy := dy / dist * wbSpeed

	spawner.SpawnSkillProjectile(sx, sy, vx, vy, w.BaseDmg*wbDamageMul, wbPierceRadius, true)
}

func (w *windBlade) ShouldSuppressFire(_ interface{}) bool { return w.Firing }
func (w *windBlade) ShouldSuppressMove(_ interface{}) bool { return false }

func (w *windBlade) GetProgress(_ interface{}) (float64, bool) {
	return w.progress(wbCooldown)
}

func (w *windBlade) GetVFX() *SkillVFX {
	if !w.Firing {
		return nil
	}
	return &SkillVFX{Type: "blades", Active: true, Points: [][2]float64{{w.OX, w.OY}}, Timer: 0.5}
}

// bladeSpawner 风刃弹生成接口（由 projectile.Pool 实现）。
type bladeSpawner interface {
	SpawnSkillProjectile(x, y, vx, vy, damage, radius float64, pierce bool)
}
