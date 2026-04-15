// selector.go — 目标选择器接口及基础实现（stub）。
//
// Selector 决定效果作用于哪些目标（当前目标/AOE/连锁/全范围/友方塔/自身）。
// 完整实现将在后续 Task 中补齐。
package descriptor

import "math"

// ── 引用类型 ─────────────────────────────────────────────

// EnemyRef 敌人引用，轻量值类型用于 Selector 查询。
// HpRatio 用于优先级选择（如连锁优先低血量目标）。
type EnemyRef struct {
	Index   int
	X, Y    float64
	HpRatio float64
	Active  bool
}

// TowerRef 塔引用，轻量值类型用于 Selector 查询。
type TowerRef struct {
	Index  int
	X, Y   float64
	Active bool
}

// ── 查询接口 ─────────────────────────────────────────────

// EnemyQuerier 敌人空间查询接口。
type EnemyQuerier interface {
	QueryRadius(cx, cy, radius float64) []EnemyRef
	AllActive() []EnemyRef
}

// TowerQuerier 友方塔空间查询接口。
type TowerQuerier interface {
	QueryRadius(cx, cy, radius float64) []TowerRef
}

// ── 选择上下文 ───────────────────────────────────────────

// SelectorCtx 选择器执行上下文。
type SelectorCtx struct {
	CurrentEnemyIdx int
	HitX, HitY      float64
	TowerX, TowerY   float64
	TowerRange       float64
	Strength         float64
	Enemies          EnemyQuerier
	Towers           TowerQuerier
}

// ── 目标结果 ─────────────────────────────────────────────

// Target 选择器返回的单个目标。
type Target struct {
	EnemyIdx int
	TowerIdx int
	X, Y     float64
}

// ── Selector 接口 ────────────────────────────────────────

// Selector 目标选择器接口。
type Selector interface {
	Select(ctx SelectorCtx) []Target
}

// ── CurrentTargetSelector ────────────────────────────────

// CurrentTargetSelector 选择当前命中的目标。
type CurrentTargetSelector struct{}

func (s CurrentTargetSelector) Select(ctx SelectorCtx) []Target {
	return []Target{{
		EnemyIdx: ctx.CurrentEnemyIdx,
		TowerIdx: -1,
		X:        ctx.HitX,
		Y:        ctx.HitY,
	}}
}

// ── AoeRadiusSelector ────────────────────────────────────

// AoeRadiusSelector 选择命中点半径内的所有敌人。
type AoeRadiusSelector struct {
	Radius Scaler
}

func (s AoeRadiusSelector) Select(ctx SelectorCtx) []Target {
	r := s.Radius.Calc(ctx.Strength)
	refs := ctx.Enemies.QueryRadius(ctx.HitX, ctx.HitY, r)
	targets := make([]Target, len(refs))
	for i, ref := range refs {
		targets[i] = Target{
			EnemyIdx: ref.Index,
			TowerIdx: -1,
			X:        ref.X,
			Y:        ref.Y,
		}
	}
	return targets
}

// ── AllInRangeSelector ───────────────────────────────────

// AllInRangeSelector 选择塔攻击范围内所有敌人。
type AllInRangeSelector struct{}

func (s AllInRangeSelector) Select(ctx SelectorCtx) []Target {
	refs := ctx.Enemies.QueryRadius(ctx.TowerX, ctx.TowerY, ctx.TowerRange)
	targets := make([]Target, len(refs))
	for i, ref := range refs {
		targets[i] = Target{
			EnemyIdx: ref.Index,
			TowerIdx: -1,
			X:        ref.X,
			Y:        ref.Y,
		}
	}
	return targets
}

// ── NearbyAlliesSelector ─────────────────────────────────

// NearbyAlliesSelector 选择塔周围指定半径内的友方塔。
type NearbyAlliesSelector struct {
	Radius float64
}

func (s NearbyAlliesSelector) Select(ctx SelectorCtx) []Target {
	refs := ctx.Towers.QueryRadius(ctx.TowerX, ctx.TowerY, s.Radius)
	targets := make([]Target, len(refs))
	for i, ref := range refs {
		targets[i] = Target{
			EnemyIdx: -1,
			TowerIdx: ref.Index,
			X:        ref.X,
			Y:        ref.Y,
		}
	}
	return targets
}

// ── SelfTowerSelector ────────────────────────────────────

// SelfTowerSelector 选择自身塔。
type SelfTowerSelector struct{}

func (s SelfTowerSelector) Select(ctx SelectorCtx) []Target {
	return []Target{{
		EnemyIdx: -1,
		TowerIdx: -1,
		X:        ctx.TowerX,
		Y:        ctx.TowerY,
	}}
}

// ── ChainSelector ────────────────────────────────────────

// ChainSelector 连锁弹射选择器。
type ChainSelector struct {
	MaxBounce  Scaler
	ChainRange float64
	DecayRatio float64
}

func (s ChainSelector) Select(ctx SelectorCtx) []Target {
	maxBounce := int(s.MaxBounce.Calc(ctx.Strength))
	if maxBounce <= 0 {
		return nil
	}

	visited := map[int]bool{ctx.CurrentEnemyIdx: true}
	allEnemies := ctx.Enemies.AllActive()

	var targets []Target
	curX, curY := ctx.HitX, ctx.HitY

	for bounce := 0; bounce < maxBounce; bounce++ {
		bestIdx := -1
		bestDist := math.MaxFloat64
		var bestRef EnemyRef

		for _, ref := range allEnemies {
			if visited[ref.Index] {
				continue
			}
			dx, dy := ref.X-curX, ref.Y-curY
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= s.ChainRange && dist < bestDist {
				bestDist = dist
				bestIdx = ref.Index
				bestRef = ref
			}
		}

		if bestIdx < 0 {
			break
		}

		visited[bestIdx] = true
		targets = append(targets, Target{
			EnemyIdx: bestIdx,
			TowerIdx: -1,
			X:        bestRef.X,
			Y:        bestRef.Y,
		})
		curX, curY = bestRef.X, bestRef.Y
	}

	return targets
}
