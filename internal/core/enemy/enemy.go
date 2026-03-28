// enemy.go — 敌人实体定义。
// 定义敌人的核心属性（位置、血量、速度、状态效果、类型等）及状态效果处理逻辑。
package enemy

import "defense2/internal/core/gamemap"

// Enemy 单个敌人实体。
type Enemy struct {
	ID         int             // 唯一标识（用于穿刺弹已命中检查）
	X, Y       float64         // 当前像素位置
	HP         float64         // 当前血量
	MaxHP      float64         // 最大血量
	Speed      float64         // 当前移动速度（像素/秒，受减速影响）
	BaseSpeed  float64         // 基础移动速度（无减速时的速度）
	Radius     float64         // 碰撞半径（像素）
	PathIndex  int             // 当前目标路径点索引
	Path       []gamemap.Point // 该敌人的行进路径（多路径地图时各敌人可能不同）
	ReachedEnd bool            // 是否已到达路径终点（基地）
	Active     bool            // 是否存活（对象池复用标记）
	Archetype  string          // 敌人原型标识（如 "normal"、"runner"、"tank"）
	Boss       bool            // 是否为 Boss
	Reward     int             // 击杀奖励金币
	StunTimer  float64         // 眩晕剩余时间（秒），>0 时无法移动
	SlowTimer  float64         // 减速剩余时间（秒）
	SlowFactor float64         // 减速倍率（0.5 表示半速）
	BleedTimer float64         // 流血剩余时间（秒）
	BleedDPS   float64         // 流血每秒伤害
	BurnTimer  float64         // 灼烧剩余时间（秒）
	BurnDPS    float64         // 灼烧每秒伤害
	ShieldHP   float64         // 护盾血量（吸收伤害直到耗尽）
	RootTimer  float64         // 定身剩余时间（秒）
	DisplayHP  float64         // 显示用血量（伤害拖尾缓慢衰减到实际 HP）
	Elite      bool            // 是否为精英怪
	HitFlash   float64         // 受击闪白剩余时间（秒，>0 时渲染白色叠加）
}

// TickStatusEffects 处理敌人身上的状态效果（减速、流血）。
// 眩晕在 movement.go 中处理。
func TickStatusEffects(e *Enemy, dt float64) {
	// 减速：倒计时归零后恢复基础速度
	if e.SlowTimer > 0 {
		e.SlowTimer -= dt
		e.Speed = e.BaseSpeed * e.SlowFactor
		if e.SlowTimer <= 0 {
			e.Speed = e.BaseSpeed
		}
	}

	// 流血：持续扣血
	if e.BleedTimer > 0 {
		e.BleedTimer -= dt
		e.HP -= e.BleedDPS * dt
	}

	// 灼烧：持续扣血
	if e.BurnTimer > 0 {
		e.BurnTimer -= dt
		e.HP -= e.BurnDPS * dt
	}

	// 定身：倒计时
	if e.RootTimer > 0 {
		e.RootTimer -= dt
	}

	// 受击闪白衰减
	if e.HitFlash > 0 {
		e.HitFlash -= dt
		if e.HitFlash < 0 {
			e.HitFlash = 0
		}
	}

	// DisplayHP 伤害拖尾衰减（每秒衰减 120% MaxHP）
	if e.DisplayHP <= 0 {
		e.DisplayHP = e.HP // 首次初始化
	}
	if e.DisplayHP > e.HP {
		e.DisplayHP -= e.MaxHP * dt * 1.2
		if e.DisplayHP < e.HP {
			e.DisplayHP = e.HP
		}
	} else {
		e.DisplayHP = e.HP // 治疗时瞬间跟上
	}
}
