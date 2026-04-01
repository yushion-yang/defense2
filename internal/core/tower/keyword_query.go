// keyword_query.go — 能力关键字查询。
// 提供对塔能力列表的查询、合并和去重操作。
package tower

// AbilityEntry 能力条目。
type AbilityEntry struct {
	Type   string                 // 能力类型标识
	Config map[string]interface{} // 能力配置
}

// legacyFieldMapping 旧字段名 → 能力类型名的映射。
var legacyFieldMapping = map[string]string{
	"bounceConfig": "bounce",
	"onHitSlow":    "onHitSlow",
	"bleedDot":     "bleedDot",
	"burn":         "burn",
	"stun":         "stun",
	"splash":       "splash",
	"crit":         "crit",
}

// GetTowerAbilities 合并能力数组与旧字段，返回去重后的能力条目列表。
// abilities 为显式声明的能力名数组，legacyFields 为旧配置中的隐式能力字段。
func GetTowerAbilities(abilities []string, legacyFields map[string]interface{}) []AbilityEntry {
	seen := make(map[string]bool, len(abilities))
	var entries []AbilityEntry

	// 先处理显式能力数组
	for _, name := range abilities {
		if seen[name] {
			continue
		}
		seen[name] = true
		entries = append(entries, AbilityEntry{Type: name})
	}

	// 合并旧字段中的隐式能力
	for field, abilType := range legacyFieldMapping {
		if seen[abilType] {
			continue
		}
		val, ok := legacyFields[field]
		if !ok {
			continue
		}

		// 检查字段值是否有效（非 nil、非 false）
		if !isLegacyFieldActive(val) {
			continue
		}

		seen[abilType] = true
		entry := AbilityEntry{Type: abilType}

		// 如果旧字段值是 map 类型，作为配置传入
		if cfgMap, ok := val.(map[string]interface{}); ok {
			entry.Config = cfgMap
		}

		entries = append(entries, entry)
	}

	return entries
}

// TowerHasAbility 检查塔是否拥有指定能力。
func TowerHasAbility(abilities []string, abilityType string) bool {
	for _, a := range abilities {
		if a == abilityType {
			return true
		}
	}
	return false
}

// GetTowerAbility 查找并返回指定类型的能力条目，不存在返回 nil。
func GetTowerAbility(abilities []string, abilityType string) *AbilityEntry {
	for _, a := range abilities {
		if a == abilityType {
			return &AbilityEntry{Type: a}
		}
	}
	return nil
}

// isLegacyFieldActive 判断旧字段值是否表示能力激活。
func isLegacyFieldActive(val interface{}) bool {
	if val == nil {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case float64:
		return v != 0
	case int:
		return v != 0
	case string:
		return v != ""
	default:
		return true
	}
}
