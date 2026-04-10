// config_rules_test.go — 配置契约测试。
// 加载实际 JSON 配置文件并校验字段范围，确保设计约束不被打破。
package contracts_test

import (
	"fmt"
	"testing"

	defense2 "defense2"
	"defense2/internal/config"
)

func init() {
	config.SetDataFS(&defense2.DataFS)
}

// TestTowerConfigRanges 验证所有塔配置字段在合法范围内。
func TestTowerConfigRanges(t *testing.T) {
	all, err := config.LoadAllTowers()
	if err != nil {
		t.Fatalf("加载塔配置失败: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("塔配置为空")
	}

	for key, tower := range all {
		t.Run(key, func(t *testing.T) {
			errs := config.ValidateTowerDef(tower)
			for _, e := range errs {
				t.Errorf("[%s] %s", key, e.Error())
			}
		})
	}
}

// TestTowerConfigFields 验证关键塔字段的具体约束。
func TestTowerConfigFields(t *testing.T) {
	all, err := config.LoadAllTowers()
	if err != nil {
		t.Fatalf("加载塔配置失败: %v", err)
	}

	cases := []struct {
		name  string
		check func(key string, t2 *config.TowerJSON) error
	}{
		{
			name: "label 非空",
			check: func(key string, t2 *config.TowerJSON) error {
				if t2.Label == "" {
					return fmt.Errorf("塔 %s 的 label 为空", key)
				}
				return nil
			},
		},
		{
			name: "buildCost 合理范围",
			check: func(key string, t2 *config.TowerJSON) error {
				if t2.BuildCost < 10 || t2.BuildCost > 200 {
					return fmt.Errorf("塔 %s 的 buildCost=%d 超出 [10,200]", key, t2.BuildCost)
				}
				return nil
			},
		},
		{
			name: "baseRange 合理范围",
			check: func(key string, t2 *config.TowerJSON) error {
				// baseRange=0 表示由 tier-presets 动态分配（如 basic 塔）
				if t2.BaseRange != 0 && (t2.BaseRange < 50 || t2.BaseRange > 10000) {
					return fmt.Errorf("塔 %s 的 baseRange=%.0f 超出 [50,10000]", key, t2.BaseRange)
				}
				return nil
			},
		},
		{
			name: "baseDamage 非负",
			check: func(key string, t2 *config.TowerJSON) error {
				if t2.BaseDamage < 0 {
					return fmt.Errorf("塔 %s 的 baseDamage=%.1f 为负", key, t2.BaseDamage)
				}
				return nil
			},
		},
		{
			name: "baseAttackSpeed 合理范围",
			check: func(key string, t2 *config.TowerJSON) error {
				if t2.BaseAttackSpeed < 0 || t2.BaseAttackSpeed > 20 {
					return fmt.Errorf("塔 %s 的 baseAttackSpeed=%.2f 超出 [0,20]", key, t2.BaseAttackSpeed)
				}
				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for key, tower := range all {
				if err := tc.check(key, tower); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

// TestEnemyConfigRanges 验证所有敌人原型配置字段在合法范围内。
func TestEnemyConfigRanges(t *testing.T) {
	archetypes, err := config.LoadEnemyArchetypes()
	if err != nil {
		t.Fatalf("加载敌人配置失败: %v", err)
	}
	if len(archetypes) == 0 {
		t.Fatal("敌人配置为空")
	}

	for key, arch := range archetypes {
		t.Run(key, func(t *testing.T) {
			errs := config.ValidateEnemyDef(arch)
			for _, e := range errs {
				t.Errorf("[%s] %s", key, e.Error())
			}
		})
	}
}

// TestEnemyConfigFields 验证关键敌人字段的具体约束。
func TestEnemyConfigFields(t *testing.T) {
	archetypes, err := config.LoadEnemyArchetypes()
	if err != nil {
		t.Fatalf("加载敌人配置失败: %v", err)
	}

	cases := []struct {
		name  string
		check func(key string, a *config.EnemyArchetype) error
	}{
		{
			name: "label 非空",
			check: func(key string, a *config.EnemyArchetype) error {
				if a.Label == "" {
					return fmt.Errorf("敌人 %s 的 label 为空", key)
				}
				return nil
			},
		},
		{
			name: "hpScale > 0.1",
			check: func(key string, a *config.EnemyArchetype) error {
				if a.HPScale <= 0.1 {
					return fmt.Errorf("敌人 %s 的 hpScale=%.2f <= 0.1", key, a.HPScale)
				}
				return nil
			},
		},
		{
			name: "speedScale >= 0",
			check: func(key string, a *config.EnemyArchetype) error {
				if a.SpeedScale < 0 {
					return fmt.Errorf("敌人 %s 的 speedScale=%.2f < 0", key, a.SpeedScale)
				}
				return nil
			},
		},
		{
			name: "rewardScale >= 0",
			check: func(key string, a *config.EnemyArchetype) error {
				if a.RewardScale < 0 {
					return fmt.Errorf("敌人 %s 的 rewardScale=%.2f < 0", key, a.RewardScale)
				}
				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for key, arch := range archetypes {
				if err := tc.check(key, arch); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

