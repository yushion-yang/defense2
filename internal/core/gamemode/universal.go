// universal.go — 配置驱动的通用游戏模式。
//
// UniversalMode 实现完整的 Mode 接口（18 个方法），所有行为参数
// 从 ModeConfig（gamemodes.json）读取，运行时扩展通过 Hook 接口注入。
//
// 替代关系：一个 UniversalMode + 配置 = 原来的 10 个独立模式文件。
//
// 胜利条件路由（cfg.Victory）：
//   "allWaves"       → 标准公式：wave >= maxWaves && !spawning
//   "never"          → 永远不胜利（无尽模式）
//   "hook:<type>"    → 委托给对应 Hook 的 CheckVictory
//
// 失败条件路由（cfg.Defeat）：
//   "livesZero"      → 生命归零
//   "never"          → 永远不失败（测试/自动对局）
package gamemode

import (
	"defense2/internal/config"
	"defense2/internal/i18n"
)

// UniversalMode 配置驱动的通用游戏模式。
type UniversalMode struct {
	id      string
	cfg     ModeConfig
	hooks   []Hook
	ruleset ConfigRuleset
}

// NewUniversalMode 从模式 ID 和配置创建通用模式实例。
// 解析 Hook 配置并创建对应的 Hook 实例。
func NewUniversalMode(id string, cfg ModeConfig) *UniversalMode {
	m := &UniversalMode{
		id:      id,
		cfg:     cfg,
		ruleset: NewConfigRuleset(cfg.Ruleset),
	}
	for _, hc := range cfg.Hooks {
		if h := NewHook(hc); h != nil {
			m.hooks = append(m.hooks, h)
		}
	}
	return m
}

// ── Mode 接口实现 ─────────────────────────────────────

func (m *UniversalMode) ID() string          { return m.id }
func (m *UniversalMode) ShouldAutoStart() bool { return m.cfg.AutoStart }
func (m *UniversalMode) EnableEvents() bool    { return m.cfg.EnableEvents }
func (m *UniversalMode) Ruleset() TowerRuleset { return m.ruleset }

// IntermissionSecs 波间休息秒数。
// 配置值为 0 时回退到 spawner 全局默认值（与 baseMode 行为一致）。
func (m *UniversalMode) IntermissionSecs() float64 {
	if m.cfg.Intermission > 0 {
		return m.cfg.Intermission
	}
	if wi := config.GlobalSpawnerConfig().Timing.WaveInterval; wi > 0 {
		return wi
	}
	return 10
}

// VictoryWaveTarget 返回目标波数。
// 有 initMaxWaves 配置且不是"永不胜利"时返回该值，否则返回 -1。
func (m *UniversalMode) VictoryWaveTarget() int {
	// Boss 竞速模式：从 hook 获取 totalBosses
	for _, h := range m.hooks {
		if bc, ok := h.(*BossCounterHook); ok {
			return bc.totalBosses
		}
	}
	return -1
}

// OnInit 模式初始化。
// 设置自定义波次上限（如果配置了 initMaxWaves），然后初始化所有 Hook。
func (m *UniversalMode) OnInit(ctx *Context) {
	if m.cfg.InitMaxWaves > 0 && ctx.SetMaxWaves != nil {
		ctx.SetMaxWaves(m.cfg.InitMaxWaves)
	}
	// 初始化 Hook（传入对应的 HookConfig）
	for i, h := range m.hooks {
		if i < len(m.cfg.Hooks) {
			h.Init(m.cfg.Hooks[i], ctx)
		}
	}
}

// OnGameStart 游戏正式开始（首波出怪前）。
func (m *UniversalMode) OnGameStart(_ *Context) {}

// OnTick 每帧更新，委托给所有 Hook。
func (m *UniversalMode) OnTick(dt float64, ctx *Context) {
	for _, h := range m.hooks {
		h.Tick(dt, ctx)
	}
}

// OnWaveStart 波次开始。
func (m *UniversalMode) OnWaveStart(_ int, _ *Context) {}

// OnWaveCleared 波次结束，发放经济奖励。
// 读取 econID 对应的经济配置，根据 perfectBonus 标志决定是否发放完美奖励。
func (m *UniversalMode) OnWaveCleared(wave int, _ *Context) WaveClearResult {
	econ := modeEcon(m.cfg.EconID)
	bonus := econ.WaveBonus.Calc(wave)
	result := WaveClearResult{
		BonusGold: bonus,
		Message:   i18n.TF("mode.wave_clear", wave, bonus),
	}
	if m.cfg.PerfectBonus {
		result.PerfectBonus = econ.PerfectBonus.Calc(wave)
	}
	return result
}

// OnEnemyKilled 敌人被击杀，委托给所有 Hook。
func (m *UniversalMode) OnEnemyKilled(boss bool, ctx *Context) {
	for _, h := range m.hooks {
		h.OnEnemyKilled(boss, ctx)
	}
}

// OnEnemyLeaked 敌人到达终点。
func (m *UniversalMode) OnEnemyLeaked(_ *Context) {}

// CheckVictory 根据配置路由胜利判定逻辑。
func (m *UniversalMode) CheckVictory(ctx *Context) bool {
	switch m.cfg.Victory {
	case "never":
		return false
	case "allWaves":
		return m.checkAllWavesVictory(ctx)
	default:
		// "hook:*" 类型：委托给匹配的 Hook
		return m.checkHookVictory(ctx)
	}
}

// checkAllWavesVictory 标准胜利公式：波次打完且无存活敌人。
// 对 maxWaves<=0 的情况特殊处理（test/autoplay 依赖地图波次）。
func (m *UniversalMode) checkAllWavesVictory(ctx *Context) bool {
	if ctx.MaxWaves <= 0 {
		return false
	}
	return ctx.Wave >= ctx.MaxWaves && !ctx.Spawning
}

// checkHookVictory 轮询所有 Hook，任一 handled && result=true 即胜利。
func (m *UniversalMode) checkHookVictory(ctx *Context) bool {
	for _, h := range m.hooks {
		result, handled := h.CheckVictory(ctx)
		if handled {
			return result
		}
	}
	return false
}

// CheckDefeat 根据配置路由失败判定逻辑。
func (m *UniversalMode) CheckDefeat(ctx *Context) bool {
	switch m.cfg.Defeat {
	case "never":
		return false
	case "livesZero":
		return ctx.Lives <= 0
	default:
		return ctx.Lives <= 0
	}
}

// GetScore 线性组合分数公式。
// 公式: wave*ScoreWave + kill*ScoreKill + leaked*ScoreLeak + lives*ScoreLives。
// Hook 可附加额外分数（如 BossCounterHook 的时间衰减分）。
func (m *UniversalMode) GetScore(ctx *Context) int {
	sc := m.cfg.Score
	wavesCleared := ctx.Wave
	if wavesCleared > ctx.MaxWaves && ctx.MaxWaves > 0 {
		wavesCleared = ctx.MaxWaves
	}
	score := wavesCleared*sc.Wave + ctx.Kills*sc.Kill + ctx.Leaked*sc.Leaked + ctx.Lives*sc.Lives
	// Hook 附加分数（如 BossRush 的时间分）
	for _, h := range m.hooks {
		if bc, ok := h.(*BossCounterHook); ok {
			score += bc.TimeScore(ctx.ElapsedTime)
		}
	}
	return score
}

// GetHUDConfig 合并基础 HUD 配置与所有 Hook 的 HUDExtra。
func (m *UniversalMode) GetHUDConfig(ctx *Context) HUDConfig {
	hud := HUDConfig{}
	for _, h := range m.hooks {
		extra := h.HUDExtra(ctx)
		mergeHUD(&hud, &extra)
	}
	return hud
}

// mergeHUD 将 src 的非零值合并到 dst。
func mergeHUD(dst, src *HUDConfig) {
	if src.ShowTimer {
		dst.ShowTimer = true
		dst.TimerSeconds = src.TimerSeconds
	}
	if src.ShowBossCount {
		dst.ShowBossCount = true
		dst.BossesKilled = src.BossesKilled
		dst.TotalBosses = src.TotalBosses
	}
	if src.ShowRules {
		dst.ShowRules = true
		dst.RulesText = src.RulesText
	}
}

// GetEndData 构建结算屏幕数据。
// 从 endFields 配置决定 Extra 中包含哪些字段，然后合并 Hook 的 EndExtra。
func (m *UniversalMode) GetEndData(ctx *Context) EndData {
	extra := m.buildEndExtra(ctx)
	// 合并 Hook 的额外数据
	for _, h := range m.hooks {
		for k, v := range h.EndExtra(ctx) {
			extra[k] = v
		}
	}
	return EndData{
		ModeName: i18n.T(m.cfg.UI.NameKey),
		Score:    m.GetScore(ctx),
		Extra:    extra,
	}
}

// buildEndExtra 根据 endFields 配置从 Context 提取对应字段值。
func (m *UniversalMode) buildEndExtra(ctx *Context) map[string]any {
	extra := make(map[string]any, len(m.cfg.EndFields))
	for _, field := range m.cfg.EndFields {
		switch field {
		case "waves":
			extra["waves"] = ctx.Wave
		case "maxWaves":
			extra["maxWaves"] = ctx.MaxWaves
		case "kills":
			extra["kills"] = ctx.Kills
		case "leaked":
			extra["leaked"] = ctx.Leaked
		case "goldEarned":
			extra["goldEarned"] = ctx.Gold
		case "lives", "livesRemaining":
			extra[field] = ctx.Lives
		case "wavesReached":
			extra["wavesReached"] = ctx.Wave
		}
		// survivedSeconds/targetSeconds/bossesKilled/totalBosses/totalTime
		// 由 Hook.EndExtra 提供，此处不处理
	}
	return extra
}
