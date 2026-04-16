// effect.go — 效果接口及 13 种具体效果实现。
//
// Effect 决定选中目标受到什么影响。每种 Effect 的 Apply 方法
// 读取 EffectCtx 中的运行时数据，返回扁平的 EffectResult。
// Phase 2 新增：CritEffect（独立暴击）、PurgeEffect（净化）。
//
// 类型定义和枚举常量在 effect_result.go 中，本文件只包含接口和实现。
package descriptor

// Effect 效果接口，所有效果类型必须实现。
type Effect interface {
	Apply(ctx EffectCtx) EffectResult
}

// ── 具体效果类型 ────────────────────────────────────────

// DamageEffect 直接伤害效果。
type DamageEffect struct {
	Mode  DamageMode
	Value Scaler
}

func (e DamageEffect) Apply(ctx EffectCtx) EffectResult {
	v := e.Value.Calc(ctx.Strength)
	var dmg float64
	switch e.Mode {
	case DmgFlat:
		dmg = v
	case DmgRatio:
		dmg = v * ctx.TowerDamage
	case DmgHpPercent:
		dmg = v * ctx.TargetMaxHp
	}
	return EffectResult{Type: EffTypeDamage, Damage: dmg, DamageMode: e.Mode}
}

// SlowEffect 减速效果。
type SlowEffect struct {
	Factor   Scaler
	Duration Scaler
}

func (e SlowEffect) Apply(ctx EffectCtx) EffectResult {
	// factor 语义：JSON 中的值表示"减速量"（0.25 = 减速 25%），
	// 需转换为速度倍率（1 - 0.25 = 0.75），与传统能力系统一致。
	return EffectResult{
		Type:       EffTypeSlow,
		SlowFactor: 1 - e.Factor.Calc(ctx.Strength),
		Duration:   e.Duration.Calc(ctx.Strength),
	}
}

// StunEffect 眩晕效果。
type StunEffect struct {
	Duration Scaler
}

func (e StunEffect) Apply(ctx EffectCtx) EffectResult {
	return EffectResult{Type: EffTypeStun, StunDur: e.Duration.Calc(ctx.Strength)}
}

// RootEffect 定身效果。
type RootEffect struct {
	Duration Scaler
}

func (e RootEffect) Apply(ctx EffectCtx) EffectResult {
	return EffectResult{Type: EffTypeRoot, RootDur: e.Duration.Calc(ctx.Strength)}
}

// DotEffect 持续伤害效果。
type DotEffect struct {
	Subtype  string
	Mode     DamageMode
	Value    Scaler
	Duration Scaler
}

func (e DotEffect) Apply(ctx EffectCtx) EffectResult {
	v := e.Value.Calc(ctx.Strength)
	var dotVal float64
	switch e.Mode {
	case DmgFlat:
		dotVal = v
	case DmgRatio:
		dotVal = v * ctx.TowerDamage
	case DmgHpPercent:
		dotVal = v * ctx.TargetMaxHp
	}
	return EffectResult{
		Type:        EffTypeDot,
		DotSubtype:  e.Subtype,
		DotMode:     e.Mode,
		DotValue:    dotVal,
		DotDuration: e.Duration.Calc(ctx.Strength),
	}
}

// WeakenEffect 削弱效果（增加目标受到的伤害）。
type WeakenEffect struct {
	Amplify  Scaler
	Duration Scaler
}

func (e WeakenEffect) Apply(ctx EffectCtx) EffectResult {
	return EffectResult{
		Type:      EffTypeWeaken,
		WeakenAmp: e.Amplify.Calc(ctx.Strength),
		WeakenDur: e.Duration.Calc(ctx.Strength),
	}
}

// SilenceEffect 沉默效果。
type SilenceEffect struct{}

func (e SilenceEffect) Apply(_ EffectCtx) EffectResult {
	return EffectResult{Type: EffTypeSilence}
}

// BuffEffect 友方增益效果。
type BuffEffect struct {
	Stat  string
	Bonus Scaler
}

func (e BuffEffect) Apply(ctx EffectCtx) EffectResult {
	return EffectResult{
		Type:      EffTypeBuff,
		BuffStat:  e.Stat,
		BuffBonus: e.Bonus.Calc(ctx.Strength),
	}
}

// SelfBuffEffect 自身增益效果。
type SelfBuffEffect struct {
	Stat  string
	Bonus Scaler
}

func (e SelfBuffEffect) Apply(ctx EffectCtx) EffectResult {
	return EffectResult{
		Type:      EffTypeSelfBuff,
		BuffStat:  e.Stat,
		BuffBonus: e.Bonus.Calc(ctx.Strength),
	}
}

// GoldEffect 金币奖励效果。
type GoldEffect struct {
	Amount Scaler
}

func (e GoldEffect) Apply(ctx EffectCtx) EffectResult {
	return EffectResult{
		Type:       EffTypeGold,
		GoldAmount: e.Amount.Calc(ctx.Strength),
	}
}

// ModifyStatEffect 属性修改效果。
type ModifyStatEffect struct {
	Stat       string
	Multiplier float64
}

func (e ModifyStatEffect) Apply(_ EffectCtx) EffectResult {
	return EffectResult{
		Type:     EffTypeModifyStat,
		BuffStat: e.Stat,
		StatMult: e.Multiplier,
	}
}

// CritEffect 独立暴击效果。
// 与 DamageEffect 的 IsCrit 标记不同，CritEffect 是一个独立效果类型，
// 可在条件管线中单独控制（如仅对 Boss 暴击）。
type CritEffect struct {
	Multiplier float64
}

func (e CritEffect) Apply(_ EffectCtx) EffectResult {
	return EffectResult{
		Type:     EffTypeCrit,
		IsCrit:   true,
		CritMult: e.Multiplier,
	}
}

// PurgeEffect 净化效果（移除敌人身上的 buff）。
// Count 指定最多移除多少个 buff，实际移除数量由战斗管线决定。
type PurgeEffect struct {
	Count int
}

func (e PurgeEffect) Apply(_ EffectCtx) EffectResult {
	return EffectResult{
		Type:       EffTypePurge,
		PurgeCount: e.Count,
	}
}

// TeleportEffect 传送/回推效果（将敌人沿路径回推指定距离）。
// 实际路径回推逻辑由战斗管线处理，这里只计算距离值。
type TeleportEffect struct {
	Distance Scaler
}

func (e TeleportEffect) Apply(ctx EffectCtx) EffectResult {
	return EffectResult{
		Type:         EffTypeTeleport,
		TeleportDist: e.Distance.Calc(ctx.Strength),
	}
}
