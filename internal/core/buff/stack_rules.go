// stack_rules.go — Buff 堆叠规则系统。
// 定义 6 种堆叠模式和 19 种 buff 类型的默认规则。
package buff

import "math"

// StackMode buff 堆叠模式。
type StackMode int

const (
	// ModeStrongest 最强值生效（同类只取绝对值最大的）
	ModeStrongest StackMode = iota
	// ModeAdditive 叠加（同类 Value 累加，可设上限）
	ModeAdditive
	// ModeMultiplicative 乘法叠加（同类 Value 相乘，可设下限）
	ModeMultiplicative
	// ModeOverride 覆盖（新 buff 替换旧同类）
	ModeOverride
	// ModeIndependent 独立（每个 buff 独立存在，如护盾）
	ModeIndependent
	// ModeIndependentPerSource 每来源独立（同来源刷新，不同来源独立，如 DOT）
	ModeIndependentPerSource
)

// StackRule 某种 buff 类型的堆叠规则。
type StackRule struct {
	Mode     StackMode // 堆叠模式
	Cap      float64   // 上限（0=无上限，用于 Additive/Strongest）
	Floor    float64   // 下限（0=无下限，用于 Multiplicative）
	Priority float64   // 优先级（Override 模式下高优先覆盖低优先）
}

// DefaultStackRules 19 种 buff 类型的默认堆叠规则。
var DefaultStackRules = map[string]StackRule{
	// 控制类
	"slow":    {Mode: ModeStrongest, Cap: 0.8}, // 最多减速80%（速度不低于20%）
	"stun":    {Mode: ModeOverride},
	"knockup": {Mode: ModeOverride},
	"silence": {Mode: ModeOverride},
	"disarm":  {Mode: ModeOverride},

	// 增益类
	"speedUp":    {Mode: ModeAdditive, Cap: 1.4},
	"damageUp":   {Mode: ModeMultiplicative},
	"damageDown": {Mode: ModeMultiplicative, Floor: 0.2},
	"fireRateUp": {Mode: ModeAdditive, Cap: 0.5},

	// 免疫类（高优先 Override，不可叠加）
	"invincible":    {Mode: ModeOverride, Priority: 99},
	"damageImmune":  {Mode: ModeOverride, Priority: 90},
	"controlImmune": {Mode: ModeOverride, Priority: 80},
	"slowImmune":    {Mode: ModeOverride, Priority: 70},
	"stunImmune":    {Mode: ModeOverride, Priority: 70},
	"untargetable":  {Mode: ModeOverride, Priority: 100},

	// 独立类
	"shield":   {Mode: ModeIndependent},
	"dot":      {Mode: ModeIndependentPerSource},
	"tenacity": {Mode: ModeMultiplicative},
}

// GetRule 获取指定 buff 类型的堆叠规则，未找到则返回默认 Override。
func GetRule(typ string, rules map[string]StackRule) StackRule {
	if r, ok := rules[typ]; ok {
		return r
	}
	return StackRule{Mode: ModeOverride}
}

// ResolveStack 根据堆叠规则解算一组同类型 buff 的最终生效值。
// 返回生效值（Strongest/Additive/Multiplicative）或 0 表示无该类型 buff。
// Independent/IndependentPerSource 模式下返回 Value 之和（简单聚合）。
func ResolveStack(typ string, buffs []Buff, rules map[string]StackRule) float64 {
	rule := GetRule(typ, rules)

	// 收集同类型的活跃 buff
	var active []Buff
	for _, b := range buffs {
		if b.Active && b.Type == typ {
			active = append(active, b)
		}
	}
	if len(active) == 0 {
		return 0
	}

	switch rule.Mode {
	case ModeStrongest:
		best := 0.0
		for _, b := range active {
			if math.Abs(b.Value) > math.Abs(best) {
				best = b.Value
			}
		}
		if rule.Cap > 0 && math.Abs(best) > rule.Cap {
			if best > 0 {
				best = rule.Cap
			} else {
				best = -rule.Cap
			}
		}
		return best

	case ModeAdditive:
		sum := 0.0
		for _, b := range active {
			sum += b.Value
		}
		if rule.Cap > 0 && sum > rule.Cap {
			sum = rule.Cap
		}
		return sum

	case ModeMultiplicative:
		product := 1.0
		for _, b := range active {
			product *= b.Value
		}
		if rule.Floor > 0 && product < rule.Floor {
			product = rule.Floor
		}
		return product

	case ModeOverride:
		// 取最后添加的（最新的）
		return active[len(active)-1].Value

	case ModeIndependent, ModeIndependentPerSource:
		sum := 0.0
		for _, b := range active {
			sum += b.Value
		}
		return sum
	}

	return 0
}
