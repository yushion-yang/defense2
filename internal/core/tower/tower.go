// tower.go — 塔实体定义。
// 定义已放置塔的核心属性：位置、攻击参数、能力列表等。
package tower

import (
	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/strength"
)

// StrengthBuyCost 购买 10 点强度的金币花费。
const StrengthBuyCost = 10

// AttackStyle 攻击方式标识。
type AttackStyle = string

const (
	StyleProjectile AttackStyle = "projectile" // 标准追踪弹
	StyleWideBeam   AttackStyle = "wideBeam"   // 宽光束（贯穿）
	StyleScatter    AttackStyle = "scatter"    // 锥形散射
	StyleSpinAoE    AttackStyle = "spin_aoe"   // 旋转范围伤害
	StyleRadial     AttackStyle = "radial"     // 360度环射穿刺弹

	// 废弃：保留常量供旧配置兼容，运行时映射到 projectile
	StyleLaser  AttackStyle = "laser"  // 废弃 → 高弹速 projectile
	StyleCharge AttackStyle = "charge" // 废弃 → 低攻速 projectile + momentum
	StyleAuraDot AttackStyle = "aura_dot" // 废弃 → 合并到 spinAoe
)

// Tower 已放置的塔实体。
type Tower struct {
	X, Y        float64  // 像素中心坐标
	Row, Col    int      // 所在网格行列
	Range       float64  // 攻击范围（像素）
	Damage      float64  // 单发伤害
	AttackSpeed float64  // 攻击速度（次/秒）
	FireTimer   float64  // 下一次射击倒计时（秒）
	Cost        int      // 总投入金币（建造+升级累计，用于计算卖价）
	Key         string   // 塔类型标识（如 "basic"、"splash"）
	InstanceKey string   // 实例唯一标识（"key_row_col"，用于弹射物来源匹配）
	Label       string   // 显示名称
	SpriteKey   string   // 精灵资源标识（根据攻击能力变化：sentinel/fortress/shotgun/...）
	Active      bool     // 是否存活（对象池复用标记）
	Abilities   []string // 该塔拥有的能力名称列表（兼容旧配置）
	// 六大类别能力槽 [0]=攻击模式 [1]=CC [2]=命中加伤 [3]=光环 [4]=DoT [5]=范围效果
	AbilitySlots [6]string
	UnlockOrder  [6]int // 能力类别解锁顺序（运行时随机）
	Level        int    // 塔等级 (1=基础, 2~7=升级)
	// 属性档位标签（展示用）
	DamageTier string
	SpeedTier  string
	RangeTier  string
	Color       [3]uint8 // 显示颜色 RGB
	Faction     string   // 阵营标识（用于资源路径）
	FireAnim    float64  // 射击动画计时器（射击时设为 0.15，逐帧衰减）
	Angle       float64  // 朝向角度（弧度，0=向上，顺时针）
	// 战力缩放参数（来自 JSON 配置，放置后不变）
	// 公式: attr = Base + Potential * (strength / 100)
	BaseDamage      float64 // JSON baseDamage
	BaseRange       float64 // JSON baseRange
	BaseSpeed       float64 // JSON baseAttackSpeed
	PotentialDamage float64 // JSON potentialDamage
	PotentialRange  float64 // JSON potentialRange
	PotentialSpeed  float64 // JSON potentialAttackSpeed

	// 攻击方式
	AttackStyleID   AttackStyle // 攻击方式（"projectile"/"laser"/...）
	ProjectileSpeed float64     // 弹射物速度（px/s，0=默认300）

	// Charge 运行时状态（由 handler_charge 管理）
	ChargeProgress float64 // 蓄力进度 0-1
	ChargeReady    bool    // 蓄力完成

	// SpinAoE 运行时状态
	SpinAngle  float64 // 旋转角度（弧度）
	SpinActive float64 // 旋转激活计时器

	// AuraDot 运行时状态
	AuraPulse float64 // 脉冲动画计时

	// GoldPassive 运行时状态
	GoldCooldown float64 // 被动产金冷却计时器

	// 分支特化
	Branch string // 分支特化标识（空=未特化，一次性选择）

	// 战力系统
	Strength *strength.StrengthData // 战力运行时数据
	Buffs    []TowerBuff            // 当前生效的 buff 列表（含来源/描述/时长）

	// 光环加成（每帧由 Ticker 能力重置+重算）
	CritBonus float64 // 暴击光环加成的暴击率（由 critAura 设置）

	// 索敌锁定
	Target              *enemy.Enemy // 当前锁定目标
	LastPercentHpTarget int          // 上次触发 percentHpDamage 的敌人 ID（切换目标首击）

	// 叠伤计数（stackDamage 能力：连续命中同目标递增）
	StackTarget int // 当前叠伤目标 ID
	StackCount  int // 叠伤层数

	// momentum (蓄势) 能力计数
	MomentumCount int // 当前攻击次数计数

	// 能力选项缓存：key=category index(0-5), value=3 个候选 AbilityDef
	// 建塔时根据全局 wavesCleared 一次性 roll 所有已解锁位；新波次解锁时追加 roll。
	PendingChoices map[int][]config.AbilityDef

	// 建造/出售动画
	BuildAnim float64 // >0 during build-in animation (seconds remaining, starts at 0.3)
	SellAnim  float64 // >0 during sell-out animation (seconds remaining, starts at 0.25)
	Selling   bool    // true when tower is in sell animation (skip gameplay logic)
}

// BuyStrength 花费金币购买 10 点永久强度。返回实际花费。
// 需要塔已挂载 StrengthData（通过 Strength 字段）。
func (t *Tower) BuyStrength() int {
	cost := StrengthBuyCost
	t.Cost += cost // 累计投入（影响卖价）
	// 通过 strength 包的接口添加永久加成（由调用方做类型断言）
	return cost
}

// RecalcStats 根据当前强度重算 Damage/AttackSpeed/Range。
// 公式与 AbilityDef 一致: value = base + potential * (strength / 100)
// 无 Strength 时使用强度100的默认值（base + potential）。
func (t *Tower) RecalcStats() {
	ratio := 1.0 // 默认强度100
	if t.Strength != nil {
		ratio = t.Strength.Effective() / 100.0
	}
	t.Damage = t.BaseDamage + t.PotentialDamage*ratio
	t.AttackSpeed = t.BaseSpeed + t.PotentialSpeed*ratio
	t.Range = t.BaseRange + t.PotentialRange*ratio
	t.CritBonus = 0 // 每帧重置，由 critAura OnTick 重新设置
}

// AllAbilities 返回所有生效能力（合并 AbilitySlots 和旧 Abilities）。
func (t *Tower) AllAbilities() []string {
	seen := map[string]bool{}
	var result []string
	for _, a := range t.AbilitySlots {
		if a != "" && !seen[a] {
			seen[a] = true
			result = append(result, a)
		}
	}
	for _, a := range t.Abilities {
		if a != "" && !seen[a] {
			seen[a] = true
			result = append(result, a)
		}
	}
	return result
}

// ResolveAttackStyle 根据 AbilitySlots[0]（攻击模式）决定实际攻击方式。
// 如果 AbilitySlots[0] 为空则回退到 AttackStyleID（兼容旧配置）。
func (t *Tower) ResolveAttackStyle() AttackStyle {
	pattern := t.AbilitySlots[0]
	switch pattern {
	case "scatter":
		return StyleScatter
	case "wideBeam":
		return StyleWideBeam
	case "spinAoe":
		return StyleSpinAoE
	case "radial":
		return StyleRadial
	default:
		// pierce/bounce/splash/multiTarget/空 → 都用 projectile（pierce 通过弹射物标志实现）
		if t.AttackStyleID != "" {
			switch t.AttackStyleID {
			case StyleLaser, StyleCharge:
				return StyleProjectile
			case StyleAuraDot:
				return StyleSpinAoE
			}
			return t.AttackStyleID
		}
		return StyleProjectile
	}
}

// abilitySpriteMap 攻击能力 → 精灵资源标识映射。
var abilitySpriteMap = map[string]string{
	"enhance":     "fortress",
	"scatter":     "shotgun",
	"wideBeam":    "prism",
	"spinAoe":     "cyclone",
	"pierce":      "railgun",
	"bounce":      "ricochet",
	"splash":      "mortar",
	"multiTarget": "hydra",
	"radial":      "nova",
}

// abilitySpriteLabels 精灵标识 → 中文名映射。
var abilitySpriteLabels = map[string]string{
	"sentinel": "哨兵",
	"fortress": "堡垒",
	"shotgun":  "霰弹",
	"prism":    "棱光",
	"cyclone":  "旋刃",
	"railgun":  "穿甲",
	"ricochet": "链弹",
	"mortar":   "轰炸",
	"hydra":    "多管",
	"nova":     "星爆",
}

// AbilitySpriteKey 根据攻击能力类型返回精灵资源标识。
func AbilitySpriteKey(abilityType string) string {
	if key, ok := abilitySpriteMap[abilityType]; ok {
		return key
	}
	return "sentinel"
}

// SpriteLabelFor 返回精灵标识对应的中文名称。
func SpriteLabelFor(spriteKey string) string {
	if label, ok := abilitySpriteLabels[spriteKey]; ok {
		return label
	}
	return "哨兵"
}

// DPS 返回当前每秒伤害。
func (t *Tower) DPS() float64 {
	return t.Damage * t.AttackSpeed
}
