// sys_spawn.go — 波次生成 + 敌人移动 + dying 动画步骤。
package pipeline

import (
	"defense2/internal/config"
	"defense2/internal/core/enemy"
)

// SysSessionTick 驱动游戏模式 session（限时倒计时等）。
type SysSessionTick struct{}

func (SysSessionTick) Tick(ctx *TickCtx) bool {
	ctx.Session.Tick(ctx.DT, ctx.BuildModeCtx())
	return false
}

// SysWardenGate 检查是否需要弹出战灵选择（第一波倒计时结束时）。
// 返回 true 中断后续步骤。
type SysWardenGate struct{}

func (SysWardenGate) Tick(ctx *TickCtx) bool {
	if ctx.CB.ShouldShowWardenSelect != nil && ctx.CB.ShouldShowWardenSelect() {
		if ctx.CB.ShowWardenSelect != nil {
			ctx.CB.ShowWardenSelect()
		}
		return true // 中断本帧
	}
	return false
}

// SysSpawn 驱动波次出怪。
type SysSpawn struct {
	prevWave int
}

func (s *SysSpawn) Tick(ctx *TickCtx) bool {
	s.prevWave = ctx.Spawner.Wave
	ctx.Spawner.Tick(ctx.Enemies, ctx.DT)
	if ctx.Spawner.Wave > s.prevWave {
		*ctx.WaveLivesSnap = *ctx.Lives
		if ctx.CB.OnWaveStart != nil {
			bossN := config.GlobalSpawnerConfig().Boss.EveryNWaves
			ctx.CB.OnWaveStart(ctx.Spawner.Wave, bossN > 0 && ctx.Spawner.Wave%bossN == 0)
		}
	}
	return false
}

// PrevWave 返回上一帧的波次号（供后续步骤使用）。
func (s *SysSpawn) PrevWave() int { return s.prevWave }

// SysEnemyStatusEffects tick 敌人状态效果（减速、流血等 DoT）。
type SysEnemyStatusEffects struct{}

func (SysEnemyStatusEffects) Tick(ctx *TickCtx) bool {
	TickEnemyStatusEffects(ctx.Enemies, ctx.DT, func(e *enemy.Enemy, dmg float64) {
		if ctx.CB.OnDamageText != nil {
			ctx.CB.OnDamageText(e.X, e.Y-10, dmg, false)
		}
	})
	return false
}

// SysEnemyMove 敌人沿路径移动 + 泄漏扣血。
type SysEnemyMove struct{}

func (SysEnemyMove) Tick(ctx *TickCtx) bool {
	ctx.Enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() || e.IsSpawning() {
			return
		}
		if enemy.MoveAlongPath(e, ctx.GameMap.Waypoints, ctx.DT) {
			*ctx.Lives--
			ctx.Enemies.KillImmediate(e)
			if ctx.CB.OnEnemyLeak != nil {
				ctx.CB.OnEnemyLeak(e)
			}
		}
	})
	return false
}

// SysSpawnAnim tick 出生动画倒计时。
type SysSpawnAnim struct{}

func (SysSpawnAnim) Tick(ctx *TickCtx) bool {
	ctx.Enemies.Each(func(e *enemy.Enemy) {
		if e.SpawnTimer > 0 {
			e.SpawnTimer -= ctx.DT
			if e.SpawnTimer < 0 {
				e.SpawnTimer = 0
			}
		}
	})
	return false
}

// SysDying tick 死亡动画倒计时。
type SysDying struct{}

func (SysDying) Tick(ctx *TickCtx) bool {
	ctx.Enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() {
			e.DyingTimer -= ctx.DT
			if e.DyingTimer <= 0 {
				ctx.Enemies.FinishDying(e)
			}
		}
	})
	return false
}
