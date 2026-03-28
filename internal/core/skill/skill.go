// skill.go — 实体无关的可插拔技能框架。
// 支持塔/战灵/召唤物挂载技能，通过注册表扩展。
package skill

import (
	"defense2/internal/core/enemy"
	"sync"
)

// CoreSkill 核心技能接口（实体无关，塔/战灵/召唤物都可挂载）。
type CoreSkill interface {
	// Name 返回技能注册名。
	Name() string
	// Init 初始化技能，绑定到持有者。
	Init(owner interface{})
	// Tick 每帧更新，返回是否压制普攻。
	Tick(owner interface{}, enemies []*enemy.Enemy, dt float64, ctx *SkillContext) bool
	// ShouldSuppressFire 是否压制持有者的普攻。
	ShouldSuppressFire(owner interface{}) bool
	// ShouldSuppressMove 是否压制持有者的移动。
	ShouldSuppressMove(owner interface{}) bool
	// GetProgress 返回技能进度（0~1）和是否就绪。
	GetProgress(owner interface{}) (ratio float64, ready bool)
}

// SkillContext 技能执行上下文。
type SkillContext struct {
	Projectiles interface{}                                      // *projectile.Pool（避免循环依赖用 interface）
	Beams       interface{}                                      // *combat.BeamPool
	OnHit       func(e *enemy.Enemy, damage float64, killed bool) // 命中回调
}

// SkillState 挂载在实体上的技能运行状态。
type SkillState struct {
	SkillName string    // 当前技能名
	Skill     CoreSkill // 技能实例
	Inited    bool      // 是否已初始化
}

// ── 全局注册表 ──

var (
	registryMu sync.RWMutex
	registry   = map[string]func() CoreSkill{} // 技能工厂注册表
)

// Register 注册技能工厂（通常在 init() 中调用）。
func Register(name string, factory func() CoreSkill) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// Get 根据名称创建新的技能实例。找不到返回 nil。
func Get(name string) CoreSkill {
	registryMu.RLock()
	defer registryMu.RUnlock()
	if f, ok := registry[name]; ok {
		return f()
	}
	return nil
}

// AssignSkill 把技能分配到 SkillState，创建实例并初始化。
// 成功返回 true，技能不存在返回 false。
func AssignSkill(state *SkillState, name string, owner interface{}) bool {
	sk := Get(name)
	if sk == nil {
		return false
	}
	state.SkillName = name
	state.Skill = sk
	state.Skill.Init(owner)
	state.Inited = true
	return true
}

// TickEntitySkill 每帧驱动实体上的技能，返回是否压制普攻。
// 如果技能未初始化或不存在，返回 false。
func TickEntitySkill(state *SkillState, owner interface{}, enemies []*enemy.Enemy, dt float64, ctx *SkillContext) bool {
	if state == nil || state.Skill == nil {
		return false
	}
	if !state.Inited {
		state.Skill.Init(owner)
		state.Inited = true
	}
	return state.Skill.Tick(owner, enemies, dt, ctx)
}

// ResetRegistry 清空注册表（仅用于测试）。
func ResetRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = map[string]func() CoreSkill{}
}
