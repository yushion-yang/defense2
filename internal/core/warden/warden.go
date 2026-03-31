// warden.go — 战灵系统框架。
// 战灵是由塔强度驱动的伴生灵体，通过行为注册表支持多种类型（金灵、火灵、水灵等）。
// 每种类型通过 RegisterBehavior 注册自己的 tick/draw 逻辑。
package warden

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/skill"
	"defense2/internal/core/tower"
)

// Warden 战灵实体。
type Warden struct {
	ID                int     // 唯一标识
	Name              string  // 显示名称
	Type              string  // 类型标识（如 "envoy"、"chain"、"skystrike"）
	SelfStrength      float64 // 自身积累强度（来自击杀/通波）
	PerceivedStrength float64 // 感知强度 = 自身 + 塔贡献
	PeakStrength      float64 // 历史最高强度（棘轮，只升不降）
	Active            bool    // 是否已激活
	Level             int     // 当前等级（1-5，由强度阈值决定）
	GrowthOnKill      float64 // 每次击杀增加的强度（从配置读取）
	GrowthOnWaveClear float64 // 每次通波增加的强度（从配置读取）
	// 技能系统
	Skill *skill.SkillState // 挂载的技能（nil=无技能）
	// 类型特定状态由 Behavior.Tick 内部管理
	State interface{} // 类型特定内部状态（由行为实现持有）
}

// TickContext 战灵 tick 时传入的上下文。
type TickContext struct {
	Enemies     *enemy.Pool                        // 场上敌人池
	Towers      *tower.Pool                        // 场上塔池
	Projectiles *projectile.Pool                   // 弹射物池（供战灵发射弹射物）
	DT          float64                            // 帧时间步长（秒）
	OnKill      func(e *enemy.Enemy)               // 击杀回调（通知场景计分/奖金）
	OnFire      func()                             // 普攻射击回调（音效）
	OnSpecial   func()                             // 特殊能力施放回调（音效）
	OnDamage    func(x, y, dmg float64, crit bool) // 伤害回调（浮字+特效，统一入口）
}

// Behavior 战灵行为接口，每种战灵类型实现一个。
type Behavior interface {
	// Type 返回行为类型标识。
	Type() string
	// Init 初始化类型特定状态，返回 State 对象。
	Init(w *Warden) interface{}
	// Tick 每帧逻辑更新。
	Tick(w *Warden, ctx *TickContext)
}

// 全局行为注册表。
var behaviors = map[string]Behavior{}

// RegisterBehavior 注册一种战灵行为。
func RegisterBehavior(b Behavior) {
	behaviors[b.Type()] = b
}

// NewWarden 创建一个战灵。
func NewWarden(id int, name, typ string) *Warden {
	w := &Warden{
		ID:                id,
		Name:              name,
		Type:              typ,
		Active:            true,
		Level:             1,
		SelfStrength:      100, // 初始强度 100
		GrowthOnKill:      2,   // 默认值，可被配置覆盖
		GrowthOnWaveClear: 5,   // 默认值，可被配置覆盖
	}
	if b, ok := behaviors[typ]; ok {
		w.State = b.Init(w)
	}
	return w
}

// BaseState 返回战灵的公共基座状态（如果 State 实现了 Stateful 接口）。
func (w *Warden) BaseState() *WardenState {
	if s, ok := w.State.(Stateful); ok {
		return s.Base()
	}
	return nil
}

// DescParams 返回当前战灵状态的占位符参数（供 HUD 替换 {key} 用）。
// 若 State 未实现 DescProvider，返回 nil。
func (w *Warden) DescParams() map[string]string {
	if dp, ok := w.State.(DescProvider); ok {
		return dp.DescParams(w)
	}
	return nil
}

// CalcStrength 根据塔池计算战灵感知强度。
// 规则: perceivedStrength = selfStrength + Σmax(0, tower.effectiveStrength - 100)
// 只有战力超过基准值(100)的部分贡献给战灵。低于100的塔不拖后腿。
func (w *Warden) CalcStrength(towers *tower.Pool) {
	total := w.SelfStrength
	towers.Each(func(t *tower.Tower) {
		if t.Strength == nil {
			return
		}
		total += t.Strength.Overflow() // max(0, effective - 100)
	})
	w.PerceivedStrength = total
	if total > w.PeakStrength {
		w.PeakStrength = total
	}
}

// Tick 驱动战灵行为。
func (w *Warden) Tick(ctx *TickContext) {
	if !w.Active {
		return
	}
	w.CalcStrength(ctx.Towers)
	if b, ok := behaviors[w.Type]; ok {
		b.Tick(w, ctx)
	}
}

// OnKill 击杀敌人时增加自身强度。
func (w *Warden) OnKill() {
	w.SelfStrength += w.GrowthOnKill
}

// OnWaveClear 波次通过时增加自身强度。
func (w *Warden) OnWaveClear() {
	w.SelfStrength += w.GrowthOnWaveClear
}
