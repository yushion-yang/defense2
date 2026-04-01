// scaling.go — 缩放类能力实现。
// 包含击杀升级、波次缩放、周期释放、邻居增益、元素切换等动态增长能力。
// 通过 init() 自注册到全局注册表。
package abilities

import (
	"fmt"
	"math"
	"math/rand"

	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

// 辅助函数（从已删除的旧能力文件迁移）
func towerAccKey(t *tower.Tower) string {
	return fmt.Sprintf("%s_%d_%d", t.Key, t.Row, t.Col)
}

func distToEnemy(t *tower.Tower, e *enemy.Enemy) float64 {
	return math.Hypot(t.X-e.X, t.Y-e.Y)
}

func distBetweenTowers(a, b *tower.Tower) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}

const auraRadius = 150.0

func init() {
	tower.Register(&KillUpgrade{})
	tower.Register(&WaveScale{})
	tower.Register(&PeriodicCast{})
	tower.Register(&NeighborBoost{})
	tower.Register(&ElementSwitch{})
}

// ---------- killUpgrade ----------

// killUpgradeStacks 每座塔的击杀堆叠数（Key+位置 → 层数）。
var killUpgradeStacks = map[string]int{}

// KillUpgrade 击杀升级：每次击杀目标 +1% 伤害（无限堆叠）。
type KillUpgrade struct{}

func (a *KillUpgrade) Name() string { return "killUpgrade" }
func (a *KillUpgrade) OnHit(t *tower.Tower, _ *projectile.Projectile, e *enemy.Enemy) *tower.HitResult {
	// 检查目标是否会被这次攻击击杀（HP 已在此之前扣除时调用）
	if e.HP <= 0 {
		key := towerAccKey(t)
		killUpgradeStacks[key]++
	}
	return nil
}
func (a *KillUpgrade) OnTick(t *tower.Tower, _ *tower.TickContext) *tower.TickResult {
	key := towerAccKey(t)
	stacks := killUpgradeStacks[key]
	if stacks <= 0 {
		return nil
	}
	// 每层 +1% 伤害
	bonusPerStack := 0.01
	return &tower.TickResult{
		DamageBoost: bonusPerStack * float64(stacks),
	}
}

// ---------- waveScale ----------

// waveScaleCounters 每座塔的波次缩放计数器。
var waveScaleCounters = map[string]int{}

// WaveScale 波次缩放：每波 +5% 伤害/射程/攻速（上限 +100%）。
type WaveScale struct{}

func (a *WaveScale) Name() string { return "waveScale" }
func (a *WaveScale) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}
func (a *WaveScale) OnTick(t *tower.Tower, _ *tower.TickContext) *tower.TickResult {
	key := towerAccKey(t)
	waves := waveScaleCounters[key]
	if waves <= 0 {
		return nil
	}

	// 每波 +5%，上限 100%
	perWave := 0.05
	maxBonus := 1.0
	bonus := math.Min(perWave*float64(waves), maxBonus)

	return &tower.TickResult{
		DamageBoost: bonus,
		SpeedBoost:  bonus,
		RangeBoost:  t.BaseRange * bonus,
	}
}

// IncrementWaveScale 波次结束时调用，递增所有拥有 waveScale 的塔的波次计数。
func IncrementWaveScale(towers *tower.Pool) {
	towers.Each(func(t *tower.Tower) {
		if tower.TowerHasAbility(t.Abilities, "waveScale") {
			key := towerAccKey(t)
			waveScaleCounters[key]++
		}
	})
}

// ---------- periodicCast ----------

// periodicCastTimers 每座塔的周期施法计时器。
var periodicCastTimers = map[string]float64{}

// periodicCastInterval 施法间隔（秒）。
const periodicCastInterval = 8.0

// periodicCastRadius 施法范围（像素）。
const periodicCastRadius = 120.0

// PeriodicCast 周期释放：每 8 秒释放一次范围效果（眩晕/伤害/增益随机）。
type PeriodicCast struct{}

func (a *PeriodicCast) Name() string { return "periodicCast" }
func (a *PeriodicCast) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}
func (a *PeriodicCast) OnTick(t *tower.Tower, ctx *tower.TickContext) *tower.TickResult {
	key := towerAccKey(t)
	periodicCastTimers[key] += ctx.DT

	if periodicCastTimers[key] < periodicCastInterval {
		return nil
	}
	periodicCastTimers[key] -= periodicCastInterval

	// 随机选择施法类型
	castType := rand.Intn(3)
	switch castType {
	case 0:
		// stunAoe：范围眩晕（走 CC 系统）
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if e.IsDying() {
				return
			}
			if distToEnemy(t, e) <= periodicCastRadius {
				combat.ApplyStun(e, 0.6, "stunAoe")
			}
		})
	case 1:
		// damageAoe：范围伤害（走伤害管线）
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if e.IsDying() {
				return
			}
			if distToEnemy(t, e) <= periodicCastRadius {
				combat.ProcessDamage(combat.DamageInput{
					Target:     e,
					RawDamage:  t.Damage * 0.5,
					DamageType: combat.DmgPhysical,
				})
			}
		})
	case 2:
		// buffAoe：自身增益
		return &tower.TickResult{
			DamageBoost: 0.25,
			SpeedBoost:  0.15,
		}
	}

	return nil
}

// ---------- neighborBoost ----------

// neighborBoostTimers 邻居增益检查计时器。
var neighborBoostTimers = map[string]float64{}

// neighborBoostInterval 检查间隔（秒）。
const neighborBoostInterval = 2.0

// NeighborBoost 邻居增益：每 2 秒检查周围最强塔，获得其 20% 属性加成。
type NeighborBoost struct{}

func (a *NeighborBoost) Name() string { return "neighborBoost" }
func (a *NeighborBoost) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}
func (a *NeighborBoost) OnTick(t *tower.Tower, ctx *tower.TickContext) *tower.TickResult {
	key := towerAccKey(t)
	neighborBoostTimers[key] += ctx.DT

	if neighborBoostTimers[key] < neighborBoostInterval {
		return nil
	}
	neighborBoostTimers[key] -= neighborBoostInterval

	// 查找周围最强邻居（以 DPS 为标准）
	var bestDPS float64
	ctx.Towers.Each(func(other *tower.Tower) {
		if other == t {
			return
		}
		if distBetweenTowers(t, other) <= auraRadius {
			dps := other.DPS()
			if dps > bestDPS {
				bestDPS = dps
			}
		}
	})

	if bestDPS <= 0 {
		return nil
	}

	// 获得最强邻居 20% 的加成
	boostRatio := 0.20
	return &tower.TickResult{
		DamageBoost: boostRatio,
	}
}

// ---------- elementSwitch ----------

// elementSwitchTimers 元素切换计时器。
var elementSwitchTimers = map[string]float64{}

// elementSwitchActiveElement 当前激活元素索引。
var elementSwitchActiveElement = map[string]int{}

// elementCycleInterval 元素切换间隔（秒）。
const elementCycleInterval = 5.0

// elementTypes 可切换的元素类型。
var elementTypes = []string{"fire", "ice", "lightning", "poison"}

// ElementSwitch 元素切换：每 5 秒循环切换元素，不同元素给予不同增益。
type ElementSwitch struct{}

func (a *ElementSwitch) Name() string { return "elementSwitch" }
func (a *ElementSwitch) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}
func (a *ElementSwitch) OnTick(t *tower.Tower, ctx *tower.TickContext) *tower.TickResult {
	key := towerAccKey(t)
	elementSwitchTimers[key] += ctx.DT

	if elementSwitchTimers[key] >= elementCycleInterval {
		elementSwitchTimers[key] -= elementCycleInterval
		elementSwitchActiveElement[key] = (elementSwitchActiveElement[key] + 1) % len(elementTypes)
	}

	// 根据当前元素给予不同增益
	idx := elementSwitchActiveElement[key]
	switch elementTypes[idx] {
	case "fire":
		// 火元素：+20% 伤害
		return &tower.TickResult{DamageBoost: 0.20}
	case "ice":
		// 冰元素：对范围内敌人施加减速
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if distToEnemy(t, e) <= t.Range {
				if e.SlowTimer < 0.5 {
					e.SlowTimer = 0.5
					e.SlowFactor = 0.7
					e.Speed = e.BaseSpeed * 0.7
				}
			}
		})
		return nil
	case "lightning":
		// 雷元素：+15% 攻速
		return &tower.TickResult{SpeedBoost: 0.15}
	case "poison":
		// 毒元素：范围内敌人每秒 2 点伤害（走伤害管线）
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if e.IsDying() {
				return
			}
			if distToEnemy(t, e) <= t.Range {
				combat.ProcessDamage(combat.DamageInput{
					Target:     e,
					RawDamage:  2.0 * ctx.DT,
					DamageType: combat.DmgMagic,
				})
			}
		})
		return nil
	}

	return nil
}

// ResetScalingState 清空所有缩放类能力的运行时状态（测试辅助函数）。
func ResetScalingState() {
	killUpgradeStacks = map[string]int{}
	waveScaleCounters = map[string]int{}
	periodicCastTimers = map[string]float64{}
	neighborBoostTimers = map[string]float64{}
	elementSwitchTimers = map[string]float64{}
	elementSwitchActiveElement = map[string]int{}
}
