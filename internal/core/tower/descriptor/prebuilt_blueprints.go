// prebuilt_blueprints.go — 预制蓝图加载器。
//
// 从 config/towers/prebuilt-blueprints.json 加载预制蓝图列表，
// 并提供转换为 TowerDef 的便捷方法。
// 替代旧的 config.LoadClassicPresets() + loadClassicTowerDefs() 路径。
//
// 关联：
//   - blueprint.go — TowerBlueprint 数据结构
//   - blueprint_to_def.go — BlueprintToTowerDef 转换
//   - config/tier_presets.go — TierPresets 查表
package descriptor

import (
	"encoding/json"
	"log"

	"defense2/internal/config"
	"defense2/internal/core/tower"
)

// prebuiltBlueprintsFile JSON 文件结构。
type prebuiltBlueprintsFile struct {
	Blueprints []TowerBlueprint `json:"blueprints"`
}

// 全局缓存（首次加载后复用）。
var globalPrebuiltBlueprints []TowerBlueprint

// LoadPrebuiltBlueprints 从 JSON 加载预制蓝图列表。
// 必须在 config.SetDataFS() 之后调用。
func LoadPrebuiltBlueprints() []TowerBlueprint {
	if globalPrebuiltBlueprints != nil {
		return globalPrebuiltBlueprints
	}

	fs := config.GetDataFS()
	if fs == nil {
		log.Printf("[descriptor] WARNING: dataFS not initialized, no prebuilt blueprints")
		return nil
	}

	data, err := fs.ReadFile("config/towers/prebuilt-blueprints.json")
	if err != nil {
		log.Printf("[descriptor] WARNING: load prebuilt-blueprints.json: %v", err)
		return nil
	}

	var file prebuiltBlueprintsFile
	if err := json.Unmarshal(data, &file); err != nil {
		log.Printf("[descriptor] WARNING: parse prebuilt-blueprints.json: %v", err)
		return nil
	}

	globalPrebuiltBlueprints = file.Blueprints
	log.Printf("[descriptor] loaded %d prebuilt blueprints", len(globalPrebuiltBlueprints))
	return globalPrebuiltBlueprints
}

// LoadPrebuiltBlueprintDefs 加载预制蓝图并转换为 TowerDef 列表。
// 替代旧的 loadClassicTowerDefs()。
func LoadPrebuiltBlueprintDefs() []tower.TowerDef {
	blueprints := LoadPrebuiltBlueprints()
	if len(blueprints) == 0 {
		log.Printf("[descriptor] WARNING: no prebuilt blueprints, fallback skipped")
		return nil
	}

	tierPresets := config.GlobalTierPresets()
	if tierPresets == nil {
		log.Printf("[descriptor] WARNING: tierPresets is nil, cannot convert prebuilt blueprints")
		return nil
	}

	budgetRules := DefaultBudgetRules()
	abilityCosts := GlobalAbilityCosts()

	defs := make([]tower.TowerDef, 0, len(blueprints))
	for _, bp := range blueprints {
		def := BlueprintToTowerDef(&bp, tierPresets, budgetRules, abilityCosts)
		// 预制蓝图保留 category 用于建造菜单分组
		def.Category = bp.Category
		defs = append(defs, def)
	}

	log.Printf("[descriptor] converted %d prebuilt blueprints to TowerDefs", len(defs))
	return defs
}
