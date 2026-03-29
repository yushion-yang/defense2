// missile_barrage.go — 导弹齐射技能。
// CD 12s → 360° 发射 6 个追踪导弹，先直飞一段距离后追踪最近敌人。
package skill

import (
	"math"

	"defense2/internal/core/enemy"
)

// ── 常量 ──

const (
	mbCooldown     = 5.0   // 冷却时间（秒）
	mbMissileCount = 6     // 导弹数量
	mbStagger      = 0.12  // 逐个发射间隔（秒）
	mbDamageMul    = 5.0   // 每颗导弹伤害倍率
	mbSpeed        = 300.0 // 导弹飞行速度
	mbLaunchOffset = 60.0  // 先直飞距离（像素）
	mbDefaultRange = 200.0 // 默认攻击范围
)

// missileSpawner 追踪导弹生成接口（由 projectile.Pool 实现）。
type missileSpawner interface {
	FireSkillMissile(sx, sy float64, target *enemy.Enemy, damage, speed float64)
}

type missileBarrageSkill struct {
	skillBase
	launchTimer float64
	launchIndex int
	enemies     []*enemy.Enemy
}

func init() {
	Register("missileBarrage", func() CoreSkill { return &missileBarrageSkill{} })
}

func (s *missileBarrageSkill) Name() string                          { return "missileBarrage" }
func (s *missileBarrageSkill) Init(_ interface{})                    { s.Timer = 0; s.Ready = false; s.Firing = false }
func (s *missileBarrageSkill) ShouldSuppressFire(_ interface{}) bool { return s.Firing }
func (s *missileBarrageSkill) ShouldSuppressMove(_ interface{}) bool { return false }

func (s *missileBarrageSkill) Tick(owner interface{}, enemies []*enemy.Enemy, dt float64, ctx *SkillContext) bool {
	if s.Firing {
		s.enemies = enemies
		s.launchTimer += dt
		for s.launchTimer >= mbStagger && s.launchIndex < mbMissileCount {
			s.launchTimer -= mbStagger
			s.launchMissile(ctx)
			s.launchIndex++
		}
		if s.launchIndex >= mbMissileCount {
			s.endFiring()
			s.enemies = nil
		}
		return true
	}
	s.tickCD(dt, mbCooldown)
	if !s.tryActivate(owner, enemies, mbDefaultRange) {
		return false
	}
	s.enemies = enemies
	s.launchTimer = 0
	s.launchIndex = 0
	notifyActivate("missileBarrage", ctx)
	return true
}

// launchMissile 发射一颗导弹：从 360° 均匀方向偏移后追踪最近敌人。
func (s *missileBarrageSkill) launchMissile(ctx *SkillContext) {
	if ctx == nil || ctx.Projectiles == nil {
		return
	}
	spawner, ok := ctx.Projectiles.(missileSpawner)
	if !ok {
		return
	}

	// 选择范围内最近的存活敌人作为追踪目标
	targets := pickRandomTargets(s.enemies, s.OX, s.OY, s.Rng, 1)
	if len(targets) == 0 {
		return
	}
	target := targets[0]

	// 360° 均匀方向发射，偏移一段距离后追踪
	angle := 2 * math.Pi * float64(s.launchIndex) / float64(mbMissileCount)
	sx := s.OX + math.Cos(angle)*mbLaunchOffset
	sy := s.OY + math.Sin(angle)*mbLaunchOffset

	spawner.FireSkillMissile(sx, sy, target, s.BaseDmg*mbDamageMul, mbSpeed)
}

func (s *missileBarrageSkill) GetProgress(_ interface{}) (float64, bool) {
	return s.progress(mbCooldown)
}

func (s *missileBarrageSkill) GetVFX() *SkillVFX {
	if !s.Firing {
		return nil
	}
	// 导弹由 projectile Pool 渲染，技能只显示发射指示
	return &SkillVFX{Type: "barrage", Active: true, Points: [][2]float64{{s.OX, s.OY}}, Timer: 0.5}
}
