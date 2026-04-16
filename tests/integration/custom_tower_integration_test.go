// custom_tower_integration_test.go — 自定义蓝图塔的集成测试。
//
// 验证自定义能力+蓝图在运行时的完整生命周期：
//   - 能力通过 AbilityStore 持久化后可注册到 tower.Registry
//   - 蓝图通过 BlueprintStore 持久化后可转换为 TowerDef
//   - TowerDef 的 Key/AttackStyle/PresetAbilities 与蓝图定义一致
//
// 测试使用 MemoryStorage 隔离，不依赖文件系统。
package integration_test

import (
	"encoding/json"
	"strings"
	"testing"

	defense2 "defense2"
	"defense2/internal/config"
	"defense2/internal/core/persistence"
	"defense2/internal/core/tower"
	"defense2/internal/core/tower/descriptor"
)

// ensureConfigLoaded 确保全局配置已加载（tier presets 等）。
// 使用 sync.Once 语义（LoadTierPresets 内部有幂等保护）。
func ensureConfigLoaded() {
	config.SetDataFS(&defense2.DataFS)
	config.LoadBalance()
	config.LoadTierPresets()
	config.LoadBuffRules()
}

// TestCustomAbilityRegistration 验证自定义能力可通过 AbilityStore → RegisterCustomAbilities
// 注册到 tower.Registry，并且通过 Lookup 可查找到。
func TestCustomAbilityRegistration(t *testing.T) {
	ensureConfigLoaded()

	store := persistence.NewMemoryStorage()
	abStore := descriptor.NewAbilityStore(store)

	// 构造一个简单的自定义能力（onHit + currentTarget + damage）
	descJSON := `{
		"id": "ca_inttest_001",
		"label": "集成测试能力",
		"cost": 4,
		"tags": ["damage"],
		"pipelines": [{
			"trigger": "onHit",
			"conditions": [],
			"selector": {"type": "currentTarget"},
			"effects": [{
				"type": "damage",
				"mode": "ratio",
				"value": {"scaler": "fixed", "value": 0.5}
			}]
		}]
	}`

	var desc descriptor.AbilityDescriptor
	if err := json.Unmarshal([]byte(descJSON), &desc); err != nil {
		t.Fatalf("解析测试能力描述符失败: %v", err)
	}

	ca := descriptor.CustomAbility{
		ID:   "ca_inttest_001",
		Name: "集成测试能力",
		Desc: desc,
	}
	if err := abStore.Save(ca); err != nil {
		t.Fatalf("保存自定义能力失败: %v", err)
	}

	// 验证 store 中有数据
	if abStore.Count() != 1 {
		t.Fatalf("期望 1 个能力，实际 %d", abStore.Count())
	}

	// 注册到 tower.Registry
	n := descriptor.RegisterCustomAbilities(abStore)
	if n != 1 {
		t.Fatalf("期望注册 1 个能力，实际 %d", n)
	}

	// 通过 Lookup 验证
	ab, ok := tower.Lookup("ca_inttest_001")
	if !ok {
		t.Fatal("自定义能力 ca_inttest_001 未在 tower.Registry 中找到")
	}
	if ab.Name() != "ca_inttest_001" {
		t.Fatalf("能力名称不匹配: 期望 ca_inttest_001，实际 %s", ab.Name())
	}

	// 清理：从 Registry 删除测试数据，避免污染其他测试
	delete(tower.Registry, "ca_inttest_001")
}

// TestBlueprintToTowerDef 验证蓝图可通过 BlueprintStore 持久化后
// 正确转换为 TowerDef，且关键字段与蓝图定义一致。
func TestBlueprintToTowerDef(t *testing.T) {
	ensureConfigLoaded()

	tierPresets := config.GlobalTierPresets()
	if tierPresets == nil {
		t.Fatal("tier presets 未加载")
	}

	store := persistence.NewMemoryStorage()
	bpStore := descriptor.NewBlueprintStore(store)

	bp := descriptor.TowerBlueprint{
		ID:          "bp_inttest_001",
		Name:        "集成测试塔",
		AttackStyle: "projectile",
		Tiers: map[string]string{
			"damage":   "B",
			"atkSpeed": "B",
			"range":    "B",
		},
		Specialty: "damage",
		Abilities: []string{"enhance"},
		Strength: descriptor.StrengthConfig{
			Cost:         50,
			Amount:       50.0,
			MaxPurchases: -1,
		},
	}

	if err := bpStore.Save(bp); err != nil {
		t.Fatalf("保存蓝图失败: %v", err)
	}

	// 重新从 store 加载（模拟 stage 初始化流程）
	bpStore2 := descriptor.NewBlueprintStore(store)
	if bpStore2.Count() != 1 {
		t.Fatalf("期望 1 个蓝图，实际 %d", bpStore2.Count())
	}

	bpLoaded := bpStore2.List()[0]
	budgetRules := descriptor.DefaultBudgetRules()
	abilityCosts := descriptor.GlobalAbilityCosts()

	def := descriptor.BlueprintToTowerDef(&bpLoaded, tierPresets, budgetRules, abilityCosts)

	// 验证关键字段
	if def.Key != "bp_inttest_001" {
		t.Fatalf("Key 不匹配: 期望 bp_inttest_001，实际 %s", def.Key)
	}
	if def.Label != "集成测试塔" {
		t.Fatalf("Label 不匹配: 期望 集成测试塔，实际 %s", def.Label)
	}
	if string(def.AttackStyleID) != "projectile" {
		t.Fatalf("AttackStyle 不匹配: 期望 projectile，实际 %s", def.AttackStyleID)
	}
	if !def.FixedTiers {
		t.Fatal("FixedTiers 应为 true")
	}
	if def.AbilityAcquireMode != "preset" {
		t.Fatalf("AbilityAcquireMode 应为 preset，实际 %s", def.AbilityAcquireMode)
	}
	if len(def.PresetAbilities) != 1 || def.PresetAbilities[0] != "enhance" {
		t.Fatalf("PresetAbilities 不匹配: 期望 [enhance]，实际 %v", def.PresetAbilities)
	}
	if def.Cost <= 0 {
		t.Fatalf("建造费用应 > 0，实际 %d", def.Cost)
	}

	// 验证属性值从 tier-presets 正确加载
	bTier := tierPresets.Damage.Tiers["B"]
	// 专精为 damage，Potential 应追加 BasePotential
	expectedPotential := bTier.Potential + tierPresets.Damage.BasePotential
	if def.PotentialDamage != expectedPotential {
		t.Fatalf("PotentialDamage 不匹配: 期望 %.1f (B tier %.1f + basePotential %.1f)，实际 %.1f",
			expectedPotential, bTier.Potential, tierPresets.Damage.BasePotential, def.PotentialDamage)
	}
}

// TestCustomAbilityRoundTrip 验证自定义能力通过 MemoryStorage
// 序列化→反序列化后 Pipeline 仍完好可用。
func TestCustomAbilityRoundTrip(t *testing.T) {
	ensureConfigLoaded()

	store := persistence.NewMemoryStorage()
	abStore := descriptor.NewAbilityStore(store)

	// 构造包含多种效果的复杂能力
	descJSON := `{
		"id": "ca_inttest_roundtrip",
		"label": "序列化测试",
		"cost": 8,
		"tags": ["cc", "damage"],
		"pipelines": [
			{
				"trigger": "onHit",
				"conditions": [],
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "damage", "mode": "ratio", "value": {"scaler": "linear", "base": 1.0, "potential": 0.5}},
					{"type": "stun", "duration": {"scaler": "fixed", "value": 0.5}}
				]
			},
			{
				"trigger": "onTick",
				"conditions": [],
				"selector": {"type": "allInRange"},
				"effects": [
					{"type": "slow", "factor": {"scaler": "fixed", "value": 0.5}, "duration": {"scaler": "fixed", "value": 0.3}}
				]
			}
		]
	}`

	var desc descriptor.AbilityDescriptor
	if err := json.Unmarshal([]byte(descJSON), &desc); err != nil {
		t.Fatalf("解析描述符失败: %v", err)
	}

	ca := descriptor.CustomAbility{
		ID:   "ca_inttest_roundtrip",
		Name: "序列化测试",
		Desc: desc,
	}
	if err := abStore.Save(ca); err != nil {
		t.Fatalf("保存失败: %v", err)
	}

	// 从同一 storage 重新创建 store（模拟重启）
	abStore2 := descriptor.NewAbilityStore(store)
	loaded, err := abStore2.Get("ca_inttest_roundtrip")
	if err != nil {
		t.Fatalf("加载失败: %v", err)
	}

	// 验证 pipeline 数量和结构
	if len(loaded.Desc.Pipelines) != 2 {
		t.Fatalf("期望 2 条 pipeline，实际 %d", len(loaded.Desc.Pipelines))
	}

	// 验证注册后可用
	n := descriptor.RegisterCustomAbilities(abStore2)
	if n != 1 {
		t.Fatalf("期望注册 1 个能力，实际 %d", n)
	}

	ab, ok := tower.Lookup("ca_inttest_roundtrip")
	if !ok {
		t.Fatal("round-trip 后能力未在 Registry 中找到")
	}
	if ab.Name() != "ca_inttest_roundtrip" {
		t.Fatalf("名称不匹配: %s", ab.Name())
	}

	// 清理
	delete(tower.Registry, "ca_inttest_roundtrip")
}

// TestBlueprintPrefixConvention 验证蓝图 ID 使用 bp_ 前缀。
func TestBlueprintPrefixConvention(t *testing.T) {
	bp := descriptor.TowerBlueprint{
		ID:   "bp_test_prefix",
		Name: "前缀测试",
	}
	if !strings.HasPrefix(bp.ID, "bp_") {
		t.Fatalf("蓝图 ID 应以 bp_ 开头: %s", bp.ID)
	}
}
