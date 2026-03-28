// skill.go — 技能系统框架。
// 管理可充能的主动/被动技能，支持充能、冷却、触发和伤害计算。
// 技能挂载到任意实体（英雄/核心/塔），通过注册表扩展。
package skill

import (
	"defense2/internal/core/enemy"
)

// SkillState 技能运行时状态。
type SkillState struct {
	Type     string  // 技能类型标识（注册表键）
	Charge   float64 // 当前充能值（0~MaxCharge）
	MaxCharge float64 // 满充能阈值
	Cooldown float64 // 冷却剩余时间（秒）
	BaseCooldown float64 // 基础冷却时间（秒）
	Damage   float64 // 技能伤害
	Range    float64 // 技能范围（像素）
	Active   bool    // 是否正在释放
}

// SkillContext 技能执行时传入的上下文。
type SkillContext struct {
	OwnerX  float64      // 持有者 X 坐标
	OwnerY  float64      // 持有者 Y 坐标
	Enemies *enemy.Pool  // 场上敌人池
	DT      float64      // 帧时间步长（秒）
}

// SkillBehavior 技能行为接口。
type SkillBehavior interface {
	Type() string                          // 技能类型标识
	Trigger(s *SkillState, ctx *SkillContext) // 触发技能效果
}

// 全局技能行为注册表。
var behaviors = map[string]SkillBehavior{}

// RegisterSkill 注册一种技能行为。
func RegisterSkill(b SkillBehavior) {
	behaviors[b.Type()] = b
}

// NewSkillState 创建一个技能状态实例。
func NewSkillState(typ string, maxCharge, cooldown, damage, rng float64) *SkillState {
	return &SkillState{
		Type:         typ,
		MaxCharge:    maxCharge,
		BaseCooldown: cooldown,
		Damage:       damage,
		Range:        rng,
	}
}

// Update 每帧更新技能状态：充能、冷却、自动触发。
func (s *SkillState) Update(ctx *SkillContext) {
	if s.Cooldown > 0 {
		s.Cooldown -= ctx.DT
		return
	}

	// 充能（每帧固定增量，满后自动触发）
	s.Charge += ctx.DT
	if s.Charge >= s.MaxCharge {
		s.Charge = 0
		s.Cooldown = s.BaseCooldown
		if b, ok := behaviors[s.Type]; ok {
			b.Trigger(s, ctx)
		}
	}
}
