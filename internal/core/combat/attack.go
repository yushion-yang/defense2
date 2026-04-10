// attack.go — 攻击方式分发框架。
// 定义 AttackHandler 接口和注册表，每种 attackStyle 实现一个 handler。
package combat

import (
	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

// AttackHandler 攻击方式处理器。
type AttackHandler interface {
	// Fire 发射攻击（标准冷却触发时调用）。
	Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext)
}

// TickHandler 自管理攻击方式（每帧调用，不走标准冷却）。
type TickHandler interface {
	AttackHandler
	// Tick 每帧更新（spin_aoe 旋转等自管理攻击方式）。
	Tick(t *tower.Tower, ctx *AttackContext)
	// SelfManaged 返回 true 表示跳过标准冷却循环。
	SelfManaged() bool
}

// AttackContext 攻击上下文，由 pipeline 传入。
type AttackContext struct {
	Enemies     *enemy.Pool
	Projectiles *projectile.Pool
	Beams       *BeamPool
	OnFire      func(t *tower.Tower, style string)                                         // 射击回调（携带塔引用和攻击方式）
	OnHit       func(e *enemy.Enemy, damage float64, killed bool, style string, crit bool) // 命中回调（携带攻击方式+暴击）
	OnCC        CCCallback                                                                 // CC 效果命中回调（可为 nil）
	DT          float64
	Style       string // 当前攻击方式（由 pipeline 设置，handler 内部可读取）
}

// HitCallback 弹射物命中回调（用于生成飘字、音效等）。
type HitCallback = func(e *enemy.Enemy, damage float64, killed bool, attackStyle string, crit bool)

// CCCallback CC 效果命中回调（用于播放 CC 音效）。
// ccType: "slow", "freeze", "stun", "burn"
type CCCallback = func(x, y float64, ccType string)

// ── 注册表 ──

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

// ProjectileHandler 标准追踪弹。
type ProjectileHandler struct{}

func (h *ProjectileHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
	bal := config.GlobalBalance().Combat
	speed := t.ProjectileSpeed
	if speed <= 0 {
		speed = bal.DefaultProjectileSpeed
	}
	ctx.Projectiles.Fire(t.X, t.Y, target.X, target.Y, t.Damage, speed, bal.DefaultProjectileRadius, target, t.InstanceKey)
}

func init() {
	Register(tower.StyleProjectile, &ProjectileHandler{})
	Register(tower.StyleWideBeam, &WideBeamHandler{})
	Register(tower.StyleScatter, &ScatterHandler{})
	Register(tower.StyleSpinAoE, &SpinAoEHandler{})
	Register(tower.StyleRadial, &RadialHandler{})
}
