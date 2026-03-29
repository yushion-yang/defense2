// channel_laser.go — 引导激光技能。
// 冷却后扫描最优方向，锁定角度引导 2 秒持续伤害激光。
// 引导期间压制普攻和移动。
package skill

import (
	"defense2/internal/core/enemy"
	"math"
)

// ── 常量 ──

const (
	clsCooldown     = 6.0   // 冷却时间（秒）
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
	timer      float64 // 冷却计时器
	ready      bool    // 是否就绪
	active     bool    // 正在引导中
	elapsed    float64 // 引导已持续时间
	damageTick float64 // 伤害 tick 计时器
	angle      float64 // 锁定的激光角度（弧度）
	baseDmg    float64 // 基础伤害
	ownerX     float64 // 引导时持有者 X（锁定）
	ownerY     float64 // 引导时持有者 Y（锁定）
	laserRange float64 // 激光射程（持有者射程 + 加成）
}

func init() {
	Register("channelLaser", func() CoreSkill {
		return &channelLaser{}
	})
}

func (l *channelLaser) Name() string { return "channelLaser" }

func (l *channelLaser) Init(_ interface{}) {
	l.timer = 0
	l.ready = false
	l.active = false
	l.elapsed = 0
	l.damageTick = 0
	l.angle = 0
}

func (l *channelLaser) Tick(owner interface{}, enemies []*enemy.Enemy, dt float64, ctx *SkillContext) bool {
	// 引导阶段
	if l.active {
		l.elapsed += dt
		if l.elapsed >= clsDuration {
			l.active = false
			return false
		}

		// 每 tickRate 造成一次伤害
		l.damageTick += dt
		for l.damageTick >= clsTickRate {
			l.damageTick -= clsTickRate
			l.applyLaserDamage(enemies, ctx)
		}
		return true // 引导期间压制普攻
	}

	// 冷却阶段
	l.timer += dt
	if l.timer >= clsCooldown {
		l.ready = true
	}

	if !l.ready {
		return false
	}

	// 尝试激活：扫描最优方向
	ox, oy, rng := getOwnerPosAndRange(owner, clsDefaultRange)
	laserRange := rng + clsRangeBonus

	if !hasEnemyInRange(enemies, ox, oy, laserRange) {
		return false
	}

	bestAngle := FindBestLaserDirection(ox, oy, enemies, laserRange, clsWidth)

	// 获取基础伤害并锁定参数
	l.baseDmg = getOwnerDamage(owner, 10.0)
	l.ownerX = ox
	l.ownerY = oy
	l.laserRange = laserRange
	l.angle = bestAngle

	// 开始引导
	l.active = true
	l.elapsed = 0
	l.damageTick = 0
	l.timer = 0
	l.ready = false
	return true
}

func (l *channelLaser) ShouldSuppressFire(_ interface{}) bool { return l.active }
func (l *channelLaser) ShouldSuppressMove(_ interface{}) bool { return l.active }

func (l *channelLaser) GetVFX() *SkillVFX {
	if !l.active {
		return nil
	}
	endX := l.ownerX + l.laserRange*math.Cos(l.angle)
	endY := l.ownerY + l.laserRange*math.Sin(l.angle)
	return &SkillVFX{
		Type:   "laser",
		Active: true,
		Points: [][2]float64{{l.ownerX, l.ownerY}, {endX, endY}},
		Timer:  clsDuration - l.elapsed,
		Radius: float64(clsWidth),
		Angle:  l.angle,
	}
}

func (l *channelLaser) GetProgress(_ interface{}) (float64, bool) {
	if l.active {
		return l.elapsed / clsDuration, false
	}
	ratio := l.timer / clsCooldown
	if ratio > 1 {
		ratio = 1
	}
	return ratio, l.ready
}

// applyLaserDamage 对激光路径上的所有敌人造成伤害。
func (l *channelLaser) applyLaserDamage(enemies []*enemy.Enemy, ctx *SkillContext) {
	dmgPerTick := l.baseDmg * clsDPSMul * clsTickRate // DPS * tickInterval = 单次伤害

	targets := GetEnemiesInLaser(l.ownerX, l.ownerY, l.angle, l.laserRange, clsWidth, enemies)
	for _, e := range targets {
		e.HP -= dmgPerTick
		e.HitFlash = 0.1
		killed := e.HP <= 0
		if killed {
			e.Active = false
		}
		if ctx != nil && ctx.OnHit != nil {
			ctx.OnHit(e, dmgPerTick, killed)
		}
	}
}

// FindBestLaserDirection 扫描 36 个方向找出覆盖最多敌人的角度。
// 返回最优方向角度（弧度）。
func FindBestLaserDirection(ox, oy float64, enemies []*enemy.Enemy, laserRange, width float64) float64 {
	bestAngle := 0.0
	bestCount := 0

	for i := 0; i < clsSectorCount; i++ {
		angle := 2 * math.Pi * float64(i) / float64(clsSectorCount)
		count := countEnemiesInLaser(ox, oy, angle, laserRange, width, enemies)
		if count > bestCount {
			bestCount = count
			bestAngle = angle
		}
	}

	return bestAngle
}

// GetEnemiesInLaser 获取激光路径上的所有存活敌人。
// 激光表示为从 (ox,oy) 出发、角度 angle、长度 laserRange、宽度 width 的矩形。
func GetEnemiesInLaser(ox, oy, angle, laserRange, width float64, enemies []*enemy.Enemy) []*enemy.Enemy {
	var result []*enemy.Enemy
	halfW := width / 2

	// 激光方向单位向量
	dirX := math.Cos(angle)
	dirY := math.Sin(angle)

	for _, e := range enemies {
		if !e.Active || e.HP <= 0 {
			continue
		}

		// 敌人相对激光原点的偏移
		dx := e.X - ox
		dy := e.Y - oy

		// 投影到激光方向上（纵向距离）
		along := dx*dirX + dy*dirY
		if along < 0 || along > laserRange {
			continue
		}

		// 投影到激光法线上（横向距离）
		perp := math.Abs(dx*(-dirY) + dy*dirX)
		if perp <= halfW+e.Radius {
			result = append(result, e)
		}
	}

	return result
}

// countEnemiesInLaser 统计激光路径上的存活敌人数量。
func countEnemiesInLaser(ox, oy, angle, laserRange, width float64, enemies []*enemy.Enemy) int {
	count := 0
	halfW := width / 2
	dirX := math.Cos(angle)
	dirY := math.Sin(angle)

	for _, e := range enemies {
		if !e.Active || e.HP <= 0 {
			continue
		}
		dx := e.X - ox
		dy := e.Y - oy
		along := dx*dirX + dy*dirY
		if along < 0 || along > laserRange {
			continue
		}
		perp := math.Abs(dx*(-dirY) + dy*dirX)
		if perp <= halfW+e.Radius {
			count++
		}
	}

	return count
}
