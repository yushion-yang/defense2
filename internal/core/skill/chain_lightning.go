// chain_lightning.go — 链式闪电技能。
// 冷却后向范围内敌人释放跳跃闪电，每跳递减伤害。
package skill

import (
	"defense2/internal/core/enemy"
	"math"
)

// ── 常量 ──

const (
	clCooldown         = 7.0   // 冷却时间（秒）
	clMaxTargets       = 8     // 最大跳跃目标数
	clDamageMultiplier = 4.0   // 伤害倍率（相对持有者基础伤害）
	clDamageDecay      = 0.85  // 每跳伤害衰减倍率
	clJumpRange        = 120.0 // 每跳搜索范围（像素）
	clJumpInterval     = 0.15  // 跳跃间隔（秒）
	clDefaultRange     = 200.0 // 默认攻击范围（持有者无范围时使用）
)

// chainLightning 链式闪电技能实现。
type chainLightning struct {
	timer      float64        // 冷却计时器
	ready      bool           // 是否就绪
	firing     bool           // 正在释放中
	jumpTimer  float64        // 当前跳跃间隔计时
	jumpIndex  int            // 当前跳跃索引
	targets    []*enemy.Enemy // 跳跃目标列表
	baseDamage float64        // 基础伤害（从持有者获取）
}

func init() {
	Register("chainLightning", func() CoreSkill {
		return &chainLightning{}
	})
}

func (c *chainLightning) Name() string { return "chainLightning" }

func (c *chainLightning) Init(_ interface{}) {
	c.timer = 0
	c.ready = false
	c.firing = false
	c.jumpIndex = 0
	c.targets = nil
	c.baseDamage = 0
}

func (c *chainLightning) Tick(owner interface{}, enemies []*enemy.Enemy, dt float64, ctx *SkillContext) bool {
	// 释放阶段：逐跳应用伤害
	if c.firing {
		c.jumpTimer += dt
		for c.jumpTimer >= clJumpInterval && c.jumpIndex < len(c.targets) {
			c.jumpTimer -= clJumpInterval
			t := c.targets[c.jumpIndex]
			if t.Active && t.HP > 0 {
				// 每跳衰减伤害
				dmg := c.baseDamage * clDamageMultiplier * math.Pow(clDamageDecay, float64(c.jumpIndex))
				t.HP -= dmg
				t.HitFlash = 0.15
				killed := t.HP <= 0
				if killed {
					t.Active = false
				}
				if ctx != nil && ctx.OnHit != nil {
					ctx.OnHit(t, dmg, killed)
				}
			}
			c.jumpIndex++
		}
		// 所有跳跃完成
		if c.jumpIndex >= len(c.targets) {
			c.firing = false
			c.targets = nil
			c.jumpIndex = 0
		}
		return true // 释放期间压制普攻
	}

	// 冷却阶段
	c.timer += dt
	if c.timer >= clCooldown {
		c.ready = true
	}

	if !c.ready {
		return false
	}

	// 尝试激活：范围内需要有敌人
	ox, oy, rng := getOwnerPosAndRange(owner, clDefaultRange)
	if !hasEnemyInRange(enemies, ox, oy, rng) {
		return false
	}

	// 获取基础伤害
	c.baseDamage = getOwnerDamage(owner, 10.0)

	// 寻找跳跃链
	c.targets = findChainTargets(enemies, ox, oy, rng, clMaxTargets, clJumpRange)
	if len(c.targets) == 0 {
		return false
	}

	// 开始释放
	c.firing = true
	c.jumpTimer = 0
	c.jumpIndex = 0
	c.timer = 0
	c.ready = false
	return true
}

func (c *chainLightning) ShouldSuppressFire(_ interface{}) bool { return c.firing }
func (c *chainLightning) ShouldSuppressMove(_ interface{}) bool { return false }

func (c *chainLightning) GetVFX() *SkillVFX {
	if !c.firing || len(c.targets) == 0 {
		return nil
	}
	pts := make([][2]float64, 0, c.jumpIndex+1)
	for i := 0; i <= c.jumpIndex && i < len(c.targets); i++ {
		t := c.targets[i]
		pts = append(pts, [2]float64{t.X, t.Y})
	}
	return &SkillVFX{Type: "lightning", Active: true, Points: pts, Timer: 0.3}
}

func (c *chainLightning) GetProgress(_ interface{}) (float64, bool) {
	if c.firing {
		return 1.0, false
	}
	ratio := c.timer / clCooldown
	if ratio > 1 {
		ratio = 1
	}
	return ratio, c.ready
}

// findChainTargets 贪心最近邻跳跃链。
// 从 (ox,oy) 范围内找第一个目标，之后每跳在 jumpRange 内找最近未访问目标。
func findChainTargets(enemies []*enemy.Enemy, ox, oy, initRange float64, maxTargets int, jumpRange float64) []*enemy.Enemy {
	if len(enemies) == 0 {
		return nil
	}

	visited := make(map[int]bool)
	result := make([]*enemy.Enemy, 0, maxTargets)

	// 第一个目标：范围内最近的存活敌人
	first := findNearest(enemies, ox, oy, initRange, visited)
	if first == nil {
		return nil
	}
	result = append(result, first)
	visited[first.ID] = true

	// 后续跳跃
	for len(result) < maxTargets {
		last := result[len(result)-1]
		next := findNearest(enemies, last.X, last.Y, jumpRange, visited)
		if next == nil {
			break
		}
		result = append(result, next)
		visited[next.ID] = true
	}

	return result
}

// findNearest 在范围内找最近的未访问存活敌人。
func findNearest(enemies []*enemy.Enemy, cx, cy, rng float64, visited map[int]bool) *enemy.Enemy {
	var best *enemy.Enemy
	bestDist := math.MaxFloat64

	for _, e := range enemies {
		if !e.Active || e.HP <= 0 || visited[e.ID] {
			continue
		}
		dx := e.X - cx
		dy := e.Y - cy
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist <= rng && dist < bestDist {
			bestDist = dist
			best = e
		}
	}

	return best
}

// ── 持有者属性提取辅助函数 ──

// ownerWithPos 具有位置的持有者接口。
type ownerWithPos interface {
	GetX() float64
	GetY() float64
}

// ownerWithRange 具有攻击范围的持有者接口。
type ownerWithRange interface {
	GetRange() float64
}

// ownerWithDamage 具有伤害属性的持有者接口。
type ownerWithDamage interface {
	GetDamage() float64
}

// getOwnerPosAndRange 从持有者提取位置和范围。
// 持有者不满足接口时使用默认值。
func getOwnerPosAndRange(owner interface{}, defaultRange float64) (float64, float64, float64) {
	var ox, oy float64
	rng := defaultRange

	if p, ok := owner.(ownerWithPos); ok {
		ox = p.GetX()
		oy = p.GetY()
	}
	if r, ok := owner.(ownerWithRange); ok {
		rng = r.GetRange()
	}

	return ox, oy, rng
}

// getOwnerDamage 从持有者提取基础伤害。
func getOwnerDamage(owner interface{}, defaultDmg float64) float64 {
	if d, ok := owner.(ownerWithDamage); ok {
		return d.GetDamage()
	}
	return defaultDmg
}

// hasEnemyInRange 检查范围内是否有存活敌人。
func hasEnemyInRange(enemies []*enemy.Enemy, cx, cy, rng float64) bool {
	rng2 := rng * rng
	for _, e := range enemies {
		if !e.Active || e.HP <= 0 {
			continue
		}
		dx := e.X - cx
		dy := e.Y - cy
		if dx*dx+dy*dy <= rng2 {
			return true
		}
	}
	return false
}
