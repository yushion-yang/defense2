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
	Assertions  []Assertion // 平衡断言（可选）
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
	TowerKeys    = []string{"basic"} // 单塔系统：所有差异化来自能力选择

	// GameModes 需要测试的游戏模式。
	GameModes = []string{"casual", "endless", "timed", "bossRush", "challenge", "test", "autoplay", "simulation"}

	// EnemyArchetypes 所有敌人原型（18 个）。
	EnemyArchetypes = []string{
		"normal", "runner", "tank", "armored", "swarm",
		"phantom", "shielder", "colossus", "ironwill", "steadfast",
		"phaser", "drainer", "healer", "buffer",
		"splitter", "summoner", "purifier", "dummy",
	}
)

// GenerateTestPlan 生成完整测试计划。
// 设计原则：每项游戏内容有且只有 1 个专项用例，去掉冗余组合。
func GenerateTestPlan() []TestCase {
	var cases []TestCase

	// 1. 地图覆盖 — 每地图 1 局 normal (8)
	cases = append(cases, generateMapCoverage()...)

	// 2. 难度覆盖 — 固定 map_01 × 4 难度 (4)
	cases = append(cases, generateDifficultyCoverage()...)

	// 3. 单塔极限 — 每种塔 1 局 focus (8)
	cases = append(cases, generateFocusTower()...)

	// 4. 敌人定向 — 每种原型 1 局 EnemyFilter (12)
	cases = append(cases, generateEnemySpecific()...)

	// 5. 战灵覆盖 — 每种战灵 1 局 (5)
	cases = append(cases, generateWardenCoverage()...)

	// 6. 游戏模式 — 各模式 1 局 (6)
	cases = append(cases, generateModeCoverage()...)

	// 7. 手动测试对照 scenario (~11)
	cases = append(cases, generateManualTestScenarios()...)

	// 8. 平衡验证 (4)
	cases = append(cases, generateBalanceTests()...)

	// 9. 特殊组合 (3)
	cases = append(cases, generateCombinationTests()...)

	// 10. 视觉目录 (2)
	cases = append(cases, generateVisualCatalog()...)

	// 11. 边界测试 (4)
	cases = append(cases, generateEdgeCases()...)

	// 12. 最强玩法全地图通关 (40 = 8地图×5风格)
	cases = append(cases, generateChampionClear()...)

	// 13. 困难模式通关 (3)
	cases = append(cases, generateHardModeClear()...)

	return cases
}

// generateChampionClear 最强玩法全地图通关测试。
func generateChampionClear() []TestCase {
	var cases []TestCase
	for _, bs := range CampaignClearScenarios() {
		cases = append(cases, TestCase{
			ID:         bs.ID,
			MapID:      bs.MapID,
			Difficulty: bs.Difficulty,
			Warden:     bs.Warden,
			Strategy:   bs.Strategy,
			Assertions: bs.Assertions,
		})
	}
	return cases
}

// generateHardModeClear 困难模式核心地图通关。
func generateHardModeClear() []TestCase {
	var cases []TestCase
	for _, bs := range HardModeClearScenarios() {
		cases = append(cases, TestCase{
			ID:         bs.ID,
			MapID:      bs.MapID,
			Difficulty: bs.Difficulty,
			Warden:     bs.Warden,
			Strategy:   bs.Strategy,
			Assertions: bs.Assertions,
		})
	}
	return cases
}

// generateMapCoverage 每张地图 1 局 normal 难度 greedy。
func generateMapCoverage() []TestCase {
	var cases []TestCase
	for i, m := range Maps {
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("map_%s", m),
			MapID:      m,
			Difficulty: "normal",
			Warden:     Wardens[i%len(Wardens)],
			Strategy:   NewGreedyStrategy(),
		})
	}
	return cases
}

// generateDifficultyCoverage 固定 map_01 × 4 难度。
func generateDifficultyCoverage() []TestCase {
	var cases []TestCase
	for _, d := range Difficulties {
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("diff_%s", d),
			MapID:      "map_01",
			Difficulty: d,
			Warden:     "prince",
			Strategy:   NewGreedyStrategy(),
		})
	}
	return cases
}

// generateWardenCoverage 每种战灵 1 局。
func generateWardenCoverage() []TestCase {
	var cases []TestCase
	for i, w := range Wardens {
		g := NewGreedyStrategy()
		g.wardenKey = w
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("warden_%s", w),
			MapID:      Maps[i%len(Maps)],
			Difficulty: "normal",
			Warden:     w,
			Strategy:   g,
		})
	}
	return cases
}

// generateManualTestScenarios 对照手动测试文档的 scenario。
func generateManualTestScenarios() []TestCase {
	scenarios := AllScenariosMap()
	var cases []TestCase
	for name, factory := range scenarios {
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("scenario_%s", name),
			MapID:      "map_01",
			Difficulty: "normal",
			Warden:     "prince",
			Strategy:   factory(),
		})
	}
	return cases
}

// generateBalanceTests 数值平衡验证（26 个场景覆盖 6 个维度）。
func generateBalanceTests() []TestCase {
	var cases []TestCase
	for _, bs := range AllBalanceScenarios() {
		cases = append(cases, TestCase{
			ID:          bs.ID,
			MapID:       bs.MapID,
			Difficulty:  bs.Difficulty,
			Warden:      bs.Warden,
			EnemyFilter: bs.EnemyFilter,
			Strategy:    bs.Strategy,
			Assertions:  bs.Assertions,
		})
	}
	return cases
}

// generateCombinationTests 特殊敌人组合。
func generateCombinationTests() []TestCase {
	return []TestCase{
		{ID: "combo_medic_tank", MapID: "map_01", Difficulty: "hard", Warden: "prince", EnemyFilter: "mixed", Strategy: NewGreedyStrategy()},
		{ID: "combo_splitter_swarm", MapID: "map_01", Difficulty: "normal", Warden: "prince", EnemyFilter: "stress", Strategy: NewGreedyStrategy()},
		{ID: "combo_stealth_runner", MapID: "map_01", Difficulty: "normal", Warden: "prince", EnemyFilter: "mixed", Strategy: NewGreedyStrategy()},
	}
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

// generateVisualCatalog 生成视觉目录用例。
// 单局建造所有塔类型，最大化视觉内容覆盖。
func generateVisualCatalog() []TestCase {
	return []TestCase{
		{
			ID: "visual_catalog_normal", MapID: "map_01", Difficulty: "normal",
			Warden: "prince", Strategy: NewVisualCatalogStrategy(),
		},
		{
			ID: "visual_catalog_hard", MapID: "map_05", Difficulty: "hard",
			Warden: "skystrike", Strategy: NewVisualCatalogStrategy(),
		},
	}
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
						ID:    FormatSessionID("focus_"+key, mapID, difficulty, run),
						MapID: mapID, Difficulty: difficulty, Warden: warden,
						Strategy: NewFocusStrategy(key),
					})
				}
			} else {
				cases = append(cases, TestCase{
					ID:    FormatSessionID(sName, mapID, difficulty, run),
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
