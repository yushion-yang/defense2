// sys_endgame.go — 波次奖励 + 胜负判定步骤。
package pipeline

import "defense2/internal/core/gamemode"

// SysWaveReward 波次完成奖励 + 事件触发。
// 需要知道 prevWave（由 SysSpawn 记录）。
type SysWaveReward struct {
	SpawnSys *SysSpawn // 引用 SysSpawn 以获取 prevWave
}

func (s SysWaveReward) Tick(ctx *TickCtx) bool {
	prevWave := s.SpawnSys.PrevWave()
	if ctx.Spawner.Wave > prevWave && prevWave > 0 {
		if ctx.CB.OnWaveCleared != nil {
			ctx.CB.OnWaveCleared(prevWave, *ctx.WaveLivesSnap)
		}
	}
	return false
}

// SysEndCondition 胜负判定（委托给游戏模式）。
type SysEndCondition struct {
	OnVictory func()
	OnDefeat  func()
}

func (s SysEndCondition) Tick(ctx *TickCtx) bool {
	if ctx.Session.CheckEndConditions(ctx.BuildModeCtx()) {
		switch ctx.Session.Status {
		case gamemode.StatusVictory:
			if s.OnVictory != nil {
				s.OnVictory()
			}
		case gamemode.StatusDefeat:
			if s.OnDefeat != nil {
				s.OnDefeat()
			}
		}
	}
	return false
}
