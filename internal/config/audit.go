// audit.go — 配置审计工具。
// 静态检查配置之间的引用完整性，用于后台预览和调试面板。
package config

// AbilityAudit 能力使用情况审计结果。
type AbilityAudit struct {
	Total       int      // 能力表总数
	Equipped    []string // 被塔装备的能力
	NotEquipped []string // 未被任何塔装备的能力（储备池）
	Orphaned    []string // 塔引用了但能力表中不存在的（配置错误）
}

// AuditAbilities 检查能力表与塔配置的引用关系。
func AuditAbilities() (*AbilityAudit, error) {
	// 加载能力表
	table, err := LoadAbilityTable()
	if err != nil {
		return nil, err
	}

	// 加载塔配置
	towers, err := LoadAllTowers()
	if err != nil {
		return nil, err
	}

	// 收集塔装备的能力
	equippedSet := make(map[string]bool)
	for _, t := range towers {
		for _, ab := range t.Abilities {
			equippedSet[ab] = true
		}
	}

	result := &AbilityAudit{
		Total: len(table),
	}

	// 分类
	for key := range table {
		if equippedSet[key] {
			result.Equipped = append(result.Equipped, key)
		} else {
			result.NotEquipped = append(result.NotEquipped, key)
		}
	}

	// 检查孤儿引用
	for ab := range equippedSet {
		if _, ok := table[ab]; !ok {
			result.Orphaned = append(result.Orphaned, ab)
		}
	}

	return result, nil
}
