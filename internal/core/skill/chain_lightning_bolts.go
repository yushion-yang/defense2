// chain_lightning_bolts.go — 闪电风暴技能。
// CD 14s → 召唤数道闪电在地图中 45° 角激射，碰边缘反弹 4 次后消失。
package skill

import (
	"math"
	"math/rand"

	"defense2/internal/core/enemy"
	"defense2/internal/core/game"
)

// ── 常量 ──

const (
	clbCooldown     = 5.0                        // 冷却时间（秒）
	clbBoltCount    = 5                          // 闪电道数
	clbDamageMul    = 4.0                        // 每次命中伤害倍率
	clbSpeed        = 600.0                      // 闪电移动速度（像素/秒）
	clbMaxBounces   = 4                          // 最大反弹次数
	clbHitRadius    = 16.0                       // 命中判定半径（像素）
	clbScreenW      = float64(game.ScreenWidth)  // 屏幕宽度
	clbScreenH      = float64(game.ScreenHeight) // 屏幕高度
	clbDefaultRange = 200.0                      // 默认攻击范围（CD检查用）
)

// bolt 单道闪电实体。
type bolt struct {
	x, y    float64
	vx, vy  float64
	bounces int
	active  bool
	hitIDs  map[int]bool // 已命中敌人 ID（同一闪电不重复命中同一敌人）
}

type chainLightningBoltsSkill struct {
	skillBase
	bolts  [clbBoltCount]bolt
	vfxPts [][2]float64 // 渲染用：所有活跃闪电位置
}

func init() {
	Register("chainLightningBolts", func() CoreSkill { return &chainLightningBoltsSkill{} })
}

func (s *chainLightningBoltsSkill) Name() string { return "chainLightningBolts" }
func (s *chainLightningBoltsSkill) Init(_ interface{}) {
	s.Timer = 0
	s.Ready = false
	s.Firing = false
}
func (s *chainLightningBoltsSkill) ShouldSuppressFire(_ interface{}) bool { return s.Firing }
func (s *chainLightningBoltsSkill) ShouldSuppressMove(_ interface{}) bool { return false }

func (s *chainLightningBoltsSkill) Tick(owner interface{}, enemies []*enemy.Enemy, dt float64, ctx *SkillContext) bool {
	if s.Firing {
		anyActive := false
		for i := range s.bolts {
			b := &s.bolts[i]
			if !b.active {
				continue
			}
			// 移动
			b.x += b.vx * dt
			b.y += b.vy * dt

			// 边缘反弹
			if b.x <= 0 || b.x >= clbScreenW {
				b.vx = -b.vx
				b.x = math.Max(0, math.Min(b.x, clbScreenW))
				b.bounces++
			}
			if b.y <= 0 || b.y >= clbScreenH {
				b.vy = -b.vy
				b.y = math.Max(0, math.Min(b.y, clbScreenH))
				b.bounces++
			}

			// 超过反弹次数则消失
			if b.bounces >= clbMaxBounces {
				b.active = false
				continue
			}

			// 命中检测
			for _, e := range enemies {
				if !e.Active || e.HP <= 0 || b.hitIDs[e.ID] {
					continue
				}
				if math.Hypot(e.X-b.x, e.Y-b.y) <= clbHitRadius {
					applySkillDamage(e, s.BaseDmg*clbDamageMul, ctx)
					b.hitIDs[e.ID] = true
				}
			}

			anyActive = true
		}

		if !anyActive {
			s.endFiring()
		}
		return true
	}

	s.tickCD(dt, clbCooldown)
	if !s.tryActivate(owner, enemies, clbDefaultRange) {
		return false
	}

	// 初始化闪电：从施法者位置出发，随机 45° 方向
	directions := [4][2]float64{
		{1, 1}, {1, -1}, {-1, 1}, {-1, -1},
	}
	for i := range s.bolts {
		dir := directions[i%4]
		// 加一点随机扰动避免完全重叠
		jitter := (rand.Float64() - 0.5) * 0.2
		s.bolts[i] = bolt{
			x:      s.OX,
			y:      s.OY,
			vx:     dir[0] * clbSpeed * math.Cos(math.Pi/4+jitter),
			vy:     dir[1] * clbSpeed * math.Sin(math.Pi/4+jitter),
			active: true,
			hitIDs: make(map[int]bool),
		}
	}
	notifyActivate("chainLightningBolts", ctx)
	return true
}

func (s *chainLightningBoltsSkill) GetProgress(_ interface{}) (float64, bool) {
	return s.progress(clbCooldown)
}

func (s *chainLightningBoltsSkill) GetVFX() *SkillVFX {
	if !s.Firing {
		return nil
	}
	pts := make([][2]float64, 0, clbBoltCount)
	for i := range s.bolts {
		if s.bolts[i].active {
			pts = append(pts, [2]float64{s.bolts[i].x, s.bolts[i].y})
		}
	}
	if len(pts) == 0 {
		return nil
	}
	return &SkillVFX{Type: "bolts", Active: true, Points: pts, Timer: 0.5}
}
