// interpreter.go — 描述符运行时解释器。
//
// Interpreter 包装一个预编译的 AbilityDescriptor，提供
// ExecOnHit / ExecOnTick 两个入口执行对应触发类型的管线。
//
// 执行流程（每条 Pipeline）：
//   1. 匹配 trigger type（跳过不匹配的管线）
//   2. 构建 ConditionCtx，EvalAll conditions
//   3. 构建 SelectorCtx，Select targets
//   4. 对每个 target，构建 EffectCtx，Apply 每个 effect
//   5. 收集所有 EffectResult
//
// 设计决策：
//   - TriggerContext 由调用方填充，解释器不持有游戏状态
//   - TargetEnemy 可为 nil（AOE 等无需主目标的场景）
//   - findNearestAllyDist 遍历所有 TowerQuerier 返回的塔，
//     无友方塔时返回 math.MaxFloat64
package descriptor

import "math"

// TriggerContext 运行时触发上下文，由调用方填充。
type TriggerContext struct {
	Type        TriggerType
	Strength    float64
	TowerDamage float64
	TowerX      float64
	TowerY      float64
	TowerRange  float64

	TargetEnemy *EnemyRef // 当前攻击目标（onHit 时非 nil，AOE 可为 nil）
	HitX        float64   // 命中坐标
	HitY        float64

	Enemies EnemyQuerier
	Towers  TowerQuerier

	DT      float64 // delta time（onTick 用）
	Elapsed float64 // 累计时间（cooldown 条件用）
}

// Interpreter 描述符运行时解释器。
type Interpreter struct {
	desc      *AbilityDescriptor
	pipelines []Pipeline // 从 desc.Pipelines 直接引用
	hasOnHit  bool
	hasOnTick bool
}

// NewInterpreter 从 AbilityDescriptor 创建解释器。
// 预扫描 pipelines 标记 hasOnHit/hasOnTick，避免运行时重复判断。
func NewInterpreter(desc *AbilityDescriptor) *Interpreter {
	interp := &Interpreter{
		desc:      desc,
		pipelines: desc.Pipelines,
	}
	for i := range interp.pipelines {
		switch interp.pipelines[i].Trigger {
		case TriggerOnHit:
			interp.hasOnHit = true
		case TriggerOnTick:
			interp.hasOnTick = true
		}
	}
	return interp
}

// HasOnHit 是否包含 onHit 管线。
func (interp *Interpreter) HasOnHit() bool { return interp.hasOnHit }

// HasOnTick 是否包含 onTick 管线。
func (interp *Interpreter) HasOnTick() bool { return interp.hasOnTick }

// ExecOnHit 执行所有 onHit 管线，返回效果列表。
func (interp *Interpreter) ExecOnHit(ctx TriggerContext) []EffectResult {
	ctx.Type = TriggerOnHit
	return interp.exec(ctx)
}

// ExecOnTick 执行所有 onTick 管线，返回效果列表。
func (interp *Interpreter) ExecOnTick(ctx TriggerContext) []EffectResult {
	ctx.Type = TriggerOnTick
	return interp.exec(ctx)
}

// exec 内部共用执行逻辑。
// 遍历所有管线，仅执行与 ctx.Type 匹配的管线。
func (interp *Interpreter) exec(ctx TriggerContext) []EffectResult {
	var results []EffectResult

	for i := range interp.pipelines {
		p := &interp.pipelines[i]

		// 步骤 1: 触发类型匹配
		if p.Trigger != ctx.Type {
			continue
		}

		// 步骤 2: 条件门 — 构建 ConditionCtx, EvalAll
		condCtx := buildConditionCtx(ctx)
		if !EvalAll(p.Conditions, condCtx) {
			continue
		}

		// 步骤 3: 目标选择 — 构建 SelectorCtx, Select
		selCtx := buildSelectorCtx(ctx)
		targets := p.Selector.Select(selCtx)

		// 步骤 4: 对每个 target 应用所有 effect
		for _, tgt := range targets {
			effCtx := buildEffectCtx(ctx, tgt)
			for _, eff := range p.Effects {
				results = append(results, eff.Apply(effCtx))
			}
		}
	}

	return results
}

// buildConditionCtx 从 TriggerContext 构建条件求值上下文。
func buildConditionCtx(ctx TriggerContext) ConditionCtx {
	cc := ConditionCtx{
		Strength:        ctx.Strength,
		Elapsed:         ctx.Elapsed,
		NearestAllyDist: findNearestAllyDist(ctx.Towers, ctx.TowerX, ctx.TowerY),
	}

	// TargetEnemy 可能为 nil（AOE/selfTower 等场景）
	if ctx.TargetEnemy != nil {
		cc.TargetHpRatio = ctx.TargetEnemy.HpRatio
		cc.TargetDistance = distance(ctx.TowerX, ctx.TowerY, ctx.TargetEnemy.X, ctx.TargetEnemy.Y)
	}

	return cc
}

// buildSelectorCtx 从 TriggerContext 构建选择器上下文。
func buildSelectorCtx(ctx TriggerContext) SelectorCtx {
	idx := -1
	if ctx.TargetEnemy != nil {
		idx = ctx.TargetEnemy.Index
	}
	return SelectorCtx{
		CurrentEnemyIdx: idx,
		HitX:            ctx.HitX,
		HitY:            ctx.HitY,
		TowerX:          ctx.TowerX,
		TowerY:          ctx.TowerY,
		TowerRange:      ctx.TowerRange,
		Strength:        ctx.Strength,
		Enemies:         ctx.Enemies,
		Towers:          ctx.Towers,
	}
}

// buildEffectCtx 从 TriggerContext + Target 构建效果上下文。
// TargetMaxHp 暂时为 0（EnemyRef 不携带 MaxHp，后续可扩展）。
func buildEffectCtx(ctx TriggerContext, _ Target) EffectCtx {
	return EffectCtx{
		Strength:    ctx.Strength,
		TowerDamage: ctx.TowerDamage,
		TargetMaxHp: 0,
	}
}

// findNearestAllyDist 查找最近友方塔的距离。
// 使用一个足够大的搜索半径查询所有附近的塔，
// 无友方塔时返回 math.MaxFloat64。
func findNearestAllyDist(towers TowerQuerier, tx, ty float64) float64 {
	if towers == nil {
		return math.MaxFloat64
	}

	// 用一个很大的半径获取所有可能的友方塔
	refs := towers.QueryRadius(tx, ty, math.MaxFloat64)
	minDist := math.MaxFloat64

	for _, ref := range refs {
		d := distance(tx, ty, ref.X, ref.Y)
		// 跳过自己（距离为 0 的通常是自身塔）
		if d < 1e-9 {
			continue
		}
		if d < minDist {
			minDist = d
		}
	}

	return minDist
}

// distance 计算两点之间的欧几里得距离。
func distance(x1, y1, x2, y2 float64) float64 {
	dx, dy := x2-x1, y2-y1
	return math.Sqrt(dx*dx + dy*dy)
}
