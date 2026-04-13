// targeting.go — 塔索敌逻辑。
//
// 索敌策略采用"粘性瞄准"（sticky targeting）：
//   - 锁定最近敌人后，持续攻击直到目标离开射程、死亡、或进入隐身/无敌状态
//   - 只有当前目标失效时才切换到新的最近目标
//   - 这样做的原因：防止塔在两个距离相近的敌人间反复切换（抖动），
//     也让玩家能更直观地预测塔的行为
//
// 过滤规则（以下敌人不可被选为目标）：
//   - IsDying: 正在播放死亡动画（已被击杀）
//   - IsSpawning: 正在播放出场动画（尚未进入战场）
//   - IsStealthed: 隐身状态（stealth 行为激活时）
//
// 与其他文件的关系：
//   - tower.go: 读取 Tower.Target（粘性锁定）和 Tower.Range（射程判断）
//   - pipeline 中的 tower system: 每帧调用 AcquireTarget 更新塔的目标
//   - combat/: 读取 AcquireTarget 返回值决定是否开火
package tower

import (
	"math"

	"defense2/internal/core/enemy"
)

// AcquireTarget 返回塔当前应攻击的敌人。
// 优先沿用已锁定的目标（存活且在射程内），否则重新选最近的。
//
// 粘性瞄准流程：
//  1. 检查当前锁定目标是否仍然有效（存活 + 非死亡动画 + 非出场动画 + 非隐身 + 在射程内）
//  2. 有效 → 直接返回，不切换（粘性锁定的核心）
//  3. 无效 → 清空 Target，调用 FindNearestEnemy 搜索新目标
func AcquireTarget(t *Tower, pool *enemy.Pool) *enemy.Enemy {
	// 检查已锁定目标是否仍然有效（5 个条件全部满足才继续攻击）
	if t.Target != nil && t.Target.Active && !t.Target.IsDying() && !t.Target.IsSpawning() && !t.Target.IsStealthed() {
		dx := t.Target.X - t.X
		dy := t.Target.Y - t.Y
		if math.Hypot(dx, dy) <= t.Range {
			return t.Target // 粘性锁定：目标仍有效，不切换
		}
		// 目标超出射程，清除锁定（敌人走远了）
		t.Target = nil
	} else {
		t.Target = nil // 目标已死亡/隐身/出场中，清除锁定
	}

	// 当前无有效目标，搜索射程内最近的敌人
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
		if e.IsDying() || e.IsSpawning() || e.IsStealthed() {
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
// 用于 multiTarget 能力：主目标由 AcquireTarget 选取，额外目标由此函数补充。
// 返回数量可能少于 count（射程内敌人不够时），排序后取距离最近的前 count 个。
// 使用冒泡排序而非 sort.Slice，因为候选数量通常很少（<10），避免 sort 的接口分配。
func FindExtraTargets(t *Tower, pool *enemy.Pool, count int, exclude *enemy.Enemy) []*enemy.Enemy {
	type candidate struct {
		e    *enemy.Enemy
		dist float64
	}
	var cands []candidate
	pool.Each(func(e *enemy.Enemy) {
		if e == exclude || e.IsDying() || e.IsSpawning() || e.IsStealthed() {
			return
		}
		dx := e.X - t.X
		dy := e.Y - t.Y
		dist := math.Hypot(dx, dy)
		if dist <= t.Range {
			cands = append(cands, candidate{e, dist})
		}
	})
	// 冒泡排序按距离升序（候选数量少，O(n^2) 可接受且零分配）
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
