// session.go — 游戏会话运行时容器。
// 封装 Mode 实例和统计数据，驱动每帧的模式回调和胜负状态机。
package gamemode

// SessionStatus 会话状态。
type SessionStatus int

const (
	StatusIdle    SessionStatus = iota // reserved for future start-game flow
	StatusPlaying                      // 进行中
	StatusVictory                      // 胜利
	StatusDefeat                       // 失败
)

// Stats 单局游戏统计。
type Stats struct {
	Kills        int // 击杀数
	Leaked       int // 泄漏数
	TowersBuilt int // 放置塔数
	WavesCleared int // 通过波数
	GoldEarned   int // 累计获得金币
	BossKills    int // Boss 击杀数
}

// Session 单局游戏会话。
type Session struct {
	Mode        Mode
	Stats       Stats
	ElapsedTime float64
	Status      SessionStatus

	// 波次期间泄漏数（用于完美波次检测）
	waveLeaked int
}

// NewSession 创建新会话。
func NewSession(mode Mode) *Session {
	return &Session{
		Mode:   mode,
		Status: StatusPlaying,
	}
}

// Tick 每帧调用，累计时间并委托给 Mode.OnTick。
// 注意：胜负判定在 CheckEndConditions 中单独调用。
func (s *Session) Tick(dt float64, ctx *Context) {
	if s.Status != StatusPlaying {
		return
	}
	s.ElapsedTime += dt
	s.Mode.OnTick(dt, ctx)
}

// CheckEndConditions 检查胜负条件，更新会话状态。
// 返回 true 表示游戏结束。
func (s *Session) CheckEndConditions(ctx *Context) bool {
	if s.Status != StatusPlaying {
		return true
	}
	if s.Mode.CheckDefeat(ctx) {
		s.Status = StatusDefeat
		return true
	}
	if s.Mode.CheckVictory(ctx) {
		s.Status = StatusVictory
		return true
	}
	return false
}

// OnWaveStart 波次开始。
func (s *Session) OnWaveStart(wave int, ctx *Context) {
	s.waveLeaked = 0
	s.Mode.OnWaveStart(wave, ctx)
}

// OnWaveCleared 波次结束，统计通过数并返回奖励。
func (s *Session) OnWaveCleared(wave int, ctx *Context) WaveClearResult {
	s.Stats.WavesCleared++
	// 完美波次检测：波次期间零泄漏
	result := s.Mode.OnWaveCleared(wave, ctx)
	if s.waveLeaked > 0 {
		result.PerfectBonus = 0
	}
	s.Stats.GoldEarned += result.BonusGold + result.PerfectBonus
	s.waveLeaked = 0
	return result
}

// OnEnemyKilled 敌人被击杀。
func (s *Session) OnEnemyKilled(boss bool, ctx *Context) {
	s.Stats.Kills++
	if boss {
		s.Stats.BossKills++
	}
	s.Mode.OnEnemyKilled(boss, ctx)
}

// OnEnemyLeaked 敌人到达终点。
func (s *Session) OnEnemyLeaked(ctx *Context) {
	s.Stats.Leaked++
	s.waveLeaked++
	s.Mode.OnEnemyLeaked(ctx)
}

// OnTowerBuilt 塔被建造。
func (s *Session) OnTowerBuilt() {
	s.Stats.TowersBuilt++
}
