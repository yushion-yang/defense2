// pool.go — 敌人对象池。
// Fixed-size array pool (256 slots). Linear scan for spawn/each.
// Chosen for: stable pointers (enemies referenced by combat/render systems),
// no GC pressure from allocations, and simple slot reuse via Active flag.
package enemy

import (
	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/game"
)

const (
	defaultHealInterval     = 2.5 // 默认治疗间隔(秒)
	bossSpawnAnimDuration   = 0.5 // Boss 出生动画时长(秒)
	normalSpawnAnimDuration = 0.3 // 普通敌人出生动画时长(秒)
)

// Pool 固定大小的敌人对象池。
type Pool struct {
	enemies []Enemy // 预分配的敌人槽位数组
	Count   int     // 当前存活敌人数量
	nextID  int     // 递增 ID 计数器
	// OnSplit 击杀时分裂回调（可选，由 stage 层注册）。
	OnSplit func(children []*Enemy)
	// OnDeathSpawn 死亡召唤回调（可选，由 stage 层注册）。
	OnDeathSpawn func(e *Enemy, count int)
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
			*e = Enemy{} // 清零所有字段，防止复用残留
			p.nextID++

			// 基础属性
			e.ID = p.nextID
			e.X = x
			e.Y = y
			e.HP = hp
			e.MaxHP = hp
			e.DisplayHP = hp
			e.Speed = speed
			e.BaseSpeed = speed
			e.Radius = cfg.Radius
			e.PathIndex = pathIndex
			e.Active = true
			e.Archetype = archetype
			e.Buffs = buff.NewDefaultBuffList() // 初始化 BuffList

			// 外观
			e.SpriteDir = cfg.Sprite
			if e.SpriteDir == "" {
				e.SpriteDir = archetype
			}

			// 基础配置
			e.Boss = cfg.Boss
			e.RewardScale = cfg.RewardScale
			if e.RewardScale <= 0 {
				e.RewardScale = 1
			}

			// 分裂
			e.SplitHPRatio = cfg.SplitHPRatio
			e.SplitSpeedScale = cfg.SplitSpeedScale

			// 传送
			e.TeleportInterval = cfg.TeleportInterval
			e.TeleportSkip = cfg.TeleportSkip
			e.TeleportTimer = cfg.TeleportInterval // 首次传送需等满间隔

			// 应用行为配置
			e.Behavior = cfg.Behavior
			if cfg.StealthDuration > 0 {
				e.Buffs.Add(buff.Buff{
					ID: buff.IDStealth, Category: buff.CatBehavior,
					Source: "archetype", Value: 1,
					Duration: cfg.StealthDuration, Remaining: cfg.StealthDuration,
				})
			}
			if cfg.SplitCount > 0 {
				e.SplitCount = cfg.SplitCount
				e.SplitScale = cfg.SplitScale
				if e.SplitScale <= 0 {
					e.SplitScale = config.GlobalBalance().Split.HpRatio
				}
			}
			if cfg.HealScale > 0 {
				e.Buffs.Add(buff.Buff{
					ID: buff.IDHealAura, Category: buff.CatBehavior, Source: "archetype",
					Value: cfg.HealScale, Value2: cfg.HealRadius,
					Duration: -1, Remaining: -1,
				})
				e.HealInterval = cfg.HealInterval
				if e.HealInterval <= 0 {
					e.HealInterval = defaultHealInterval
				}
				e.HealCooldown = 0
			}
			if cfg.AuraRange > 0 {
				e.Buffs.Add(buff.Buff{
					ID: buff.IDBufferAura, Category: buff.CatBehavior, Source: "archetype",
					Value: cfg.AuraSpeedUp, Value2: cfg.AuraRange,
					Duration: -1, Remaining: -1,
				})
			}

			// 行为 buff（原型级）
			if cfg.DamageReduceRatio > 0 {
				e.Buffs.Add(buff.Buff{
					ID: buff.IDDamageReduce, Category: buff.CatDefense, Source: "archetype",
					Value: cfg.DamageReduceRatio, Duration: -1, Remaining: -1,
				})
			}
			if cfg.BerserkThreshold > 0 {
				e.Buffs.Add(buff.Buff{
					ID: buff.IDBerserk, Category: buff.CatBehavior, Source: "archetype",
					Value: cfg.BerserkSpeedScale, Value2: cfg.BerserkThreshold,
					Duration: -1, Remaining: -1,
				})
			}
			if cfg.RegenRatio > 0 {
				e.Buffs.Add(buff.Buff{
					ID: buff.IDRegen, Category: buff.CatBehavior, Source: "archetype",
					Value: e.MaxHP * cfg.RegenRatio, Duration: -1, Remaining: -1,
				})
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
			if cfg.PhaseDuration > 0 {
				e.Buffs.Add(buff.Buff{
					ID: buff.IDPhaseShift, Category: buff.CatBehavior, Source: "archetype",
					Value: cfg.PhaseDuration, Value2: cfg.PhaseCooldown,
					Duration: -1, Remaining: -1,
				})
				e.PhaseTimer = cfg.PhaseCooldown // 首次需等满冷却（runtime state）
			}
			e.StrDrainRatio = cfg.StrDrainRatio
			e.StrDrainInterval = cfg.StrDrainInterval
			e.StrDrainDuration = cfg.StrDrainDuration
			e.StrDrainTimer = cfg.StrDrainInterval
			e.DeathSpawnCount = cfg.DeathSpawnCount
			e.DeathSpawnArch = cfg.DeathSpawnArch
			e.PurgeInterval = cfg.PurgeInterval
			e.PurgeImmuneDur = cfg.PurgeImmuneDur
			e.PurgeTimer = cfg.PurgeInterval // 首次净化需等满间隔
			e.AbilityIDs = cfg.AbilityIDs
			if cfg.CCImmune {
				e.IsControlImmune = true
				e.IsStunImmune = true
				e.IsSlowImmune = true
			}
			if cfg.SlowImmune {
				e.IsSlowImmune = true
			}

			// 出生动画
			if cfg.Boss {
				e.SpawnTimer = bossSpawnAnimDuration
			} else {
				e.SpawnTimer = normalSpawnAnimDuration
			}
			e.SpawnDuration = e.SpawnTimer

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
		if e.SplitCount > 0 {
			children := OnSplitterDeath(e, p)
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
					// 召唤小怪奖励削减
					if bal.DeathSpawn.RewardScale > 0 {
						child.RewardScale *= bal.DeathSpawn.RewardScale
					}
				}
			}
			if p.OnDeathSpawn != nil {
				p.OnDeathSpawn(e, e.DeathSpawnCount)
			}
		}

		dying := config.GlobalBalance().Dying
		e.DyingTimer = dying.NormalDuration
		e.DyingDuration = dying.NormalDuration
		if e.Boss {
			bossDur := config.GlobalSpawnerConfig().Boss.DyingDuration
			if bossDur <= 0 {
				bossDur = dying.BossDuration // fallback
			}
			e.DyingTimer = bossDur
			e.DyingDuration = bossDur
		}
		// Clear stealth so death animation renders at full alpha
		e.Buffs.RemoveByID(buff.IDStealth)
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
		p.enemies[i] = Enemy{}
	}
	p.Count = 0
}
