// condition.go — 条件门接口及 11 种内置条件类型。
//
// Condition 是 descriptor 引擎的核心抽象之一，作为能力管线中的"门"，
// 决定后续 Effect 是否触发。多个条件通过 EvalAll 进行 AND 组合。
//
// 内置条件类型：
//   - ChanceCondition:       概率触发（rate 随强度缩放）
//   - CooldownCondition:     冷却计时（有状态，累计 elapsed）
//   - HpBelowCondition:      目标血量低于阈值
//   - HpAboveCondition:      目标血量高于阈值
//   - DistanceMinCondition:  目标距离不小于指定值
//   - NoNearbyTowerCondition: 附近无友方塔
//   - IsBossCondition:       目标为 Boss（Phase 2）
//   - NotBossCondition:      目标非 Boss（Phase 2）
//   - EveryCondition:        每 N 次触发（有状态，Phase 2）
//   - BuffActiveCondition:   目标存在指定 buff（Phase 2）
//   - BuffAbsentCondition:   目标不存在指定 buff（Phase 2）
package descriptor

import "math/rand/v2"

// Condition 条件门接口 — 根据上下文判断是否放行。
type Condition interface {
	Eval(ctx ConditionCtx) bool
}

// ConditionCtx 条件求值上下文，由管线每次触发时填充。
type ConditionCtx struct {
	Strength        float64 // 塔的当前强度
	TargetHpRatio   float64 // 目标血量比例 0-1
	TargetDistance   float64 // 到目标的距离
	NearestAllyDist float64 // 最近友方塔距离，无盟友时为 math.MaxFloat64
	Elapsed         float64 // 自上次管线触发以来的秒数（供冷却条件使用）

	// ── Phase 2 新增字段 ──
	IsBoss        bool     // 目标是否为 Boss
	ActiveBuffIDs []string // 目标身上的活跃 buff ID 列表
	HitCount      int      // 当前塔的累计命中次数（用于 every）
}

// EvalAll AND 组合器：所有条件都通过才返回 true。空列表返回 true。
func EvalAll(conds []Condition, ctx ConditionCtx) bool {
	for _, c := range conds {
		if !c.Eval(ctx) {
			return false
		}
	}
	return true
}

// ── ChanceCondition ─────────────────────────────────────────

// ChanceCondition 概率触发条件。
// Rate 通过 Scaler 计算，使得触发概率可随强度缩放。
type ChanceCondition struct {
	Rate Scaler
}

func (c ChanceCondition) Eval(ctx ConditionCtx) bool {
	rate := c.Rate.Calc(ctx.Strength)
	// rate >= 1.0 必定通过，rate <= 0.0 必定失败
	if rate >= 1.0 {
		return true
	}
	if rate <= 0.0 {
		return false
	}
	return rand.Float64() < rate
}

// ── CooldownCondition ───────────────────────────────────────

// CooldownCondition 冷却条件（有状态）。
// 首次调用无条件通过，之后累计 elapsed 直到达到冷却时间。
// 通过时重置累计器。
type CooldownCondition struct {
	Seconds   float64 // 冷却秒数
	fired     bool    // 是否已经触发过
	cumulated float64 // 自上次通过以来累计的时间
}

// NewCooldownCondition 创建指定冷却秒数的条件。
func NewCooldownCondition(seconds float64) *CooldownCondition {
	return &CooldownCondition{Seconds: seconds}
}

func (c *CooldownCondition) Eval(ctx ConditionCtx) bool {
	// 首次触发无条件通过
	if !c.fired {
		c.fired = true
		c.cumulated = 0
		return true
	}

	c.cumulated += ctx.Elapsed
	if c.cumulated >= c.Seconds {
		c.cumulated = 0 // 重置累计
		return true
	}
	return false
}

// ── HpBelowCondition ────────────────────────────────────────

// HpBelowCondition 目标血量低于阈值时通过（严格小于）。
type HpBelowCondition struct {
	Threshold Scaler
}

func (c HpBelowCondition) Eval(ctx ConditionCtx) bool {
	return ctx.TargetHpRatio < c.Threshold.Calc(ctx.Strength)
}

// ── HpAboveCondition ────────────────────────────────────────

// HpAboveCondition 目标血量高于阈值时通过（严格大于）。
type HpAboveCondition struct {
	Threshold Scaler
}

func (c HpAboveCondition) Eval(ctx ConditionCtx) bool {
	return ctx.TargetHpRatio > c.Threshold.Calc(ctx.Strength)
}

// ── DistanceMinCondition ────────────────────────────────────

// DistanceMinCondition 目标距离不小于指定值时通过（大于等于）。
type DistanceMinCondition struct {
	Distance float64
}

func (c DistanceMinCondition) Eval(ctx ConditionCtx) bool {
	return ctx.TargetDistance >= c.Distance
}

// ── NoNearbyTowerCondition ──────────────────────────────────

// NoNearbyTowerCondition 附近无友方塔时通过（最近盟友距离严格大于半径）。
type NoNearbyTowerCondition struct {
	Radius float64
}

func (c NoNearbyTowerCondition) Eval(ctx ConditionCtx) bool {
	return ctx.NearestAllyDist > c.Radius
}

// ── IsBossCondition ───────────────────────────────────────

// IsBossCondition 目标为 Boss 时通过。
type IsBossCondition struct{}

func (c IsBossCondition) Eval(ctx ConditionCtx) bool {
	return ctx.IsBoss
}

// ── NotBossCondition ──────────────────────────────────────

// NotBossCondition 目标非 Boss 时通过。
type NotBossCondition struct{}

func (c NotBossCondition) Eval(ctx ConditionCtx) bool {
	return !ctx.IsBoss
}

// ── EveryCondition ────────────────────────────────────────

// EveryCondition 每 N 次调用通过一次（有状态）。
// N <= 0 时永远不通过。
type EveryCondition struct {
	N       int // 每 N 次触发
	counter int // 内部计数器
}

// NewEveryCondition 创建每 N 次触发的条件。
func NewEveryCondition(n int) *EveryCondition {
	return &EveryCondition{N: n}
}

func (c *EveryCondition) Eval(_ ConditionCtx) bool {
	if c.N <= 0 {
		return false
	}
	c.counter++
	if c.counter >= c.N {
		c.counter = 0
		return true
	}
	return false
}

// ── BuffActiveCondition ───────────────────────────────────

// BuffActiveCondition 目标身上存在指定 buff 时通过。
type BuffActiveCondition struct {
	BuffID string
}

func (c BuffActiveCondition) Eval(ctx ConditionCtx) bool {
	for _, id := range ctx.ActiveBuffIDs {
		if id == c.BuffID {
			return true
		}
	}
	return false
}

// ── BuffAbsentCondition ───────────────────────────────────

// BuffAbsentCondition 目标身上不存在指定 buff 时通过。
type BuffAbsentCondition struct {
	BuffID string
}

func (c BuffAbsentCondition) Eval(ctx ConditionCtx) bool {
	for _, id := range ctx.ActiveBuffIDs {
		if id == c.BuffID {
			return false
		}
	}
	return true
}
