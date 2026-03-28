// modes.go — 内置游戏模式实现。
// 包含战役、Boss Rush、无尽、限时防守、挑战 5 种模式。
// 通过 init() 自注册到全局模式注册表。
package gamemode

func init() {
	Register(&CampaignMode{})
	Register(&BossRushMode{})
	Register(&EndlessMode{})
	Register(&TimedDefenseMode{})
	Register(&ChallengeMode{})
}

// ─── 战役模式 ─────────────────────────────────────────

// CampaignMode 标准战役模式：通过全部波次即胜利。
type CampaignMode struct{}

func (m *CampaignMode) Name() string                        { return "campaign" }
func (m *CampaignMode) OnInit(_ *ModeContext)                {}
func (m *CampaignMode) OnWaveStart(_ int, _ *ModeContext)    {}
func (m *CampaignMode) OnWaveEnd(_ int, _ *ModeContext)      {}
func (m *CampaignMode) OnUpdate(_ float64, _ *ModeContext)   {}
func (m *CampaignMode) IsWon(ctx *ModeContext) bool          { return ctx.Wave >= ctx.MaxWaves }
func (m *CampaignMode) IsLost(ctx *ModeContext) bool         { return ctx.Lives <= 0 }

// ─── Boss Rush 模式 ───────────────────────────────────

// BossRushMode Boss 连战模式：每波只出 Boss，击败全部即胜利。
type BossRushMode struct{}

func (m *BossRushMode) Name() string                        { return "bossRush" }
func (m *BossRushMode) OnInit(_ *ModeContext)                {}
func (m *BossRushMode) OnWaveStart(_ int, _ *ModeContext)    {}
func (m *BossRushMode) OnWaveEnd(_ int, _ *ModeContext)      {}
func (m *BossRushMode) OnUpdate(_ float64, _ *ModeContext)   {}
func (m *BossRushMode) IsWon(ctx *ModeContext) bool          { return ctx.Wave >= ctx.MaxWaves }
func (m *BossRushMode) IsLost(ctx *ModeContext) bool         { return ctx.Lives <= 0 }

// ─── 无尽模式 ─────────────────────────────────────────

// EndlessMode 无尽模式：永不停波，看能撑多久。
type EndlessMode struct{}

func (m *EndlessMode) Name() string                        { return "endless" }
func (m *EndlessMode) OnInit(ctx *ModeContext)              { ctx.MaxWaves = 999 }
func (m *EndlessMode) OnWaveStart(_ int, _ *ModeContext)    {}
func (m *EndlessMode) OnWaveEnd(_ int, _ *ModeContext)      {}
func (m *EndlessMode) OnUpdate(_ float64, _ *ModeContext)   {}
func (m *EndlessMode) IsWon(_ *ModeContext) bool            { return false } // 无尽不可胜
func (m *EndlessMode) IsLost(ctx *ModeContext) bool         { return ctx.Lives <= 0 }

// ─── 限时防守模式 ─────────────────────────────────────

// TimedDefenseMode 限时防守模式：坚守指定时间即胜利。
type TimedDefenseMode struct {
	TimeLimit float64 // 时间限制（秒）
	Elapsed   float64 // 已用时间（秒）
}

func (m *TimedDefenseMode) Name() string { return "timedDefense" }
func (m *TimedDefenseMode) OnInit(_ *ModeContext) {
	m.TimeLimit = 300 // 5 分钟
	m.Elapsed = 0
}
func (m *TimedDefenseMode) OnWaveStart(_ int, _ *ModeContext) {}
func (m *TimedDefenseMode) OnWaveEnd(_ int, _ *ModeContext)   {}
func (m *TimedDefenseMode) OnUpdate(dt float64, _ *ModeContext) {
	m.Elapsed += dt
}
func (m *TimedDefenseMode) IsWon(_ *ModeContext) bool  { return m.Elapsed >= m.TimeLimit }
func (m *TimedDefenseMode) IsLost(ctx *ModeContext) bool { return ctx.Lives <= 0 }

// ─── 挑战模式 ─────────────────────────────────────────

// ChallengeMode 挑战模式：特殊约束（如限金、限塔数等），通过全波即胜利。
type ChallengeMode struct{}

func (m *ChallengeMode) Name() string                        { return "challenge" }
func (m *ChallengeMode) OnInit(_ *ModeContext)                {}
func (m *ChallengeMode) OnWaveStart(_ int, _ *ModeContext)    {}
func (m *ChallengeMode) OnWaveEnd(_ int, _ *ModeContext)      {}
func (m *ChallengeMode) OnUpdate(_ float64, _ *ModeContext)   {}
func (m *ChallengeMode) IsWon(ctx *ModeContext) bool          { return ctx.Wave >= ctx.MaxWaves }
func (m *ChallengeMode) IsLost(ctx *ModeContext) bool         { return ctx.Lives <= 0 }
