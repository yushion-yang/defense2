// targeting.go — 塔索敌逻辑。
// 默认策略：锁定最近敌人，持续攻击直到离开射程或死亡，再切换新目标。
package tower

import (
	"math"

	"defense2/internal/core/enemy"
)

// AcquireTarget 返回塔当前应攻击的敌人。
// 优先沿用已锁定的目标（存活且在射程内），否则重新选最近的。
func AcquireTarget(t *Tower, pool *enemy.Pool) *enemy.Enemy {
	// 检查已锁定目标是否仍然有效
	if t.Target != nil && t.Target.Active && !t.Target.IsDying() && !t.Target.Stealthed {
		dx := t.Target.X - t.X
		dy := t.Target.Y - t.Y
		if math.Hypot(dx, dy) <= t.Range {
			return t.Target
		}
		// 目标超出射程，清除锁定
		t.Target = nil
	} else {
		t.Target = nil
	}

	// 重新选最近目标
	best := FindNearestEnemy(t, pool)
	if best != nil {
		t.Target = best
	}
	return best
}

// FindNearestEnemy 返回塔射程内最近的存活敌人，无目标时返回 nil。
// 纯搜索函数，不修改塔状态。
func FindNearestEnemy(t *Tower, pool *enemy.Pool) *enemy.Enemy {
	var best *enemy.Enemy
	bestDist := math.MaxFloat64

	pool.Each(func(e *enemy.Enemy) {
		if e.IsDying() || e.Stealthed {
			return
		}
		dx := e.X - t.X
		dy := e.Y - t.Y
		dist := math.Hypot(dx, dy)
		if dist <= t.Range && dist < bestDist {
			bestDist = dist
			best = e
		}
	})
	return best
}

// FindExtraTargets 返回塔射程内除 exclude 外最近的 count 个敌人。
// 用于多目标攻击：主目标由 AcquireTarget 选取，额外目标由此函数选取。
func FindExtraTargets(t *Tower, pool *enemy.Pool, count int, exclude *enemy.Enemy) []*enemy.Enemy {
	type candidate struct {
		e    *enemy.Enemy
		dist float64
	}
	var cands []candidate
	pool.Each(func(e *enemy.Enemy) {
		if e == exclude || e.IsDying() || e.Stealthed {
			return
		}
		dx := e.X - t.X
		dy := e.Y - t.Y
		dist := math.Hypot(dx, dy)
		if dist <= t.Range {
			cands = append(cands, candidate{e, dist})
		}
	})
	// 按距离排序，取前 count 个
	for i := 0; i < len(cands); i++ {
		for j := i + 1; j < len(cands); j++ {
			if cands[j].dist < cands[i].dist {
				cands[i], cands[j] = cands[j], cands[i]
			}
		}
	}
	result := make([]*enemy.Enemy, 0, count)
	for i := 0; i < count && i < len(cands); i++ {
		result = append(result, cands[i].e)
	}
	return result
}
