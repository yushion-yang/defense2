// nuke_bomb.go — 核弹技能。
// 冷却后向敌人最密集区域发射飞弹，爆炸造成距离衰减 AoE 伤害。
package skill

import (
	"defense2/internal/core/enemy"
	"math"
)

// ── 常量 ──

const (
	nbCooldown          = 12.0  // 冷却时间（秒）
	nbDamageMultiplier  = 15.0  // 伤害倍率
	nbBlastRadius       = 150.0 // 爆炸半径（像素）
	nbEdgeDamageFactor  = 0.3   // 爆炸边缘伤害系数（中心100%，边缘30%）
	nbBombSpeed         = 240.0 // 飞弹速度（像素/秒）
	nbScanRadius        = 180.0 // 密度扫描半径（像素）
	nbDefaultRange      = 300.0 // 默认攻击范围
	nbBossWeight        = 5.0   // Boss 在密度计算中的权重
)

// nukeBomb 核弹技能实现。
type nukeBomb struct {
	timer     float64 // 冷却计时器
	ready     bool    // 是否就绪
	flying    bool    // 飞弹飞行中
	posX      float64 // 飞弹当前 X
	posY      float64 // 飞弹当前 Y
	targetX   float64 // 目标点 X
	targetY   float64 // 目标点 Y
	exploding bool    // 正在爆炸
	expTimer  float64 // 爆炸持续计时
	expPosX   float64 // 爆炸中心 X
	expPosY   float64 // 爆炸中心 Y
	baseDmg   float64 // 基础伤害
}

func init() {
	Register("nukeBomb", func() CoreSkill {
		return &nukeBomb{}
	})
}

func (n *nukeBomb) Name() string { return "nukeBomb" }

func (n *nukeBomb) Init(_ interface{}) {
	n.timer = 0
	n.ready = false
	n.flying = false
	n.exploding = false
}

func (n *nukeBomb) Tick(owner interface{}, enemies []*enemy.Enemy, dt float64, ctx *SkillContext) bool {
	// 爆炸阶段：应用 AoE 伤害（单帧）
	if n.exploding {
		n.expTimer += dt
		if n.expTimer >= 0.3 { // 爆炸视觉持续 0.3 秒
			n.exploding = false
		}
		return false
	}

	// 飞弹飞行阶段
	if n.flying {
		dx := n.targetX - n.posX
		dy := n.targetY - n.posY
		dist := math.Sqrt(dx*dx + dy*dy)
		step := nbBombSpeed * dt

		if step >= dist {
			// 到达目标：爆炸
			n.flying = false
			n.exploding = true
			n.expTimer = 0
			n.expPosX = n.targetX
			n.expPosY = n.targetY
			n.applyExplosion(enemies, ctx)
		} else {
			// 继续飞行
			n.posX += dx / dist * step
			n.posY += dy / dist * step
		}
		return true // 飞行期间压制普攻
	}

	// 冷却阶段
	n.timer += dt
	if n.timer >= nbCooldown {
		n.ready = true
	}

	if !n.ready {
		return false
	}

	// 尝试激活：扫描最密集区域
	ox, oy, _ := getOwnerPosAndRange(owner, nbDefaultRange)
	tx, ty, found := findDensestCluster(enemies, nbScanRadius)
	if !found {
		return false
	}

	// 获取基础伤害
	n.baseDmg = getOwnerDamage(owner, 10.0)

	// 发射飞弹
	n.posX = ox
	n.posY = oy
	n.targetX = tx
	n.targetY = ty
	n.flying = true
	n.timer = 0
	n.ready = false
	return true
}

func (n *nukeBomb) ShouldSuppressFire(_ interface{}) bool { return n.flying }
func (n *nukeBomb) ShouldSuppressMove(_ interface{}) bool { return false }

func (n *nukeBomb) GetProgress(_ interface{}) (float64, bool) {
	if n.flying || n.exploding {
		return 1.0, false
	}
	ratio := n.timer / nbCooldown
	if ratio > 1 {
		ratio = 1
	}
	return ratio, n.ready
}

// applyExplosion 爆炸伤害：距离衰减公式 dmg * (1 - (1-edgeFactor) * dist/blastRadius)。
func (n *nukeBomb) applyExplosion(enemies []*enemy.Enemy, ctx *SkillContext) {
	totalDmg := n.baseDmg * nbDamageMultiplier

	for _, e := range enemies {
		if !e.Active || e.HP <= 0 {
			continue
		}
		dx := e.X - n.expPosX
		dy := e.Y - n.expPosY
		dist := math.Sqrt(dx*dx + dy*dy)

		if dist > nbBlastRadius {
			continue
		}

		// 距离衰减：中心满伤害，边缘 edgeDamageFactor 倍
		factor := 1.0 - (1.0-nbEdgeDamageFactor)*(dist/nbBlastRadius)
		dmg := totalDmg * factor

		e.HP -= dmg
		e.HitFlash = 0.2
		killed := e.HP <= 0
		if killed {
			e.Active = false
		}
		if ctx != nil && ctx.OnHit != nil {
			ctx.OnHit(e, dmg, killed)
		}
	}
}

// findDensestCluster 寻找敌人最密集的位置（O(n^2)，Boss 权重 5 倍）。
// 返回密度最高的敌人坐标作为爆炸中心。
func findDensestCluster(enemies []*enemy.Enemy, scanRadius float64) (float64, float64, bool) {
	bestScore := 0.0
	var bestX, bestY float64
	found := false
	scanR2 := scanRadius * scanRadius

	for _, center := range enemies {
		if !center.Active || center.HP <= 0 {
			continue
		}

		score := 0.0
		for _, other := range enemies {
			if !other.Active || other.HP <= 0 {
				continue
			}
			dx := other.X - center.X
			dy := other.Y - center.Y
			if dx*dx+dy*dy <= scanR2 {
				if other.Boss {
					score += nbBossWeight
				} else {
					score += 1.0
				}
			}
		}

		if score > bestScore {
			bestScore = score
			bestX = center.X
			bestY = center.Y
			found = true
		}
	}

	return bestX, bestY, found
}
