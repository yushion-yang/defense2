// telemetry.go — 全局遥测计数器。
// 各游戏子系统在运行时写入，autoplay 系统读取以追踪代码路径覆盖率。
// 零依赖设计：不导入任何游戏包，仅用基础类型，避免循环依赖。
package telemetry

import "sync"

// T 全局遥测实例。
var T = New()

// Telemetry 遥测数据收集器。
type Telemetry struct {
	mu sync.Mutex

	// 伤害管线步骤（8 步）
	PipelineSteps map[string]int // "immunity_check", "boss_hp_cap", "attacker_buff", "target_debuff", "damage_cap", "shield_absorb", "hp_deduct", "death"

	// 伤害类型
	DamageTypes map[string]int // "physical", "magic", "true", "pure"

	// Buff 类型（已应用的）
	BuffTypesApplied map[string]int // "slow", "stun", "root", "shield", "dot", etc.

	// Buff 堆叠模式（已使用的）
	BuffStackModes map[string]int // "strongest", "additive", "multiplicative", "override", "independent", "independentPerSource"

	// 敌人 Buff 模板（已应用的）
	EnemyBuffTemplates map[string]int // "berserk", "regen", "healAura", etc.

	// Boss 模板/类型
	BossSpawned map[string]int // archetype key of boss enemies

	// 交互模式（已进入的）
	InteractionModes map[string]int // "idle", "buildMenu", "buildPlace", etc.

	// CC 类型（已施加的）
	CCApplied map[string]int // "slow", "stun", "root", "silence", "knockup"
}

// New 创建空的遥测实例。
func New() *Telemetry {
	return &Telemetry{
		PipelineSteps:      make(map[string]int),
		DamageTypes:        make(map[string]int),
		BuffTypesApplied:   make(map[string]int),
		BuffStackModes:     make(map[string]int),
		EnemyBuffTemplates: make(map[string]int),
		BossSpawned:        make(map[string]int),
		InteractionModes:   make(map[string]int),
		CCApplied:          make(map[string]int),
	}
}

// Reset 清除所有遥测数据（每局开始时调用）。
func (t *Telemetry) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.PipelineSteps = make(map[string]int)
	t.DamageTypes = make(map[string]int)
	t.BuffTypesApplied = make(map[string]int)
	t.BuffStackModes = make(map[string]int)
	t.EnemyBuffTemplates = make(map[string]int)
	t.BossSpawned = make(map[string]int)
	t.InteractionModes = make(map[string]int)
	t.CCApplied = make(map[string]int)
}

// Record 记录一个维度的一次触发。
func (t *Telemetry) Record(dimension string, key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	switch dimension {
	case "pipeline":
		t.PipelineSteps[key]++
	case "damage_type":
		t.DamageTypes[key]++
	case "buff_type":
		t.BuffTypesApplied[key]++
	case "buff_stack_mode":
		t.BuffStackModes[key]++
	case "enemy_template":
		t.EnemyBuffTemplates[key]++
	case "boss":
		t.BossSpawned[key]++
	case "imode":
		t.InteractionModes[key]++
	case "cc":
		t.CCApplied[key]++
	}
}

// Snapshot 返回当前遥测数据的只读副本。
func (t *Telemetry) Snapshot() TelemetrySnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	return TelemetrySnapshot{
		PipelineSteps:      copyMap(t.PipelineSteps),
		DamageTypes:        copyMap(t.DamageTypes),
		BuffTypesApplied:   copyMap(t.BuffTypesApplied),
		BuffStackModes:     copyMap(t.BuffStackModes),
		EnemyBuffTemplates: copyMap(t.EnemyBuffTemplates),
		BossSpawned:        copyMap(t.BossSpawned),
		InteractionModes:   copyMap(t.InteractionModes),
		CCApplied:          copyMap(t.CCApplied),
	}
}

// TelemetrySnapshot 遥测数据只读快照。
type TelemetrySnapshot struct {
	PipelineSteps      map[string]int `json:"pipeline_steps"`
	DamageTypes        map[string]int `json:"damage_types"`
	BuffTypesApplied   map[string]int `json:"buff_types_applied"`
	BuffStackModes     map[string]int `json:"buff_stack_modes"`
	EnemyBuffTemplates map[string]int `json:"enemy_buff_templates"`
	BossSpawned        map[string]int `json:"boss_spawned"`
	InteractionModes   map[string]int `json:"interaction_modes"`
	CCApplied          map[string]int `json:"cc_applied"`
}

// Keys 返回某个 map 的所有键。
func Keys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func copyMap(src map[string]int) map[string]int {
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
