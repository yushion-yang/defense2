// dimension_meta.go — 能力维度缩放元数据。
// 定义各维度（伤害/射程/冷却等）的缩放方式，用于能力升级时按规则计算数值。
package tower

// ScaleType 能力维度缩放类型。
type ScaleType int

const (
	ScaleMultiply     ScaleType = iota // 乘法（值越大越好）
	ScaleAddCapped                     // 加法带上限
	ScaleInverse                       // 反比（值越小越好，如冷却）
	ScaleInverseRatio                  // 反比带下限（如减速因子）
	ScaleNone                          // 不缩放
)

// DimensionMeta 单个维度的缩放元数据。
type DimensionMeta struct {
	Scale ScaleType // 缩放类型
	Cap   float64   // 上限（ScaleAddCapped 使用）
	Floor float64   // 下限（ScaleInverse/ScaleInverseRatio 使用）
}

// DimensionRegistry 预定义所有维度的缩放规则。
var DimensionRegistry = map[string]DimensionMeta{
	// 乘法类维度
	"radius":       {Scale: ScaleMultiply},
	"damage":       {Scale: ScaleMultiply},
	"duration":     {Scale: ScaleMultiply},
	"bounces":      {Scale: ScaleMultiply},
	"gold":         {Scale: ScaleMultiply},
	"baseDamage":   {Scale: ScaleMultiply},
	"distance":     {Scale: ScaleMultiply},
	"hpThreshold":  {Scale: ScaleMultiply},
	"armorIgnore":  {Scale: ScaleMultiply},

	// 加法带上限
	"chance": {Scale: ScaleAddCapped, Cap: 0.6},

	// 反比类维度（值越小越好）
	"cooldown": {Scale: ScaleInverse, Floor: 0.5},
	"interval": {Scale: ScaleInverse, Floor: 2},

	// 反比带下限（减速因子越小越强，但不能低于下限）
	"factor":      {Scale: ScaleInverseRatio, Floor: 0.30},
	"damageDecay": {Scale: ScaleInverseRatio, Floor: 0.10},

	// 不缩放
	"type":  {Scale: ScaleNone},
	"range": {Scale: ScaleNone},
}

// ScaleValue 按维度规则缩放单个数值。
// key 为维度名，base 为基础值，multiplier 为缩放乘数。
func ScaleValue(key string, base, multiplier float64) float64 {
	meta, ok := DimensionRegistry[key]
	if !ok {
		// 未注册维度默认乘法
		return base * multiplier
	}

	switch meta.Scale {
	case ScaleMultiply:
		return base * multiplier

	case ScaleAddCapped:
		result := base + (multiplier - 1)
		if meta.Cap > 0 && result > meta.Cap {
			result = meta.Cap
		}
		return result

	case ScaleInverse:
		// 反比：乘数越大冷却越短
		if multiplier <= 0 {
			return base
		}
		result := base / multiplier
		if meta.Floor > 0 && result < meta.Floor {
			result = meta.Floor
		}
		return result

	case ScaleInverseRatio:
		// 反比带下限：减速因子随乘数递减
		if multiplier <= 0 {
			return base
		}
		result := base / multiplier
		if meta.Floor > 0 && result < meta.Floor {
			result = meta.Floor
		}
		return result

	case ScaleNone:
		return base

	default:
		return base * multiplier
	}
}

// ScaleAbility 对整个能力配置 map 按乘数缩放，返回新 map（不修改原 map）。
func ScaleAbility(ab map[string]float64, multiplier float64) map[string]float64 {
	result := make(map[string]float64, len(ab))
	for key, base := range ab {
		result[key] = ScaleValue(key, base, multiplier)
	}
	return result
}
