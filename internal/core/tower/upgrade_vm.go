// upgrade_vm.go — 升级系统 ViewModel。
// 为 HUD 提供纯值数据结构，解耦渲染层与核心逻辑。
package tower

import "defense2/internal/config"

// CategoryOption 类别选择面板的一个选项。
type CategoryOption struct {
	Index    int    // 类别索引 (0-5)
	Name     string // 中文名
	Icon     string // 图标名
	Used     bool   // 是否已被占用
	Current  string // 当前能力名（已占用时）
}

// AbilityOption 能力选择面板的一个选项。
type AbilityOption struct {
	Type    string  // 能力类型标识
	Label   string  // 显示名
	Icon    string  // 图标名
	Desc    string  // 描述文本
	Value   string  // 当前强度下的数值预览
}

// categoryIcons 类别图标映射。
var categoryIcons = [config.AbilityCatCount]string{
	"multishot",     // 攻击模式
	"stun",          // 控制效果
	"stat-damage",   // 命中加伤
	"tower-aura",    // 增益光环
	"burn",          // 持续伤害
	"tower-poison",  // 范围效果
}

// BuildCategoryOptions 构建类别选择面板数据。
func BuildCategoryOptions(t *Tower) []CategoryOption {
	opts := make([]CategoryOption, config.AbilityCatCount)
	for i := 0; i < config.AbilityCatCount; i++ {
		opts[i] = CategoryOption{
			Index:   i,
			Name:    CategoryName(i),
			Icon:    categoryIcons[i],
			Used:    t.AbilitySlots[i] != "",
			Current: t.AbilitySlots[i],
		}
	}
	return opts
}

// BuildAbilityOptions 构建指定类别的能力选择面板数据。
func BuildAbilityOptions(category int, t *Tower) []AbilityOption {
	defs := AbilitiesForCategory(category)
	str := 100.0
	if t.Strength != nil {
		str = t.Strength.Effective()
	}

	opts := make([]AbilityOption, 0, len(defs))
	for _, def := range defs {
		opts = append(opts, AbilityOption{
			Type:  def.Type,
			Label: def.Label,
			Icon:  def.Icon,
			Desc:  def.Display,
			Value: def.FormatScale(str),
		})
	}
	return opts
}

// UpgradeInfo 升级信息摘要（给 info_panel 展示用）。
type UpgradeInfo struct {
	Pending     int // 待选择的能力槽位数
	UsedSlots   int // 已使用的能力槽位数
	MaxSlots    int // 最大能力槽位数
	Slots       [config.AbilityCatCount]SlotInfo
}

// SlotInfo 单个能力槽位信息。
type SlotInfo struct {
	CategoryName string
	AbilityLabel string // 空=未选
	AbilityIcon  string
}

// BuildUpgradeInfo 构建升级信息摘要。
func BuildUpgradeInfo(t *Tower, wavesCleared int) UpgradeInfo {
	table := config.GlobalAbilityTable()
	used := 0
	for _, a := range t.AbilitySlots {
		if a != "" {
			used++
		}
	}
	info := UpgradeInfo{
		Pending:   t.PendingSlots(wavesCleared),
		UsedSlots: used,
		MaxSlots:  MaxAbilitySlots,
	}
	for i := 0; i < config.AbilityCatCount; i++ {
		info.Slots[i].CategoryName = CategoryName(i)
		aType := t.AbilitySlots[i]
		if aType != "" && table != nil {
			if def, ok := table[aType]; ok {
				info.Slots[i].AbilityLabel = def.Label
				info.Slots[i].AbilityIcon = def.Icon
			}
		}
	}
	return info
}
