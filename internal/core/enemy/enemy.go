// enemy.go — 敌人实体定义。
// 定义敌人的核心属性（位置、血量、速度、状态效果、类型等）及状态效果处理逻辑。
package enemy

import "defense2/internal/core/gamemap"

// Threshold HP阈值触发器。
// 当敌人 HP 比例降到 Ratio 以下时触发一次。
type Threshold struct {
	Type      string  // 触发器类型标识
	Ratio     float64 // 触发比例（如 0.5 = 50% HP）
	Triggered bool    // 是否已触发
}

// AddThreshold 注册一个 HP 阈值触发器。
func AddThreshold(e *Enemy, typ string, ratio float64) {
	if e.Thresholds == nil {
		e.Thresholds = make([]Threshold, 0, 4)
	}
	e.Thresholds = append(e.Thresholds, Threshold{
		Type:  typ,
		Ratio: ratio,
	})
}

// CheckThresholds 检查并返回本次伤害触发的阈值列表。
func CheckThresholds(e *Enemy) []Threshold {
	if len(e.Thresholds) == 0 || e.MaxHP <= 0 {
		return nil
	}
	ratio := e.HP / e.MaxHP
	var triggered []Threshold
	for i := range e.Thresholds {
		th := &e.Thresholds[i]
		if !th.Triggered && ratio <= th.Ratio {
			th.Triggered = true
			triggered = append(triggered, *th)
		}
	}
	return triggered
}

// Enemy 单个敌人实体。
type Enemy struct {
	ID           int             // 唯一标识（用于穿刺弹已命中检查）
	X, Y         float64         // 当前像素位置
	HP           float64         // 当前血量
	MaxHP        float64         // 最大血量
	Speed        float64         // 当前移动速度（像素/秒，受减速影响）
	BaseSpeed    float64         // 基础移动速度（无减速时的速度）
	Radius       float64         // 碰撞半径（像素）
	PathIndex    int             // 当前目标路径点索引
	Path         []gamemap.Point // 该敌人的行进路径（多路径地图时各敌人可能不同）
	ReachedEnd   bool            // 是否已到达路径终点（基地）
	Active       bool            // 是否存活（对象池复用标记）
	Archetype    string          // 敌人原型标识（如 "normal"、"runner"、"tank"）
	Boss         bool            // 是否为 Boss
	Reward       int             // 击杀奖励金币
	RewardScale  float64         // 原型奖励倍率（如 tank=1.35, runner=0.72）
	StunTimer    float64         // 眩晕剩余时间（秒），>0 时无法移动
	SlowTimer    float64         // 减速剩余时间（秒）
	SlowFactor   float64         // 减速倍率（0.5 表示半速）
	BleedTimer   float64         // 流血剩余时间（秒）
	BleedDPS     float64         // 流血每秒伤害
	PoisonTimer  float64         // 中毒剩余时间（秒）
	PoisonDPS    float64         // 中毒每秒伤害
	BurnTimer    float64         // 灼烧剩余时间（秒）
	BurnDPS      float64         // 灼烧每秒伤害
	DotTickTimer float64         // DoT 触发计时器（每 DotTickInterval 触发一次伤害）
	LastDotDmg   float64         // 上次 DoT tick 的伤害量（>0 时由 pipeline 弹浮字后清零）
	ZoneDmgAccum float64         // 区域能力（curseZone/poisonZone）每帧累积伤害，DotTick 时结算
	RootTimer    float64         // 定身剩余时间（秒）
	DisplayHP    float64         // 显示用血量（伤害拖尾缓慢衰减到实际 HP）
	Elite        bool            // 是否为精英怪
	HitFlash     float64         // 受击闪白剩余时间（秒，>0 时渲染白色叠加）
	Age          float64         // 存活时间（秒），用于出生保护期
	AnimCur      string          // 当前动画名（per-instance）
	AnimFrame    int             // 当前帧索引
	AnimTimer    float64         // 帧计时器
	AnimDone     bool            // 非循环动画是否播完

	// ── 死亡动画 ──
	DyingTimer    float64 // >0 means dying animation in progress (seconds remaining)
	DyingDuration float64 // total dying time (for progress calculation)

	// ── 伤害管线扩展字段 ──

	DamageCap          float64 // 单次伤害上限（0=无上限，如铁甲怪 60）
	DamageCapPercent   float64 // 单次伤害百分比上限（0=无上限，如巨人 0.08=8%maxHP）
	Silenced           bool    // 是否被沉默（沉默时 DamageCap 失效）
	DamageAmplify      float64 // 受伤增加倍率（weaken/weakenZone 施加）
	DamageAmplifyTimer float64 // weaken OnHit 的持续时间（秒），zone 型每帧由区域重设
	IsInvincible       bool    // 无敌状态（pure 伤害可穿透）
	IsDamageImmune     bool    // 伤害免疫（pure 伤害可穿透）
	IsUntargetable     bool    // 不可选中

	Thresholds []Threshold // HP阈值触发器列表

	// ── 控制减免 ──
	Tenacity            float64 // 韧性（0~1，减少控制效果持续时间）
	ControlImmuneTimer  float64 // 控制免疫剩余时间（秒，>0 时免疫所有控制效果）
	IsControlImmune     bool    // 控制免疫
	IsStunImmune    bool    // 眩晕免疫
	IsSlowImmune    bool    // 减速免疫
	IsRootImmune    bool    // 定身免疫

	// ── 生命周期 ──
	Lifecycle *LifecycleHandlers // 生命周期回调

	// ── 行为 ──
	Behavior          string  // 行为类型标识（"healer"/"stealth"/"splitter"/"buffer"/"regenerator"/""）
	BerserkThreshold  float64 // 狂暴触发血量比例（如0.5=50%HP）
	BerserkSpeedScale float64 // 狂暴速度倍率
	BerserkTriggered  bool    // 狂暴是否已触发（一次性）
	RegenPerSec       float64 // 每秒回血量
	HealPower         float64 // 治疗光环治疗量
	HealRadius        float64 // 治疗光环范围
	HealInterval      float64 // 治疗光环间隔（秒）
	HealCooldown      float64 // 治疗光环当前冷却

	// ── 隐身 ──
	Stealthed    bool    // 当前是否隐身
	StealthTimer float64 // 隐身剩余持续时间（秒）

	// ── 分裂 ──
	SplitCount      int     // 死亡分裂子体数量（0=不分裂）
	SplitScale      float64 // 子体血量倍率（相对父体 MaxHP）
	SplitHPRatio    float64 // 子体 HP 占父体 MaxHP 的比例
	SplitSpeedScale float64 // 子体速度倍率

	// ── 传送 ──
	TeleportInterval float64 // 传送间隔（秒，0=不传送）
	TeleportSkip     int     // 每次传送跳过的路径段数
	TeleportTimer    float64 // 传送冷却倒计时

	// ── 旗手光环 ──
	BuffRadius  float64 // 光环加速范围（像素）
	BuffAmount  float64 // 光环移速加成（如 0.2 = +20%）
	SpeedBuff   float64 // 当前帧受到的光环加速值（由 buffer 每帧写入，movement 读取）
	AuraRange   float64 // 光环范围（像素，0=无光环）
	AuraSpeedUp float64 // 光环加速比例（如 0.2 = +20%）

	// ── 减伤 ──
	DamageReduceRatio float64 // 受伤减免比例（0~1，由 buff 模板设置）

	// ── 反伤 ──
	ReflectPercent float64 // 反伤比例（0~1，0=无反伤）

	// ── 复活 ──
	ReviveHPPercent float64 // 复活时 HP 占 MaxHP 比例（0=不复活）
	ReviveUsed      bool    // 是否已使用过复活

	// ── Boss 行为 ──
	BossData *BossState // Boss 专属行为状态（非 Boss 为 nil）

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
	e.Age += dt

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

	// DoT（流血/灼烧/中毒/区域伤害）按固定周期触发
	hasDot := e.BleedTimer > 0 || e.BurnTimer > 0 || e.PoisonTimer > 0 || e.ZoneDmgAccum > 0
	if hasDot {
		// 首次施加 DOT 时初始化计时器，不立即触发（保证 3s/0.5s = 6 次）
		if e.DotTickTimer <= 0 {
			e.DotTickTimer = DotTickInterval
		}
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
			if e.PoisonTimer > 0 {
				dotDmg += e.PoisonDPS * DotTickInterval
			}
			// 区域伤害（curseZone/poisonZone 每帧累积，tick 时一次性结算）
			if e.ZoneDmgAccum > 0 {
				dotDmg += e.ZoneDmgAccum
				e.ZoneDmgAccum = 0
			}
			// 无敌/伤害免疫时跳过 HP 扣减（DoT 仍正常倒计时以便状态图标消失）
			if !e.IsInvincible && !e.IsDamageImmune {
				e.HP -= dotDmg
				e.LastDotDmg = dotDmg
			}
		}
		// 倒计时递减
		if e.BleedTimer > 0 {
			e.BleedTimer -= dt
		}
		if e.BurnTimer > 0 {
			e.BurnTimer -= dt
		}
		if e.PoisonTimer > 0 {
			e.PoisonTimer -= dt
		}
	} else {
		e.DotTickTimer = 0
	}

	// 虚弱(weaken OnHit)：倒计时归零后清除增伤
	if e.DamageAmplifyTimer > 0 {
		e.DamageAmplifyTimer -= dt
		if e.DamageAmplifyTimer <= 0 {
			e.DamageAmplifyTimer = 0
			e.DamageAmplify = 0
		}
	}

	// 定身：倒计时
	if e.RootTimer > 0 {
		e.RootTimer -= dt
	}

	// 控制免疫：倒计时归零后清除免疫标志
	if e.ControlImmuneTimer > 0 {
		e.ControlImmuneTimer -= dt
		if e.ControlImmuneTimer <= 0 {
			e.ControlImmuneTimer = 0
			e.IsControlImmune = false
			e.IsStunImmune = false
			e.IsSlowImmune = false
		}
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
