// hook.go — 模式钩子系统（运行时扩展点）。
//
// Hook 接口允许模式在不修改 UniversalMode 核心逻辑的前提下，
// 注入自定义的运行时行为（倒计时、Boss 计数等）。
//
// 当前实现：
//   - CountdownHook: 倒计时，时间到且存活=胜利（限时模式）
//   - BossCounterHook: Boss 击杀计数，达标=胜利（Boss 竞速模式）
//
// Hook 的生命周期：
//   Init → (每帧 Tick + OnEnemyKilled) → CheckVictory → HUDExtra/EndExtra
package gamemode

import "math"

// Hook 模式运行时钩子接口。
// 每个 Hook 维护自己的状态，由 UniversalMode 在合适的时机调用。
type Hook interface {
	// Init 初始化钩子状态（游戏开始时调用一次）。
	Init(cfg HookConfig, ctx *Context)

	// Tick 每帧更新（dt=帧间隔秒数）。
	Tick(dt float64, ctx *Context)

	// OnEnemyKilled 敌人被击杀时回调。
	OnEnemyKilled(boss bool, ctx *Context)

	// CheckVictory 检查此钩子是否触发胜利条件。
	// 返回 (result, handled)：handled=true 表示此钩子接管了胜利判定。
	CheckVictory(ctx *Context) (result bool, handled bool)

	// HUDExtra 返回此钩子需要在 HUD 上显示的额外信息。
	HUDExtra(ctx *Context) HUDConfig

	// EndExtra 返回此钩子在结算屏幕上的额外数据。
	EndExtra(ctx *Context) map[string]any
}

// ── CountdownHook ─────────────────────────────────────

// CountdownHook 倒计时钩子。
// 维护 remainingTime，每帧递减。时间到且生命>0 = 胜利。
type CountdownHook struct {
	targetSeconds float64 // 目标存活时间
	remainingTime float64 // 剩余时间（运行时递减）
}

func (h *CountdownHook) Init(cfg HookConfig, _ *Context) {
	h.targetSeconds = cfg.TargetSeconds
	if h.targetSeconds <= 0 {
		h.targetSeconds = 300 // 默认 5 分钟
	}
	h.remainingTime = h.targetSeconds
}

func (h *CountdownHook) Tick(dt float64, _ *Context) {
	h.remainingTime -= dt
}

func (h *CountdownHook) OnEnemyKilled(_ bool, _ *Context) {}

// CheckVictory 倒计时归零且玩家仍存活=胜利。
func (h *CountdownHook) CheckVictory(ctx *Context) (bool, bool) {
	if h.remainingTime <= 0 && ctx.Lives > 0 {
		return true, true
	}
	// 时间未到，不接管判定（让其他逻辑继续）
	return false, false
}

// HUDExtra 在 HUD 上显示倒计时，钳制最小值为 0。
func (h *CountdownHook) HUDExtra(_ *Context) HUDConfig {
	return HUDConfig{
		ShowTimer:    true,
		TimerSeconds: math.Max(0, h.remainingTime),
	}
}

// EndExtra 返回实际存活时长和目标时长。
func (h *CountdownHook) EndExtra(_ *Context) map[string]any {
	survived := h.targetSeconds - math.Max(0, h.remainingTime)
	return map[string]any{
		"survivedSeconds": math.Round(survived),
		"targetSeconds":   h.targetSeconds,
	}
}

// ── BossCounterHook ───────────────────────────────────

// BossCounterHook Boss 击杀计数钩子。
// 维护 bossesKilled 计数器，达到 totalBosses 时触发胜利。
// 分数包含时间衰减：timeBase - elapsedTime * timeDecay。
type BossCounterHook struct {
	totalBosses  int     // 需击败的 Boss 总数
	bossesKilled int     // 已击败计数
	timeBase     int     // 时间分基数（如 10000）
	timeDecay    float64 // 每秒扣分（如 10）
}

func (h *BossCounterHook) Init(cfg HookConfig, ctx *Context) {
	h.totalBosses = cfg.TotalBosses
	if h.totalBosses <= 0 {
		h.totalBosses = 5
	}
	h.bossesKilled = 0
	h.timeBase = cfg.TimeBase
	h.timeDecay = cfg.TimeDecay
	// Boss 竞速：将最大波次设为 Boss 总数
	if ctx.SetMaxWaves != nil {
		ctx.SetMaxWaves(h.totalBosses)
	}
}

func (h *BossCounterHook) Tick(_ float64, _ *Context) {}

// OnEnemyKilled 只统计 Boss 击杀。
func (h *BossCounterHook) OnEnemyKilled(boss bool, _ *Context) {
	if boss {
		h.bossesKilled++
	}
}

// CheckVictory 击败全部 Boss 即胜利。
func (h *BossCounterHook) CheckVictory(_ *Context) (bool, bool) {
	if h.bossesKilled >= h.totalBosses {
		return true, true
	}
	return false, false
}

// HUDExtra 显示 Boss 击杀进度。
func (h *BossCounterHook) HUDExtra(_ *Context) HUDConfig {
	return HUDConfig{
		ShowBossCount: true,
		BossesKilled:  h.bossesKilled,
		TotalBosses:   h.totalBosses,
	}
}

// EndExtra 返回 Boss 击杀统计和时间分。
func (h *BossCounterHook) EndExtra(ctx *Context) map[string]any {
	extra := map[string]any{
		"bossesKilled": h.bossesKilled,
		"totalBosses":  h.totalBosses,
		"totalTime":    math.Round(ctx.ElapsedTime),
	}
	return extra
}

// TimeScore 返回时间衰减分数：timeBase - elapsed * timeDecay，最低为 0。
// 由 UniversalMode.GetScore 在计算总分时调用。
func (h *BossCounterHook) TimeScore(elapsed float64) int {
	if h.timeBase <= 0 {
		return 0
	}
	s := float64(h.timeBase) - elapsed*h.timeDecay
	return int(math.Max(0, s))
}

// ── 工厂 ──────────────────────────────────────────────

// NewHook 根据 HookConfig.Type 创建对应的 Hook 实例。
// 未知类型返回 nil。
func NewHook(cfg HookConfig) Hook {
	switch cfg.Type {
	case "countdown":
		return &CountdownHook{}
	case "bossCounter":
		return &BossCounterHook{}
	default:
		return nil
	}
}
