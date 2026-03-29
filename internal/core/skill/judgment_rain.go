// judgment_rain.go — 审判之雨技能。
// CD 20s → 全图持续一段时间随机位置召唤穿透雨滴，从天而降造成伤害。
package skill

import (
	"math"
	"math/rand"

	"defense2/internal/core/enemy"
	"defense2/internal/core/game"
)

// ── 常量 ──

const (
	jrCooldown     = 5.0                        // 冷却时间（秒）
	jrDuration     = 2.0                        // 持续生成雨滴时间（秒）
	jrSpawnRate    = 0.1                        // 雨滴生成间隔（秒）
	jrDamageMul    = 3.0                        // 每滴伤害倍率
	jrDropSpeed    = 500.0                      // 雨滴下落速度（像素/秒）
	jrHitRadius    = 10.0                       // 命中判定半径（像素）
	jrScreenW      = float64(game.ScreenWidth)  // 屏幕宽度
	jrScreenH      = float64(game.ScreenHeight) // 屏幕高度
	jrMaxDrops     = 30                         // 最大同时存在雨滴数
	jrDefaultRange = 200.0                      // 默认攻击范围（CD检查用）
)

// raindrop 单个雨滴实体。
type raindrop struct {
	x, y   float64
	active bool
	hitIDs map[int]bool // 穿透：已命中的敌人不再重复命中
}

type judgmentRainSkill struct {
	skillBase
	elapsed    float64 // 已持续时间
	spawnTimer float64 // 生成计时
	drops      [jrMaxDrops]raindrop
}

func init() {
	Register("judgmentRain", func() CoreSkill { return &judgmentRainSkill{} })
}

func (s *judgmentRainSkill) Name() string                          { return "judgmentRain" }
func (s *judgmentRainSkill) Init(_ interface{})                    { s.Timer = 0; s.Ready = false; s.Firing = false }
func (s *judgmentRainSkill) ShouldSuppressFire(_ interface{}) bool { return false }
func (s *judgmentRainSkill) ShouldSuppressMove(_ interface{}) bool { return false }

func (s *judgmentRainSkill) Tick(owner interface{}, enemies []*enemy.Enemy, dt float64, ctx *SkillContext) bool {
	if s.Firing {
		s.elapsed += dt

		// 持续时间内生成新雨滴
		if s.elapsed < jrDuration {
			s.spawnTimer += dt
			for s.spawnTimer >= jrSpawnRate {
				s.spawnTimer -= jrSpawnRate
				s.spawnDrop()
			}
		}

		// 更新所有雨滴
		anyActive := false
		for i := range s.drops {
			d := &s.drops[i]
			if !d.active {
				continue
			}
			d.y += jrDropSpeed * dt

			// 飞出屏幕底部则消失
			if d.y > jrScreenH+20 {
				d.active = false
				continue
			}

			// 命中检测（穿透）
			for _, e := range enemies {
				if !e.Active || e.HP <= 0 || d.hitIDs[e.ID] {
					continue
				}
				if math.Hypot(e.X-d.x, e.Y-d.y) <= jrHitRadius {
					applySkillDamage(e, s.BaseDmg*jrDamageMul, ctx)
					d.hitIDs[e.ID] = true
				}
			}

			anyActive = true
		}

		// 生成期结束且所有雨滴飞完
		if s.elapsed >= jrDuration && !anyActive {
			s.endFiring()
		}
		return true
	}

	s.tickCD(dt, jrCooldown)
	if !s.tryActivate(owner, enemies, jrDefaultRange) {
		return false
	}

	s.elapsed = 0
	s.spawnTimer = 0
	// 清空雨滴
	for i := range s.drops {
		s.drops[i].active = false
	}
	notifyActivate("judgmentRain", ctx)
	return true
}

// spawnDrop 在全图随机 X 位置生成一个雨滴（从屏幕顶部出发）。
func (s *judgmentRainSkill) spawnDrop() {
	for i := range s.drops {
		if !s.drops[i].active {
			s.drops[i] = raindrop{
				x:      rand.Float64() * jrScreenW,
				y:      -10, // 从屏幕顶部上方
				active: true,
				hitIDs: make(map[int]bool),
			}
			return
		}
	}
	// 槽位满则跳过
}

func (s *judgmentRainSkill) GetProgress(_ interface{}) (float64, bool) {
	return s.progress(jrCooldown)
}

func (s *judgmentRainSkill) GetVFX() *SkillVFX {
	if !s.Firing {
		return nil
	}
	pts := make([][2]float64, 0, jrMaxDrops)
	for i := range s.drops {
		if s.drops[i].active {
			pts = append(pts, [2]float64{s.drops[i].x, s.drops[i].y})
		}
	}
	if len(pts) == 0 {
		return nil
	}
	return &SkillVFX{Type: "rain", Active: true, Points: pts, Timer: jrDuration - s.elapsed}
}
