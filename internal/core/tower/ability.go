// ability.go — 塔能力系统核心接口与注册表。
//
// 本文件定义了能力系统的"骨架"，不包含任何具体能力的实现。
// 具体能力（scatter/bounce/splash/stun 等 32 种）在 abilities/ 子包中实现，
// 通过 init() + blank import 模式自注册到全局 Registry。
//
// 注册模式说明（init() + blank import）：
//  1. 每个能力文件的 init() 调用 tower.Register(myAbility{})
//  2. 主程序通过 _ "defense2/internal/core/tower/abilities" 触发所有 init()
//  3. 运行时通过 Lookup("scatter") 获取能力实例
//     优点：新增能力只需添加一个文件 + 一行 blank import，无需修改注册中心
//
// 两种能力接口：
//   - Ability（必须）：OnHit 在弹射物命中时触发，返回 HitResult 描述效果
//   - Ticker（可选）：OnTick 每帧调用，用于光环/区域/经济等持续性能力
//
// HitResult 是能力效果的"菜单"：每个字段是可选的效果项，
// 战斗管线（combat/apply_hit.go）遍历所有非 nil 字段逐一应用。
package tower

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
)

// HitResult 描述弹射物命中时触发的能力效果。
// 采用"效果菜单"模式：每个字段是一种可选效果，nil 表示不触发。
// 战斗管线（combat/apply_hit.go）会遍历所有非 nil 字段逐一应用。
// 一次命中可以同时触发多种效果（如 BonusDamage + Slow + Bleed）。
type HitResult struct {
	// ── 伤害类 ──
	BonusDamage    float64 // 额外伤害（叠加到主伤害上，与主伤害共享 damageCap 上限）
	SeparateDamage float64 // 独立伤害（单独走一次伤害管线，有自己的 damageCap，用于绕过主伤害上限）
	IsCrit         bool    // 是否暴击（触发暴击倍率和暴击 VFX）

	// ── 效果类（指针，nil=不触发）──
	Splash *SplashEffect // 范围溅射：以命中点为圆心，对半径内其他敌人造成比例伤害
	Slow   *SlowEffect   // 减速：降低敌人移动速度
	Stun   *StunEffect   // 眩晕：冻结敌人移动
	Bleed  *BleedEffect  // 流血 DoT：持续物理伤害（tick 间隔由 balance.json combat.dotTickInterval.bleed 控制）
	Burn   *BleedEffect  // 灼烧 DoT：持续火焰伤害（结构同 Bleed 但独立计时和 tick 间隔）
	Bounce *BounceEffect // 弹射链：弹射物跳跃到附近敌人（每次跳跃伤害递减）
}

// SplashEffect 范围溅射伤害。
type SplashEffect struct {
	Radius float64 // 溅射半径（像素）
	Ratio  float64 // 伤害比例（相对弹射物伤害，0.4 = 40%）
}

// SlowEffect 减速效果。
type SlowEffect struct {
	Factor   float64 // 速度倍率（0.5 表示减至 50% 速度）
	Duration float64 // 持续时间（秒）
}

// StunEffect 眩晕效果（冻结敌人移动）。
type StunEffect struct {
	Duration float64 // 持续时间（秒）
}

// BleedEffect 持续流血伤害。
type BleedEffect struct {
	DPS      float64 // 每秒伤害
	Duration float64 // 持续时间（秒）
}

// BounceEffect 弹射链效果（弹射物命中后跳跃到附近敌人）。
// 弹射过程：命中目标 A → 搜索 Range 内最近未命中敌人 B → 发射新弹射物到 B → 重复。
// 每次弹射伤害 = SrcDamage × DamageRatio，最多弹射 MaxBounces 次。
type BounceEffect struct {
	MaxBounces  int     // 最大弹射次数（弹射计数器递减到 0 时停止）
	Range       float64 // 弹射搜索范围（像素，从当前命中点搜索下一个目标）
	DamageRatio float64 // 弹射伤害比例（相对来源塔伤害，0.8 = 80%）
	SrcDamage   float64 // 来源塔的当前伤害（快照值，弹射期间塔属性变化不影响已发射的弹射链）
}

// Ability 塔能力接口（所有能力必须实现）。
// 战斗管线在弹射物命中时遍历塔的 AllAbilities()，对每个已注册能力调用 OnHit。
// 能力通过 HitResult 返回效果，返回 nil 表示本次命中不触发任何效果。
type Ability interface {
	// Name 返回能力唯一标识符（与 abilities.json 中的 type 字段一致）。
	Name() string
	// OnHit 在弹射物命中敌人时调用。
	// 参数：t=发射塔, p=命中的弹射物, e=被命中的敌人。
	// 返回 nil 表示不触发效果（如概率未命中、目标已有同类 debuff 等）。
	OnHit(t *Tower, p *projectile.Projectile, e *enemy.Enemy) *HitResult
}

// Ticker 可选接口，用于需要每帧 tick 更新的能力（光环/区域/经济类）。
// pipeline 的 tower system 通过类型断言 if ticker, ok := ability.(Ticker) 检查，
// 仅实现 Ability 接口的能力不受影响（如 splash/bounce 只需 OnHit）。
type Ticker interface {
	OnTick(t *Tower, ctx *TickContext) *TickResult
}

// TickContext tick 调用时传入的上下文。
type TickContext struct {
	Enemies *enemy.Pool
	Towers  *Pool
	DT      float64 // 本帧时间步长（秒）
}

// TickResult tick 返回的临时效果。
// 字段为零值表示无对应效果。
type TickResult struct {
	GoldEarned int // 本 tick 获得的金币
}

// Registry 全局能力注册表（能力名 → 能力实例）。
// 由各能力文件的 init() 函数填充，程序运行后内容不变（事实上的只读）。
// 生产代码应通过 Lookup() 读取（语义更清晰）；测试代码可直接访问 Registry[] 做断言。
var Registry = map[string]Ability{}

// Register 向全局注册表添加一个能力（仅在 init() 中调用）。
func Register(a Ability) {
	Registry[a.Name()] = a
}

// Lookup 从注册表查找能力（推荐的只读访问方式）。
// 返回 ok=false 表示该能力名未注册（可能是配置错误或能力文件未 blank import）。
func Lookup(name string) (Ability, bool) {
	a, ok := Registry[name]
	return a, ok
}

// TowerAbility 塔与能力的绑定关系，支持自定义参数（用于 JSON 序列化/反序列化）。
// 目前主要用于旧配置兼容，新系统直接用 AbilitySlots[cat] = abilityType 字符串。
type TowerAbility struct {
	Name   string             `json:"name"`             // 能力名称（与 Registry 中的 key 一致）
	Params map[string]float64 `json:"params,omitempty"` // 自定义参数（可选，用于覆盖能力默认值）
}
