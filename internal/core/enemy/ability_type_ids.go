// ability_type_ids.go — 怪物能力类型唯一标识符常量。
// 消灭跨文件魔法字符串，所有敌人能力引用必须使用这些常量。
// 配置权威源：config/enemies/abilities.json
package enemy

// 防御类能力
const (
	AbilProjectileBlock = "projectileBlock"
	AbilArmorPlating    = "armorPlating"
	AbilDamageCap       = "damageCap"
	AbilDamageCapPct    = "damageCapPercent"
	AbilDamageReduce    = "damageReduce"
)

// 被动类能力
const (
	AbilEvasion = "evasion"
	AbilBerserk = "berserk"
	AbilRegen   = "regen"
)

// 抗性类能力
const (
	AbilCCImmune   = "ccImmune"
	AbilSlowImmune = "slowImmune"
	AbilPurge      = "purge"
)

// 移动类能力
const (
	AbilDashOnHit  = "dashOnHit"
	AbilPhaseShift = "phaseShift"
	AbilStealth    = "stealth"
	AbilTeleport   = "teleport"
)

// 攻击类能力
const (
	AbilStrengthDrain = "strengthDrain"
)

// 辅助类能力
const (
	AbilHealAura  = "healAura"
	AbilSpeedAura = "speedAura"
)

// 死亡类能力
const (
	AbilDeathSplit = "deathSplit"
	AbilDeathSpawn = "deathSpawn"
)
