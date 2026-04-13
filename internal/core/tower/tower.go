// tower.go — 塔实体定义。
//
// 本文件是塔系统的核心数据结构，定义了已放置塔的完整状态。
// 游戏中只有 1 种基础塔（basic），玩家通过选择 AbilitySlots[0] 的攻击能力
// 使其变形为 11 种视觉变体（sentinel/fortress/shotgun/prism/cyclone 等）。
//
// 关键设计：
//   - 属性公式: attr = Base + Potential × (Strength / 100)
//   - Tower struct 的字段按职责分组：身份→战斗参数→动画→攻击方式状态→战力→buff→瞄准→统计→建造出售
//   - RecalcStats() 是属性计算的唯一入口，所有强度/光环变化后必须调用
//   - StatsDirty 脏标记优化：仅在强度/buff 变化时才重算，避免每帧全塔计算
//
// 依赖关系：
//   - pool.go: 管理 Tower 实例的生命周期（创建/销毁）
//   - targeting.go: 读取 Tower.Target 和 Tower.Range 进行索敌
//   - upgrade.go: 操作 AbilitySlots/UnlockOrder/PendingChoices 实现能力选择
//   - randomize.go: 建塔时设置 Base*/Potential*/Specialty 随机属性
package tower

import (
	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/enemy"
	"defense2/internal/core/strength"
	"defense2/internal/i18n"
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
	StyleBarrage    AttackStyle = "barrage"    // 连射（多弹丸微延迟）
)

// Tower 已放置的塔实体。
//
// 字段按职责分为 10 组：身份标识 → 战斗属性 → 能力系统 → 属性档位 → 视觉动画
// → 战力缩放参数 → 攻击方式运行时 → 战力与buff → 索敌与叠伤 → 能力选项缓存 → 统计与生命周期。
// 对象池复用时由 initTower() 清零所有字段，防止残留数据。
type Tower struct {
	// ── 身份标识 ──
	X, Y        float64  // 像素中心坐标（逻辑坐标系 1200×540）
	Row, Col    int      // 所在网格行列（用于空间索引 O(1) 查找）
	Key         string   // 塔类型标识（如 "basic"，建造时从 TowerDef 复制）
	InstanceKey string   // 实例唯一标识（格式 "key_row_col"，用于弹射物来源匹配和 ByInstanceKey 反查）
	Label       string   // 显示名称（随攻击能力变化，如 "Sentinel" → "Fortress"）
	SpriteKey   string   // 精灵资源标识（根据 AbilitySlots[0] 的攻击能力决定：sentinel/fortress/shotgun/...）
	Active      bool     // 是否存活（对象池复用标记，false 表示槽位空闲可回收）
	Color       [3]uint8 // 显示颜色 RGB（建造时从 TowerDef 复制）

	// ── 战斗属性（RecalcStats 计算的最终值，直接用于战斗逻辑）──
	Range       float64 // 攻击范围（像素，索敌和 UI 显示都读此值）
	Damage      float64 // 单发伤害（弹射物/光束的基础伤害）
	AttackSpeed float64 // 攻击速度（次/秒，决定 FireTimer 的重置值）
	FireTimer   float64 // 下一次射击倒计时（秒，每帧递减，≤0 时可开火）
	Cost        int     // 总投入金币（建造+升级+购买强度累计，出售时按比例退还）

	// ── 能力系统 ──
	Abilities []string // 该塔拥有的能力名称列表（兼容旧配置，AllAbilities() 合并新旧两套）
	// 六大类别能力槽：[0]=攻击模式 [1]=CC [2]=命中加伤 [3]=光环 [4]=DoT [5]=范围效果
	// AbilitySlots[0] 最为关键：决定了塔的攻击方式（projectile/wideBeam/scatter/spin_aoe/radial/barrage）
	// 以及精灵外观（SpriteKey）和显示名称（Label）。其余 5 个槽位提供被动/触发效果。
	AbilitySlots [6]string
	UnlockOrder  [6]int // 能力类别解锁顺序（建塔时随机生成，[0] 始终是攻击模式）
	Level        int    // 塔等级（1=基础，每选择一个能力 +1，最高 7 = 1基础 + 6能力）
	PaidUnlocks  int    // 付费解锁能力槽位的累计次数（用于索引 TowerDef.UpgradeCosts 计算下次费用）

	// ── 属性档位（展示用，randomize.go 建塔时设置）──
	DamageTier string // "S"/"B"/"D" — 伤害属性档位
	SpeedTier  string // "S"/"B"/"D" — 攻速属性档位
	RangeTier  string // "S"/"B"/"D" — 射程属性档位
	Specialty  int    // 专精属性索引（0=damage, 1=speed, 2=range），专精项 Potential 额外加成

	// ── 视觉/动画 ──
	FireAnim float64 // 射击动画计时器（开火时设为 0.15s，逐帧衰减至 0，渲染层据此播放射击特效）
	Angle    float64 // 朝向角度（弧度，0=向上，顺时针。索敌时更新，渲染层用于旋转精灵）

	// ── 战力缩放参数（来自 JSON 配置 + randomize.go，放置后固定不变）──
	// 属性公式: final = Base + Potential × (Strength / 100)
	// Base* 是强度 0 时的底线属性，Potential* 是每 100 强度的增量。
	// 例如 BaseDamage=10, PotentialDamage=20, Strength=150 → Damage = 10 + 20×1.5 = 40
	BaseDamage      float64 // 伤害基础值（强度为 0 时的伤害底线）
	BaseRange       float64 // 射程基础值
	BaseSpeed       float64 // 攻速基础值
	PotentialDamage float64 // 伤害潜力值（每 100 强度的增量）
	PotentialRange  float64 // 射程潜力值
	PotentialSpeed  float64 // 攻速潜力值

	// ── 攻击方式与运行时状态 ──
	AttackStyleID   AttackStyle // 当前攻击方式（由 ResolveAttackStyle() 从 AbilitySlots[0] 推导）
	ProjectileSpeed float64     // 弹射物飞行速度（px/s，0=默认 300）

	// SpinAoE（旋转范围伤害）运行时状态
	SpinAngle  float64 // 当前旋转角度（弧度，持续递增）
	SpinActive float64 // 旋转激活计时器（>0 时正在旋转攻击，逐帧递减）

	// Barrage（连射/加特林）运行时状态
	// 连射是 self-managed 模式：开火后进入 burst，每 0.08s 发一颗 50% 伤害追踪弹
	BarrageBurst  int          // 剩余连射发射数（>0 表示正在 burst 中）
	BarrageTimer  float64      // 下一弹倒计时（秒，burst 内每弹间隔 0.08s）
	BarrageTarget *enemy.Enemy // 连射锁定目标（burst 期间不切目标，目标死亡时才重选）

	// GoldPassive（被动产金能力）运行时状态
	GoldCooldown float64 // 产金冷却计时器（秒，归零时产金并重置）

	// ── 战力系统与 Buff ──
	Strength *strength.StrengthData // 战力运行时数据（三层结构：base=100 + permanent + temp）
	// Buffs 是统一的 buff 容器，光环效果以 0.3s 短 buff 形式存在。
	// RecalcStats() 从 Buffs 聚合 pctDamage/pctSpeed/flatRange/crit/damageAmp。
	Buffs *buff.BuffList

	// 光环聚合值（RecalcStats 从 BuffList 读取后缓存到这里，战斗代码直接读取）
	CritBonus float64 // 暴击光环加成的暴击率增量（来自 aura:crit buff）
	DamageAmp float64 // 伤害增幅比例（来自 aura:damageAmp buff，最终伤害 ×(1+DamageAmp)）

	// ── 索敌与叠伤 ──
	Target              *enemy.Enemy // 当前锁定目标（粘性瞄准：有效时不切换，详见 targeting.go）
	LastPercentHpTarget int          // 上次触发 percentHpDamage 的敌人 ID（防止对同一目标重复触发首击效果）

	// ── 能力选项缓存 ──
	// key=类别索引(0-5), value=3 个候选 AbilityDef。
	// 生命周期：建塔时 RollAndCachePendingChoices 一次性 roll 所有已解锁位 →
	// 玩家选择后 ClearPendingChoice 删除对应 key → 新波次解锁时追加 roll 新位。
	PendingChoices map[int][]config.AbilityDef

	// ── 统计 ──
	Kills int // 累计击杀数（弹射物命中/直接攻击击杀时在 apply_hit.go 中递增）

	// ── 建造/出售动画 ──
	BuildAnim float64 // 建造动画剩余时间（秒，初始 0.3s，>0 时跳过战斗逻辑）
	SellAnim  float64 // 出售动画剩余时间（秒，初始 0.25s，动画结束后 Remove）
	Selling   bool    // 是否正在出售动画中（true 时跳过所有游戏逻辑，仅播放消失动画）

	// ── 性能优化 ──
	// 脏标记：仅当强度/buff 变化时标记为 true，pipeline 中检查此标记决定是否调用 RecalcStats。
	// 避免每帧对所有塔执行属性重算（大多数帧塔的属性不变）。
	StatsDirty bool
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
//
// 计算流程：
//  1. 从 Strength 获取当前强度比率 ratio = Strength.Ratio()（无 Strength 时默认 1.0 即强度 100）
//  2. 从 BuffList 聚合光环加成：pctDamage/pctSpeed（百分比）、flatRange（固定值）
//  3. 应用公式：
//     - Damage    = Base + Potential × ratio          （无百分比修饰，DamageAmp 在战斗时单独乘）
//     - Speed     = (Base + Potential × ratio) × (1 + pctSpeed)
//     - Range     = (Base + Potential × ratio) + flatRange
//  4. 各属性有保底值：Damage/Range 不低于 Base*，AttackSpeed 不低于 balance.json 的 floor
//  5. 同步更新 CritBonus/DamageAmp 缓存供 apply_hit.go 直接读取
//
// 调用时机：建塔、购买强度、战灵设置临时强度、光环 buff 刷新后（通过 StatsDirty 标记触发）。
func (t *Tower) RecalcStats() {
	ratio := 1.0 // 默认强度 100（Ratio = Strength/100，所以 100 对应 1.0）
	if t.Strength != nil {
		ratio = t.Strength.Ratio()
	}

	// 从 BuffList 聚合所有光环 buff 的数值修饰
	// 光环以 0.3s 短 buff 形式存在，每帧由光环塔的 Ticker 刷新
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

	// 伤害计算：不乘 pctDamage（DamageAmp 在 apply_hit.go 战斗时单独应用，避免重复计算）
	baseDmg := t.BaseDamage + t.PotentialDamage*ratio
	t.Damage = baseDmg
	if t.Damage < t.BaseDamage {
		t.Damage = t.BaseDamage // 保底：敌人 debuff 削减强度时，伤害不低于基础值
	}

	// 攻速计算：应用百分比光环加成
	baseSpd := t.BaseSpeed + t.PotentialSpeed*ratio
	t.AttackSpeed = baseSpd * (1 + pctSpeed)
	if floor := config.GlobalBalance().Tower.AttackSpeedFloor; t.AttackSpeed < floor {
		t.AttackSpeed = floor // 保底：防止极端减速使塔完全停火（balance.json tower.attackSpeedFloor）
	}

	// 射程计算：应用固定值光环加成
	baseRng := t.BaseRange + t.PotentialRange*ratio
	t.Range = baseRng + flatRange
	if t.Range < t.BaseRange {
		t.Range = t.BaseRange // 保底：射程不低于基础值
	}
}

// AllAbilities 返回所有生效能力（合并 AbilitySlots 和旧 Abilities）。
// 新系统用 AbilitySlots[0..5]，旧系统用 Abilities[]。
// 两套并存是为了兼容旧配置，去重后返回统一列表供战斗代码遍历。
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

// ResolveAttackStyle 根据 AbilitySlots[0]（攻击模式能力）推导实际攻击方式。
//
// 映射规则：scatter→StyleScatter, wideBeam→StyleWideBeam, 等等。
// 特殊情况：bounce/splash/multiTarget 这三种能力虽然在攻击类别（slot[0]），
// 但它们不改变攻击方式，仍使用 projectile 发射弹射物，通过能力的 OnHit 实现弹射/溅射/多目标。
// AbilitySlots[0] 为空时回退到 AttackStyleID（兼容旧配置或初始状态）。
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
	case AbilityBarrage:
		return StyleBarrage
	default:
		// bounce/splash/multiTarget/空 → 都用 projectile（这些能力通过弹射物命中后的 OnHit 生效）
		if t.AttackStyleID != "" {
			return t.AttackStyleID
		}
		return StyleProjectile
	}
}

// abilitySpriteMap 攻击能力 → 精灵资源标识映射。
// 基础塔默认精灵为 "sentinel"（无攻击能力时），选择攻击能力后变形为对应精灵。
// 注意：enhance（强化）虽然不改变攻击方式，但会把外观变为 "fortress"。
var abilitySpriteMap = map[string]string{
	AbilityEnhance:     "fortress",
	AbilityScatter:     "shotgun",
	AbilityWideBeam:    "prism",
	AbilitySpinAoe:     "cyclone",
	AbilityBounce:      "ricochet",
	AbilitySplash:      "mortar",
	AbilityMultiTarget: "hydra",
	AbilityRadial:      "nova",
	AbilityBarrage:     "gatling",
}

// spriteLabelKeys 精灵标识列表（运行时通过 i18n.T 查询本地化名称）。
var spriteLabelKeys = []string{
	"sentinel", "fortress", "shotgun", "prism", "cyclone",
	"ricochet", "mortar", "hydra", "nova", "gatling",
}

// AbilitySpriteKey 根据攻击能力类型返回精灵资源标识。
func AbilitySpriteKey(abilityType string) string {
	if key, ok := abilitySpriteMap[abilityType]; ok {
		return key
	}
	return "sentinel"
}

// SpriteLabelFor 返回精灵标识对应的本地化名称。
func SpriteLabelFor(spriteKey string) string {
	return i18n.T("tower.sprite." + spriteKey)
}
