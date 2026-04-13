// strategy_simulation.go — 仿真测试场景矩阵。
// 定义 ~68 个仿真场景（mortal 模式），覆盖地图/难度/战灵/能力/经济维度。
package autoplay

import "fmt"

// SimScenario 仿真测试场景。
type SimScenario struct {
	ID          string
	MapID       string
	Difficulty  string
	Warden      string
	EnemyFilter string
	Strategy    Strategy
	Assertions  []Assertion
}

// ToTestCase 转换为 TestCase（ModeID=simulation）。
func (ss SimScenario) ToTestCase() TestCase {
	return TestCase{
		ID:          ss.ID,
		MapID:       ss.MapID,
		Difficulty:  ss.Difficulty,
		Warden:      ss.Warden,
		ModeID:      "simulation",
		EnemyFilter: ss.EnemyFilter,
		Strategy:    ss.Strategy,
		Assertions:  ss.Assertions,
	}
}

// AllSimScenarios 返回全部仿真测试场景。
func AllSimScenarios() []SimScenario {
	var all []SimScenario
	all = append(all, simBaseScenarios()...)
	all = append(all, simWardenScenarios()...)
	all = append(all, simAbilityScenarios()...)
	all = append(all, simEconomyScenarios()...)
	all = append(all, simMultiPathScenarios()...)
	return all
}

// ── A. 基础覆盖：8 地图 × 4 难度 = 32 场景 ──

func simBaseScenarios() []SimScenario {
	var scenarios []SimScenario
	for _, m := range Maps {
		for _, d := range Difficulties {
			id := fmt.Sprintf("sim_%s_%s", m, d)
			scenarios = append(scenarios, SimScenario{
				ID:         id,
				MapID:      m,
				Difficulty: d,
				Warden:     "prince",
				Strategy:   NewCompetentStrategy(WithCompetentSeed(stableHash(id))),
				Assertions: simDifficultyAssertions(d),
			})
		}
	}
	return scenarios
}

// ── B. 战灵维度：5 战灵 × 2 代表地图 = 10 场景 ──

func simWardenScenarios() []SimScenario {
	var scenarios []SimScenario
	reps := []struct {
		mapID, diff string
	}{
		{"map_01", "easy"},
		{"map_05", "hard"},
	}
	for _, w := range Wardens {
		for _, r := range reps {
			id := fmt.Sprintf("sim_warden_%s_%s_%s", w, r.mapID, r.diff)
			scenarios = append(scenarios, SimScenario{
				ID:         id,
				MapID:      r.mapID,
				Difficulty: r.diff,
				Warden:     w,
				Strategy: NewCompetentStrategy(
					WithCompetentWarden(w),
					WithCompetentSeed(stableHash(id)),
				),
				Assertions: simDifficultyAssertions(r.diff),
			})
		}
	}
	return scenarios
}

// ── C. 能力类别：6 类别 × 2 难度 = 12 场景 ──

func simAbilityScenarios() []SimScenario {
	categories := map[string][]string{
		"attack": {"scatter", "wideBeam", "spinAoe", "bounce", "splash", "multiTarget"},
		"cc":     {"slowPower", "slowDuration", "stunChance", "stunDuration"},
		"damage": {"crit", "distanceDamage", "executionBonus", "flatDamage", "momentum"},
		"dot":    {"burn", "bleedDot", "poison", "weaken"},
		"buff":   {"damageUpAura", "attackSpeedAura", "rangeAura", "critAura", "soloBoost"},
		"zone":   {"poisonZone", "silenceZone", "curseZone", "weakenZone"},
	}

	var scenarios []SimScenario
	diffs := []string{"normal", "hard"}
	for cat, abilities := range categories {
		for _, d := range diffs {
			id := fmt.Sprintf("sim_abil_%s_%s", cat, d)
			scenarios = append(scenarios, SimScenario{
				ID:         id,
				MapID:      "map_03",
				Difficulty: d,
				Warden:     "prince",
				Strategy: NewCompetentStrategy(
					WithCompetentAbilities(abilities),
					WithCompetentSeed(stableHash(id)),
				),
				Assertions: simDifficultyAssertions(d),
			})
		}
	}
	return scenarios
}

// ── D. 经济策略：4 策略 × 2 难度 = 8 场景 ──

func simEconomyScenarios() []SimScenario {
	type econVariant struct {
		suffix    string
		maxTowers int
	}
	variants := []econVariant{
		{"build_heavy", 8},  // 多建塔
		{"upgrade_heavy", 2}, // 少建多升
		{"minimal", 1},       // 极限单塔
		{"max_towers", 0},    // 不限制（自动计算）
	}

	var scenarios []SimScenario
	diffs := []string{"normal", "hard"}
	for _, v := range variants {
		for _, d := range diffs {
			id := fmt.Sprintf("sim_econ_%s_%s", v.suffix, d)
			scenarios = append(scenarios, SimScenario{
				ID:         id,
				MapID:      "map_01",
				Difficulty: d,
				Warden:     "prince",
				Strategy: NewCompetentStrategy(
					WithCompetentMaxTowers(v.maxTowers),
					WithCompetentSeed(stableHash(id)),
				),
				Assertions: simDifficultyAssertions(d),
			})
		}
	}
	return scenarios
}

// ── E. 多路径地图：map_02/05/08 × 2 难度 = 6 场景 ──

func simMultiPathScenarios() []SimScenario {
	multiPathMaps := []string{"map_02", "map_05", "map_08"}
	var scenarios []SimScenario
	diffs := []string{"normal", "hard"}
	for _, m := range multiPathMaps {
		for _, d := range diffs {
			id := fmt.Sprintf("sim_multipath_%s_%s", m, d)
			scenarios = append(scenarios, SimScenario{
				ID:         id,
				MapID:      m,
				Difficulty: d,
				Warden:     "prince",
				Strategy:   NewCompetentStrategy(WithCompetentSeed(stableHash(id))),
				Assertions: simDifficultyAssertions(d),
			})
		}
	}
	return scenarios
}

// ── 断言定义 ──

func simDifficultyAssertions(diff string) []Assertion {
	switch diff {
	case "easy":
		return []Assertion{
			{Name: "sim_victory", Type: "victory"},
			{Name: "sim_lives_gte_10", Type: "final_lives_gte", Expected: 10},
		}
	case "normal":
		return []Assertion{
			{Name: "sim_victory", Type: "victory"},
			{Name: "sim_lives_gte_3", Type: "final_lives_gte", Expected: 3},
			{Name: "sim_takes_damage", Type: "final_lives_lte", Expected: 18},
		}
	case "hard":
		return []Assertion{
			{Name: "sim_survive_60pct", Type: "waves_survived_gte", Expected: 8}, // 大多数地图 12-15 波
			{Name: "sim_takes_heavy_damage", Type: "final_lives_lte", Expected: 12},
		}
	case "extreme":
		return []Assertion{
			{Name: "sim_survive_40pct", Type: "waves_survived_gte", Expected: 5},
			{Name: "sim_heavy_damage", Type: "final_lives_lte", Expected: 5},
		}
	}
	return nil
}

// stableHash 生成稳定的整数种子（确保跨运行可复现）。
func stableHash(s string) int64 {
	h := int64(0)
	for _, c := range s {
		h = h*31 + int64(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}
