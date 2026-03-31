// skill.go — 技能系统 stub。
// TODO: 完整实现技能框架。
package skill

import "defense2/internal/core/enemy"

// SkillState 技能运行时状态。
type SkillState struct {
	SkillName string  // 技能名称
	Inited    bool    // 是否已初始化
	Cooldown  float64 // 冷却剩余
	Active    bool    // 是否正在释放
}

// SkillContext 技能执行上下文。
type SkillContext struct {
	Projectiles interface{} // *projectile.Pool
	Beams       interface{} // *combat.BeamPool
	OnHit       func(e *enemy.Enemy, dmg float64, killed bool)
	OnActivate  func(skillKey string)
}

// AssignSkill 为实体挂载技能（stub）。
func AssignSkill(state *SkillState, name string, owner interface{}) {
	state.SkillName = name
	state.Inited = true
}

// TickEntitySkill 每帧驱动实体技能（stub）。
func TickEntitySkill(state *SkillState, owner interface{}, enemies []*enemy.Enemy, dt float64, ctx *SkillContext) {
	// TODO: implement
}
