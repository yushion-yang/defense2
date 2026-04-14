// attack.go — 攻击方式分发框架。
// 定义 AttackHandler 接口和注册表，每种 attackStyle 实现一个 handler。
//
// 本文件是战斗系统的入口层，负责将 6 种攻击方式（projectile/wideBeam/scatter/
// spin_aoe/radial/barrage）统一注册到全局注册表，供 pipeline/orchestrator 查表调用。
//
// 攻击方式分两类：
//   - 冷却触发型（cooldown-based）：外部 pipeline 管理冷却，时间到了调 Fire()。
//     包括 projectile、wideBeam、scatter、radial。
//   - 自管理型（self-managed）：handler 自己管理冷却和状态，pipeline 每帧调 Tick()。
//     包括 spin_aoe（持续旋转打击）、barrage（连射模式）。
//
// 调用关系：pipeline → IsSelfManaged() 判断类型 → Fire() 或 TickSelfManaged()
// 命中后统一走 apply_hit.go 的 ApplyHit() 处理伤害和能力触发。
package combat

import (
	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

// AttackHandler 攻击方式处理器。
// 所有攻击方式（6 种）都实现此接口。冷却触发型仅实现 Fire；
// 自管理型额外实现 TickHandler，pipeline 会通过类型断言区分。
type AttackHandler interface {
	// Fire 发射攻击（标准冷却触发时调用）。
	Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext)
}

// TickHandler 自管理攻击方式（每帧调用，不走标准冷却）。
// spin_aoe 和 barrage 实现此接口，pipeline 每帧调 Tick() 而非等冷却到了才调 Fire()。
// 这两种方式有持续状态（旋转角度、连射队列），需要自己管理时序。
type TickHandler interface {
	AttackHandler
	// Tick 每帧更新（spin_aoe 旋转等自管理攻击方式）。
	Tick(t *tower.Tower, ctx *AttackContext)
	// SelfManaged 返回 true 表示跳过标准冷却循环。
	SelfManaged() bool
}

// AttackContext 攻击上下文，由 pipeline 传入。
// 携带当前帧所需的全部运行时引用（对象池、回调函数、时间步长），
// 避免 handler 直接依赖全局状态，便于测试和解耦。
type AttackContext struct {
	Enemies     *enemy.Pool
	Projectiles *projectile.Pool
	Beams       *BeamPool
	OnFire      func(t *tower.Tower, style string)                                         // 射击回调（携带塔引用和攻击方式）
	OnHit       func(e *enemy.Enemy, damage float64, killed bool, style string, crit bool) // 命中回调（携带攻击方式+暴击）
	OnCC        CCCallback                                                                 // CC 效果命中回调（可为 nil）
	OnSplashVFX func(x, y, radius float64)                                                 // 溅射 VFX 回调（可为 nil）
	DT          float64
	Style       string // 当前攻击方式（由 pipeline 设置，handler 内部可读取）
}

// HitCallback 弹射物命中回调（用于生成飘字、音效等）。
type HitCallback = func(e *enemy.Enemy, damage float64, killed bool, attackStyle string, crit bool)

// CCCallback CC 效果命中回调（用于播放 CC 音效）。
// ccType: "slow", "freeze", "stun", "burn"
type CCCallback = func(x, y float64, ccType string)

// getAbilityDef 从全局能力表中获取指定能力的定义。未找到返回 nil。
func getAbilityDef(abilType string) *config.AbilityDef {
	if table := config.GlobalAbilityTable(); table != nil {
		return table[abilType]
	}
	return nil
}

// ── 注册表 ──
// 全局 map，init() 中注册所有 6 种攻击方式。
// pipeline 通过 Get(style) 查表获取 handler，无需 switch-case。

var handlers = map[tower.AttackStyle]AttackHandler{}

// Register 注册攻击方式处理器。
func Register(style tower.AttackStyle, h AttackHandler) {
	handlers[style] = h
}

// Get 获取指定 style 的处理器。
func Get(style tower.AttackStyle) AttackHandler {
	return handlers[style]
}

// IsSelfManaged 检查某 style 是否自管理（跳过标准冷却）。
func IsSelfManaged(style tower.AttackStyle) bool {
	h := handlers[style]
	if th, ok := h.(TickHandler); ok {
		return th.SelfManaged()
	}
	return false
}

// TickSelfManaged 对自管理 handler 调用 Tick。
func TickSelfManaged(t *tower.Tower, ctx *AttackContext) {
	h := handlers[t.AttackStyleID]
	if th, ok := h.(TickHandler); ok {
		th.Tick(t, ctx)
	}
}

// ProjectileHandler 标准追踪弹（最基础的攻击方式）。
// 向目标位置发射一颗追踪弹射物，弹速和半径从配置读取。
// 弹射物命中后由 projectile pool 的碰撞检测触发 ApplyHit。
type ProjectileHandler struct{}

func (h *ProjectileHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
	bal := config.GlobalBalance().Combat
	speed := t.ProjectileSpeed
	if speed <= 0 {
		speed = bal.DefaultProjectileSpeed
	}
	ctx.Projectiles.Fire(t.X, t.Y, target.X, target.Y, t.Damage, speed, bal.DefaultProjectileRadius, target, t.InstanceKey)
}

// init 注册全部 6 种攻击方式到全局注册表。
// 其中 SpinAoEHandler 和 BarrageHandler 实现 TickHandler（自管理），
// 其余 4 种仅实现 AttackHandler（冷却触发型）。
func init() {
	Register(tower.StyleProjectile, &ProjectileHandler{})
	Register(tower.StyleWideBeam, &WideBeamHandler{})
	Register(tower.StyleScatter, &ScatterHandler{})
	Register(tower.StyleSpinAoE, &SpinAoEHandler{})
	Register(tower.StyleRadial, &RadialHandler{})
	Register(tower.StyleBarrage, &BarrageHandler{})
}
