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
	BurnTimer    float64         // 灼烧剩余时间（秒）
	BurnDPS      float64         // 灼烧每秒伤害
	DotTickTimer float64         // DoT 触发计时器（每 DotTickInterval 触发一次伤害）
	LastDotDmg   float64         // 上次 DoT tick 的伤害量（>0 时由 pipeline 弹浮字后清零）
	ZoneDmgAccum float64         // 区域能力（curseZone/poisonZone）每帧累积伤害，DotTick 时结算
	ShieldHP   float64         // 护盾血量（吸收伤害直到耗尽）
	RootTimer  float64         // 定身剩余时间（秒）
	DisplayHP  float64         // 显示用血量（伤害拖尾缓慢衰减到实际 HP）
	Elite      bool            // 是否为精英怪
	HitFlash   float64         // 受击闪白剩余时间（秒，>0 时渲染白色叠加）
	AnimCur    string          // 当前动画名（per-instance）
	AnimFrame  int             // 当前帧索引
	AnimTimer  float64         // 帧计时器
	AnimDone   bool            // 非循环动画是否播完

	// ── 死亡动画 ──
	DyingTimer    float64 // >0 means dying animation in progress (seconds remaining)
	DyingDuration float64 // total dying time (for progress calculation)

	// ── 伤害管线扩展字段 ──

	DamageCap        float64 // 单次伤害上限（0=无上限，如铁甲怪 60）
	DamageCapPercent float64 // 单次伤害百分比上限（0=无上限，如巨人 0.08=8%maxHP）
	Silenced         bool    // 是否被沉默（沉默时 DamageCap 失效）
	IsInvincible     bool    // 无敌状态（pure 伤害可穿透）
	IsDamageImmune   bool    // 伤害免疫（pure 伤害可穿透）
	IsUntargetable   bool    // 不可选中

	// 多重护盾系统
	Shields    []Shield    // 多层护盾列表（按剩余时间升序消耗）
	Thresholds []Threshold // HP阈值触发器列表

	// ── 控制减免 ──
	Tenacity        float64 // 韧性（0~1，减少控制效果持续时间）
	IsControlImmune bool    // 控制免疫
	IsStunImmune    bool    // 眩晕免疫
	IsSlowImmune    bool    // 减速免疫
	IsRootImmune    bool    // 定身免疫

	// ── 生命周期 ──
	Lifecycle *LifecycleHandlers // 生命周期回调

	// ── 行为 ──
	BerserkThreshold  float64 // 狂暴触发血量比例（如0.5=50%HP）
	BerserkSpeedScale float64 // 狂暴速度倍率
	BerserkTriggered  bool    // 狂暴是否已触发（一次性）
	RegenPerSec       float64 // 每秒回血量
	HealPower         float64 // 治疗光环治疗量
	HealRadius        float64 // 治疗光环范围
	HealInterval      float64 // 治疗光环间隔（秒）
	HealCooldown      float64 // 治疗光环当前冷却

	// ── 飞行 ──
	MovementType string // 移动类型（"ground"/"flying"）
}

// IsDying returns true if the enemy is playing its death animation.
// Dying enemies are still Active (for rendering) but should be skipped by
// targeting, collision, movement, and status-effect systems.
func (e *Enemy) IsDying() bool { return e.DyingTimer > 0 }

// MinSpeedRatio 全局减速下限：速度不低于初始速度的 20%。
const MinSpeedRatio = 0.2

// DotTickInterval DoT（流血/灼烧）伤害触发周期（秒）。
const DotTickInterval = 0.5

// TickStatusEffects 处理敌人身上的状态效果（减速、流血）。
// 眩晕在 movement.go 中处理。
func TickStatusEffects(e *Enemy, dt float64) {
	// 减速：倒计时归零后恢复基础速度（受全局减速下限约束）
	if e.SlowTimer > 0 {
		e.SlowTimer -= dt
		factor := e.SlowFactor
		if factor < MinSpeedRatio {
			factor = MinSpeedRatio
		}
		e.Speed = e.BaseSpeed * factor
		if e.SlowTimer <= 0 {
			e.Speed = e.BaseSpeed
		}
	}

	// DoT（流血/灼烧/区域伤害）按固定周期触发
	hasDot := e.BleedTimer > 0 || e.BurnTimer > 0 || e.ZoneDmgAccum > 0
	if hasDot {
		e.DotTickTimer -= dt
		if e.DotTickTimer <= 0 {
			e.DotTickTimer += DotTickInterval
			dotDmg := 0.0
			if e.BleedTimer > 0 {
				dotDmg += e.BleedDPS * DotTickInterval
			}
			if e.BurnTimer > 0 {
				dotDmg += e.BurnDPS * DotTickInterval
			}
			// 区域伤害（curseZone/poisonZone 每帧累积，tick 时一次性结算）
			if e.ZoneDmgAccum > 0 {
				dotDmg += e.ZoneDmgAccum
				e.ZoneDmgAccum = 0
			}
			e.HP -= dotDmg
			e.LastDotDmg = dotDmg
		}
		// 倒计时递减
		if e.BleedTimer > 0 {
			e.BleedTimer -= dt
		}
		if e.BurnTimer > 0 {
			e.BurnTimer -= dt
		}
	} else {
		e.DotTickTimer = 0
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
