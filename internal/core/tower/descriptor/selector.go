// selector.go — 目标选择器接口及 9 种实现。
//
// Selector 决定效果作用于哪些目标。
// 基础 6 种：当前目标/AOE/连锁/全范围/友方塔/自身。
// Phase 2 新增 3 种：锥形/360环形/随机。
package descriptor

import (
	"math"
	"math/rand/v2"
)

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
	if ctx.Enemies == nil {
		return nil
	}
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
	if ctx.Enemies == nil {
		return nil
	}
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
	if ctx.Towers == nil {
		return nil
	}
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
	if ctx.Enemies == nil {
		return nil
	}
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

// ── ConeSelector ─────────────────────────────────────────

// ConeSelector 锥形区域选择器。
// 以塔位置为起点、命中点方向为中心线，选取锥角范围内且在半径内的敌人。
type ConeSelector struct {
	Angle  float64 // 锥角（度数），例如 90 表示前方 ±45°
	Radius Scaler
}

func (s ConeSelector) Select(ctx SelectorCtx) []Target {
	if ctx.Enemies == nil {
		return nil
	}

	r := s.Radius.Calc(ctx.Strength)
	halfAngle := s.Angle / 2.0 * (math.Pi / 180.0) // 半锥角（弧度）

	// 塔到命中点的方向向量
	refDx := ctx.HitX - ctx.TowerX
	refDy := ctx.HitY - ctx.TowerY
	refLen := math.Sqrt(refDx*refDx + refDy*refDy)
	if refLen < 1e-9 {
		// 命中点与塔重合，无法确定方向，退化为全范围
		halfAngle = math.Pi
	}

	refs := ctx.Enemies.QueryRadius(ctx.TowerX, ctx.TowerY, r)
	var targets []Target

	for _, ref := range refs {
		dx := ref.X - ctx.TowerX
		dy := ref.Y - ctx.TowerY
		dist := math.Sqrt(dx*dx + dy*dy)

		// 在塔位置上的敌人总是被选中
		if dist < 1e-9 {
			targets = append(targets, Target{
				EnemyIdx: ref.Index, TowerIdx: -1, X: ref.X, Y: ref.Y,
			})
			continue
		}

		// 计算 (塔→敌人) 与 (塔→命中点) 之间的夹角
		if refLen >= 1e-9 {
			// cos(θ) = dot / (|a|*|b|)
			dot := dx*refDx + dy*refDy
			cosAngle := dot / (dist * refLen)
			// 钳制浮点误差
			if cosAngle > 1 {
				cosAngle = 1
			} else if cosAngle < -1 {
				cosAngle = -1
			}
			angle := math.Acos(cosAngle)
			if angle > halfAngle {
				continue
			}
		}

		targets = append(targets, Target{
			EnemyIdx: ref.Index, TowerIdx: -1, X: ref.X, Y: ref.Y,
		})
	}

	return targets
}

// ── Ring360Selector ──────────────────────────────────────

// Ring360Selector 360 度均匀方向选择器。
// 生成 Count 个均匀分布在塔周围的方向性目标点（类似 radial 攻击模式）。
// 返回的 Target 坐标在塔的攻击范围边界上，无具体敌人绑定。
type Ring360Selector struct {
	Count Scaler
}

func (s Ring360Selector) Select(ctx SelectorCtx) []Target {
	n := int(s.Count.Calc(ctx.Strength))
	if n <= 0 {
		return nil
	}

	targets := make([]Target, n)
	step := 2.0 * math.Pi / float64(n)

	for i := 0; i < n; i++ {
		angle := float64(i) * step
		targets[i] = Target{
			EnemyIdx: -1,
			TowerIdx: -1,
			X:        ctx.TowerX + ctx.TowerRange*math.Cos(angle),
			Y:        ctx.TowerY + ctx.TowerRange*math.Sin(angle),
		}
	}

	return targets
}

// ── RandomSelector ───────────────────────────────────────

// RandomSelector 随机选择器。
// 从塔周围半径内的敌人中随机选取最多 Count 个。
type RandomSelector struct {
	Count  Scaler
	Radius float64
}

func (s RandomSelector) Select(ctx SelectorCtx) []Target {
	if ctx.Enemies == nil {
		return nil
	}

	n := int(s.Count.Calc(ctx.Strength))
	if n <= 0 {
		return nil
	}

	refs := ctx.Enemies.QueryRadius(ctx.TowerX, ctx.TowerY, s.Radius)
	if len(refs) == 0 {
		return nil
	}

	// 复制一份避免修改原始数据，Fisher-Yates 洗牌后取前 n 个
	shuffled := make([]EnemyRef, len(refs))
	copy(shuffled, refs)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	if n > len(shuffled) {
		n = len(shuffled)
	}

	targets := make([]Target, n)
	for i := 0; i < n; i++ {
		targets[i] = Target{
			EnemyIdx: shuffled[i].Index,
			TowerIdx: -1,
			X:        shuffled[i].X,
			Y:        shuffled[i].Y,
		}
	}

	return targets
}
