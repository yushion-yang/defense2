// derive_ability_table.go — 从描述符自动派生 AbilityDef 元数据表。
//
// 将描述符中的 label/icon/tags/display 映射为 config.AbilityDef，
// 使 config.GlobalAbilityTable() 能同时包含预制和自定义能力。
// 核心目的：让 AddAbility() 能通过 AbilityTable 识别所有能力（包括自定义）。
//
// 关联：
//   - descriptor.go — AbilityDescriptor 数据结构
//   - config/ability_config.go — AbilityDef / AbilityTable 类型
//   - init.go — 在 InitDescriptorAbilities 中调用
package descriptor

import (
	"encoding/json"

	"defense2/internal/config"
)

// jsonUnmarshal 是 json.Unmarshal 的包内别名，避免与 descriptor 包的自定义 UnmarshalJSON 冲突。
var jsonUnmarshal = json.Unmarshal

// DeriveAbilityTable 从全局描述符表生成 AbilityTable。
// 每个描述符的 tags[0] 映射为 category，直接使用描述符的 display 模板。
func DeriveAbilityTable() config.AbilityTable {
	table := config.AbilityTable{}
	for _, desc := range GlobalDescriptorTable() {
		table[desc.ID] = deriveAbilityDef(desc)
	}
	return table
}

// DeriveAbilityDefFromDescriptor 从单个描述符生成 AbilityDef（供自定义能力注册用）。
func DeriveAbilityDefFromDescriptor(desc *AbilityDescriptor) *config.AbilityDef {
	return deriveAbilityDef(desc)
}

// deriveAbilityDef 从描述符派生 AbilityDef 元数据。
// 核心字段（category/icon/label/display）直接从描述符复制。
// base/potential 从管线的主 effect scaler 提取。
func deriveAbilityDef(desc *AbilityDescriptor) *config.AbilityDef {
	def := &config.AbilityDef{
		Type:    desc.ID,
		Label:   desc.Label,
		Icon:    desc.Icon,
		Display: desc.Display,
	}

	// category 从 tags[0] 取
	if len(desc.Tags) > 0 {
		def.Category = desc.Tags[0]
	}

	// 从管线中提取 scaler 参数
	if len(desc.Pipelines) > 0 {
		extractPipelineParams(&desc.Pipelines[0], def)
	}

	// attack 类能力的 attackParams 中提取主 scaler
	if desc.AttackParams != nil {
		extractAttackParamsScaler(desc, def)
	}

	return def
}

// extractPipelineParams 从第一条管线提取 scaler 参数到 AbilityDef。
// 优先从 effect 提取主 scaler，然后从 condition/selector 提取参数。
func extractPipelineParams(p *Pipeline, def *config.AbilityDef) {
	// 从 condition 提取参数（chance/cooldown 等）
	for _, cond := range p.Conditions {
		extractConditionParams(cond, def)
	}

	// 从 selector 提取参数（aoe radius 等）
	extractSelectorParams(p.Selector, def)

	// 遍历所有 effects 提取参数。
	// 第一个 effect 填主槽（Param），后续 effect 填次要槽（Param2）。
	for _, eff := range p.Effects {
		extractEffectParams(eff, def)
	}
}

// extractEffectParams 从 effect 提取 base/potential 到 AbilityDef。
func extractEffectParams(eff Effect, def *config.AbilityDef) {
	switch e := eff.(type) {
	case DamageEffect:
		if def.ScaleDim == "" {
			// 主 scaler 槽空闲：填充为主 scaler
			extractScaler(e.Value, def, true)
			switch e.Mode {
			case DmgFlat:
				def.ScaleDim = "damage"
			case DmgRatio:
				def.ScaleDim = "ratio"
			case DmgHpPercent:
				def.ScaleDim = "hpPercent"
			}
		} else {
			// 主 scaler 已被 condition/selector 占据（如 crit 的 ChanceCondition），
			// 将 damage value 放入 Param 次要槽位，避免丢失。
			extractScaler(e.Value, def, false)
			switch e.Mode {
			case DmgFlat:
				def.ParamDim = "damage"
			case DmgRatio:
				def.ParamDim = "ratio"
			case DmgHpPercent:
				def.ParamDim = "hpPercent"
			}
		}
	case SlowEffect:
		// slowPower: factor 是主 scaler，duration 是参数
		// slowDuration: duration 是主 scaler，factor 是参数
		// 用 LinearScaler 判断哪个是主 scaler
		if isLinear(e.Factor) {
			extractScaler(e.Factor, def, true)
			def.ScaleDim = "factor"
			extractScaler(e.Duration, def, false)
			def.ParamDim = "duration"
		} else {
			extractScaler(e.Duration, def, true)
			def.ScaleDim = "duration"
			extractScaler(e.Factor, def, false)
			def.ParamDim = "factor"
		}
	case StunEffect:
		extractScaler(e.Duration, def, true)
		def.ScaleDim = "duration"
	case RootEffect:
		extractScaler(e.Duration, def, true)
		def.ScaleDim = "duration"
	case DotEffect:
		extractScaler(e.Value, def, true)
		switch e.Mode {
		case DmgFlat:
			def.ScaleDim = "dps"
		case DmgRatio:
			def.ScaleDim = "ratio"
		case DmgHpPercent:
			def.ScaleDim = "hpPercent"
		}
		extractScaler(e.Duration, def, false)
		def.ParamDim = "duration"
	case WeakenEffect:
		extractScaler(e.Amplify, def, true)
		def.ScaleDim = "amplify"
		extractScaler(e.Duration, def, false)
		def.ParamDim = "duration"
	case SilenceEffect:
		def.ScaleDim = "none"
	case BuffEffect:
		extractScaler(e.Bonus, def, true)
		def.ScaleDim = "bonus"
	case SelfBuffEffect:
		extractScaler(e.Bonus, def, true)
		def.ScaleDim = "bonus"
	case GoldEffect:
		extractScaler(e.Amount, def, true)
		def.ScaleDim = "amount"
	case CritEffect:
		// crit: chance 在 condition 中，multiplier 在 effect 中
		def.Param = e.Multiplier
		def.ParamDim = "multiplier"
	case ModifyStatEffect:
		// Multiplier 是总倍率（如 1.8 = ×1.8），存入时转为增量（0.8 = +80%），
		// 使 {p%}/{p2%} 模板（Param*100）显示正确的百分比。
		// range stat 填 Param2 槽位（对应 {p2%}），其他填 Param 槽位（对应 {p%}）。
		inc := e.Multiplier - 1
		if e.Stat == "range" {
			if def.Param2 == 0 {
				def.Param2 = inc
				def.Param2Dim = "statBoost"
			}
		} else {
			if def.Param == 0 {
				def.Param = inc
				def.ParamDim = "statBoost"
			}
		}
	case TeleportEffect:
		extractScaler(e.Distance, def, true)
		def.ScaleDim = "distance"
	}
}

// extractConditionParams 从 condition 提取参数。
func extractConditionParams(cond Condition, def *config.AbilityDef) {
	switch c := cond.(type) {
	case ChanceCondition:
		// chance 通常是主 scaler（如 crit、stunChance）
		// 但如果 effect 已设置了主 scaler，则放到 param
		if def.Base == 0 && def.Potential == 0 {
			extractScaler(c.Rate, def, true)
			def.ScaleDim = "chance"
		} else {
			extractScaler(c.Rate, def, false)
			def.ParamDim = "chance"
		}
	case *CooldownCondition:
		def.Param = c.Seconds
		def.ParamDim = "interval"
	case NoNearbyTowerCondition:
		def.Param = c.Radius
		def.ParamDim = "checkRadius"
	}
}

// extractSelectorParams 从 selector 提取参数。
func extractSelectorParams(sel Selector, def *config.AbilityDef) {
	switch s := sel.(type) {
	case NearbyAlliesSelector:
		// aura 类能力的 radius 放到 param
		if def.Param == 0 {
			def.Param = s.Radius
			def.ParamDim = "radius"
		}
	case AoeRadiusSelector:
		if def.Param == 0 {
			extractScaler(s.Radius, def, false)
			def.ParamDim = "radius"
		}
	case ChainSelector:
		// bounce 类：maxBounce 是主 scaler
		extractScaler(s.MaxBounce, def, true)
		def.ScaleDim = "maxBounces"
		def.Param = s.DecayRatio
		def.ParamDim = "damageDecay"
		def.Param2 = s.ChainRange
		def.Param2Dim = "bounceRange"
	}
}

// extractScaler 从 Scaler 提取数值到 AbilityDef。
// primary=true 写入 Base/Potential，primary=false 写入 Param。
func extractScaler(s Scaler, def *config.AbilityDef, primary bool) {
	if s == nil {
		return
	}
	switch sc := s.(type) {
	case FixedScaler:
		if primary {
			def.Base = sc.Value
		} else {
			def.Param = sc.Value
		}
	case LinearScaler:
		if primary {
			def.Base = sc.Base
			def.Potential = sc.Potential
		} else {
			def.Param = sc.Base
		}
	case DiminishingScaler:
		if primary {
			def.Base = sc.Base
			def.Potential = sc.Potential
		}
	case CappedScaler:
		if primary {
			def.Base = sc.Base
			def.Potential = sc.Potential
		}
	}
}

// isLinear 判断 scaler 是否是 LinearScaler。
func isLinear(s Scaler) bool {
	_, ok := s.(LinearScaler)
	return ok
}

// atkParamScaler attackParams 中单个参数的 JSON 结构。
type atkParamScaler struct {
	Scaler    string  `json:"scaler"`
	Value     float64 `json:"value"`
	Base      float64 `json:"base"`
	Potential float64 `json:"potential"`
}

// extractAttackParamsScaler 从攻击参数 JSON 中提取主 scaler。
// attack 类能力的管线通常为空，参数在 attackParams 中。
func extractAttackParamsScaler(desc *AbilityDescriptor, def *config.AbilityDef) {
	if desc.AttackParams == nil {
		return
	}

	var params map[string]atkParamScaler
	if err := jsonUnmarshal(desc.AttackParams, &params); err != nil {
		return
	}

	// 按 attackStyle 确定主参数名
	mainParam := attackStyleMainParam(desc.AttackStyle, params)
	if mainParam == "" {
		return
	}

	if s, ok := params[mainParam]; ok {
		switch s.Scaler {
		case "fixed":
			def.Base = s.Value
		case "linear":
			def.Base = s.Base
			def.Potential = s.Potential
		}
		def.ScaleDim = mainParam
	}

	// 提取次要参数
	for name, s := range params {
		if name == mainParam {
			continue
		}
		val := s.Value
		if s.Scaler == "linear" {
			val = s.Base
		}
		if def.Param == 0 {
			def.Param = val
			def.ParamDim = name
		} else if def.Param2 == 0 {
			def.Param2 = val
			def.Param2Dim = name
		}
	}
}

// attackStyleMainParam 返回攻击方式的主参数名。
// 当 style 无特定主参数时，返回 attackParams 中的第一个 key。
func attackStyleMainParam(style string, params map[string]atkParamScaler) string {
	switch style {
	case "scatter":
		return "pellets"
	case "wideBeam":
		return "beamWidth"
	case "spin_aoe":
		return "damageRatio"
	case "radial":
		return "shots"
	case "barrage":
		return "bullets"
	default:
		// multiTarget 等：取 attackParams 中唯一/第一个 key 作为主参数
		for k := range params {
			return k
		}
		return ""
	}
}
