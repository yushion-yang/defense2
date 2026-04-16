// mode.go — 游戏模式框架。
//
// 游戏模式系统是本游戏的核心扩展点之一。通过 Mode 接口定义了
// 十种游戏玩法，全部由 gamemodes.json 配置驱动。
//
// 设计决策：
//   - 配置驱动：所有模式通过 UniversalMode + JSON 配置实现，无需独立 Go 文件
//   - Context 注入：回调时传入快照+修改器，避免模式直接依赖 Stage
//   - 延迟加载注册表：首次访问时从 gamemodes.json 加载，兼容测试环境的 dataFS 初始化时序
package gamemode

import "sync"

// Mode 游戏模式接口。
// 每种模式实现自己的胜负判定、经济规则、HUD 提示和分数计算。
type Mode interface {
	// ID 返回模式标识（"casual", "classic", "coop", "test" 等）。
	ID() string

	// OnInit 模式初始化（Session 创建后调用，可通过 ctx 修改初始状态）。
	OnInit(ctx *Context)

	// OnGameStart 游戏正式开始（首波出怪前）。
	OnGameStart(ctx *Context)

	// OnTick 每帧更新（用于模式特有逻辑，如限时倒计时）。
	OnTick(dt float64, ctx *Context)

	// OnWaveStart 波次开始时回调。
	OnWaveStart(wave int, ctx *Context)

	// OnWaveCleared 波次结束时回调，返回奖励信息。
	OnWaveCleared(wave int, ctx *Context) WaveClearResult

	// ShouldAutoStart 是否自动开始下一波。
	ShouldAutoStart() bool

	// IntermissionSecs 波间休息秒数。
	IntermissionSecs() float64

	// CheckVictory 判断是否满足胜利条件。
	CheckVictory(ctx *Context) bool

	// CheckDefeat 判断是否满足失败条件。
	CheckDefeat(ctx *Context) bool

	// OnEnemyKilled 敌人被击杀（boss 标记 Boss 击杀）。
	OnEnemyKilled(boss bool, ctx *Context)

	// OnEnemyLeaked 敌人到达终点。
	OnEnemyLeaked(ctx *Context)

	// GetScore 计算当前分数。
	GetScore(ctx *Context) int

	// GetHUDConfig 返回模式专属 HUD 提示配置。
	GetHUDConfig(ctx *Context) HUDConfig

	// GetEndData 返回结算屏幕数据。
	GetEndData(ctx *Context) EndData

	// VictoryWaveTarget 返回胜利目标波数（-1 表示无目标，如无尽模式）。
	VictoryWaveTarget() int

	// EnableEvents 返回是否启用关卡事件系统（奖励波次弹出事件选择）。
	EnableEvents() bool

	// Ruleset 返回此模式的塔建造规则策略。
	// 控制能力解锁方式、强度上限、道具掉落等塔相关行为。
	Ruleset() TowerRuleset
}

// Context 模式回调时的游戏状态快照 + 状态修改器。
// 由 StageScene 每帧构建并传入，闭包函数可安全修改 Stage 状态。
type Context struct {
	Wave         int     // 当前波次
	MaxWaves     int     // 总波次数（地图定义）
	Lives        int     // 剩余生命
	Gold         int     // 当前金币
	Kills        int     // 累计击杀
	Leaked       int     // 累计泄漏
	TowersBuilt  int     // 放置塔数
	EnemiesAlive int     // 存活敌人数
	ElapsedTime  float64 // 游戏已用时间
	Spawning     bool    // spawner 是否还在出怪/还有存活敌人

	// 状态修改器（闭包，由 StageScene 注入）
	SetLives    func(int)
	SetGold     func(int)
	AddGold     func(int)
	SetMaxWaves func(int)
}

// WaveClearResult 波次通关奖励。
type WaveClearResult struct {
	BonusGold    int    // 通关金币奖励
	PerfectBonus int    // 完美波次额外奖励（零泄漏）
	Message      string // 通知文本
}

// HUDConfig 模式专属 HUD 提示。
type HUDConfig struct {
	ShowTimer     bool    // 是否显示计时器
	TimerSeconds  float64 // 剩余/已用秒数
	ShowBossCount bool    // 是否显示 Boss 计数
	BossesKilled  int     // 已击杀 Boss 数
	TotalBosses   int     // Boss 总数
	ShowRules     bool    // 是否显示挑战规则
	RulesText     string  // 规则描述文本
}

// EndData 结算屏幕数据。
type EndData struct {
	ModeName string         // 模式显示名称
	Score    int            // 最终分数
	Extra    map[string]any // 模式特有数据（如 Boss 击杀时间）
}

// ── 全局模式注册表 ───────────────────────────────────
// 延迟初始化：首次访问时从 gamemodes.json 加载配置并注册所有模式。
// 延迟初始化是必须的——测试环境中 config.SetDataFS() 在 init() 之后才被调用，
// 如果在 init() 中加载会因 dataFS=nil 而失败。

var (
	registry     = map[string]Mode{}
	registryOnce sync.Once
)

// ensureRegistry 确保注册表已初始化（懒加载）。
func ensureRegistry() {
	registryOnce.Do(func() {
		configs, err := LoadModeConfigs()
		if err != nil {
			// 配置加载失败时注册一个最小可用的 casual 模式
			registry["casual"] = &UniversalMode{id: "casual"}
			return
		}
		for id, cfg := range configs {
			registry[id] = NewUniversalMode(id, cfg)
		}
	})
}

// Register 注册一种游戏模式（仅用于测试覆盖）。
func Register(m Mode) {
	ensureRegistry()
	registry[m.ID()] = m
}

// Get 获取指定 ID 的模式，未找到返回 nil。
func Get(id string) Mode {
	ensureRegistry()
	return registry[id]
}

// GetOrDefault 获取指定 ID 的模式，未找到返回 casual。
// 三级回退保证始终返回有效模式：指定ID → casual 实例 → 临时 baseMode。
func GetOrDefault(id string) Mode {
	ensureRegistry()
	if m := registry[id]; m != nil {
		return m
	}
	if m := registry["casual"]; m != nil {
		return m
	}
	return &baseMode{id: "casual"}
}

// List 返回所有已注册模式的 ID。
func List() []string {
	ensureRegistry()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// ResetRegistry 重置注册表（仅用于测试）。
func ResetRegistry() {
	registryOnce = sync.Once{}
	registry = map[string]Mode{}
}
