// effect_result.go — 统一效果输出结构与上下文。
//
// EffectResult 是 descriptor 引擎的核心输出类型，
// 对标现有 tower.HitResult + tower.TickResult 的超集。
// 每个 Effect.Apply() 返回一个 EffectResult，
// 后续适配层负责将其转换回 HitResult/TickResult 供战斗管线消费。
//
// EffectCtx 携带 Apply 所需的运行时上下文（塔强度、塔伤害、目标血量），
// 由调用方在命中/tick 时构造并传入。
package descriptor

// EffectType 效果类型常量。
type EffectType int

const (
	EffTypeDamage     EffectType = iota // 直接伤害
	EffTypeSlow                         // 减速
	EffTypeStun                         // 眩晕
	EffTypeRoot                         // 定身
	EffTypeDot                          // 持续伤害（burn/bleed/poison）
	EffTypeWeaken                       // 易伤（增加受到的伤害）
	EffTypeSilence                      // 沉默（禁用敌人能力）
	EffTypeBuff                         // 增益友方塔
	EffTypeSelfBuff                     // 自我增益
	EffTypeGold                         // 获得金币
	EffTypeModifyStat                   // 修改属性倍率
	EffTypeCrit                         // 独立暴击效果
	EffTypePurge                        // 净化（移除敌人 buff）
)

// DamageMode 伤害模式。
type DamageMode int

const (
	DmgFlat      DamageMode = iota // 固定伤害
	DmgRatio                       // 塔伤害百分比
	DmgHpPercent                   // 目标最大HP百分比
)

// EffectResult 统一效果输出。
// 采用扁平结构而非多态嵌套：所有字段都是可选的，
// 由 EffectType 决定哪些字段有效。这与 HitResult 的"效果菜单"模式一脉相承，
// 方便战斗管线按字段遍历、零分配。
type EffectResult struct {
	Type       EffectType
	Damage     float64
	DamageMode DamageMode
	IsCrit     bool
	CritMult   float64

	// ── CC ──
	SlowFactor float64
	StunDur    float64
	RootDur    float64
	Duration   float64 // 通用持续时间（Slow/Weaken 等共用）

	// ── DoT ──
	DotSubtype  string     // burn/bleed/poison
	DotValue    float64    // 每秒伤害值（已计算模式）
	DotDuration float64    // DoT 持续时间
	DotMode     DamageMode // DoT 的伤害模式（记录用）

	// ── Weaken ──
	WeakenAmp float64
	WeakenDur float64

	// ── Buff ──
	BuffStat  string  // damage/speed/range/crit
	BuffBonus float64 // 增益数值

	// ── Gold ──
	GoldAmount float64

	// ── ModifyStat ──
	StatMult float64 // 属性倍率

	// ── Purge ──
	PurgeCount int // 净化移除的 buff 数量
}

// EffectCtx 效果执行上下文。
// 由调用方在命中/tick 时构造，提供 Effect.Apply 所需的运行时数据。
type EffectCtx struct {
	Strength    float64 // 塔的当前强度值
	TowerDamage float64 // 塔的当前伤害值
	TargetMaxHp float64 // 目标的最大生命值
}
