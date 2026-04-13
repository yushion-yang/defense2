// sys_spawn.go — 敌人生命周期相关的 TickSystem 集合。
//
// 包含 6 个 System，覆盖敌人从出生到死亡的完整生命周期：
//
//	SysSessionTick      → 游戏模式 session 驱动（限时倒计时等）
//	SysWardenGate       → 战灵选择门控（可中断本帧）
//	SysSpawn            → 波次出怪调度
//	SysEnemyStatusEffects → 状态效果 tick（DoT/CC）
//	SysEnemyMove        → 敌人沿路径移动 + 泄漏检测
//	SysSpawnAnim / SysDying → 出生/死亡动画计时器
//
// 设计说明：
//
//	这些 System 都是无状态或轻状态的 struct（仅 SysSpawn 记录 prevWave），
//	因此用 struct 而非 Func() 适配，便于 stage.go 持有引用读取状态（如 PrevWave()）。
package pipeline

import (
	"defense2/internal/config"
	"defense2/internal/core/enemy"
)

// SysSessionTick 驱动游戏模式 session 的时间推进。
// 不同模式有不同的 session 逻辑（战役=波次推进，限时=倒计时，无尽=无限波次）。
// 必须在出怪之前执行，因为 session 可能决定是否允许下一波。
type SysSessionTick struct{}

func (SysSessionTick) Tick(ctx *TickCtx) bool {
	ctx.Session.Tick(ctx.DT, ctx.BuildModeCtx())
	return false
}

// SysWardenGate 战灵选择门控——唯一会返回 true 中断编排器的 System。
// 在第一波倒计时结束、玩家尚未选择战灵时触发，弹出战灵选择覆盖层。
// 中断后整帧的后续 System（出怪/战斗/结算）全部跳过，游戏逻辑冻结。
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

// SysSpawn 驱动波次出怪调度器。
// prevWave 记录上一帧的波次号，用于检测"新波次开始"事件。
// 这是唯一需要持有状态的轻量 System，因此用 struct + 指针接收者。
type SysSpawn struct {
	prevWave int
}

func (s *SysSpawn) Tick(ctx *TickCtx) bool {
	s.prevWave = ctx.Spawner.Wave
	ctx.Spawner.Tick(ctx.Enemies, ctx.DT)

	// 检测波次递增——Spawner.Tick 内部会自增 Wave
	if ctx.Spawner.Wave > s.prevWave {
		// 记录本波开始时的生命值快照，波次结束时用于判定"完美波次"（无泄漏）
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

// SysEnemyStatusEffects tick 敌人的 BuffList 状态效果（CC + DoT）。
// 必须在 SysEnemyMove 之前执行：减速效果需要在本帧移动前生效。
// DoT 伤害通过回调上报（飘字），实际扣血在 TickEnemyStatusEffects 内完成。
type SysEnemyStatusEffects struct{}

func (SysEnemyStatusEffects) Tick(ctx *TickCtx) bool {
	TickEnemyStatusEffects(ctx.Enemies, ctx.DT, func(e *enemy.Enemy, dmg float64) {
		if ctx.CB.OnDamageText != nil {
			ctx.CB.OnDamageText(e.X, e.Y-10, dmg, false)
		}
	})
	return false
}

// SysEnemyMove 敌人沿路径点移动。
// 到达终点的敌人视为"泄漏"：扣生命 → 立即移除（不走死亡动画）。
// KillImmediate 与 Kill 的区别：前者不触发击杀奖励和死亡效果（如分裂）。
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
// 出生期间 (SpawnTimer > 0) 敌人不可被攻击、不移动，仅播放淡入动画。
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
// 死亡期间 (IsDying=true) 敌人已不参与碰撞和索敌，仅播放消散动画。
// DyingTimer 归零后 FinishDying 将其从池中彻底回收。
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
