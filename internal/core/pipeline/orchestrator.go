// orchestrator.go — Tick 编排器。
// 将 stage.go 的 updatePlaying() 分解为有序步骤链。
// 每个步骤是 TickSystem 接口，通过 TickCtx 共享运行时状态。
package pipeline

import (
	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/internal/core/gamemode"
	"defense2/internal/core/gamemap"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
	"defense2/internal/core/warden"
)

// TickSystem 是一个可插入编排器的 tick 步骤。
type TickSystem interface {
	// Tick 执行本步骤的逻辑。返回 true 表示应中断后续步骤（如 hit-stop）。
	Tick(ctx *TickCtx) bool
}

// TickCtx 包含一帧内所有子系统共享的运行时数据。
// stage.go 负责在每帧开始前填充。
type TickCtx struct {
	// 核心实体池
	Enemies     *enemy.Pool
	Towers      *tower.Pool
	Projectiles *projectile.Pool
	Beams       *combat.BeamPool

	// 战灵
	WardenUnit  *warden.Warden
	WardenReady bool
	WardenType  string

	// 游戏系统
	Session *gamemode.Session
	Spawner *enemy.Spawner
	GameMap *gamemap.GameMap

	// 帧参数
	DT        float64 // gameDT（已乘以倍速）
	Frame     int
	GameSpeed int

	// 可变游戏状态（指针，System 直接修改）
	Gold  *int
	Lives *int
	Kills *int

	// 波次快照
	WaveLivesSnap *int

	BuildModeCtx func() *gamemode.Context

	// 回调（连接 stage.go 的音效/VFX/渲染副作用）
	CB TickCallbacks
}

// TickCallbacks 所有渲染/音效副作用通过回调注入，保持 System 纯逻辑。
type TickCallbacks struct {
	// 敌人
	OnDamageText    func(x, y, dmg float64, crit bool)
	OnEnemyLeak     func(e *enemy.Enemy)
	OnKill          func(isBoss bool, source string)

	// 塔
	OnTowerFire     func(t *tower.Tower, style string)
	OnTowerDirectHit func(e *enemy.Enemy, dmg float64, killed bool, style string, crit bool)

	// 弹射物
	OnProjectileHit func(e *enemy.Enemy, dmg float64, killed bool, style string, crit bool)

	// CC 效果（减速/眩晕/灼烧等命中时回调）
	OnCC func(x, y float64, ccType string)

	// 战灵
	OnWardenFire    func()
	OnWardenSpecial func()
	OnWardenDamage  func(x, y, dmg float64, crit bool)
	OnWardenKill    func(e *enemy.Enemy)

	// 波次
	OnWaveStart     func(wave int, isBoss bool)
	OnWaveCleared   func(prevWave int, livesSnap int)

	// 战灵选择检查
	ShouldShowWardenSelect func() bool
	ShowWardenSelect       func()
}

// Orchestrator 按顺序执行一系列 TickSystem。
type Orchestrator struct {
	systems []namedSystem
}

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

// Func 将函数适配为 TickSystem 接口。
func Func(fn func(ctx *TickCtx) bool) TickSystem {
	return tickFunc(fn)
}

type tickFunc func(ctx *TickCtx) bool

func (f tickFunc) Tick(ctx *TickCtx) bool { return f(ctx) }
