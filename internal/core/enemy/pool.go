// pool.go — 敌人对象池：生成、击杀、遍历。
//
// 固定大小数组池（256 槽位），不做 GC 分配。设计选型理由：
//   - 稳定指针：combat/render 系统持有 *Enemy 引用，不能因 slice 扩容失效
//   - 零 GC 压力：预分配所有槽位，Spawn 仅清零+赋值
//   - 简单复用：Active 标记控制槽位可用性
//
// 性能优化 — ActiveList：
//
//	activeIdx 维护当前活跃（含 dying）敌人的 slot 索引列表。
//	EachActive 只遍历此列表，跳过空槽位。在 256 槽中仅有 20 个活跃时，
//	性能从 O(256) 降至 O(20)。注意 Each() 仍然全量扫描，保留兼容旧代码。
//
// 生命周期状态机：
//
//	Spawn → Active(存活) → Kill(dying动画) → FinishDying(回收)
//	                     → KillImmediate(直接回收，用于泄漏/非战斗移除)
//
// Kill vs KillImmediate vs FinishDying：
//   - Kill：启动死亡动画，Count 立即减少（gameplay 视角已"死"），但槽位保持 Active 供渲染
//   - KillImmediate：立即回收槽位（泄漏到基地时使用），无死亡动画
//   - FinishDying：死亡动画播完后由 pipeline 调用，真正回收槽位
//
// 关联文件：
//   - enemy.go: Enemy struct 定义
//   - spawner.go: 波次控制器，调用 Spawn 生成敌人
//   - spawn_config.go: SpawnConfig 原型参数
//   - lifecycle.go: 死亡/出生事件回调
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
	enemies   []Enemy // 预分配的敌人槽位数组
	activeIdx []int   // 活跃敌人的 slot 索引列表（含 dying），EachActive 只遍历此列表
	Count     int     // 当前存活敌人数量（不含 dying）
	nextID    int     // 递增 ID 计数器
	// OnSplit 击杀时分裂回调（可选，由 stage 层注册）。
	OnSplit func(children []*Enemy)
	// OnDeathSpawn 死亡召唤回调（可选，由 stage 层注册）。
	OnDeathSpawn func(e *Enemy, count int)
}

// NewPool 创建指定容量的敌人对象池。
func NewPool(cap int) *Pool {
	return &Pool{
		enemies:   make([]Enemy, cap),
		activeIdx: make([]int, 0, cap),
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
//
// 初始化阶段（按顺序）：
//  1. 查找空闲槽位（Active==false），清零所有字段
//  2. 基础属性：ID/位置/HP/速度/半径/路径索引
//  3. 外观：SpriteDir（优先 cfg.Sprite，回退到 archetype）
//  4. 经济/Boss/分裂/传送 配置
//  5. 行为 buff 注入：stealth/healAura/bufferAura/damageReduce/berserk/regen
//     （通过 BuffList.Add 写入，Duration=-1 表示永久）
//  6. 能力系统字段：damageCap/armor/evasion/dash/phase/strDrain/purge 等
//  7. 免疫标记：CCImmune/SlowImmune
//  8. 出生动画：Boss 0.5s / 普通 0.3s
//  9. 加入 activeIdx + Count++
func (p *Pool) Spawn(x, y, baseHP, baseSpeed float64, pathIndex int, archetype string, cfg *SpawnConfig) *Enemy {
	if cfg == nil {
		cfg = DefaultSpawnConfig()
	}

	// 平台缩放（Web 端降低敌人强度，桌面端 1.0 无影响）
	plat := config.GlobalPlatform()
	hp := baseHP * cfg.HpScale * plat.EnemyHPScale
	speed := baseSpeed * cfg.SpeedScale * plat.EnemySpeedScale

	// 线性扫描找空闲槽位（典型负载 <30 个活跃，扫描很快）
	for i := range p.enemies {
		if !p.enemies[i].Active {
			e := &p.enemies[i]
			*e = Enemy{} // 清零所有字段，防止复用残留（关键：避免上一个敌人的 buff/状态遗留）
			p.nextID++

			// ── 阶段 1：基础属性 ──
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

			// ── 阶段 2：外观 ──
			e.SpriteDir = cfg.Sprite
			if e.SpriteDir == "" {
				e.SpriteDir = archetype
			}

			// ── 阶段 3：基础配置 ──
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

			// ── 阶段 4：行为 buff 注入 ──
			// 行为参数通过 BuffList.Add 写入，Duration=-1 表示永久 buff。
			// 原型可同时拥有多个行为 buff（如 tank 同时有 damageReduce + berserk）。
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

			// ── 阶段 4b：原型级行为 buff（与上面的行为配置分开，因为这些不依赖特定 Behavior 字段）──
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

			// ── 阶段 5：能力系统字段 ──
			// 直接从 SpawnConfig 拷贝到 Enemy 字段，后续 ApplyAbilityPotentials 会叠加波次增量
			e.DamageCap = cfg.DamageCap * plat.EnemyDamageCapScale
			e.DamageCapPercent = cfg.DamageCapPercent
			e.ProjectileBlockChance = cfg.ProjectileBlockChance
			e.ArmorFlat = cfg.ArmorFlat * plat.EnemyArmorScale
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

			// ── 阶段 6：出生动画 ──
			// 出生动画期间敌人 Active 但不可被选中/伤害（IsSpawning()=true）
			if cfg.Boss {
				e.SpawnTimer = bossSpawnAnimDuration
			} else {
				e.SpawnTimer = normalSpawnAnimDuration
			}
			e.SpawnDuration = e.SpawnTimer

			p.activeIdx = append(p.activeIdx, i)
			p.Count++
			return e
		}
	}
	return nil
}

// Kill 启动死亡动画。Count 立即减少（gameplay 视角已"死"），但 Active 保持为 true，
// 供渲染层继续绘制死亡动画，直到 FinishDying 被调用。
//
// 执行顺序：
//  1. 分裂体检查：SplitCount>0 时在当前位置生成子体（必须在 dying 标记前，否则子体无法获取父体路径）
//  2. 死亡召唤：DeathSpawnCount>0 时在当前位置生成指定原型的小怪
//  3. 设置死亡动画计时器（Boss 用 spawner.json 配置，普通用 balance.json）
//  4. 清除隐身 buff（死亡动画需全不透明渲染）
//  5. Count-- （此后 pool.Count 不再计入此敌人）
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

// KillImmediate 立即回收槽位，无死亡动画。
// 用于泄漏到基地的敌人或其他非战斗移除场景。
// 注意：如果敌人已在 dying 状态（Kill 已调用），不重复减 Count。
func (p *Pool) KillImmediate(e *Enemy) {
	if e.Active {
		if e.DyingTimer <= 0 {
			p.Count-- // only decrement if not already dying (Kill already decremented)
		}
		e.Active = false
		e.DyingTimer = 0
		p.removeFromActive(p.slotIndexOf(e))
	}
}

// FinishDying completes the dying animation and deactivates the enemy slot.
func (p *Pool) FinishDying(e *Enemy) {
	e.Active = false
	e.DyingTimer = 0
	p.removeFromActive(p.slotIndexOf(e))
}

// removeFromActive 从 activeIdx 中移除指定 slot 索引。
// 使用 swap-remove：将末尾元素覆盖到被删位置，O(n) 扫描 + O(1) 删除。
// 不保证顺序，但 EachActive 不依赖遍历顺序。
func (p *Pool) removeFromActive(slotIdx int) {
	for i, idx := range p.activeIdx {
		if idx == slotIdx {
			last := len(p.activeIdx) - 1
			p.activeIdx[i] = p.activeIdx[last]
			p.activeIdx = p.activeIdx[:last]
			return
		}
	}
}

// slotIndexOf 返回敌人在池中的 slot 索引。
func (p *Pool) slotIndexOf(e *Enemy) int {
	for i := range p.enemies {
		if &p.enemies[i] == e {
			return i
		}
	}
	return -1
}

// Len 返回池的槽位总数（非存活数量）。
func (p *Pool) Len() int { return len(p.enemies) }

// ByIndex 返回指定索引的敌人指针（不检查 Active 状态）。
func (p *Pool) ByIndex(i int) *Enemy { return &p.enemies[i] }

// Each 遍历所有存活敌人并执行回调。
// 全量扫描 256 槽位（O(cap)），兼容旧代码。新代码优先用 EachActive。
func (p *Pool) Each(fn func(e *Enemy)) {
	for i := range p.enemies {
		if p.enemies[i].Active {
			fn(&p.enemies[i])
		}
	}
}

// EachActive 仅遍历活跃索引列表中的敌人（含 dying）。
// 比 Each 快：跳过空槽位，只访问实际存活/dying 的敌人。
func (p *Pool) EachActive(fn func(e *Enemy)) {
	for _, idx := range p.activeIdx {
		fn(&p.enemies[idx])
	}
}

// ClearAll 清空所有敌人（重置对象池）。
func (p *Pool) ClearAll() {
	for i := range p.enemies {
		p.enemies[i] = Enemy{}
	}
	p.Count = 0
	p.activeIdx = p.activeIdx[:0]
}
