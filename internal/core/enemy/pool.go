// pool.go — 敌人对象池。
// 固定大小数组实现零分配对象池，通过 Active 标记复用槽位。
package enemy

import "defense2/internal/core/game"

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
			e.Elite = cfg.HpScale >= 4
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
			// 治疗光环（从 SpawnConfig 计算）
			if cfg.HealScale > 0 {
				e.HealPower = hp * cfg.HealScale // healScale 是 maxHP 的比例
				e.HealRadius = cfg.HealRadius
				e.HealInterval = cfg.HealInterval
				if e.HealInterval <= 0 {
					e.HealInterval = 2.0
				}
			} else {
				e.HealPower = 0
				e.HealRadius = 0
				e.HealInterval = 0
			}
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
			e.ReflectPercent = 0
			e.ReviveHPPercent = 0
			e.ReviveUsed = false
			e.BossData = nil
			e.MovementType = cfg.MovementType

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
					e.SplitScale = 0.3
				}
			}
			if cfg.HealScale > 0 {
				e.HealPower = hp * cfg.HealScale // 治疗量 = 本体HP * healScale
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
		// 复活检查：首次死亡时复活
		if e.ReviveHPPercent > 0 && !e.ReviveUsed {
			e.ReviveUsed = true
			e.HP = e.MaxHP * e.ReviveHPPercent
			e.DisplayHP = e.HP
			e.HitFlash = 0.3 // 复活闪烁
			return            // 不进入死亡流程
		}
		// 分裂体死亡时生成子体（必须在 dying 标记前执行，否则子体无法获取父体路径）
		if e.Behavior == "splitter" && e.SplitCount > 0 {
			HandleSplitterDeath(e, p)
		} else if e.SplitCount > 0 {
			children := SpawnSplitChildren(e, p)
			if p.OnSplit != nil && len(children) > 0 {
				p.OnSplit(children)
			}
		}

		e.DyingTimer = 0.3
		e.DyingDuration = 0.3
		if e.Boss {
			e.DyingTimer = 0.5
			e.DyingDuration = 0.5
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
