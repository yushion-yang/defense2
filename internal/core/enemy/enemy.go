// enemy.go — 敌人实体定义。
// 定义敌人的核心属性（位置、血量、速度、状态效果、类型等）及状态效果处理逻辑。
package enemy

import (
	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/gamemap"
)

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

// StatusEffects 敌人身上的状态效果。
// CC/DoT/debuff 由 BuffList 管理，此结构体保留 pipeline 输出、zone 计时和原型属性。
type StatusEffects struct {
	// DoT pipeline 输出
	LastDotDmg   float64 // 上次 DoT tick 的伤害量（>0 时由 pipeline 弹浮字后清零）
	ZoneDmgAccum float64 // 区域能力（curseZone/poisonZone）每帧累积伤害，DotTick 时结算
	ZoneDotTimer float64 // 区域伤害 DoT tick 计时器（与 BuffList.dotTimer 独立）

	// 状态标记
	Silenced        bool // 是否被沉默（沉默时 DamageCap 失效）
	AbilitySilenced bool // 当前帧是否被沉默（每帧重置）
	IsInvincible    bool // 无敌状态（pure 伤害可穿透）
	IsDamageImmune  bool // 伤害免疫（pure 伤害可穿透）
	IsUntargetable  bool // 不可选中

	// 控制减免（原型属性，Spawn 时设置，不由 BuffList 管理）
	Tenacity        float64 // 韧性（0~1，减少控制效果持续时间）
	IsControlImmune bool    // 控制免疫
	IsStunImmune    bool    // 眩晕免疫
	IsSlowImmune    bool    // 减速免疫
	IsRootImmune    bool    // 定身免疫
}

// VisualState 敌人的视觉/渲染状态（闪光、飘字、血条拖尾）。
// 嵌入 Enemy 中，外部代码通过 e.HitFlash 等直接访问。
type VisualState struct {
	HitFlash     float64 // 受击闪白剩余时间（秒，>0 时渲染白色叠加）
	BlockFlash   float64 // 弹幕盾格挡闪光
	DodgeFlash   float64 // 闪避残影
	ArmorSpark   float64 // 装甲火花
	PurgeFlash   float64 // 净化脉冲
	DamageCapHit float64 // 坚韧触发闪光
	DisplayHP    float64 // 显示用血量（伤害拖尾缓慢衰减到实际 HP）

	// 飘字系统
	FloatText      string  // 当前飘字内容（空=无）
	FloatTextTimer float64 // 飘字剩余时间
	FloatTextR     uint8   // 飘字颜色
	FloatTextG     uint8
	FloatTextB     uint8
}

// AbilityFields 敌人能力系统字段（防御/移动/攻击/死亡/净化等）。
// 嵌入 Enemy 中，外部代码通过 e.DamageCap 等直接访问。
type AbilityFields struct {
	// defense
	DamageCap             float64 // 单次伤害上限（0=无上限，如铁甲怪 60）
	DamageCapPercent      float64 // 单次伤害百分比上限（0=无上限，如巨人 0.08=8%maxHP）
	ProjectileBlockChance float64 // 弹幕盾：阻挡弹射物概率（0=无）
	ArmorFlat             float64 // 装甲：每次受击固定减免
	EvasionChance         float64 // 闪避：完全闪避概率（0=无）
	DamageReduceRatio     float64 // 受伤减免比例（0~1，由 buff 模板设置）

	// movement
	DashSpeedBoost float64 // 受击冲刺：速度提升比例
	DashDuration   float64 // 受击冲刺：提升持续时间（秒）
	DashCooldown   float64 // 受击冲刺：冷却时间（秒）
	DashCooldownT  float64 // 受击冲刺：当前冷却倒计时
	DashActiveT    float64 // 受击冲刺：当前激活倒计时
	PhaseDuration  float64 // 相位偏移：免伤持续时间（秒）
	PhaseCooldown  float64 // 相位偏移：冷却时间（秒）
	PhaseTimer     float64 // 相位偏移：当前计时（>0 免伤中, <0 冷却中）
	PhaseActive    bool    // 相位偏移：当前是否免伤

	// offense — 削强
	StrDrainRatio    float64 // 减益比例（0.5 = -50%）
	StrDrainInterval float64 // 施加间隔（秒）
	StrDrainDuration float64 // 减益持续时间（秒）
	StrDrainTimer    float64 // 冷却倒计时
	StrDrainTargetRC [2]int  // 连接的塔 [row,col]（[0,0]=无连接）
	StrDrainActiveT  float64 // 减益剩余持续时间（>0 表示连接中）

	// death
	DeathSpawnCount int    // 死亡召唤：召唤数量（0=不召唤）
	DeathSpawnArch  string // 死亡召唤：召唤原型（默认 "normal"）

	// resist
	PurgeInterval  float64 // 净化：清除间隔（秒，0=无净化）
	PurgeImmuneDur float64 // 净化：清除后免疫持续时间（秒）
	PurgeTimer     float64 // 净化：当前计时

	// 能力系统
	AbilityIDs []string // 装配的能力类型 ID 列表（用于 HUD 展示）
}

// Enemy 单个敌人实体。
type Enemy struct {
	// ── 身份 ──
	ID        int    // 唯一标识（用于穿透弹已命中检查）
	Active    bool   // 是否存活（对象池复用标记）
	Archetype string // 敌人原型标识（如 "normal"、"runner"、"tank"）
	SpriteDir string // 精灵目录名（加载贴图用，可与 Archetype 不同）
	Boss      bool   // 是否为 Boss
	IsDummy   bool   // 是否为木桩怪（不移动）

	// ── 核心属性 ──
	X, Y      float64         // 当前像素位置
	HP        float64         // 当前血量
	MaxHP     float64         // 最大血量
	Speed     float64         // 当前移动速度（像素/秒，受减速影响）
	BaseSpeed float64         // 基础移动速度（无减速时的速度）
	Radius    float64         // 碰撞半径（像素）
	PathIndex int             // 当前目标路径点索引
	Path      []gamemap.Point // 该敌人的行进路径（多路径地图时各敌人可能不同）

	// ── 经济 ──
	Reward      int     // 击杀奖励金币
	RewardScale float64 // 原型奖励倍率（如 tank=1.35, runner=0.72）
	ReachedEnd  bool    // 是否已到达路径终点（基地）

	// ── 时间/动画 ──
	Age       float64 // 存活时间（秒），用于出生保护期
	AnimCur   string  // 当前动画名（per-instance）
	AnimFrame int     // 当前帧索引
	AnimTimer float64 // 帧计时器
	AnimDone  bool    // 非循环动画是否播完

	// ── 出生动画 ──
	SpawnTimer    float64 // >0 means spawning animation in progress (seconds remaining)
	SpawnDuration float64 // total spawn animation time (for progress calc)

	// ── 死亡动画 ──
	DyingTimer    float64 // >0 means dying animation in progress (seconds remaining)
	DyingDuration float64 // total dying time (for progress calculation)

	// ── HP阈值触发器 ──
	Thresholds []Threshold // HP阈值触发器列表

	// ── 生命周期 ──
	Lifecycle *LifecycleHandlers // 生命周期回调

	// ── Buff 系统 ──
	Buffs *buff.BuffList // 统一状态效果管理（CC/DoT/debuff）

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

	// ── 嵌入子结构体 ──
	StatusEffects
	VisualState
	AbilityFields
}

// IsSpawning returns true if the enemy is playing its spawn-in animation.
// Spawning enemies are Active and visible but should be skipped by targeting
// and combat systems (invulnerable during materialisation).
func (e *Enemy) IsSpawning() bool { return e.SpawnTimer > 0 }

// IsDying returns true if the enemy is playing its death animation.
// Dying enemies are still Active (for rendering) but should be skipped by
// targeting, collision, movement, and status-effect systems.
func (e *Enemy) IsDying() bool { return e.DyingTimer > 0 }

// SetFloatText 设置飘字（会覆盖已有的飘字）。
func (e *Enemy) SetFloatText(text string, r, g, b uint8) {
	e.FloatText = text
	e.FloatTextTimer = 0.6
	e.FloatTextR = r
	e.FloatTextG = g
	e.FloatTextB = b
}

// ── BuffList 查询方法 ──

// IsStunned returns true if the enemy has an active stun buff.
func (e *Enemy) IsStunned() bool { return e.Buffs != nil && e.Buffs.Has("stun") }

// IsSlowed returns true if the enemy has an active slow buff.
func (e *Enemy) IsSlowed() bool { return e.Buffs != nil && e.Buffs.Has("slow") }

// IsRooted returns true if the enemy has an active root buff.
func (e *Enemy) IsRooted() bool { return e.Buffs != nil && e.Buffs.Has("root") }

// IsBleeding returns true if the enemy has an active bleed buff.
func (e *Enemy) IsBleeding() bool { return e.Buffs != nil && e.Buffs.Has("bleed") }

// IsBurning returns true if the enemy has an active burn buff.
func (e *Enemy) IsBurning() bool { return e.Buffs != nil && e.Buffs.Has("burn") }

// IsWeakened returns true if the enemy has an active weaken buff.
func (e *Enemy) IsWeakened() bool { return e.Buffs != nil && e.Buffs.Has("weaken") }

// HasControlImmunity returns true if the enemy has a temporary controlImmune buff.
func (e *Enemy) HasControlImmunity() bool { return e.Buffs != nil && e.Buffs.Has("controlImmune") }

// GetSlowFactor returns the slow factor from BuffList (1.0 = no slow).
func (e *Enemy) GetSlowFactor() float64 {
	if e.Buffs == nil {
		return 1.0
	}
	b, ok := e.Buffs.Get("slow")
	if !ok {
		return 1.0
	}
	return b.Value
}

// GetWeakenAmplify returns the weaken damage amplify value from BuffList (0 = no weaken).
func (e *Enemy) GetWeakenAmplify() float64 {
	if e.Buffs == nil {
		return 0
	}
	b, ok := e.Buffs.Get("weaken")
	if !ok {
		return 0
	}
	return b.Value
}

// MinSpeedRatio 返回全局减速下限（从 balance.json 实时读取，不再冻结于 init 时刻）。
func MinSpeedRatio() float64 { return config.GlobalBalance().Combat.MinSpeedRatio }

// DotTickInterval 返回 DoT 伤害触发周期（从 balance.json 实时读取，不再冻结于 init 时刻）。
func DotTickInterval() float64 { return config.GlobalBalance().Combat.DotTickInterval }

// TickStatusEffects 处理敌人身上的状态效果。
// BuffList.Tick 统一处理 CC/DoT/debuff 的倒计时和过期清理。
// Legacy 字段从 BuffList 同步，保证下游 render/stage/autoplay 代码向后兼容。
func TickStatusEffects(e *Enemy, dt float64) {
	bal := config.GlobalBalance()
	e.Age += dt

	// ── BuffList tick: 统一处理所有 buff 倒计时和过期 ──
	if e.Buffs != nil {
		// Track pre-tick state to detect buff expiry
		wasSlowed := e.Buffs.Has("slow")
		hadControlImmune := e.Buffs.Has("controlImmune")

		// DoT damage via BuffList (must call BEFORE Tick so expiring DoTs still deal damage)
		dotInterval := bal.Combat.DotTickInterval
		buffDotDmg := e.Buffs.TickDoT(dt, dotInterval)

		// Zone damage uses its own timer (not a buff)
		if e.ZoneDmgAccum > 0 {
			if e.ZoneDotTimer <= 0 {
				e.ZoneDotTimer = dotInterval
			}
			e.ZoneDotTimer -= dt
			if e.ZoneDotTimer <= 0 {
				e.ZoneDotTimer += dotInterval
				buffDotDmg += e.ZoneDmgAccum
				e.ZoneDmgAccum = 0
			}
		} else if !e.Buffs.Has("bleed") && !e.Buffs.Has("burn") && !e.Buffs.Has("poison") {
			e.ZoneDotTimer = 0
		}

		if buffDotDmg > 0 {
			e.LastDotDmg = buffDotDmg
		}

		// Tick all buff timers (decrements Remaining, removes expired)
		e.Buffs.Tick(dt)

		// ── Post-tick: handle buff-driven side effects ──

		// Slow: keep Speed in sync with slow buff
		if b, ok := e.Buffs.Get("slow"); ok {
			factor := b.Value
			if factor < bal.Combat.MinSpeedRatio {
				factor = bal.Combat.MinSpeedRatio
			}
			e.Speed = e.BaseSpeed * factor
		} else if wasSlowed {
			// Slow just expired — restore speed
			e.Speed = e.BaseSpeed
		}

		// ControlImmune: when buff expires, clear temporary immunity flags
		// (archetype immunity set in Spawn is never cleared here)
		if hadControlImmune && !e.Buffs.Has("controlImmune") {
			e.IsControlImmune = false
			e.IsStunImmune = false
			e.IsSlowImmune = false
			e.IsRootImmune = false
		}
	}

	// 受击闪白衰减
	if e.HitFlash > 0 {
		e.HitFlash -= dt
		if e.HitFlash < 0 {
			e.HitFlash = 0
		}
	}
	// 能力视觉特效衰减
	if e.BlockFlash > 0 {
		e.BlockFlash -= dt
	}
	if e.DodgeFlash > 0 {
		e.DodgeFlash -= dt
	}
	if e.ArmorSpark > 0 {
		e.ArmorSpark -= dt
	}
	if e.DamageCapHit > 0 {
		e.DamageCapHit -= dt
	}
	if e.FloatTextTimer > 0 {
		e.FloatTextTimer -= dt
		if e.FloatTextTimer <= 0 {
			e.FloatText = ""
		}
	}
	if e.PurgeFlash > 0 {
		e.PurgeFlash -= dt
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
