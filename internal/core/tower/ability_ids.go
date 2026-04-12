// ability_ids.go — 能力类型唯一标识符常量。
// 消灭跨文件魔法字符串，所有能力引用必须使用这些常量。
package tower

// 攻击类能力
const (
	AbilitySplash         = "splash"
	AbilityCrit           = "crit"
	AbilityBounce         = "bounce"
	AbilityMomentum       = "momentum"
	AbilityExecutionBonus = "executionBonus"
	AbilityFlatDamage     = "flatDamage"
	AbilityDistanceDamage = "distanceDamage"
	AbilityMultiTarget    = "multiTarget"
	AbilityDeathMark      = "deathMark"
	AbilityEnhance        = "enhance"
)

// CC 类能力
const (
	AbilitySlowPower    = "slowPower"
	AbilitySlowDuration = "slowDuration"
	AbilityStun         = "stun"
	AbilityStunChance   = "stunChance"
	AbilityStunDuration = "stunDuration"
)

// DoT 类能力
const (
	AbilityBleedDot = "bleedDot"
	AbilityBurn     = "burn"
	AbilityPoison   = "poison"
	AbilityWeaken   = "weaken"
)

// 光环类能力
const (
	AbilityDamageUpAura    = "damageUpAura"
	AbilityAttackSpeedAura = "attackSpeedAura"
	AbilityRangeAura       = "rangeAura"
	AbilityCritAura        = "critAura"
	AbilitySoloBoost       = "soloBoost"
)

// 区域类能力
const (
	AbilityPoisonZone  = "poisonZone"
	AbilitySilenceZone = "silenceZone"
	AbilityCurseZone   = "curseZone"
	AbilityWeakenZone  = "weakenZone"
)

// 经济/被动类能力
const (
	AbilityGoldPassive = "goldPassive"
)

// 攻击方式覆盖（config_ability.go 中引用，对应 ResolveAttackStyle）
const (
	AbilityScatter  = "scatter"
	AbilityWideBeam = "wideBeam"
	AbilitySpinAoe  = "spinAoe"
	AbilityRadial   = "radial"
	AbilityBarrage  = "barrage"
)

// 已禁用能力（实现未完成）
const (
	AbilityElementSwitch = "elementSwitch"
	AbilityPeriodicCast  = "periodicCast"
)
