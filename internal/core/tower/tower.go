// tower.go — 塔实体定义。
// 定义已放置塔的核心属性：位置、攻击参数、能力列表等。
package tower

import (
	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/enemy"
	"defense2/internal/core/strength"
)

// StrengthBuyCost 返回购买强度的金币花费（从 balance.json 实时读取）。
func StrengthBuyCost() int { return config.GlobalBalance().Tower.StrengthBuyCost }

// AttackStyle 攻击方式标识。
type AttackStyle = string

const (
	StyleProjectile AttackStyle = "projectile" // 标准追踪弹
	StyleWideBeam   AttackStyle = "wideBeam"   // 宽光束（贯穿）
	StyleScatter    AttackStyle = "scatter"    // 锥形散射
	StyleSpinAoE    AttackStyle = "spin_aoe"   // 旋转范围伤害
	StyleRadial     AttackStyle = "radial"     // 360度环射弹
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
	Color      [3]uint8 // 显示颜色 RGB
	FireAnim   float64  // 射击动画计时器（射击时设为 0.15，逐帧衰减）
	Angle      float64  // 朝向角度（弧度，0=向上，顺时针）
	// 战力缩放参数（来自 JSON 配置，放置后不变）
	// 公式: attr = Base + Potential * (strength / 100)
	BaseDamage      float64 // JSON baseDamage
	BaseRange       float64 // JSON baseRange
	BaseSpeed       float64 // JSON baseAttackSpeed
	PotentialDamage float64 // JSON potentialDamage
	PotentialRange  float64 // JSON potentialRange
	PotentialSpeed  float64 // JSON potentialAttackSpeed

	// 攻击方式
	AttackStyleID   AttackStyle // 攻击方式（"projectile"/"wideBeam"/...）
	ProjectileSpeed float64     // 弹射物速度（px/s，0=默认300）

	// SpinAoE 运行时状态
	SpinAngle  float64 // 旋转角度（弧度）
	SpinActive float64 // 旋转激活计时器

	// GoldPassive 运行时状态
	GoldCooldown float64 // 被动产金冷却计时器

	// 战力系统
	Strength *strength.StrengthData // 战力运行时数据
	Buffs    *buff.BuffList         // 当前生效的 buff 列表（统一 BuffList 容器）

	// 光环加成（由 RecalcStats 从 BuffList 读取）
	CritBonus float64 // 暴击光环加成的暴击率（aura:crit）
	DamageAmp float64 // 伤害增幅（aura:damageAmp，所有伤害输出 ×(1+DamageAmp)）

	// 索敌锁定
	Target              *enemy.Enemy // 当前锁定目标
	LastPercentHpTarget int          // 上次触发 percentHpDamage 的敌人 ID（切换目标首击）

	// 叠伤计数（stackDamage 能力：连续命中同目标递增）
	StackTarget int // 当前叠伤目标 ID
	StackCount  int // 叠伤层数

	// 能力选项缓存：key=category index(0-5), value=3 个候选 AbilityDef
	// 建塔时根据全局 wavesCleared 一次性 roll 所有已解锁位；新波次解锁时追加 roll。
	PendingChoices map[int][]config.AbilityDef

	// 统计
	Kills int // 累计击杀数（弹射物/直接攻击）

	// 建造/出售动画
	BuildAnim float64 // >0 during build-in animation (seconds remaining, starts at 0.3)
	SellAnim  float64 // >0 during sell-out animation (seconds remaining, starts at 0.25)
	Selling   bool    // true when tower is in sell animation (skip gameplay logic)
}

// BuyStrength 花费金币购买永久强度。返回实际花费。
func (t *Tower) BuyStrength() int {
	bal := config.GlobalBalance()
	cost := bal.Tower.StrengthBuyCost
	t.Cost += cost // 累计投入（影响卖价）
	if t.Strength != nil {
		t.Strength.AddPermanent(bal.Tower.StrengthBuyAmount)
	}
	t.RecalcStats()
	return cost
}

// RecalcStats 根据当前强度和 BuffList 光环加成重算 Damage/AttackSpeed/Range。
// 公式: final = (Base + Potential × str/100) × (1 + PctMod) + FlatMod
// 光环修饰从 Buffs.SumByID() 读取；CritBonus/DamageAmp 同步更新供战斗代码读取。
// 无 Strength 时使用强度100的默认值。
func (t *Tower) RecalcStats() {
	ratio := 1.0 // 默认强度100
	if t.Strength != nil {
		ratio = t.Strength.Ratio()
	}

	// Aura modifiers from BuffList
	var pctDamage, pctSpeed, flatRange float64
	if t.Buffs != nil {
		pctDamage = t.Buffs.SumByID(buff.IDAuraDamageAmp) // soloBoost + damageUpAura → pctDamage
		pctSpeed = t.Buffs.SumByID(buff.IDAuraPctSpeed)
		flatRange = t.Buffs.SumByID(buff.IDAuraFlatRange)
		t.CritBonus = t.Buffs.SumByID(buff.IDAuraCrit)
		t.DamageAmp = pctDamage // also exposed for combat code (apply_hit.go)
	} else {
		t.CritBonus = 0
		t.DamageAmp = 0
	}

	baseDmg := t.BaseDamage + t.PotentialDamage*ratio
	t.Damage = baseDmg
	if t.Damage < t.BaseDamage {
		t.Damage = t.BaseDamage // 伤害不低于基础值
	}

	baseSpd := t.BaseSpeed + t.PotentialSpeed*ratio
	t.AttackSpeed = baseSpd * (1 + pctSpeed)
	if floor := config.GlobalBalance().Tower.AttackSpeedFloor; t.AttackSpeed < floor {
		t.AttackSpeed = floor // 攻速保底
	}

	baseRng := t.BaseRange + t.PotentialRange*ratio
	t.Range = baseRng + flatRange
	if t.Range < t.BaseRange {
		t.Range = t.BaseRange // 射程不低于基础值
	}
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
	case AbilityScatter:
		return StyleScatter
	case AbilityWideBeam:
		return StyleWideBeam
	case AbilitySpinAoe:
		return StyleSpinAoE
	case AbilityRadial:
		return StyleRadial
	default:
		// bounce/splash/multiTarget/空 → 都用 projectile
		if t.AttackStyleID != "" {
			return t.AttackStyleID
		}
		return StyleProjectile
	}
}

// abilitySpriteMap 攻击能力 → 精灵资源标识映射。
var abilitySpriteMap = map[string]string{
	AbilityEnhance:     "fortress",
	AbilityScatter:     "shotgun",
	AbilityWideBeam:    "prism",
	AbilitySpinAoe:     "cyclone",
	AbilityBounce:      "ricochet",
	AbilitySplash:      "mortar",
	AbilityMultiTarget: "hydra",
	AbilityRadial:      "nova",
}

// abilitySpriteLabels 精灵标识 → 中文名映射。
var abilitySpriteLabels = map[string]string{
	"sentinel": "哨兵",
	"fortress": "堡垒",
	"shotgun":  "霰弹",
	"prism":    "棱光",
	"cyclone":  "旋刃",
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
