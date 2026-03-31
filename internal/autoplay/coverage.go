// coverage.go — 覆盖矩阵与测试计划生成器。
// 通过 pairwise 组合 + 定向用例实现 ~100 个测试覆盖所有游戏内容。
package autoplay

import "fmt"

// TestCase 单个测试用例。
type TestCase struct {
	ID          string
	MapID       string
	Difficulty  string
	Warden      string
	ModeID      string // 游戏模式（空=autoplay）
	EnemyFilter string // 敌人过滤器（空=默认）
	Strategy    Strategy
}

// EffectiveModeID 返回实际使用的游戏模式 ID。
func (tc TestCase) EffectiveModeID() string {
	if tc.ModeID == "" {
		return "autoplay"
	}
	return tc.ModeID
}

// ─── 内容清单 ───

var (
	Maps         = []string{"map_01", "map_02", "map_03", "map_04", "map_05", "map_06", "map_07", "map_08"}
	Difficulties = []string{"easy", "normal", "hard", "extreme"}
	Wardens      = []string{"prince", "core", "chain", "skystrike", "envoy"}
	TowerKeys    = []string{"laser", "freeze", "electric", "hunter", "en-04", "en-05", "en-08", "wl-02"}

	// SkillNames 所有可用技能。
	SkillNames = []string{
		"chainLightning", "nukeBomb", "windBlade", "channelLaser",
		"missileBarrage", "judgmentBeam", "chainLightningBolts", "judgmentRain", "thunderSmite",
	}

	// GameModes 需要测试的游戏模式。
	GameModes = []string{"campaign", "endless", "timed", "bossRush", "challenge", "test"}

	// EnemyArchetypes 所有敌人原型。
	EnemyArchetypes = []string{
		"normal", "runner", "tank", "armored", "shielded", "swarm",
		"stealth", "splitter", "teleporter", "healer", "buffer", "flying", "dummy",
	}
)

// GenerateTestPlan 生成完整测试计划。
func GenerateTestPlan() []TestCase {
	var cases []TestCase

	// 1. Pairwise 组合覆盖 (~32)
	cases = append(cases, generatePairwise()...)

	// 2. 单塔极限 (8)
	cases = append(cases, generateFocusTower()...)

	// 3. 敌人定向 — 使用 EnemyFilter (13)
	cases = append(cases, generateEnemySpecific()...)

	// 4. 技能覆盖 (9)
	cases = append(cases, generateSkillCoverage()...)

	// 5. 游戏模式覆盖 (6)
	cases = append(cases, generateModeCoverage()...)

	// 6. 交互场景 (~4)
	cases = append(cases, generateInteractionScenarios()...)

	// 7. 边界测试 (~4)
	cases = append(cases, generateEdgeCases()...)

	return cases
}

// generatePairwise 生成 pairwise 组合（map × difficulty 全覆盖，warden/strategy 轮换）。
func generatePairwise() []TestCase {
	strategies := []string{"random", "greedy"}
	var cases []TestCase
	idx := 0
	for _, m := range Maps {
		for _, d := range Difficulties {
			w := Wardens[idx%len(Wardens)]
			sName := strategies[idx%len(strategies)]
			cases = append(cases, TestCase{
				ID:         fmt.Sprintf("pw_%03d_%s_%s", idx, m, d),
				MapID:      m,
				Difficulty: d,
				Warden:     w,
				Strategy:   makeStrategy(sName, ""),
			})
			idx++
		}
	}
	return cases
}

// generateFocusTower 为每种塔生成单塔极限用例。
func generateFocusTower() []TestCase {
	var cases []TestCase
	for i, key := range TowerKeys {
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("focus_%s", key),
			MapID:      Maps[i%len(Maps)],
			Difficulty: "normal",
			Warden:     Wardens[i%len(Wardens)],
			Strategy:   NewFocusStrategy(key),
		})
	}
	return cases
}

// generateEnemySpecific 为每种敌人原型生成定向用例（通过 EnemyFilter）。
func generateEnemySpecific() []TestCase {
	// 原型 → 对应的 EnemyFilter 映射
	filterMap := map[string]string{
		"flying": "flying-only",
		"dummy":  "dummy",
	}
	var cases []TestCase
	for i, arch := range EnemyArchetypes {
		filter := arch // 默认用原型名作 filter
		if f, ok := filterMap[arch]; ok {
			filter = f
		}
		cases = append(cases, TestCase{
			ID:          fmt.Sprintf("enemy_%s", arch),
			MapID:       Maps[i%len(Maps)],
			Difficulty:  "normal",
			Warden:      "prince",
			EnemyFilter: filter,
			Strategy:    NewGreedyStrategy(),
		})
	}
	return cases
}

// generateSkillCoverage 为每种技能生成测试用例。
// 策略：建塔后给第一座塔装技能，给战灵装另一个技能。
func generateSkillCoverage() []TestCase {
	var cases []TestCase
	for i, sk := range SkillNames {
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("skill_%s", sk),
			MapID:      Maps[i%len(Maps)],
			Difficulty: "normal",
			Warden:     Wardens[i%len(Wardens)],
			Strategy:   NewSkillTestStrategy(sk),
		})
	}
	return cases
}

// generateModeCoverage 为每种游戏模式生成测试用例。
func generateModeCoverage() []TestCase {
	var cases []TestCase
	for i, mode := range GameModes {
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("mode_%s", mode),
			MapID:      Maps[i%len(Maps)],
			Difficulty: "normal",
			Warden:     "prince",
			ModeID:     mode,
			Strategy:   NewGreedyStrategy(),
		})
	}
	return cases
}

// generateInteractionScenarios 生成交互 FSM 场景。
func generateInteractionScenarios() []TestCase {
	scenarios := AllScenariosMap()
	var cases []TestCase
	for name, factory := range scenarios {
		cases = append(cases, TestCase{
			ID:       fmt.Sprintf("scenario_%s", name),
			MapID:    "map_01",
			Difficulty: "normal",
			Warden:  "prince",
			Strategy: factory(),
		})
	}
	return cases
}

// generateEdgeCases 生成边界测试用例。
func generateEdgeCases() []TestCase {
	return []TestCase{
		{
			ID: "edge_zero_gold", MapID: "map_01", Difficulty: "extreme",
			Warden: "prince", Strategy: ZeroGoldBuildScenario(),
		},
		{
			ID: "edge_rapid_actions", MapID: "map_01", Difficulty: "normal",
			Warden: "chain", Strategy: RapidActionScenario(),
		},
		{
			ID: "edge_high_speed_random", MapID: "map_03", Difficulty: "hard",
			Warden: "skystrike", Strategy: NewRandomStrategy(42),
		},
		{
			ID: "edge_greedy_extreme", MapID: "map_08", Difficulty: "extreme",
			Warden: "envoy", Strategy: NewGreedyStrategy(),
		},
	}
}

// makeStrategy 根据名称创建策略。
func makeStrategy(name, towerKey string) Strategy {
	switch name {
	case "random":
		return NewRandomStrategy(0)
	case "greedy":
		return NewGreedyStrategy()
	case "focus":
		return NewFocusStrategy(towerKey)
	default:
		return NewRandomStrategy(0)
	}
}

// ParseCLICases 从 CLI 参数生成测试用例。
func ParseCLICases(runs int, strategies, mapID, difficulty, warden string) []TestCase {
	var cases []TestCase
	stratNames := splitCSV(strategies)
	for run := 0; run < runs; run++ {
		for _, sName := range stratNames {
			if sName == "focus" {
				for _, key := range TowerKeys {
					cases = append(cases, TestCase{
						ID: FormatSessionID("focus_"+key, mapID, difficulty, run),
						MapID: mapID, Difficulty: difficulty, Warden: warden,
						Strategy: NewFocusStrategy(key),
					})
				}
			} else {
				cases = append(cases, TestCase{
					ID: FormatSessionID(sName, mapID, difficulty, run),
					MapID: mapID, Difficulty: difficulty, Warden: warden,
					Strategy: makeStrategy(sName, ""),
				})
			}
		}
	}
	return cases
}

// ScenarioCase 创建单个场景测试用例。
func ScenarioCase(name, mapID string) TestCase {
	scenarios := AllScenariosMap()
	factory, ok := scenarios[name]
	if !ok {
		factory = func() Strategy { return BuildFlowScenario() }
	}
	return TestCase{
		ID: fmt.Sprintf("scenario_%s", name), MapID: mapID,
		Difficulty: "normal", Warden: "prince", Strategy: factory(),
	}
}

func splitCSV(s string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if c == ',' {
			if current != "" {
				result = append(result, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
