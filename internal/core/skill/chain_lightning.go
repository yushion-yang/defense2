// chain_lightning.go — 链式闪电技能。
// 冷却后向范围内敌人释放跳跃闪电，每跳递减伤害。
package skill

import (
	"defense2/internal/core/enemy"
	"math"
)

// ── 常量 ──

const (
	clCooldown         = 5.0   // 冷却时间（秒）
	clMaxTargets       = 8     // 最大跳跃目标数
	clDamageMultiplier = 4.0   // 伤害倍率（相对持有者基础伤害）
	clJumpRange        = 120.0 // 每跳搜索范围（像素）
	clJumpInterval     = 0.15  // 跳跃间隔（秒）
	clDefaultRange     = 200.0 // 默认攻击范围（持有者无范围时使用）
)

// chainLightning 链式闪电技能实现。
type chainLightning struct {
	skillBase
	jumpTimer float64        // 当前跳跃间隔计时
	jumpIndex int            // 当前跳跃索引
	targets   []*enemy.Enemy // 跳跃目标列表
}

func init() {
	Register("chainLightning", func() CoreSkill {
		return &chainLightning{}
	})
}

func (c *chainLightning) Name() string { return "chainLightning" }

func (c *chainLightning) Init(_ interface{}) {
	c.Timer = 0
	c.Ready = false
	c.Firing = false
	c.jumpIndex = 0
	c.targets = nil
}

func (c *chainLightning) Tick(owner interface{}, enemies []*enemy.Enemy, dt float64, ctx *SkillContext) bool {
	if c.Firing {
		c.jumpTimer += dt
		for c.jumpTimer >= clJumpInterval && c.jumpIndex < len(c.targets) {
			c.jumpTimer -= clJumpInterval
			t := c.targets[c.jumpIndex]
			if t.Active && t.HP > 0 {
				dmg := c.BaseDmg * clDamageMultiplier
				applySkillDamage(t, dmg, ctx)
			}
			c.jumpIndex++
		}
		if c.jumpIndex >= len(c.targets) {
			c.endFiring()
			c.targets = nil
			c.jumpIndex = 0
		}
		return true
	}
	c.tickCD(dt, clCooldown)
	if !c.tryActivate(owner, enemies, clDefaultRange) {
		return false
	}
	c.targets = findChainTargets(enemies, c.OX, c.OY, c.Rng, clMaxTargets, clJumpRange)
	if len(c.targets) == 0 {
		c.endFiring()
		return false
	}
	c.jumpTimer = 0
	c.jumpIndex = 0
	notifyActivate("chainLightning", ctx)
	return true
}

func (c *chainLightning) ShouldSuppressFire(_ interface{}) bool { return c.Firing }
func (c *chainLightning) ShouldSuppressMove(_ interface{}) bool { return false }

func (c *chainLightning) GetVFX() *SkillVFX {
	if !c.Firing || len(c.targets) == 0 {
		return nil
	}
	// 首个点为施法者位置，后续为跳跃目标
	pts := make([][2]float64, 0, c.jumpIndex+2)
	pts = append(pts, [2]float64{c.OX, c.OY})
	for i := 0; i <= c.jumpIndex && i < len(c.targets); i++ {
		t := c.targets[i]
		pts = append(pts, [2]float64{t.X, t.Y})
	}
	return &SkillVFX{Type: "lightning", Active: true, Points: pts, Timer: 0.3}
}

func (c *chainLightning) GetProgress(_ interface{}) (float64, bool) {
	return c.progress(clCooldown)
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
