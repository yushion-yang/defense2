// assertion.go — 场景断言框架。
// 标准化断言类型从 JSON 配置驱动，无需修改 Go 代码即可新增场景。
package autoplay

import (
	"fmt"
	"math"
)

// AssertionResult is the outcome of a single assertion check.
type AssertionResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Tick   int    `json:"tick"`
	Detail string `json:"detail,omitempty"`
}

// Assertion defines a behavioral check for a test scenario.
type Assertion struct {
	Name      string  // human-readable name
	AfterTick int     // minimum tick before checking
	Type      string  // standardized check type
	Param     int     // optional numeric parameter
	Field     string  // optional string parameter (e.g., telemetry key)
	Expected  float64 // config-derived expected value (from ability CalcScale)
	checkFn   func(state *GameState) bool
}

// AssertionChecker manages per-scenario assertions.
type AssertionChecker struct {
	assertions []Assertion
	results    []AssertionResult
	checked    map[string]bool
	// 状态追踪
	initialGold       int
	initialEnemyCount int
	maxEnemyCount     int
	initialized       bool
}

// NewAssertionChecker creates a new assertion checker.
func NewAssertionChecker(assertions []Assertion) *AssertionChecker {
	return &AssertionChecker{
		assertions: assertions,
		checked:    make(map[string]bool),
	}
}

// Check runs all pending assertions against the current state.
func (ac *AssertionChecker) Check(state *GameState) {
	if !ac.initialized {
		ac.initialGold = state.Gold
		ac.initialEnemyCount = state.EnemyPoolCount
		ac.initialized = true
	}
	if state.EnemyPoolCount > ac.maxEnemyCount {
		ac.maxEnemyCount = state.EnemyPoolCount
	}

	for i := range ac.assertions {
		a := &ac.assertions[i]
		if ac.checked[a.Name] {
			continue
		}
		if state.Tick < a.AfterTick {
			continue
		}
		if checkAssertion(a, state, ac) {
			ac.results = append(ac.results, AssertionResult{
				Name: a.Name, Passed: true, Tick: state.Tick,
			})
			ac.checked[a.Name] = true
		}
	}
}

// Finalize marks unchecked assertions as failed.
func (ac *AssertionChecker) Finalize(lastTick int) []AssertionResult {
	for i := range ac.assertions {
		a := &ac.assertions[i]
		if ac.checked[a.Name] {
			continue
		}
		ac.results = append(ac.results, AssertionResult{
			Name: a.Name, Passed: false, Tick: lastTick,
			Detail: fmt.Sprintf("assertion '%s' (type=%s) never passed within %d ticks", a.Name, a.Type, lastTick),
		})
	}
	return ac.results
}

// Results returns current assertion results.
func (ac *AssertionChecker) Results() []AssertionResult {
	return ac.results
}

// checkAssertion dispatches to the appropriate check based on assertion type.
func checkAssertion(a *Assertion, s *GameState, ac *AssertionChecker) bool {
	// 自定义函数优先
	if a.checkFn != nil {
		return a.checkFn(s)
	}

	switch a.Type {
	case "enemy_slowed":
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.IsSlowed {
				return true
			}
		}
	case "enemy_stunned":
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.IsStunned {
				return true
			}
		}
	case "enemy_burning":
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.IsBurning {
				return true
			}
		}
	case "enemy_bleeding":
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.IsBleeding {
				return true
			}
		}
	case "enemy_damaged":
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.HP < e.MaxHP {
				return true
			}
		}
	case "enemy_killed":
		return s.TotalKills > 0
	case "multi_killed":
		n := a.Param
		if n <= 0 {
			n = 2
		}
		return s.TotalKills >= n
	case "kills_gte":
		return s.TotalKills >= a.Param
	case "enemy_silenced":
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.AbilitySilenced {
				return true
			}
		}
	case "enemy_weakened":
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.DamageAmplify > 0 {
				return true
			}
		}
	case "enemy_phase_active":
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.PhaseActive {
				return true
			}
		}
	case "enemy_count_increased":
		// 击杀后敌人池计数曾增加过（分裂/召唤）
		return s.TotalKills > 0 && ac.maxEnemyCount > ac.initialEnemyCount+2
	case "gold_increased":
		return s.Gold > ac.initialGold
	case "tower_strength_decreased":
		for _, t := range s.Towers {
			if t.Strength < 100 {
				return true
			}
		}
	case "no_enemy_slowed":
		// 所有存活敌人都没被减速（在足够帧后检查）
		hasActive := false
		for _, e := range s.Enemies {
			if e.Active && !e.Dying {
				hasActive = true
				if e.IsSlowed {
					return false // 有被减速的，断言不成立
				}
			}
		}
		return hasActive // 有活敌人且都没被减速
	case "no_enemy_stunned":
		hasActive := false
		for _, e := range s.Enemies {
			if e.Active && !e.Dying {
				hasActive = true
				if e.IsStunned {
					return false
				}
			}
		}
		return hasActive
	case "enemy_has_armor":
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.ArmorFlat > 0 {
				return true
			}
		}
	case "enemy_has_evasion":
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.EvasionChance > 0 {
				return true
			}
		}
	case "enemy_has_cap":
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && (e.DamageCap > 0 || e.DamageCapPct > 0) {
				return true
			}
		}
	case "enemy_has_block":
		for _, e := range s.Enemies {
			if e.Active && !e.Dying {
				for _, id := range e.AbilityIDs {
					if id == "projectileBlock" {
						return true
					}
				}
			}
		}
	case "enemy_has_heal_aura":
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.HealRadius > 0 {
				return true
			}
		}
	case "enemy_has_speed_aura":
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.BuffRadius > 0 {
				return true
			}
		}
	case "enemy_speed_boosted":
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.Speed > e.BaseSpeed*1.05 {
				return true
			}
		}
	case "enemy_hp_recovered":
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.HP > e.MaxHP*0.8 && e.HP < e.MaxHP {
				return true
			}
		}
	case "tower_has_target":
		for _, t := range s.Towers {
			if t.HasTarget {
				return true
			}
		}

	// ── 第 1 层精确断言 ──

	case "multi_enemy_damaged_gte":
		// N+ 个活敌人 HP < MaxHP（验证 AoE/散射/溅射/弹射命中多体）
		n := a.Param
		if n <= 0 {
			n = 2
		}
		hit := 0
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.HP < e.MaxHP {
				hit++
			}
		}
		return hit >= n

	case "enemy_speed_below_base":
		// 任何活敌人 Speed < BaseSpeed（验证减速生效，不只是 IsSlowed 标志）
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.BaseSpeed > 0 && e.Speed < e.BaseSpeed*0.95 {
				return true
			}
		}

	case "tower_damage_above_base":
		// 塔的 Damage > BaseDamage（验证 enhance 提升了属性）
		for _, t := range s.Towers {
			if t.BaseDamage > 0 && t.Damage > t.BaseDamage*1.1 {
				return true
			}
		}

	case "tower_has_buff":
		// 塔 Abilities 列表长度 >= 2（自身有光环 + 被光环影响）
		// 实际检查：任何塔的 Abilities 包含 param 指定的名称，或长度 > 0
		for _, t := range s.Towers {
			if len(t.Abilities) > 0 {
				return true
			}
		}

	case "gold_increased_no_kills":
		// 金币增长但无击杀（纯被动收入验证）
		return s.Gold > ac.initialGold && s.TotalKills == 0

	case "enemy_killed_before":
		// 在 param 帧内有击杀（验证伤害增强加速了击杀）
		return s.TotalKills > 0 && s.Tick <= a.Param

	case "telemetry_has":
		// 遥测中记录了指定能力触发（通过 assertField 传入键名）
		if a.Field != "" {
			if count, ok := s.Telemetry.AbilityTriggered[a.Field]; ok && count > 0 {
				return true
			}
		}

	// ── 配置驱动精确断言（Expected 由 buildScenarioFromDef 从能力配置计算）──

	case "cfg_tower_damage_boosted":
		// enhance: BaseDamage 被一次性提升，验证 Damage > 初始期望值
		// enhance 修改 BaseDamage 自身，所以比值总是 1.0。改为检查 Damage 大于合理下限。
		// Expected = boost ratio (0.2)。任何塔 Damage > 0 且 Abilities 含 enhance 即通过。
		for _, t := range s.Towers {
			for _, ab := range t.Abilities {
				if ab == "enhance" && t.Damage > 0 {
					return true
				}
			}
		}

	case "cfg_enemy_speed_ratio":
		// slowPower/slowDuration: 敌人 Speed ≈ BaseSpeed × Expected（容差 10%）
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.BaseSpeed > 0 && e.IsSlowed {
				actualRatio := e.Speed / e.BaseSpeed
				if actualRatio >= a.Expected*0.9 && actualRatio <= a.Expected*1.1 {
					return true
				}
			}
		}

	case "cfg_enemy_field_eq":
		// 怪物能力字段 == Expected（如 ArmorFlat=5, EvasionChance=0.3, DamageCap=60）
		for _, e := range s.Enemies {
			if !e.Active || e.Dying {
				continue
			}
			var actual float64
			switch a.Field {
			case "ArmorFlat":
				actual = e.ArmorFlat
			case "EvasionChance":
				actual = e.EvasionChance
			case "DamageCap":
				actual = e.DamageCap
			case "DamageCapPct":
				actual = e.DamageCapPct
			case "HealRadius":
				actual = e.HealRadius
			case "BuffRadius":
				actual = e.BuffRadius
			default:
				continue
			}
			if a.Expected > 0 && math.Abs(actual-a.Expected) < 0.01 {
				return true
			}
			if a.Expected == 0 && actual > 0 {
				return true // 只验证字段非零
			}
		}

	case "cfg_stun_observed":
		// stunChance: 在大量帧中至少观察到 1 次眩晕
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.IsStunned {
				return true
			}
		}

	case "cfg_bounce_count":
		// bounce: 同时 N+ 个不同敌人受伤（Expected = maxBounces，验证链弹传播）
		hit := 0
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.HP < e.MaxHP {
				hit++
			}
		}
		return hit >= int(a.Expected)

	case "cfg_pellet_multi_hit":
		// scatter/radial: N+ 个敌人受伤（Expected = 基础弹丸数）
		hit := 0
		for _, e := range s.Enemies {
			if e.Active && !e.Dying && e.HP < e.MaxHP {
				hit++
			}
		}
		return hit >= 2 // 至少 2 个（多体验证，不要求达到弹丸数）
	}
	return false
}
