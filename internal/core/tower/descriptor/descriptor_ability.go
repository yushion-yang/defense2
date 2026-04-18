// descriptor_ability.go — 描述符驱动能力的 tower.Ability/Ticker 适配器。
//
// 本文件是描述符引擎与现有塔能力系统的桥接层。
// 核心挑战：Go 不支持运行时动态实现接口，而 pipeline/tick_abilities.go
// 通过类型断言 `ability.(tower.Ticker)` 判断能力是否需要每帧 tick。
// 因此仅有 onHit 的描述符不能实现 Ticker，否则 pipeline 会白白每帧调用它。
//
// 解决方案：两种具体类型 + 工厂函数分派：
//   - DescriptorAbilityHit:  仅实现 tower.Ability（无 onTick 管线时）
//   - DescriptorAbilityFull: 实现 tower.Ability + tower.Ticker（有 onTick 管线时）
//   - NewDescriptorAbility:  根据描述符内容返回正确的类型
//
// 池适配器（poolEnemyQuerier / poolTowerQuerier）将 enemy.Pool / tower.Pool
// 适配到 EnemyQuerier / TowerQuerier 接口，供 onTick 管线执行空间查询。
package descriptor

import (
	"defense2/internal/core/buff"
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

// ── 共享基类 ─────────────────────────────────────────────────

// descriptorAbilityBase 描述符能力的共享字段。
type descriptorAbilityBase struct {
	desc   *AbilityDescriptor
	interp *Interpreter
}

// Name 返回描述符 ID（与 tower.Ability 接口契约一致）。
func (a *descriptorAbilityBase) Name() string { return a.desc.ID }

// buildHitTriggerContext 从 tower/projectile/enemy 构建 TriggerContext。
// enemy 可为 nil（AOE 等无需主目标的场景）。
func (a *descriptorAbilityBase) buildHitTriggerContext(
	t *tower.Tower, _ *projectile.Projectile, e *enemy.Enemy,
) TriggerContext {
	ctx := TriggerContext{
		Type:        TriggerOnHit,
		Strength:    t.EffectiveStrength(),
		TowerDamage: t.Damage,
		TowerX:      t.X,
		TowerY:      t.Y,
		TowerRange:  t.Range,
	}
	if e != nil {
		ref := EnemyRef{
			X:       e.X,
			Y:       e.Y,
			HpRatio: safeHpRatio(e.HP, e.MaxHP),
			MaxHp:   e.MaxHP,
			Active:  true,
			Index:   e.ID,
		}
		ctx.TargetEnemy = &ref
		ctx.HitX = e.X
		ctx.HitY = e.Y
	}
	return ctx
}

// buildTickTriggerContext 从 tower/TickContext 构建 TriggerContext。
// onTick 没有特定目标敌人，但需要 Enemies/Towers 查询接口。
func (a *descriptorAbilityBase) buildTickTriggerContext(
	t *tower.Tower, ctx *tower.TickContext,
) TriggerContext {
	tc := TriggerContext{
		Type:        TriggerOnTick,
		Strength:    t.EffectiveStrength(),
		TowerDamage: t.Damage,
		TowerX:      t.X,
		TowerY:      t.Y,
		TowerRange:  t.Range,
		DT:          ctx.DT,
	}
	if ctx.Enemies != nil {
		tc.Enemies = NewPoolEnemyQuerier(ctx.Enemies)
	}
	if ctx.Towers != nil {
		tc.Towers = NewPoolTowerQuerier(ctx.Towers, t)
	}
	return tc
}

// safeHpRatio 安全计算 HP 比例（防止除零）。
func safeHpRatio(hp, maxHp float64) float64 {
	if maxHp <= 0 {
		return 0
	}
	return hp / maxHp
}

// ── DescriptorAbilityHit（仅 Ability）────────────────────────

// DescriptorAbilityHit 仅实现 tower.Ability（无 onTick 管线时使用）。
// pipeline 的类型断言 `ability.(tower.Ticker)` 会返回 false，
// 跳过每帧 tick 调用。
type DescriptorAbilityHit struct {
	descriptorAbilityBase
}

// onHitCommon 是 OnHit 的共享实现。
// 先检测 splash 模式（合成 HitResult.Splash），再走通用 interpreter 路径。
func (a *descriptorAbilityBase) onHitCommon(
	t *tower.Tower, p *projectile.Projectile, e *enemy.Enemy,
) *tower.HitResult {
	// splash 模式：aoeRadius selector 在 onHit 上下文无 Enemies 池，
	// 直接合成 Splash 交给战斗管线处理
	if hr := TrySynthSplash(a.desc, t.EffectiveStrength()); hr != nil {
		return hr
	}
	ctx := a.buildHitTriggerContext(t, p, e)
	results := a.interp.ExecOnHit(ctx)
	if len(results) == 0 {
		return nil
	}
	return AdaptToHitResult(results)
}

// OnHit 在弹射物命中敌人时执行 onHit 管线。
func (a *DescriptorAbilityHit) OnHit(
	t *tower.Tower, p *projectile.Projectile, e *enemy.Enemy,
) *tower.HitResult {
	return a.onHitCommon(t, p, e)
}

// ── DescriptorAbilityFull（Ability + Ticker）─────────────────

// DescriptorAbilityFull 实现 tower.Ability + tower.Ticker（有 onTick 管线时使用）。
type DescriptorAbilityFull struct {
	descriptorAbilityBase
}

// OnHit 在弹射物命中敌人时执行 onHit 管线。
func (a *DescriptorAbilityFull) OnHit(
	t *tower.Tower, p *projectile.Projectile, e *enemy.Enemy,
) *tower.HitResult {
	return a.onHitCommon(t, p, e)
}

// OnTick 每帧调用，执行 onTick 管线。
// Elapsed 传入 DT（帧增量），CooldownCondition 内部自行累计。
//
// Buff/SelfBuff 效果直接应用到目标塔（不走 TickResult），
// 复用现有 ConfigAbility 的 0.3s 短 buff 模式。
// Silence/Weaken/Root 等敌人 debuff 也在此直接应用到目标敌人。
func (a *DescriptorAbilityFull) OnTick(
	t *tower.Tower, ctx *tower.TickContext,
) *tower.TickResult {
	trigCtx := a.buildTickTriggerContext(t, ctx)
	trigCtx.Elapsed = ctx.DT // CooldownCondition 内部累加 DT
	results := a.interp.ExecOnTick(trigCtx)
	if len(results) == 0 {
		return nil
	}

	// 直接应用 buff/debuff 效果，不走 TickResult 适配
	applyTickBuffEffects(t, ctx, results)

	return AdaptToTickResult(results)
}

// buffStatToID 将描述符的 stat 名映射到塔光环 buff ID。
var buffStatToID = map[string]string{
	"damage": buff.IDAuraDamageAmp,
	"speed":  buff.IDAuraPctSpeed,
	"range":  buff.IDAuraFlatRange,
	"crit":   buff.IDAuraCrit,
}

// applyTickBuffEffects 直接应用 tick 管线中的 Buff/SelfBuff/敌人 debuff 效果。
//
// SelfBuff: 以 0.3s 短 buff 形式施加到自身塔（同 ConfigAbility 光环模式）。
// Buff: 通过 TargetX/TargetY 匹配目标塔，施加 0.3s 短 buff。
// Silence/Weaken/Root: 通过 TargetEnemyIdx 匹配敌人，直接施加。
func applyTickBuffEffects(self *tower.Tower, ctx *tower.TickContext, results []EffectResult) {
	for i := range results {
		r := &results[i]
		switch r.Type {
		case EffTypeSelfBuff:
			// 自身 buff：0.3s 短时效，每帧由 tick 续期
			buffID, ok := buffStatToID[r.BuffStat]
			if !ok {
				continue
			}
			srcKey := "desc_" + self.InstanceKey
			self.Buffs.Add(buff.Buff{
				ID: buffID, Category: buff.CatAura,
				Source: srcKey, Value: r.BuffBonus,
				Duration: 0.3, Remaining: 0.3,
			})

		case EffTypeBuff:
			// 友方 buff：通过坐标匹配目标塔
			if ctx.Towers == nil {
				continue
			}
			buffID, ok := buffStatToID[r.BuffStat]
			if !ok {
				continue
			}
			srcKey := "desc_" + self.InstanceKey
			tx, ty := r.TargetX, r.TargetY
			ctx.Towers.Each(func(other *tower.Tower) {
				if other == self || other.Selling {
					return
				}
				// 坐标精确匹配（塔位置基于网格，不会有浮点误差）
				if other.X == tx && other.Y == ty {
					other.Buffs.Add(buff.Buff{
						ID: buffID, Category: buff.CatAura,
						Source: srcKey, Value: r.BuffBonus,
						Duration: 0.3, Remaining: 0.3,
					})
				}
			})

		case EffTypeSilence:
			// 沉默：设置敌人帧级标记
			if ctx.Enemies == nil {
				continue
			}
			idx := r.TargetEnemyIdx
			ctx.Enemies.EachActive(func(e *enemy.Enemy) {
				if e.ID == idx {
					e.Silenced = true
					e.AbilitySilenced = true
				}
			})

		case EffTypeWeaken:
			// 易伤：施加短时效 debuff
			if ctx.Enemies == nil {
				continue
			}
			idx := r.TargetEnemyIdx
			srcKey := "desc_" + self.InstanceKey
			ctx.Enemies.EachActive(func(e *enemy.Enemy) {
				if e.ID == idx {
					e.Buffs.Add(buff.Buff{
						ID:        buff.IDWeaken,
						Category:  buff.CatDebuff,
						Source:    srcKey,
						Value:     r.WeakenAmp,
						Duration:  0.2,
						Remaining: 0.2,
					})
				}
			})

		case EffTypeRoot:
			// 定身：施加 CC buff
			if ctx.Enemies == nil {
				continue
			}
			idx := r.TargetEnemyIdx
			srcKey := "desc_" + self.InstanceKey
			ctx.Enemies.EachActive(func(e *enemy.Enemy) {
				if e.ID == idx && !e.IsControlImmune && !e.HasControlImmunity() {
					dur := r.RootDur * (1 - e.Tenacity)
					if dur > 0 {
						e.Buffs.Add(buff.Buff{
							ID:        buff.IDRoot,
							Category:  buff.CatCC,
							Source:    srcKey,
							Duration:  dur,
							Remaining: dur,
						})
					}
				}
			})
		}
	}
}

// ── 工厂函数 ─────────────────────────────────────────────────

// NewDescriptorAbility 创建描述符驱动的能力实例。
// 根据描述符是否包含 onTick 管线返回不同的具体类型：
//   - 无 onTick → *DescriptorAbilityHit（不实现 Ticker）
//   - 有 onTick → *DescriptorAbilityFull（实现 Ticker）
func NewDescriptorAbility(desc *AbilityDescriptor) tower.Ability {
	interp := NewInterpreter(desc)
	base := descriptorAbilityBase{desc: desc, interp: interp}
	if interp.HasOnTick() {
		return &DescriptorAbilityFull{descriptorAbilityBase: base}
	}
	return &DescriptorAbilityHit{descriptorAbilityBase: base}
}

// ── 池适配器 ─────────────────────────────────────────────────

// poolEnemyQuerier 将 enemy.Pool 适配到 EnemyQuerier 接口。
type poolEnemyQuerier struct {
	pool *enemy.Pool
}

// NewPoolEnemyQuerier 创建 enemy.Pool 的 EnemyQuerier 适配器。
func NewPoolEnemyQuerier(pool *enemy.Pool) EnemyQuerier {
	return &poolEnemyQuerier{pool: pool}
}

// QueryRadius 查询圆形区域内所有活跃（非 dying/spawning）敌人。
func (q *poolEnemyQuerier) QueryRadius(cx, cy, radius float64) []EnemyRef {
	r2 := radius * radius
	var refs []EnemyRef
	q.pool.EachActive(func(e *enemy.Enemy) {
		if e.IsDying() || e.IsSpawning() {
			return
		}
		dx, dy := e.X-cx, e.Y-cy
		if dx*dx+dy*dy <= r2 {
			refs = append(refs, enemyToRef(e))
		}
	})
	return refs
}

// AllActive 返回所有活跃（非 dying/spawning）敌人引用。
func (q *poolEnemyQuerier) AllActive() []EnemyRef {
	var refs []EnemyRef
	q.pool.EachActive(func(e *enemy.Enemy) {
		if e.IsDying() || e.IsSpawning() {
			return
		}
		refs = append(refs, enemyToRef(e))
	})
	return refs
}

// enemyToRef 将 Enemy 指针转为轻量 EnemyRef 值类型。
func enemyToRef(e *enemy.Enemy) EnemyRef {
	return EnemyRef{
		Index:   e.ID,
		X:       e.X,
		Y:       e.Y,
		HpRatio: safeHpRatio(e.HP, e.MaxHP),
		MaxHp:   e.MaxHP,
		Active:  true,
	}
}

// poolTowerQuerier 将 tower.Pool 适配到 TowerQuerier 接口。
// self 指向调用方自身塔，QueryRadius 结果中排除它。
type poolTowerQuerier struct {
	pool *tower.Pool
	self *tower.Tower // 排除自身（距离为 0 的不算"友方"）
}

// NewPoolTowerQuerier 创建 tower.Pool 的 TowerQuerier 适配器。
// self 为调用方塔（查询时排除），可为 nil。
func NewPoolTowerQuerier(pool *tower.Pool, self *tower.Tower) TowerQuerier {
	return &poolTowerQuerier{pool: pool, self: self}
}

// QueryRadius 查询圆形区域内所有友方塔（排除自身）。
func (q *poolTowerQuerier) QueryRadius(cx, cy, radius float64) []TowerRef {
	r2 := radius * radius
	var refs []TowerRef
	q.pool.Each(func(t *tower.Tower) {
		// 排除自身
		if q.self != nil && t == q.self {
			return
		}
		// 排除正在出售的塔
		if t.Selling {
			return
		}
		dx, dy := t.X-cx, t.Y-cy
		if dx*dx+dy*dy <= r2 {
			refs = append(refs, TowerRef{
				Index:  -1, // 塔没有 pool 级 index，用 -1 占位
				X:      t.X,
				Y:      t.Y,
				Active: true,
			})
		}
	})
	return refs
}

