// i18n_resolve.go — 配置加载后统一覆盖显示文本。
// 在 loading.go 中 i18n.Init() 和所有 config 加载完成后调用 ResolveConfigLabels()，
// 将所有配置 struct 的 Label/Name/Description 等字段通过 i18n.ResolveKey 覆盖。
// 这确保运行时所有显示文本统一走 i18n 管线，不遗漏配置 JSON 中的硬编码文本。
package config

import (
	"defense2/internal/i18n"
)

// ResolveConfigLabels 遍历所有已加载的配置，用 i18n 覆盖显示文本字段。
// 必须在 i18n.Init() 和所有 Load*() 调用之后执行。
func ResolveConfigLabels() {
	resolveAbilityLabels()
	resolveEnemyLabels()
	resolveWardenLabels()
	resolveItemLabels()
	resolveTowerLabels()
}

func resolveAbilityLabels() {
	table := GlobalAbilityTable()
	if table == nil {
		return
	}
	for key, def := range table {
		def.Label = i18n.ResolveKey("ability."+key+".label", def.Label)
		def.Display = i18n.ResolveKey("ability."+key+".display", def.Display)
	}
}

func resolveEnemyLabels() {
	table := GlobalEnemyArchetypes()
	if table == nil {
		return
	}
	for _, arch := range table {
		arch.Label = i18n.ResolveKey("enemy."+arch.ID+".label", arch.Label)
	}
}

func resolveWardenLabels() {
	if globalWardenConfigs == nil {
		return
	}
	for k, w := range globalWardenConfigs {
		w.Name = i18n.ResolveKey("warden."+k+".name", w.Name)
		w.Description = i18n.ResolveKey("warden."+k+".description", w.Description)
		w.AttackName = i18n.ResolveKey("warden."+k+".attack_name", w.AttackName)
		w.AttackDesc = i18n.ResolveKey("warden."+k+".attack_desc", w.AttackDesc)
		w.SpecialName = i18n.ResolveKey("warden."+k+".special_name", w.SpecialName)
		w.SpecialDesc = i18n.ResolveKey("warden."+k+".special_desc", w.SpecialDesc)
		w.StrengthDesc = i18n.ResolveKey("warden."+k+".strength_desc", w.StrengthDesc)
		w.CounterTip = i18n.ResolveKey("warden."+k+".counter_tip", w.CounterTip)
		globalWardenConfigs[k] = w // 值类型需要回写
	}
}

func resolveItemLabels() {
	bal := GlobalBalance()
	if bal == nil {
		return
	}
	for i := range bal.Items {
		item := &bal.Items[i]
		item.Label = i18n.ResolveKey("item."+item.Kind+".label", item.Label)
		item.Description = i18n.ResolveKey("item."+item.Kind+".desc", item.Description)
	}
}

func resolveTowerLabels() {
	table := GlobalTowerTable()
	if table == nil {
		return
	}
	for key, t := range table {
		t.Label = i18n.ResolveKey("tower."+key+".label", t.Label)
		t.ShortLabel = i18n.ResolveKey("tower."+key+".short", t.ShortLabel)
		t.Description = i18n.ResolveKey("tower."+key+".desc", t.Description)
	}
}
