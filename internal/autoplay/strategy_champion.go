// strategy_champion.go — "最强玩法"策略。
// 模拟高手玩家：精确控制建塔数量+集中升级+合理能力搭配。
// 用于验证战役模式全 8 张地图能否通关。
package autoplay

import "math"

// ChampionStyle 最强玩法的风格类型。
type ChampionStyle int

const (
	// StyleBalanced 均衡型：4塔+均匀升级，攻击+控制+伤害混搭。适合大多数地图。
	StyleBalanced ChampionStyle = iota
	// StyleElite 精英型：2塔+极限升级，单塔强度拉满。适合小地图/少格子。
	StyleElite
	// StyleSwarm 铺开型：6塔+广覆盖，光环+区域能力为主。适合多格子大地图。
	StyleSwarm
	// StyleCC 控制型：3塔+控制能力为主，减速+眩晕+减伤。适合高难度。
	StyleCC
	// StyleDPS 纯输出型：3塔+全输出能力，暴击+溅射+追伤。速通风格。
	StyleDPS
)

// ChampionStrategy 最强玩法策略。
type ChampionStrategy struct {
	style      ChampionStyle
	wardenKey  string
	targetTowers int // 目标塔数
	phase      int
	builtCount int
	lastBuild  int
	upgradeTick int
	abilityPlan [][]string // 每座塔的能力分配计划
	abilityAssigned map[[2]int]int // [row,col] -> 已分配能力数
}

func NewChampionStrategy(style ChampionStyle, warden string) *ChampionStrategy {
	s := &ChampionStrategy{
		style:           style,
		wardenKey:       warden,
		abilityAssigned: make(map[[2]int]int),
	}
	switch style {
	case StyleBalanced:
		s.targetTowers = 4
		s.abilityPlan = [][]string{
			{"scatter", "crit", "burn"},               // 塔1: 散射+暴击+灼烧
			{"slowPower", "slowDuration", "weaken"},    // 塔2: 减速+减速延长+削弱
			{"splash", "flatDamage", "momentum"},       // 塔3: 溅射+固伤+动量
			{"damageUpAura", "attackSpeedAura", "critAura"}, // 塔4: 三光环
		}
	case StyleElite:
		s.targetTowers = 2
		s.abilityPlan = [][]string{
			{"crit", "flatDamage", "executionBonus", "momentum", "distanceDamage"}, // 塔1: 全输出
			{"slowPower", "stunChance", "weaken", "burn", "bleedDot"},               // 塔2: 全控制+DoT
		}
	case StyleSwarm:
		s.targetTowers = 6
		s.abilityPlan = [][]string{
			{"damageUpAura", "attackSpeedAura", "critAura"}, // 塔1: 三光环核心
			{"scatter", "crit", "burn"},                      // 塔2: 输出
			{"slowPower", "stunChance", "weaken"},            // 塔3: 控制
			{"splash", "flatDamage", "poisonZone"},           // 塔4: 范围
			{"bleedDot", "poison", "weakenZone"},             // 塔5: DoT+区域
			{"silenceZone", "curseZone", "soloBoost"},        // 塔6: 区域+独立
		}
	case StyleCC:
		s.targetTowers = 3
		s.abilityPlan = [][]string{
			{"slowPower", "slowDuration", "stunChance", "stunDuration"}, // 塔1: 极限控制
			{"weaken", "burn", "bleedDot", "poison"},                    // 塔2: 削弱+DoT
			{"damageUpAura", "critAura", "crit", "flatDamage"},          // 塔3: 输出
		}
	case StyleDPS:
		s.targetTowers = 3
		s.abilityPlan = [][]string{
			{"crit", "flatDamage", "executionBonus", "splash"},     // 塔1: 爆发
			{"scatter", "momentum", "distanceDamage", "stackDamage"}, // 塔2: 持续
			{"damageUpAura", "attackSpeedAura", "critAura", "burn"},  // 塔3: 光环+灼烧
		}
	}
	return s
}

func (s *ChampionStrategy) Name() string {
	names := []string{"champion_balanced", "champion_elite", "champion_swarm", "champion_cc", "champion_dps"}
	if int(s.style) < len(names) {
		return names[s.style]
	}
	return "champion"
}

func (s *ChampionStrategy) Init(_ *GameState) {}

func (s *ChampionStrategy) Decide(state *GameState) []Action {
	if state.GameOver {
		return nil
	}

	var actions []Action

	// 战灵选择
	if !state.WardenReady {
		return []Action{{Type: ActionSelectWarden, WardenKey: s.wardenKey}}
	}

	// 开波：立即开
	if !state.WaveActive && state.Wave < state.MaxWaves {
		actions = append(actions, Action{Type: ActionStartWave})
	}

	// 能力分配（每帧检查新建的塔）
	actions = append(actions, s.assignAbilities(state)...)

	// 阶段判定
	progress := 0.0
	if state.MaxWaves > 0 {
		progress = float64(state.Wave) / float64(state.MaxWaves)
	}

	if s.builtCount < s.targetTowers {
		// 建塔阶段：达到目标数量前持续建
		actions = append(actions, s.decideBuild(state)...)
	}

	// 升级：建完塔后全力升级；建塔中期也穿插升级
	if s.builtCount >= s.targetTowers || (progress > 0.2 && s.builtCount >= 2) {
		actions = append(actions, s.decideUpgrade(state)...)
	}

	return actions
}

func (s *ChampionStrategy) decideBuild(state *GameState) []Action {
	if len(state.BuildCells) == 0 || len(state.TowerDefs) == 0 {
		return nil
	}
	best := state.TowerDefs[0]
	if best.Cost > state.Gold {
		return nil
	}
	// 冷却：至少间隔 60 tick（1秒）
	if state.Tick-s.lastBuild < 60 && s.builtCount > 0 {
		return nil
	}

	// 选位置：优先靠近地图中心
	cell := s.bestCell(state)
	s.lastBuild = state.Tick
	s.builtCount++
	return []Action{{Type: ActionBuild, TowerKey: best.Key, Cell: cell}}
}

func (s *ChampionStrategy) bestCell(state *GameState) Cell {
	centerX := state.MapPixelW / 2
	centerY := state.MapPixelH / 2
	if centerX == 0 {
		centerX = 600
	}
	if centerY == 0 {
		centerY = 270
	}
	best := state.BuildCells[0]
	bestDist := math.MaxFloat64
	for _, c := range state.BuildCells {
		d := math.Hypot(c.X-centerX, c.Y-centerY)
		if d < bestDist {
			bestDist = d
			best = c
		}
	}
	return best
}

func (s *ChampionStrategy) decideUpgrade(state *GameState) []Action {
	if len(state.Towers) == 0 || state.Gold < 10 {
		return nil
	}
	// 冷却：至少间隔 30 tick
	if state.Tick-s.upgradeTick < 30 {
		return nil
	}
	// 找强度最低的塔升级（均匀提升）
	var target TowerInfo
	minStr := math.MaxFloat64
	for _, t := range state.Towers {
		if float64(t.Strength) < minStr {
			minStr = float64(t.Strength)
			target = t
		}
	}
	s.upgradeTick = state.Tick
	return []Action{{Type: ActionUpgrade, Row: target.Row, Col: target.Col}}
}

func (s *ChampionStrategy) assignAbilities(state *GameState) []Action {
	var actions []Action
	for i, t := range state.Towers {
		if i >= len(s.abilityPlan) {
			break
		}
		key := [2]int{t.Row, t.Col}
		assigned := s.abilityAssigned[key]
		plan := s.abilityPlan[i]
		for j := assigned; j < len(plan); j++ {
			actions = append(actions, Action{
				Type: ActionAddAbility, Row: t.Row, Col: t.Col,
				AbilityName: plan[j],
			})
		}
		s.abilityAssigned[key] = len(plan)
	}
	return actions
}

// ─── 全地图通关场景 ───

// CampaignClearScenarios 生成战役模式全 8 张地图×5种风格的通关测试场景。
func CampaignClearScenarios() []BalanceScenario {
	maps := []string{"map_01", "map_02", "map_03", "map_04", "map_05", "map_06", "map_07", "map_08"}
	styles := []struct {
		style  ChampionStyle
		name   string
		warden string
	}{
		{StyleBalanced, "balanced", "prince"},
		{StyleElite, "elite", "core"},
		{StyleSwarm, "swarm", "chain"},
		{StyleCC, "cc", "skystrike"},
		{StyleDPS, "dps", "envoy"},
	}

	var scenarios []BalanceScenario
	for _, m := range maps {
		for _, st := range styles {
			scenarios = append(scenarios, BalanceScenario{
				ID:         "campaign_" + m + "_" + st.name,
				MapID:      m,
				Difficulty: "normal",
				Warden:     st.warden,
				Strategy:   NewChampionStrategy(st.style, st.warden),
				Assertions: []Assertion{
					{Name: m + "_" + st.name + "_victory", Type: "victory"},
					{Name: m + "_" + st.name + "_no_stall", Type: "no_economy_stall", Param: 5},
				},
			})
		}
	}
	return scenarios
}

// HardModeClearScenarios 困难模式下 3 张核心地图的通关验证。
func HardModeClearScenarios() []BalanceScenario {
	return []BalanceScenario{
		{
			ID: "hard_map01_balanced", MapID: "map_01", Difficulty: "hard", Warden: "prince",
			Strategy:   NewChampionStrategy(StyleBalanced, "prince"),
			Assertions: []Assertion{
				{Name: "hard01_survive", Type: "waves_survived_gte", Param: 8},
				{Name: "hard01_kills", Type: "total_kills_gte", Param: 20},
			},
		},
		{
			ID: "hard_map02_cc", MapID: "map_02", Difficulty: "hard", Warden: "skystrike",
			Strategy:   NewChampionStrategy(StyleCC, "skystrike"),
			Assertions: []Assertion{
				{Name: "hard02_survive", Type: "waves_survived_gte", Param: 8},
			},
		},
		{
			ID: "hard_map03_dps", MapID: "map_03", Difficulty: "hard", Warden: "envoy",
			Strategy:   NewChampionStrategy(StyleDPS, "envoy"),
			Assertions: []Assertion{
				{Name: "hard03_survive", Type: "waves_survived_gte", Param: 6},
			},
		},
	}
}
