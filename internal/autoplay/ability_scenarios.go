// ability_scenarios.go — JSON 驱动的能力级测试场景。
// 从 config/ability_tests.json 加载测试定义，自动生成 ScenarioStrategy + Assertion。
// 新增场景只需编辑 JSON，不需改 Go 代码。
package autoplay

import (
	"encoding/json"
	"fmt"
	"log"

	"defense2/internal/config"
)

// abilityTestsJSON 顶层结构。
type abilityTestsJSON struct {
	Scenarios []abilityTestDef `json:"scenarios"`
}

// abilityTestDef 单个测试场景定义。
type abilityTestDef struct {
	Name           string   `json:"name"`
	Desc           string   `json:"desc"`
	TowerAbility   string   `json:"towerAbility,omitempty"`   // 单个塔能力
	TowerAbilities []string `json:"towerAbilities,omitempty"` // 多个塔能力（组合测试）
	TowerAbility2  string   `json:"towerAbility2,omitempty"`  // 第二座塔的能力（交叉测试用）
	EnemyAbility   string   `json:"enemyAbility,omitempty"`   // 单个怪物能力（通过 enemyFilter 筛选原型）
	EnemyAbilities []string `json:"enemyAbilities,omitempty"` // 多个怪物能力（选择装备了这些能力的原型）
	WaitTicks      int      `json:"waitTicks"`
	Assert         string   `json:"assert"`
	AssertParam    int      `json:"assertParam,omitempty"`
	AssertField    string   `json:"assertField,omitempty"` // 字符串参数（如遥测键名）
	Comment        string   `json:"_comment,omitempty"`    // 跳过
}

// AbilityScenario 打包的测试场景。
type AbilityScenario struct {
	Name       string
	Strategy   Strategy
	Assertions []Assertion
}

// loadAbilityTestDefs 从嵌入 FS 加载测试定义。
func loadAbilityTestDefs() []abilityTestDef {
	fs := config.GetDataFS()
	if fs == nil {
		return nil
	}
	data, err := fs.ReadFile("config/ability_tests.json")
	if err != nil {
		log.Printf("[ability_tests] load failed: %v", err)
		return nil
	}
	var root abilityTestsJSON
	if err := json.Unmarshal(data, &root); err != nil {
		log.Printf("[ability_tests] parse failed: %v", err)
		return nil
	}
	// 过滤注释行（name 为空的）
	var defs []abilityTestDef
	for _, d := range root.Scenarios {
		if d.Name != "" {
			defs = append(defs, d)
		}
	}
	return defs
}

// GenerateAbilityScenarios 从 JSON 配置生成所有能力测试场景。
func GenerateAbilityScenarios() []AbilityScenario {
	defs := loadAbilityTestDefs()
	if len(defs) == 0 {
		return nil
	}

	var scenarios []AbilityScenario
	for _, d := range defs {
		s := buildScenarioFromDef(d)
		if s != nil {
			scenarios = append(scenarios, *s)
		}
	}
	return scenarios
}

// AbilityScenarioNames 返回所有场景名称。
func AbilityScenarioNames() []string {
	defs := loadAbilityTestDefs()
	names := make([]string, 0, len(defs))
	for _, d := range defs {
		names = append(names, d.Name)
	}
	return names
}

// buildScenarioFromDef 从定义构建场景。
func buildScenarioFromDef(d abilityTestDef) *AbilityScenario {
	// 收集所有塔能力
	var towerAbils []string
	if d.TowerAbility != "" {
		towerAbils = append(towerAbils, d.TowerAbility)
	}
	towerAbils = append(towerAbils, d.TowerAbilities...)

	// 收集怪物能力（决定 enemyFilter）
	var enemyAbils []string
	if d.EnemyAbility != "" {
		enemyAbils = append(enemyAbils, d.EnemyAbility)
	}
	enemyAbils = append(enemyAbils, d.EnemyAbilities...)

	// 确定 enemyFilter：有怪物能力时按原型筛选
	enemyFilter := enemyFilterForAbilities(enemyAbils)

	// 构建步骤
	steps := buildSteps(towerAbils, d.TowerAbility2, enemyFilter)

	strat := NewScenarioStrategy(d.Name, steps)

	// 构建断言
	assertion := Assertion{
		Name:      d.Name + ":" + d.Assert,
		AfterTick: d.WaitTicks,
		Type:      d.Assert,
		Param:     d.AssertParam,
		Field:     d.AssertField,
	}

	return &AbilityScenario{
		Name:       d.Name,
		Strategy:   strat,
		Assertions: []Assertion{assertion},
	}
}

// enemyFilterForAbilities 根据怪物能力选择对应的原型过滤器。
// 返回空字符串表示不过滤（使用默认波次出怪）。
func enemyFilterForAbilities(abilities []string) string {
	if len(abilities) == 0 {
		return ""
	}
	// 能力 → 原型映射（来自 enemies-core.json）
	abilToArch := map[string]string{
		"projectileBlock":  "shielder",
		"armorPlating":     "armored",
		"damageCap":        "tank",
		"damageCapPercent": "colossus",
		"evasion":          "phantom",
		"ccImmune":         "ironwill",
		"slowImmune":       "steadfast",
		"dashOnHit":        "runner",
		"phaseShift":       "phaser",
		"strengthDrain":    "drainer",
		"healAura":         "healer",
		"speedAura":        "buffer",
		"deathSplit":       "splitter",
		"deathSpawn":       "summoner",
		"purge":            "purifier",
	}

	// 取第一个能力对应的原型
	if arch, ok := abilToArch[abilities[0]]; ok {
		return arch
	}
	return ""
}

// buildSteps 生成通用场景步骤序列。
func buildSteps(towerAbils []string, towerAbility2 string, enemyFilter string) []ScenarioStep {
	var builtRow, builtCol int

	var steps []ScenarioStep

	// Step 0: 选战灵（如果已就绪则跳过）
	wardenSelected := false
	steps = append(steps, ScenarioStep{
		WaitUntil: func(s *GameState) bool {
			if s.WardenReady {
				wardenSelected = true // 已就绪，跳过选择
				return true
			}
			return true // 总是通过，尝试选择
		},
		Actions: func(s *GameState) []Action {
			if wardenSelected {
				return nil
			}
			wardenSelected = true
			return []Action{{Type: ActionSelectWarden, WardenKey: "prince"}}
		},
	})

	// Step 1: 建塔
	steps = append(steps, ScenarioStep{
		WaitUntil: func(s *GameState) bool {
			return len(s.BuildCells) > 0 && len(s.TowerDefs) > 0 && s.Gold >= s.TowerDefs[0].Cost
		},
		Actions: func(s *GameState) []Action {
			cell := s.BuildCells[0]
			builtRow, builtCol = cell.Row, cell.Col
			return []Action{{Type: ActionBuild, TowerKey: s.TowerDefs[0].Key, Cell: cell}}
		},
	})

	// Step 2: 等待塔出现
	steps = append(steps, ScenarioStep{
		WaitUntil: func(s *GameState) bool {
			for _, t := range s.Towers {
				if t.Row == builtRow && t.Col == builtCol {
					return true
				}
			}
			return false
		},
	})

	// Step 3: 添加塔能力
	for _, abil := range towerAbils {
		ab := abil // capture
		steps = append(steps, ScenarioStep{
			Actions: func(_ *GameState) []Action {
				return []Action{{Type: ActionAddAbility, Row: builtRow, Col: builtCol, AbilityName: ab}}
			},
		})
	}

	// Step 3b: 如果有第二座塔（交叉测试用），建第二座 + 添加能力
	if towerAbility2 != "" {
		var built2Row, built2Col int
		steps = append(steps, ScenarioStep{
			WaitUntil: func(s *GameState) bool {
				return len(s.BuildCells) > 1 && s.Gold >= s.TowerDefs[0].Cost
			},
			Actions: func(s *GameState) []Action {
				cell := s.BuildCells[1]
				built2Row, built2Col = cell.Row, cell.Col
				return []Action{{Type: ActionBuild, TowerKey: s.TowerDefs[0].Key, Cell: cell}}
			},
		})
		steps = append(steps, ScenarioStep{
			WaitUntil: func(s *GameState) bool {
				for _, t := range s.Towers {
					if t.Row == built2Row && t.Col == built2Col {
						return true
					}
				}
				return false
			},
			Actions: func(_ *GameState) []Action {
				return []Action{{Type: ActionAddAbility, Row: built2Row, Col: built2Col, AbilityName: towerAbility2}}
			},
		})
	}

	// Step 4: 开波
	steps = append(steps, ScenarioStep{
		WaitUntil: func(s *GameState) bool { return !s.WaveActive },
		Actions: func(_ *GameState) []Action {
			return []Action{{Type: ActionStartWave}}
		},
	})

	// enemyFilter 会通过 StageOptions.EnemyFilter 在 autoplay 启动时设置
	// 这里不需要额外步骤
	_ = enemyFilter

	return steps
}

// AbilityTestEnemyFilter 返回指定场景的 enemyFilter。
// autoplay 启动时需要将此设置到 StageOptions。
func AbilityTestEnemyFilter(scenarioName string) string {
	defs := loadAbilityTestDefs()
	for _, d := range defs {
		if d.Name != scenarioName {
			continue
		}
		var enemyAbils []string
		if d.EnemyAbility != "" {
			enemyAbils = append(enemyAbils, d.EnemyAbility)
		}
		enemyAbils = append(enemyAbils, d.EnemyAbilities...)
		return enemyFilterForAbilities(enemyAbils)
	}
	return ""
}

// AbilityScenarioDesc 返回场景描述。
func AbilityScenarioDesc(scenarioName string) string {
	defs := loadAbilityTestDefs()
	for _, d := range defs {
		if d.Name == scenarioName {
			if d.Desc != "" {
				return d.Desc
			}
			return fmt.Sprintf("%s (assert: %s)", d.Name, d.Assert)
		}
	}
	return scenarioName
}
