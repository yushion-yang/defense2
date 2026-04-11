// ability.go — 塔能力系统核心。
// 定义能力接口、命中效果结构体、全局能力注册表。
// 具体能力实现在 abilities/ 子包中通过 init() 自注册。
package tower

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
)

// HitResult 描述弹射物命中时触发的能力效果。
type HitResult struct {
	BonusDamage float64       // 额外伤害
	Splash      *SplashEffect // 溅射效果（可选）
	Slow        *SlowEffect   // 减速效果（可选）
	Stun        *StunEffect   // 眩晕效果（可选）
	Bleed       *BleedEffect  // 流血效果（可选）
	Burn        *BleedEffect  // 灼烧效果（可选，结构同 Bleed 但独立计时）
	Bounce      *BounceEffect // 弹射效果（可选）
	IsCrit      bool          // 是否暴击
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

// BounceEffect 弹射链效果（弹射物跳跃到附近敌人）。
type BounceEffect struct {
	MaxBounces  int     // 最大弹射次数
	Range       float64 // 弹射搜索范围（像素）
	DamageRatio float64 // 弹射伤害比例（相对塔伤害，0.8 = 80%）
	SrcDamage   float64 // 来源塔的当前伤害（用于计算弹射伤害）
}

// Ability 塔能力接口。
type Ability interface {
	// Name 返回能力唯一标识符。
	Name() string
	// OnHit 在弹射物命中敌人时调用，返回触发的效果（无效果返回 nil）。
	OnHit(t *Tower, p *projectile.Projectile, e *enemy.Enemy) *HitResult
}

// Ticker 可选接口，用于需要每帧 tick 更新的能力（光环、区域、经济类）。
// 通过类型断言检查，不影响仅实现 OnHit 的能力。
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
// 生产代码应通过 Lookup() 读取；测试代码可直接访问。
var Registry = map[string]Ability{}

// Register 向全局注册表添加一个能力。
func Register(a Ability) {
	Registry[a.Name()] = a
}

// Lookup 从注册表查找能力（推荐的只读访问方式）。
func Lookup(name string) (Ability, bool) {
	a, ok := Registry[name]
	return a, ok
}

// TowerAbility 塔与能力的绑定关系，支持自定义参数。
type TowerAbility struct {
	Name   string             `json:"name"`             // 能力名称
	Params map[string]float64 `json:"params,omitempty"` // 自定义参数（可选）
}
