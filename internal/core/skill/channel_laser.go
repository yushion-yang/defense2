// channel_laser.go — 引导激光技能。
// CD 6s → 锁定最密集方向引导 2s 激光，DPS = 基础伤害 × 8。
package skill

import (
	"math"

	"defense2/internal/core/enemy"
)

// ── 常量 ──

const (
	clsCooldown     = 5.0   // 冷却时间（秒）
	clsDuration     = 2.0   // 引导持续时间（秒）
	clsDPSMul       = 8.0   // DPS 倍率
	clsRangeBonus   = 40.0  // 额外射程加成（像素）
	clsWidth        = 28.0  // 激光宽度（像素）
	clsTickRate     = 0.1   // 伤害 tick 间隔（秒）
	clsSectorCount  = 36    // 方向扫描扇区数（每 10 度）
	clsDefaultRange = 200.0 // 默认射程
)

// channelLaser 引导激光技能实现。
type channelLaser struct {
	skillBase
	elapsed    float64 // 引导已持续时间
	damageTick float64 // 伤害 tick 计时器
	angle      float64 // 锁定的激光角度（弧度）
	laserRange float64 // 激光射程
}

func init() {
	Register("channelLaser", func() CoreSkill {
		return &channelLaser{}
	})
}

func (l *channelLaser) Name() string { return "channelLaser" }

func (l *channelLaser) Init(_ interface{}) {
	l.Timer = 0
	l.Ready = false
	l.Firing = false
}

func (l *channelLaser) Tick(owner interface{}, enemies []*enemy.Enemy, dt float64, ctx *SkillContext) bool {
	if l.Firing {
		l.elapsed += dt
		l.damageTick += dt
		for l.damageTick >= clsTickRate {
			l.damageTick -= clsTickRate
			dps := l.BaseDmg * clsDPSMul
			dmgPerTick := dps * clsTickRate
			l.applyLaserDamage(enemies, dmgPerTick, ctx)
		}
		if l.elapsed >= clsDuration {
			l.endFiring()
		}
		return true
	}
	l.tickCD(dt, clsCooldown)
	if !l.tryActivate(owner, enemies, clsDefaultRange) {
		return false
	}
	l.laserRange = l.Rng + clsRangeBonus
	l.angle = FindBestLaserDirection(enemies, l.OX, l.OY, l.laserRange, clsWidth)
	l.elapsed = 0
	l.damageTick = 0
	notifyActivate("channelLaser", ctx)
	return true
}

func (l *channelLaser) ShouldSuppressFire(_ interface{}) bool { return l.Firing }
func (l *channelLaser) ShouldSuppressMove(_ interface{}) bool { return l.Firing }

func (l *channelLaser) GetProgress(_ interface{}) (float64, bool) {
	return l.progress(clsCooldown)
}

func (l *channelLaser) GetVFX() *SkillVFX {
	if !l.Firing {
		return nil
	}
	endX := l.OX + l.laserRange*math.Cos(l.angle)
	endY := l.OY + l.laserRange*math.Sin(l.angle)
	return &SkillVFX{
		Type: "laser", Active: true,
		Points: [][2]float64{{l.OX, l.OY}, {endX, endY}},
		Timer:  clsDuration - l.elapsed, Radius: float64(clsWidth), Angle: l.angle,
	}
}

// applyLaserDamage 对激光路径上的敌人造成伤害。
func (l *channelLaser) applyLaserDamage(enemies []*enemy.Enemy, dmg float64, ctx *SkillContext) {
	cosA := math.Cos(l.angle)
	sinA := math.Sin(l.angle)
	halfW := clsWidth / 2.0

	for _, e := range enemies {
		if !e.Active || e.HP <= 0 {
			continue
		}
		dx := e.X - l.OX
		dy := e.Y - l.OY
		along := dx*cosA + dy*sinA
		perp := math.Abs(-dx*sinA + dy*cosA)
		if along >= 0 && along <= l.laserRange && perp <= halfW {
			applySkillDamage(e, dmg, ctx)
		}
	}
}

// FindBestLaserDirection 扫描扇区找到覆盖最多敌人的激光方向。（公开供测试使用）
func FindBestLaserDirection(enemies []*enemy.Enemy, ox, oy, length, width float64) float64 {
	bestAngle := 0.0
	bestCount := 0
	halfW := width / 2.0

	for i := 0; i < clsSectorCount; i++ {
		a := float64(i) * 2 * math.Pi / float64(clsSectorCount)
		cosA := math.Cos(a)
		sinA := math.Sin(a)
		count := 0
		for _, e := range enemies {
			if !e.Active || e.HP <= 0 {
				continue
			}
			dx := e.X - ox
			dy := e.Y - oy
			along := dx*cosA + dy*sinA
			perp := math.Abs(-dx*sinA + dy*cosA)
			if along >= 0 && along <= length && perp <= halfW {
				count++
			}
		}
		if count > bestCount {
			bestCount = count
			bestAngle = a
		}
	}
	return bestAngle
}

// GetEnemiesInLaser 返回激光路径上的所有敌人。（公开供测试使用）
func GetEnemiesInLaser(enemies []*enemy.Enemy, ox, oy, angle, length, width float64) []*enemy.Enemy {
	cosA := math.Cos(angle)
	sinA := math.Sin(angle)
	halfW := width / 2.0
	var result []*enemy.Enemy
	for _, e := range enemies {
		if !e.Active || e.HP <= 0 {
			continue
		}
		dx := e.X - ox
		dy := e.Y - oy
		along := dx*cosA + dy*sinA
		perp := math.Abs(-dx*sinA + dy*cosA)
		if along >= 0 && along <= length && perp <= halfW {
			result = append(result, e)
		}
	}
	return result
}
