// ids.go — Buff ID 唯一标识符常量。
// 消灭跨文件魔法字符串，所有 buff 引用必须使用这些常量。
package buff

// CC 类 debuff
const (
	IDStun          = "stun"
	IDSlow          = "slow"
	IDRoot          = "root"
	IDControlImmune = "controlImmune"
)

// DoT 类 debuff
const (
	IDBleed  = "bleed"
	IDBurn   = "burn"
	IDPoison = "poison"
)

// 其他 debuff
const (
	IDWeaken = "weaken"
)

// 敌人行为 buff
const (
	IDStealth      = "stealth"
	IDBerserk      = "berserk"
	IDRegen        = "regen"
	IDHealAura     = "healAura"
	IDBufferAura   = "bufferAura"
	IDSpeedUp      = "speedUp"
	IDPhaseShift   = "phaseShift"
	IDDamageReduce = "damageReduce"
)

// 塔光环 buff（通过 BuffList 施加到相邻塔）
const (
	IDAuraDamageAmp = "aura:damageAmp"
	IDAuraPctSpeed  = "aura:pctSpeed"
	IDAuraFlatRange = "aura:flatRange"
	IDAuraCrit      = "aura:crit"
)
