// thunder_smite.go — 天罚雷击技能。
// CD 后随机选 3 个敌人，各造成 %MaxHP 伤害（Boss 5% cap）。
package skill

import "defense2/internal/core/enemy"

// ── 常量 ──

const (
	tsCooldown       = 5.0    // 冷却时间（秒）
	tsMaxTargets     = 3      // 雷击目标数
	tsDamagePercent  = 0.10   // MaxHP 伤害比例（10%）
	tsBossPercentCap = 0.05   // Boss %HP 伤害上限（5%）
	tsHitInterval    = 0.12   // 逐个命中间隔（秒）
	tsDefaultRange   = 9999.0 // 全图范围（不受射程限制）
)

type thunderSmiteSkill struct {
	skillBase
	hitTimer float64
	hitIndex int
	targets  []*enemy.Enemy
	vfxPts   [][2]float64
	fireT    float64
}

func init() {
	Register("thunderSmite", func() CoreSkill { return &thunderSmiteSkill{} })
}

func (s *thunderSmiteSkill) Name() string                          { return "thunderSmite" }
func (s *thunderSmiteSkill) Init(_ interface{})                    { s.Timer = 0; s.Ready = false; s.Firing = false }
func (s *thunderSmiteSkill) ShouldSuppressFire(_ interface{}) bool { return false }
func (s *thunderSmiteSkill) ShouldSuppressMove(_ interface{}) bool { return false }

func (s *thunderSmiteSkill) Tick(owner interface{}, enemies []*enemy.Enemy, dt float64, ctx *SkillContext) bool {
	if s.Firing {
		s.hitTimer += dt
		for s.hitTimer >= tsHitInterval && s.hitIndex < len(s.targets) {
			s.hitTimer -= tsHitInterval
			t := s.targets[s.hitIndex]
			if t.Active && t.HP > 0 {
				dmg := t.MaxHP * tsDamagePercent
				if t.Boss && dmg > t.MaxHP*tsBossPercentCap {
					dmg = t.MaxHP * tsBossPercentCap
				}
				applySkillDamage(t, dmg, ctx)
				s.vfxPts = append(s.vfxPts, [2]float64{t.X, t.Y})
			}
			s.hitIndex++
		}
		s.fireT -= dt
		// 不在命中完毕时立刻结束，让闪电 VFX 持续展示到 fireT 耗尽
		if s.fireT <= 0 {
			s.endFiring()
			s.targets = nil
		}
		return false // 不压制普攻
	}
	s.tickCD(dt, tsCooldown)
	if !s.tryActivate(owner, enemies, tsDefaultRange) {
		return false
	}
	s.targets = pickRandomTargets(enemies, s.OX, s.OY, tsDefaultRange, tsMaxTargets) // 全图选目标
	if len(s.targets) == 0 {
		s.endFiring()
		return false
	}
	s.hitTimer = 0
	s.hitIndex = 0
	s.fireT = 1.0
	s.vfxPts = [][2]float64{{s.OX, s.OY}}
	notifyActivate("thunderSmite", ctx)
	return false
}

func (s *thunderSmiteSkill) GetProgress(_ interface{}) (float64, bool) {
	return s.progress(tsCooldown)
}

func (s *thunderSmiteSkill) GetVFX() *SkillVFX {
	if !s.Firing || len(s.vfxPts) == 0 {
		return nil
	}
	return &SkillVFX{Type: "thunder_strike", Active: true, Points: s.vfxPts, Timer: s.fireT}
}
