// strategy_balance.go — 数值平衡测试策略与场景。
// 可配置的贪心策略 + 26 个预定义平衡测试场景，覆盖难度/经济/能力/敌人/Boss/强弱对比。
package autoplay

// BalanceGreedyStrategy 可配置的平衡测试贪心策略。
type BalanceGreedyStrategy struct {
	wardenKey       string
	maxTowers       int    // 最多建几座塔（0=无限）
	upgradeFirst    bool   // 优先升级
	abilityPref     string // 偏好能力类别（attack/cc/damage/dot/buff/zone，空=随机）
	noSell          bool   // 禁止卖塔
	sellRebuy       bool   // 频繁卖/买
	noTowers        bool   // 不建塔（纯战灵测试）
	phase           int
	lastBuild       int
	builtCount      int
	sellCooldown    int
	lastSellTick    int
	abilityAssigned map[[2]int][]string // [row,col] -> 已分配能力
}

// BalanceOpt 策略配置选项。
type BalanceOpt func(*BalanceGreedyStrategy)

func WithMaxTowers(n int) BalanceOpt { return func(s *BalanceGreedyStrategy) { s.maxTowers = n } }
func WithUpgradeFirst(v bool) BalanceOpt {
	return func(s *BalanceGreedyStrategy) { s.upgradeFirst = v }
}
func WithAbilityPref(p string) BalanceOpt {
	return func(s *BalanceGreedyStrategy) { s.abilityPref = p }
}
func WithNoSell(v bool) BalanceOpt      { return func(s *BalanceGreedyStrategy) { s.noSell = v } }
func WithSellRebuy(v bool) BalanceOpt   { return func(s *BalanceGreedyStrategy) { s.sellRebuy = v } }
func WithNoTowers(v bool) BalanceOpt    { return func(s *BalanceGreedyStrategy) { s.noTowers = v } }
func WithWardenKey(k string) BalanceOpt { return func(s *BalanceGreedyStrategy) { s.wardenKey = k } }

// NewBalanceGreedyStrategy 创建可配置的平衡测试策略。
func NewBalanceGreedyStrategy(opts ...BalanceOpt) *BalanceGreedyStrategy {
	s := &BalanceGreedyStrategy{
		wardenKey:       "prince",
		abilityAssigned: make(map[[2]int][]string),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *BalanceGreedyStrategy) Name() string      { return "balance_greedy" }
func (s *BalanceGreedyStrategy) Init(_ *GameState) {}

func (s *BalanceGreedyStrategy) Decide(state *GameState) []Action {
	if state.GameOver {
		return nil
	}

	var actions []Action

	// 战灵选择
	if !state.WardenReady {
		return []Action{{Type: ActionSelectWarden, WardenKey: s.wardenKey}}
	}

	// 开波
	if !state.WaveActive && state.Wave < state.MaxWaves {
		actions = append(actions, Action{Type: ActionStartWave})
	}

	// 纯战灵模式
	if s.noTowers {
		return actions
	}

	// 频繁卖/买模式
	if s.sellRebuy && len(state.Towers) > 0 && state.Tick-s.lastSellTick > 300 {
		t := state.Towers[state.Tick%len(state.Towers)]
		actions = append(actions, Action{Type: ActionSell, Row: t.Row, Col: t.Col})
		s.lastSellTick = state.Tick
		s.builtCount--
		if s.builtCount < 0 {
			s.builtCount = 0
		}
	}

	// 阶段判定
	progress := 0.0
	if state.MaxWaves > 0 {
		progress = float64(state.Wave) / float64(state.MaxWaves)
	}
	switch {
	case progress < 0.33:
		s.phase = 0
	case progress < 0.66:
		s.phase = 1
	default:
		s.phase = 2
	}

	// 能力分配（有偏好时）
	if s.abilityPref != "" {
		actions = append(actions, s.assignAbilities(state)...)
	}

	if s.upgradeFirst {
		// 优先升级
		actions = append(actions, s.decideUpgrade(state)...)
		if s.phase < 2 {
			actions = append(actions, s.decideBuild(state)...)
		}
	} else {
		// 默认：按阶段决策
		switch s.phase {
		case 0:
			actions = append(actions, s.decideBuild(state)...)
		case 1:
			actions = append(actions, s.decideUpgrade(state)...)
			if state.Tick-s.lastBuild > 300 {
				actions = append(actions, s.decideBuild(state)...)
			}
		case 2:
			actions = append(actions, s.decideUpgrade(state)...)
		}
	}

	return actions
}

func (s *BalanceGreedyStrategy) decideBuild(state *GameState) []Action {
	if s.maxTowers > 0 && s.builtCount >= s.maxTowers {
		return nil
	}
	if len(state.BuildCells) == 0 || len(state.TowerDefs) == 0 {
		return nil
	}
	best := state.TowerDefs[0]
	if best.Cost > state.Gold {
		return nil
	}
	cell := state.BuildCells[0]
	s.lastBuild = state.Tick
	s.builtCount++
	return []Action{{Type: ActionBuild, TowerKey: best.Key, Cell: cell}}
}

func (s *BalanceGreedyStrategy) decideUpgrade(state *GameState) []Action {
	if len(state.Towers) == 0 || state.Gold < 10 {
		return nil
	}
	best := state.Towers[0]
	bestDmg := -1.0
	for _, t := range state.Towers {
		if t.Damage > bestDmg {
			bestDmg = t.Damage
			best = t
		}
	}
	return []Action{{Type: ActionUpgrade, Row: best.Row, Col: best.Col}}
}

// assignAbilities 为还没分配偏好能力的塔分配能力。
func (s *BalanceGreedyStrategy) assignAbilities(state *GameState) []Action {
	// 每种类别的代表能力
	abilityMap := map[string][]string{
		"attack": {"scatter", "wideBeam", "spinAoe", "bounce", "splash", "multiTarget"},
		"cc":     {"slowPower", "slowDuration", "stunChance", "stunDuration"},
		"damage": {"crit", "distanceDamage", "executionBonus", "flatDamage", "momentum"},
		"dot":    {"burn", "bleedDot", "poison", "weaken"},
		"buff":   {"damageUpAura", "attackSpeedAura", "rangeAura", "critAura", "soloBoost"},
		"zone":   {"poisonZone", "silenceZone", "curseZone", "weakenZone"},
	}

	abils, ok := abilityMap[s.abilityPref]
	if !ok {
		return nil
	}

	var actions []Action
	for _, t := range state.Towers {
		key := [2]int{t.Row, t.Col}
		assigned := s.abilityAssigned[key]
		if len(assigned) >= 6 {
			continue // 已满
		}
		// 分配该类别所有能力（最多6个）
		for i := len(assigned); i < len(abils) && i < 6; i++ {
			actions = append(actions, Action{
				Type: ActionAddAbility, Row: t.Row, Col: t.Col,
				AbilityName: abils[i],
			})
			assigned = append(assigned, abils[i])
		}
		s.abilityAssigned[key] = assigned
	}
	return actions
}

// ─── 平衡测试场景定义 ───

// BalanceScenario 平衡测试场景定义。
type BalanceScenario struct {
	ID          string
	MapID       string
	Difficulty  string
	Warden      string
	EnemyFilter string
	Strategy    Strategy
	Assertions  []Assertion
}

// AllBalanceScenarios 返回所有 26 个平衡测试场景。
func AllBalanceScenarios() []BalanceScenario {
	var scenarios []BalanceScenario
	scenarios = append(scenarios, balanceDifficultyScenarios()...)
	scenarios = append(scenarios, balanceEconomyScenarios()...)
	scenarios = append(scenarios, balanceAbilityScenarios()...)
	scenarios = append(scenarios, balanceEnemyScenarios()...)
	scenarios = append(scenarios, balanceBossScenarios()...)
	scenarios = append(scenarios, balancePowerScenarios()...)
	return scenarios
}

// ─── 维度 A: 难度梯度验证 ───

func balanceDifficultyScenarios() []BalanceScenario {
	return []BalanceScenario{
		{
			ID: "bal_easy_winnable", MapID: "map_01", Difficulty: "easy", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(),
			Assertions: []Assertion{
				{Name: "easy_victory", Type: "victory"},
				{Name: "easy_lives_gte_15", Type: "final_lives_gte", Param: 15},
				{Name: "easy_pace_ok", Type: "pace_not_boring"},
			},
		},
		{
			ID: "bal_normal_balanced", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(),
			Assertions: []Assertion{
				{Name: "normal_victory", Type: "victory"},
				{Name: "normal_lives_lte_20", Type: "final_lives_lte", Param: 20},
				{Name: "normal_dps_growth", Type: "dps_growth"},
			},
		},
		{
			ID: "bal_hard_challenging", MapID: "map_01", Difficulty: "hard", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(),
			Assertions: []Assertion{
				{Name: "hard_survive_15", Type: "waves_survived_gte", Param: 15},
				{Name: "hard_kills_gte_30", Type: "total_kills_gte", Param: 30},
			},
		},
		{
			ID: "bal_extreme_punishing", MapID: "map_01", Difficulty: "extreme", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(),
			Assertions: []Assertion{
				{Name: "extreme_not_easy_win", Type: "waves_survived_lte", Param: 20},
				{Name: "extreme_has_kills", Type: "total_kills_gte", Param: 10},
			},
		},
	}
}

// ─── 维度 B: 经济平衡 ───

func balanceEconomyScenarios() []BalanceScenario {
	return []BalanceScenario{
		{
			ID: "bal_econ_build_heavy", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(WithMaxTowers(6)),
			Assertions: []Assertion{
				{Name: "build_survive_10", Type: "waves_survived_gte", Param: 10},
				{Name: "build_no_stall", Type: "no_economy_stall", Param: 1},
			},
		},
		{
			ID: "bal_econ_upgrade_heavy", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(WithMaxTowers(2), WithUpgradeFirst(true)),
			Assertions: []Assertion{
				{Name: "upgrade_survive_10", Type: "waves_survived_gte", Param: 10},
				{Name: "upgrade_dps_growth", Type: "dps_growth"},
			},
		},
		{
			ID: "bal_econ_no_sell", MapID: "map_03", Difficulty: "normal", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(WithNoSell(true)),
			Assertions: []Assertion{
				{Name: "nosell_survive_8", Type: "waves_survived_gte", Param: 8},
				{Name: "nosell_gold_ok", Type: "gold_never_negative"},
			},
		},
		{
			ID: "bal_econ_sell_rebuy", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(WithSellRebuy(true)),
			Assertions: []Assertion{
				{Name: "sellrebuy_gold_ok", Type: "gold_never_negative"},
				{Name: "sellrebuy_has_kills", Type: "total_kills_gte", Param: 5},
			},
		},
	}
}

// ─── 维度 C: 单能力类型强弱 ───

func balanceAbilityScenarios() []BalanceScenario {
	return []BalanceScenario{
		{
			ID: "bal_abil_attack_only", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(WithAbilityPref("attack")),
			Assertions: []Assertion{
				{Name: "atkonly_survive_12", Type: "waves_survived_gte", Param: 12},
				{Name: "atkonly_kills_gte_40", Type: "total_kills_gte", Param: 40},
			},
		},
		{
			ID: "bal_abil_cc_only", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(WithAbilityPref("cc")),
			Assertions: []Assertion{
				{Name: "cconly_survive_10", Type: "waves_survived_gte", Param: 10},
				{Name: "cconly_slow_seen", Type: "enemy_slowed"},
			},
		},
		{
			ID: "bal_abil_damage_only", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(WithAbilityPref("damage")),
			Assertions: []Assertion{
				{Name: "dmgonly_survive_12", Type: "waves_survived_gte", Param: 12},
				{Name: "dmgonly_kills_gte_40", Type: "total_kills_gte", Param: 40},
			},
		},
		{
			ID: "bal_abil_dot_only", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(WithAbilityPref("dot")),
			Assertions: []Assertion{
				{Name: "dotonly_kills_gte_20", Type: "total_kills_gte", Param: 20},
				{Name: "dotonly_burn_seen", Type: "enemy_burning"},
			},
		},
		{
			ID: "bal_abil_buff_only", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(WithAbilityPref("buff")),
			Assertions: []Assertion{
				{Name: "buffonly_survive_10", Type: "waves_survived_gte", Param: 10},
				{Name: "buffonly_dps_growth", Type: "dps_growth"},
			},
		},
		{
			ID: "bal_abil_zone_only", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(WithAbilityPref("zone")),
			Assertions: []Assertion{
				{Name: "zoneonly_kills_gte_15", Type: "total_kills_gte", Param: 15},
				{Name: "zoneonly_has_effect", Type: "enemy_damaged"},
			},
		},
	}
}

// ─── 维度 D: 敌人能力有效性 ───

func balanceEnemyScenarios() []BalanceScenario {
	return []BalanceScenario{
		{
			ID: "bal_enemy_tank", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			EnemyFilter: "tank",
			Strategy:    NewBalanceGreedyStrategy(),
			Assertions: []Assertion{
				{Name: "tank_has_cap", Type: "enemy_has_cap"},
				{Name: "tank_survive_5", Type: "waves_survived_gte", Param: 5},
			},
		},
		{
			ID: "bal_enemy_armor", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			EnemyFilter: "armored",
			Strategy:    NewBalanceGreedyStrategy(),
			Assertions: []Assertion{
				{Name: "armor_has_armor", Type: "enemy_has_armor"},
				{Name: "armor_still_killable", Type: "total_kills_gte", Param: 10},
			},
		},
		{
			ID: "bal_enemy_evasion", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			EnemyFilter: "phantom",
			Strategy:    NewBalanceGreedyStrategy(),
			Assertions: []Assertion{
				{Name: "evasion_has_evasion", Type: "enemy_has_evasion"},
				{Name: "evasion_still_killable", Type: "total_kills_gte", Param: 10},
			},
		},
		{
			ID: "bal_enemy_healer", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			EnemyFilter: "healer",
			Strategy:    NewBalanceGreedyStrategy(),
			Assertions: []Assertion{
				{Name: "healer_has_aura", Type: "enemy_has_heal_aura"},
				{Name: "healer_survive_5", Type: "waves_survived_gte", Param: 5},
			},
		},
		{
			ID: "bal_enemy_splitter", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			EnemyFilter: "splitter",
			Strategy:    NewBalanceGreedyStrategy(),
			Assertions: []Assertion{
				{Name: "split_count_inc", Type: "enemy_count_increased"},
				{Name: "split_survive_5", Type: "waves_survived_gte", Param: 5},
			},
		},
		{
			ID: "bal_enemy_stealth", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			EnemyFilter: "phantom",
			Strategy:    NewBalanceGreedyStrategy(),
			Assertions: []Assertion{
				{Name: "stealth_evasion_active", Type: "enemy_has_evasion"},
				{Name: "stealth_survive_5", Type: "waves_survived_gte", Param: 5},
			},
		},
	}
}

// ─── 维度 E: Boss 生存时间 ───

func balanceBossScenarios() []BalanceScenario {
	return []BalanceScenario{
		{
			ID: "bal_boss_easy", MapID: "map_01", Difficulty: "easy", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(WithMaxTowers(4)),
			Assertions: []Assertion{
				{Name: "boss_easy_alive_5", Type: "boss_alive_gte", Param: 5},
				{Name: "boss_easy_alive_lte_30", Type: "boss_alive_lte", Param: 30},
			},
		},
		{
			ID: "bal_boss_normal", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(WithMaxTowers(3)),
			Assertions: []Assertion{
				{Name: "boss_normal_alive_5", Type: "boss_alive_gte", Param: 5},
				{Name: "boss_normal_alive_lte_60", Type: "boss_alive_lte", Param: 60},
			},
		},
		{
			ID: "bal_boss_extreme", MapID: "map_01", Difficulty: "extreme", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(WithMaxTowers(2)),
			Assertions: []Assertion{
				{Name: "boss_extreme_alive_10", Type: "boss_alive_gte", Param: 10},
			},
		},
	}
}

// ─── 维度 F: 强弱对比 ───

func balancePowerScenarios() []BalanceScenario {
	return []BalanceScenario{
		{
			ID: "bal_tower_overpower", MapID: "map_01", Difficulty: "easy", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(WithMaxTowers(6), WithUpgradeFirst(true)),
			Assertions: []Assertion{
				{Name: "overpower_victory", Type: "victory"},
				{Name: "overpower_low_leak", Type: "leak_rate_lte", Expected: 0.1},
			},
		},
		{
			ID: "bal_tower_underpower", MapID: "map_01", Difficulty: "extreme", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(WithMaxTowers(1)),
			Assertions: []Assertion{
				{Name: "underpower_limited", Type: "waves_survived_lte", Param: 8},
				{Name: "underpower_has_kills", Type: "total_kills_gte", Param: 3},
			},
		},
		{
			ID: "bal_warden_solo", MapID: "map_01", Difficulty: "normal", Warden: "prince",
			Strategy: NewBalanceGreedyStrategy(WithNoTowers(true)),
			Assertions: []Assertion{
				{Name: "solo_survive_3", Type: "waves_survived_gte", Param: 3},
				{Name: "solo_has_kills", Type: "total_kills_gte", Param: 1},
			},
		},
	}
}

// AllBalanceScenariosMap 返回平衡场景名称→策略映射（供 AllScenariosMap 合并）。
func AllBalanceScenariosMap() map[string]func() Strategy {
	m := make(map[string]func() Strategy)
	for _, bs := range AllBalanceScenarios() {
		name := bs.ID
		strat := bs.Strategy
		m[name] = func() Strategy { return strat }
	}
	return m
}

// BalanceAssertionsMap 返回平衡场景名称→断言列表映射。
func BalanceAssertionsMap() map[string][]Assertion {
	m := make(map[string][]Assertion)
	for _, bs := range AllBalanceScenarios() {
		m[bs.ID] = bs.Assertions
	}
	return m
}
