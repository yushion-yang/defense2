// pool.go — 敌人对象池。
// 固定大小数组实现零分配对象池，通过 Active 标记复用槽位。
package enemy

import (
	"defense2/internal/config"
	"defense2/internal/core/game"
)

// Pool 固定大小的敌人对象池。
type Pool struct {
	enemies []Enemy // 预分配的敌人槽位数组
	Count   int     // 当前存活敌人数量
	nextID  int     // 递增 ID 计数器
	// OnSplit 击杀时分裂回调（可选，由 stage 层注册）。
	OnSplit func(children []*Enemy)
}

// NewPool 创建指定容量的敌人对象池。
func NewPool(cap int) *Pool {
	return &Pool{
		enemies: make([]Enemy, cap),
	}
}

// DefaultPool 创建默认容量（MaxEnemies=256）的敌人对象池。
func DefaultPool() *Pool {
	return NewPool(game.MaxEnemies)
}

// Spawn 激活一个空闲槽位并初始化敌人属性。
// baseHP/baseSpeed 为当前波次的基准值，cfg 中的倍率会应用于它们。
// archetype 标识敌人原型（如 "normal"、"runner"、"tank"）。
// cfg 为 nil 时使用默认配置（hpScale=1, speedScale=1, radius=8）。
// pathIndex 通常为 1（敌人从 waypoint[0] 出生，朝 waypoint[1] 移动）。
// 池满时返回 nil。
func (p *Pool) Spawn(x, y, baseHP, baseSpeed float64, pathIndex int, archetype string, cfg *SpawnConfig) *Enemy {
	if cfg == nil {
		cfg = DefaultSpawnConfig()
	}

	hp := baseHP * cfg.HpScale
	speed := baseSpeed * cfg.SpeedScale

	for i := range p.enemies {
		if !p.enemies[i].Active {
			e := &p.enemies[i]
			p.nextID++
			e.ID = p.nextID
			e.X = x
			e.Y = y
			e.HP = hp
			e.MaxHP = hp
			e.Speed = speed
			e.BaseSpeed = speed
			e.Radius = cfg.Radius
			e.PathIndex = pathIndex
			e.ReachedEnd = false
			e.Active = true
			e.Path = nil
			e.Archetype = archetype
			e.SpriteDir = cfg.Sprite
			if e.SpriteDir == "" {
				e.SpriteDir = archetype
			}
			e.Boss = cfg.Boss
			e.Reward = cfg.Reward
			e.RewardScale = cfg.RewardScale
			if e.RewardScale <= 0 {
				e.RewardScale = 1
			}
			e.StunTimer = 0
			e.SlowTimer = 0
			e.SlowFactor = 1
			e.BleedTimer = 0
			e.BleedDPS = 0
			e.PoisonTimer = 0
			e.PoisonDPS = 0
			e.BurnTimer = 0
			e.BurnDPS = 0
			e.RootTimer = 0
			e.DamageAmplify = 0
			e.DamageAmplifyTimer = 0
			e.DisplayHP = hp
			// Elite 已移除
			e.DyingTimer = 0
			e.DyingDuration = 0
			e.HitFlash = 0
			e.Age = 0
			e.AnimCur = ""
			e.AnimFrame = 0
			e.AnimTimer = 0
			e.AnimDone = false
			e.Silenced = false
			e.ZoneDmgAccum = 0
			e.DotTickTimer = 0
			e.LastDotDmg = 0
			e.DamageCap = 0
			e.DamageCapPercent = 0
			e.IsInvincible = false
			e.IsDamageImmune = false
			e.IsUntargetable = false
			e.Thresholds = nil
			e.Tenacity = 0
			e.ControlImmuneTimer = 0
			e.IsControlImmune = false
			e.IsStunImmune = false
			e.IsSlowImmune = false
			e.IsRootImmune = false
			e.Lifecycle = nil
			e.Behavior = ""
			e.BerserkThreshold = 0
			e.BerserkSpeedScale = 0
			e.BerserkTriggered = false
			e.RegenPerSec = 0
			// 治疗光环初始化为零值，下方"应用行为配置"段按 cfg 设置实际值
			e.HealPower = 0
			e.HealRadius = 0
			e.HealInterval = 0
			e.HealCooldown = 0
			e.Stealthed = false
			e.StealthTimer = 0
			e.SplitCount = 0
			e.SplitScale = 0
			e.SplitHPRatio = cfg.SplitHPRatio
			e.SplitSpeedScale = cfg.SplitSpeedScale
			// 传送
			e.TeleportInterval = cfg.TeleportInterval
			e.TeleportSkip = cfg.TeleportSkip
			e.TeleportTimer = cfg.TeleportInterval // 首次传送需等满间隔
			// 旗手光环
			e.BuffRadius = 0
			e.BuffAmount = 0
			e.SpeedBuff = 0
			e.AuraRange = cfg.AuraRange
			e.AuraSpeedUp = cfg.AuraSpeedUp
			// 减伤
			e.DamageReduceRatio = 0
			// 已迁移到能力系统的字段不再在此设置

			// 应用行为配置
			e.Behavior = cfg.Behavior
			if cfg.StealthDuration > 0 {
				e.Stealthed = true
				e.StealthTimer = cfg.StealthDuration
			}
			if cfg.SplitCount > 0 {
				e.SplitCount = cfg.SplitCount
				e.SplitScale = cfg.SplitScale
				if e.SplitScale <= 0 {
					e.SplitScale = config.GlobalBalance().Split.HpRatio
				}
			}
			if cfg.HealScale > 0 {
				e.HealPower = cfg.HealScale // 治疗比例（如0.05=5%目标MaxHP）
				e.HealRadius = cfg.HealRadius
				e.HealInterval = cfg.HealInterval
				if e.HealInterval <= 0 {
					e.HealInterval = 2.5
				}
			}
			if cfg.AuraRange > 0 {
				e.BuffRadius = cfg.AuraRange
				e.BuffAmount = cfg.AuraSpeedUp
			}

			// 能力系统字段
			e.DamageCap = cfg.DamageCap
			e.DamageCapPercent = cfg.DamageCapPercent
			e.ProjectileBlockChance = cfg.ProjectileBlockChance
			e.ArmorFlat = cfg.ArmorFlat
			e.EvasionChance = cfg.EvasionChance
			e.DashSpeedBoost = cfg.DashSpeedBoost
			e.DashDuration = cfg.DashDuration
			e.DashCooldown = cfg.DashCooldown
			e.DashCooldownT = 0
			e.DashActiveT = 0
			e.PhaseDuration = cfg.PhaseDuration
			e.PhaseCooldown = cfg.PhaseCooldown
			e.PhaseTimer = cfg.PhaseCooldown // 首次需等满冷却
			e.PhaseActive = false
			e.StrDrainRatio = cfg.StrDrainRatio
			e.StrDrainInterval = cfg.StrDrainInterval
			e.StrDrainDuration = cfg.StrDrainDuration
			e.StrDrainTimer = cfg.StrDrainInterval
			e.DeathSpawnCount = cfg.DeathSpawnCount
			e.DeathSpawnArch = cfg.DeathSpawnArch
			e.PurgeInterval = cfg.PurgeInterval
			e.PurgeImmuneDur = cfg.PurgeImmuneDur
			e.PurgeTimer = 0
			e.AbilityIDs = cfg.AbilityIDs
			e.AbilitySilenced = false
			if cfg.CCImmune {
				e.IsControlImmune = true
				e.IsStunImmune = true
				e.IsSlowImmune = true
				e.IsRootImmune = true
			}
			if cfg.SlowImmune {
				e.IsSlowImmune = true
			}

			p.Count++
			return e
		}
	}
	return nil
}

// Kill starts the dying animation for an enemy. Count drops immediately so
// gameplay systems see the enemy as "gone", but the enemy stays Active for
// rendering until FinishDying is called.
// If the enemy is a splitter, children are spawned at the same position.
func (p *Pool) Kill(e *Enemy) {
	if e.Active && e.DyingTimer <= 0 {
		// TODO: 复活能力将通过能力系统实现
		// 分裂体死亡时生成子体（必须在 dying 标记前执行，否则子体无法获取父体路径）
		if e.Behavior == "splitter" && e.SplitCount > 0 {
			HandleSplitterDeath(e, p)
		} else if e.SplitCount > 0 {
			children := SpawnSplitChildren(e, p)
			if p.OnSplit != nil && len(children) > 0 {
				p.OnSplit(children)
			}
		}

		// 死亡召唤（deathSpawn 能力）
		if e.DeathSpawnCount > 0 {
			bal := config.GlobalBalance()
			arch := e.DeathSpawnArch
			if arch == "" {
				arch = bal.DeathSpawn.DefaultArch
			}
			for i := 0; i < e.DeathSpawnCount; i++ {
				child := p.Spawn(e.X+float64(i)*bal.DeathSpawn.ChildOffset, e.Y, e.MaxHP*bal.DeathSpawn.HpRatio, e.BaseSpeed, e.PathIndex, arch, DefaultSpawnConfig())
				if child != nil {
					child.Path = e.Path
				}
			}
		}

		dying := config.GlobalBalance().Dying
		e.DyingTimer = dying.NormalDuration
		e.DyingDuration = dying.NormalDuration
		if e.Boss {
			e.DyingTimer = dying.BossDuration
			e.DyingDuration = dying.BossDuration
		}
		p.Count--
	}
}

// KillImmediate deactivates an enemy instantly without a dying animation.
// Used for enemies that reach the base (leaked) or other non-combat removal.
func (p *Pool) KillImmediate(e *Enemy) {
	if e.Active {
		if e.DyingTimer <= 0 {
			p.Count-- // only decrement if not already dying (Kill already decremented)
		}
		e.Active = false
		e.DyingTimer = 0
	}
}

// FinishDying completes the dying animation and deactivates the enemy slot.
func (p *Pool) FinishDying(e *Enemy) {
	e.Active = false
	e.DyingTimer = 0
}

// Each 遍历所有存活敌人并执行回调。
func (p *Pool) Each(fn func(e *Enemy)) {
	for i := range p.enemies {
		if p.enemies[i].Active {
			fn(&p.enemies[i])
		}
	}
}

// ClearAll 清空所有敌人（重置对象池）。
func (p *Pool) ClearAll() {
	for i := range p.enemies {
		p.enemies[i].Active = false
	}
	p.Count = 0
}
