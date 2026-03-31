// skill_helpers.go — 技能共用基座 + 辅助函数。
package skill

import (
	"math"
	"math/rand"

	"defense2/internal/core/enemy"
)

// ── skillBase: 所有技能共用的 CD / 激活 / 属性基座 ──

// skillBase 8 个技能共有的状态字段和通用方法。
// 嵌入到每个技能 struct 中，减少重复代码。
type skillBase struct {
	Timer   float64 // CD 计时器（递增到 Cooldown 就绪）
	Ready   bool    // CD 好了
	Firing  bool    // 正在释放中
	BaseDmg float64 // 基础伤害（从 owner 获取）
	OX, OY  float64 // 释放时 owner 位置
	Rng     float64 // 释放时 owner 射程
}

// tickCD 递增 CD 计时器，到达 cooldown 后标记 ready。返回 true 表示 ready。
func (b *skillBase) tickCD(dt, cooldown float64) bool {
	b.Timer += dt
	if b.Timer >= cooldown {
		b.Ready = true
	}
	return b.Ready
}

// tryActivate 尝试激活技能：检查 ready + 范围内有敌人 + 提取 owner 属性。
// 成功返回 true 并重置 CD。
func (b *skillBase) tryActivate(owner interface{}, enemies []*enemy.Enemy, defaultRange float64) bool {
	if !b.Ready {
		return false
	}
	ox, oy, rng := getOwnerPosAndRange(owner, defaultRange)
	if !hasEnemyInRange(enemies, ox, oy, rng) {
		return false
	}
	b.BaseDmg = getOwnerDamage(owner, 10)
	b.OX, b.OY, b.Rng = ox, oy, rng
	b.Timer = 0
	b.Ready = false
	b.Firing = true
	return true
}

// endFiring 结束释放。
func (b *skillBase) endFiring() {
	b.Firing = false
}

// progress 返回 CD 进度（0~1）和是否 ready。释放中返回 (1, false)。
func (b *skillBase) progress(cooldown float64) (float64, bool) {
	if b.Firing {
		return 1, false
	}
	r := b.Timer / cooldown
	if r > 1 {
		r = 1
	}
	return r, b.Ready
}

// notifyActivate 通知技能激活（触发音效等）。
func notifyActivate(skillKey string, ctx *SkillContext) {
	if ctx != nil && ctx.OnActivate != nil {
		ctx.OnActivate(skillKey)
	}
}

// applySkillDamage 技能伤害统一入口。
// 注意：不直接设 e.Active=false，由 TickEnemyStatusEffects 安全网统一调 Pool.Kill()
// 确保 Pool.Count 正确递减，否则 CheckVictory 永远不触发。
func applySkillDamage(e *enemy.Enemy, dmg float64, ctx *SkillContext) {
	e.HP -= dmg
	if e.HitFlash < 0.06 {
		e.HitFlash = 0.12
	}
	killed := e.HP <= 0
	if ctx != nil && ctx.OnHit != nil {
		ctx.OnHit(e, dmg, killed)
	}
}

// pickRandomTargets 从范围内随机选 n 个存活敌人。
func pickRandomTargets(enemies []*enemy.Enemy, cx, cy, rng float64, n int) []*enemy.Enemy {
	var inRange []*enemy.Enemy
	for _, e := range enemies {
		if !e.Active || e.HP <= 0 {
			continue
		}
		if math.Hypot(e.X-cx, e.Y-cy) <= rng {
			inRange = append(inRange, e)
		}
	}
	if len(inRange) <= n {
		return inRange
	}
	perm := rand.Perm(len(inRange))
	result := make([]*enemy.Enemy, n)
	for i := 0; i < n; i++ {
		result[i] = inRange[perm[i]]
	}
	return result
}

// beamDamage 对光束路径上的敌人造成伤害。
func beamDamage(enemies []*enemy.Enemy, ox, oy, angle, length, width, dmg float64, ctx *SkillContext) {
	cosA, sinA := math.Cos(angle), math.Sin(angle)
	halfW := width / 2
	for _, e := range enemies {
		if !e.Active || e.HP <= 0 {
			continue
		}
		dx, dy := e.X-ox, e.Y-oy
		along := dx*cosA + dy*sinA
		perp := math.Abs(-dx*sinA + dy*cosA)
		if along >= 0 && along <= length && perp <= halfW {
			applySkillDamage(e, dmg, ctx)
		}
	}
}

// findBestAngle 找到覆盖最多敌人的射击角度（36 方向扫描）。
func findBestAngle(enemies []*enemy.Enemy, ox, oy float64) float64 {
	bestAngle := 0.0
	bestCount := 0
	for i := 0; i < 36; i++ {
		a := float64(i) * math.Pi / 18
		count := 0
		cosA, sinA := math.Cos(a), math.Sin(a)
		for _, e := range enemies {
			if !e.Active || e.HP <= 0 {
				continue
			}
			dx, dy := e.X-ox, e.Y-oy
			along := dx*cosA + dy*sinA
			perp := math.Abs(-dx*sinA + dy*cosA)
			if along > 0 && perp < 30 {
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
