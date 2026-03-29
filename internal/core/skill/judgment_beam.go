// judgment_beam.go — 审判光束技能。
// CD 18s → 全图召唤两道纵向宽幅光束左右来回移动，对触碰敌人造成伤害。
package skill

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/game"
)

// ── 常量 ──

const (
	jbCooldown     = 5.0                        // 冷却时间（秒）
	jbDuration     = 1.0                        // 光束持续时间（秒）
	jbBeamWidth    = 30.0                       // 光束宽度（像素）
	jbDPSMul       = 8.0                        // DPS 倍率
	jbTickRate     = 0.1                        // 伤害 tick 间隔（秒）
	jbScreenW      = float64(game.ScreenWidth)  // 屏幕宽度
	jbScreenH      = float64(game.ScreenHeight) // 屏幕高度
	jbDefaultRange = 200.0                      // 默认攻击范围（CD检查用）
)

type judgmentBeamSkill struct {
	skillBase
	elapsed float64 // 已持续时间
	dmgTick float64 // 伤害 tick 计时
	beam1X  float64 // 光束1 当前 X（从左向右）
	beam2X  float64 // 光束2 当前 X（从右向左）
	speed1  float64 // 光束1 移动速度
	speed2  float64 // 光束2 移动速度
}

func init() {
	Register("judgmentBeam", func() CoreSkill { return &judgmentBeamSkill{} })
}

func (s *judgmentBeamSkill) Name() string                          { return "judgmentBeam" }
func (s *judgmentBeamSkill) Init(_ interface{})                    { s.Timer = 0; s.Ready = false; s.Firing = false }
func (s *judgmentBeamSkill) ShouldSuppressFire(_ interface{}) bool { return false }
func (s *judgmentBeamSkill) ShouldSuppressMove(_ interface{}) bool { return false }

func (s *judgmentBeamSkill) Tick(owner interface{}, enemies []*enemy.Enemy, dt float64, ctx *SkillContext) bool {
	if s.Firing {
		s.elapsed += dt

		// 移动光束
		s.beam1X += s.speed1 * dt
		s.beam2X += s.speed2 * dt

		// 伤害 tick
		s.dmgTick += dt
		for s.dmgTick >= jbTickRate {
			s.dmgTick -= jbTickRate
			dmgPerTick := s.BaseDmg * jbDPSMul * jbTickRate
			s.applyBeamDamage(enemies, s.beam1X, dmgPerTick, ctx)
			s.applyBeamDamage(enemies, s.beam2X, dmgPerTick, ctx)
		}

		if s.elapsed >= jbDuration {
			s.endFiring()
		}
		return true
	}

	s.tickCD(dt, jbCooldown)
	if !s.tryActivate(owner, enemies, jbDefaultRange) {
		return false
	}

	// 光束1 从左端向右，光束2 从右端向左
	s.beam1X = 0
	s.beam2X = jbScreenW
	s.speed1 = jbScreenW / jbDuration // 1s 扫完全屏
	s.speed2 = -jbScreenW / jbDuration
	s.elapsed = 0
	s.dmgTick = 0
	notifyActivate("judgmentBeam", ctx)
	return true
}

// applyBeamDamage 对纵向光束矩形内的敌人造成伤害。
func (s *judgmentBeamSkill) applyBeamDamage(enemies []*enemy.Enemy, beamX, dmg float64, ctx *SkillContext) {
	halfW := jbBeamWidth / 2
	for _, e := range enemies {
		if !e.Active || e.HP <= 0 {
			continue
		}
		if e.X >= beamX-halfW && e.X <= beamX+halfW {
			applySkillDamage(e, dmg, ctx)
		}
	}
}

func (s *judgmentBeamSkill) GetProgress(_ interface{}) (float64, bool) {
	return s.progress(jbCooldown)
}

func (s *judgmentBeamSkill) GetVFX() *SkillVFX {
	if !s.Firing {
		return nil
	}
	return &SkillVFX{
		Type:   "sweep_beam",
		Active: true,
		Points: [][2]float64{{s.beam1X, 0}, {s.beam2X, 0}},
		Timer:  jbDuration - s.elapsed,
		Radius: jbBeamWidth,
	}
}
