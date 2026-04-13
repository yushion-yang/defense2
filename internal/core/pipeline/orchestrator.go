// orchestrator.go — Pipeline 编排器：每帧 tick 的核心调度中枢。
//
// 职责：将 stage.go 的 updatePlaying() 拆分为 ~20 个有序步骤（TickSystem），
// 按注册顺序依次执行。这是 ECS-like 的"S"（System）层。
//
// 关系：
//   - stage.go 每帧构造 TickCtx 并调用 Orchestrator.Tick()
//   - sys_spawn.go / tick_combat.go / tick_abilities.go 提供具体 System 实现
//   - TickCtx 是所有 System 共享的可变上下文（类似帧级 DI 容器）
//
// 设计决策：
//   - TickSystem.Tick() 返回 bool 而非 error，因为需要支持"中断本帧"语义
//     （如战灵选择界面弹出时必须暂停后续所有 System）
//   - 回调模式（TickCallbacks）隔离 core 与 render/audio 依赖，
//     保持 System 可单元测试
package pipeline

import (
	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/internal/core/gamemap"
	"defense2/internal/core/gamemode"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
	"defense2/internal/core/warden"
)

// TickSystem 是一个可插入编排器的 tick 步骤。
// 返回值语义：true = 中断本帧后续所有步骤。
// 目前唯一使用中断的场景是 SysWardenGate（弹出战灵选择时冻结游戏逻辑）。
// 大多数 System 始终返回 false。
type TickSystem interface {
	Tick(ctx *TickCtx) bool
}

// TickCtx 包含一帧内所有子系统共享的运行时数据。
// stage.go 负责在每帧开始前填充，生命周期仅限当帧。
//
// 设计要点：
//   - 实体池（Enemies/Towers 等）是指针——多个 System 可读写同一池
//   - Gold/Lives/Kills 是指向 stage 字段的指针——System 直接修改游戏状态
//   - CB 回调将副作用（音效/飘字/VFX）延迟到 System 执行时触发，
//     避免 core 包 import render/audio
type TickCtx struct {
	// ── 核心实体池 ──
	Enemies     *enemy.Pool
	Towers      *tower.Pool
	Projectiles *projectile.Pool
	Beams       *combat.BeamPool // wideBeam 等光束攻击方式的渲染数据池

	// ── 战灵 ──
	WardenUnit  *warden.Warden // 当前激活的战灵实例（未选择时 nil）
	WardenReady bool           // 战灵技能冷却完毕可释放
	WardenType  string         // 战灵类型 ID（如 "nexus"/"forge" 等）

	// ── 游戏系统 ──
	Session *gamemode.Session // 当前游戏模式 session（战役/无尽/限时等）
	Spawner *enemy.Spawner    // 波次出怪调度器
	GameMap *gamemap.GameMap  // 当前地图（路径点、格子等）

	// ── 帧参数 ──
	DT        float64 // 本帧 delta time，已乘倍速（1x/2x/3x）
	Frame     int     // 帧计数器（从关卡开始累计）
	GameSpeed int     // 当前倍速（1/2/3）

	// ── 可变游戏状态（指针直接修改 stage 字段）──
	Gold  *int // 当前金币，塔能力产出/击杀奖励直接加
	Lives *int // 剩余生命，敌人泄漏时直接减
	Kills *int // 击杀总数

	// ── 波次状态 ──
	WaveLivesSnap *int // 波次开始时的生命快照，用于"完美通关波次"判定

	BuildModeCtx func() *gamemode.Context // 延迟构造 mode context，避免每帧分配

	// ── 回调（core → stage/render 的副作用桥梁）──
	CB TickCallbacks
}

// TickCallbacks 所有渲染/音效副作用通过回调注入，保持 System 纯逻辑。
// stage.go 在构造 TickCtx 时将闭包填入这些字段，System 仅在需要时调用。
// 任何回调均可为 nil，调用方须判空。
type TickCallbacks struct {
	// ── 敌人相关 ──
	OnDamageText func(x, y, dmg float64, crit bool) // 伤害飘字（位置 + 数值 + 是否暴击）
	OnEnemyLeak  func(e *enemy.Enemy)               // 敌人到达终点泄漏（触发扣血音效/屏幕抖动）
	OnKill       func(isBoss bool, source string)   // 击杀（source: "tower"/"warden"/"dot" 等）

	// ── 塔相关 ──
	OnTowerFire      func(t *tower.Tower, style string)                                      // 塔开火（触发射击音效）
	OnTowerDirectHit func(e *enemy.Enemy, dmg float64, killed bool, style string, crit bool) // 光束等无弹射物的直接命中

	// ── 弹射物相关 ──
	OnProjectileHit func(e *enemy.Enemy, dmg float64, killed bool, style string, crit bool) // 弹射物命中敌人

	// ── CC / 溅射 ──
	OnCC        func(x, y float64, ccType string) // CC 命中（触发冰冻/眩晕等 VFX）
	OnSplashVFX func(x, y, radius float64)        // 溅射爆炸 VFX（圆形范围指示）

	// ── 战灵相关 ──
	OnWardenFire    func()                             // 战灵普攻
	OnWardenSpecial func()                             // 战灵技能释放
	OnWardenDamage  func(x, y, dmg float64, crit bool) // 战灵造成伤害
	OnWardenKill    func(e *enemy.Enemy)               // 战灵击杀

	// ── 波次事件 ──
	OnWaveStart   func(wave int, isBoss bool)       // 新波次开始（波次号 + 是否 Boss 波）
	OnWaveCleared func(prevWave int, livesSnap int) // 波次清空（用于结算完美奖励）

	// ── 战灵选择门控 ──
	// 第一波倒计时结束时检查是否需要弹出战灵选择覆盖层。
	// SysWardenGate 调用这两个回调来实现"暂停游戏→选战灵→恢复"的流程。
	ShouldShowWardenSelect func() bool
	ShowWardenSelect       func()
}

// Orchestrator 按注册顺序依次执行一系列 TickSystem。
// 顺序至关重要：出怪 → 状态效果 → 战斗 → 弹道 → 结算，不可乱序。
type Orchestrator struct {
	systems []namedSystem
}

// namedSystem 为调试/日志保留名称（如 "spawn"、"combat" 等）。
type namedSystem struct {
	name   string
	system TickSystem
}

// NewOrchestrator 创建空编排器。
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{}
}

// Add 添加一个命名的 tick 步骤。
func (o *Orchestrator) Add(name string, sys TickSystem) {
	o.systems = append(o.systems, namedSystem{name: name, system: sys})
}

// Tick 按顺序执行所有步骤。任何步骤返回 true 时中断后续步骤。
func (o *Orchestrator) Tick(ctx *TickCtx) {
	for _, ns := range o.systems {
		if ns.system.Tick(ctx) {
			return
		}
	}
}

// SystemCount 返回已注册的步骤数。
func (o *Orchestrator) SystemCount() int {
	return len(o.systems)
}

// Func 将普通函数适配为 TickSystem 接口。
// 用于不需要维护状态的简单步骤（如时间缩放、教程检查），
// 避免为每个轻量步骤都定义 struct。
func Func(fn func(ctx *TickCtx) bool) TickSystem {
	return tickFunc(fn)
}

type tickFunc func(ctx *TickCtx) bool

func (f tickFunc) Tick(ctx *TickCtx) bool { return f(ctx) }
