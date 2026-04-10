// cross_system_contracts_test.go — 跨配置系统一致性契约测试。
// 验证多个 config JSON 之间的引用关系和数据一致性。
package contracts_test

import (
	"math"
	"testing"

	"defense2/internal/config"
)

// TestEveryTowerAbilityExistsInConfig 验证 towers.json 中每个塔的 abilities 引用都存在于 abilities.json。
func TestEveryTowerAbilityExistsInConfig(t *testing.T) {
	towers, err := config.LoadAllTowers()
	if err != nil {
		t.Fatalf("加载塔配置失败: %v", err)
	}
	table := config.GlobalAbilityTable()
	if table == nil || len(table) == 0 {
		t.Fatal("能力表为空")
	}
	for key, tw := range towers {
		for _, abilName := range tw.Abilities {
			if _, ok := table[abilName]; !ok {
				t.Errorf("塔 %q 引用能力 %q 不存在于 abilities.json", key, abilName)
			}
		}
	}
}

// TestBalanceAndEconomyKillRewardConsistent 验证 balance.json 和 economy.json 的击杀奖励一致。
func TestBalanceAndEconomyKillRewardConsistent(t *testing.T) {
	bal := config.GlobalBalance()
	spec := config.GlobalEconomySpec()
	if spec == nil || spec.Global.KillReward == 0 {
		t.Skip("economy spec 未加载或 killReward=0")
	}
	if int(bal.Economy.KillReward) != spec.Global.KillReward {
		t.Errorf("balance KillReward=%.0f != economy spec KillReward=%d",
			bal.Economy.KillReward, spec.Global.KillReward)
	}
}

// TestBalanceAndEconomySellRatioConsistent 验证 balance.json 和 economy.json 的卖回比例一致。
func TestBalanceAndEconomySellRatioConsistent(t *testing.T) {
	bal := config.GlobalBalance()
	spec := config.GlobalEconomySpec()
	if spec == nil || spec.Global.SellRefundRatio == 0 {
		t.Skip("economy spec 未加载或 sellRefundRatio=0")
	}
	if math.Abs(bal.Economy.SellRefundRatio-spec.Global.SellRefundRatio) > 1e-9 {
		t.Errorf("balance SellRefundRatio=%.2f != economy spec SellRefundRatio=%.2f",
			bal.Economy.SellRefundRatio, spec.Global.SellRefundRatio)
	}
}

// TestAllConfigsLoadWithoutError 验证所有运行时 config 加载函数均无错误。
func TestAllConfigsLoadWithoutError(t *testing.T) {
	loaders := []struct {
		name string
		fn   func() error
	}{
		{"LoadBalance", func() error { _, err := config.LoadBalance(); return err }},
		{"LoadAllTowers", func() error { _, err := config.LoadAllTowers(); return err }},
		{"LoadEnemyArchetypes", func() error { _, err := config.LoadEnemyArchetypes(); return err }},
		{"LoadEnemyAbilities", func() error { _, err := config.LoadEnemyAbilities(); return err }},
		{"LoadAbilityTable", func() error { _, err := config.LoadAbilityTable(); return err }},
		{"LoadWardenConfigs", func() error { _, err := config.LoadWardenConfigs(); return err }},
		{"LoadLevelList", func() error { _, err := config.LoadLevelList(); return err }},
		{"LoadTierPresets", func() error { _, err := config.LoadTierPresets(); return err }},
		{"LoadVFXCatalog", func() error { _, err := config.LoadVFXCatalog(); return err }},
		{"LoadScenarios", func() error { _, err := config.LoadScenarios(); return err }},
	}
	for _, l := range loaders {
		t.Run(l.name, func(t *testing.T) {
			if err := l.fn(); err != nil {
				t.Errorf("%s 失败: %v", l.name, err)
			}
		})
	}
}

// TestAllScenariosLoadAndHaveRequiredFields 验证所有场景文件加载成功且必填字段有效。
func TestAllScenariosLoadAndHaveRequiredFields(t *testing.T) {
	scenarios, err := config.LoadScenarios()
	if err != nil {
		t.Fatalf("LoadScenarios 失败: %v", err)
	}
	if len(scenarios) == 0 {
		t.Skip("无场景文件")
	}
	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			if s.Name == "" {
				t.Error("场景 name 为空")
			}
			if s.MapID == "" {
				t.Error("场景 mapID 为空")
			}
			if s.Waves <= 0 {
				t.Errorf("场景 waves=%d 应 > 0", s.Waves)
			}
		})
	}
}
