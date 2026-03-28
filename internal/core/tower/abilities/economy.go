// economy.go — 经济类能力实现。
// 提供金币生成和费用减免等经济效果。
// goldOnKill 通过 OnHit 返回 kill-check 逻辑（仍返回 nil，由经济管线处理）。
// goldPassive 通过 Ticker 接口每 3 秒产出 1 金币。
// interest 为 no-op，由波次结算系统处理。
package abilities

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

func init() {
	tower.Register(&GoldOnKill{})
	tower.Register(&GoldPassive{})
	tower.Register(&Interest{})
}

// GoldOnKill 击杀赏金：该塔击杀敌人时额外获得金币。
// 击杀判定在弹射物命中管线中完成，此处 OnHit 返回 nil。
// OnTick 用于检查最近击杀并返回额外金币。
type GoldOnKill struct{}

func (a *GoldOnKill) Name() string { return "goldOnKill" }
func (a *GoldOnKill) OnHit(_ *tower.Tower, _ *projectile.Projectile, e *enemy.Enemy) *tower.HitResult {
	// 击杀赏金在命中管线中由经济系统处理，此处不触发额外效果。
	return nil
}

// goldPassiveInterval 被动产金间隔（秒）。
const goldPassiveInterval = 3.0

// goldPassiveAccumulators 每座塔的被动产金累计时间。
// 使用 map[*tower.Tower] 跟踪各塔的独立计时器。
// 注意：塔被出售后其条目会被 GC 回收（pool 复用时指针仍指向同一地址，
// 但 pool.Place 会重置 Active，TickTowerAbilities 只遍历 Active 塔，安全）。
var goldPassiveAccumulators = map[string]float64{}

// GoldPassive 被动产金：每 3 秒获得 1 金币。
type GoldPassive struct{}

func (a *GoldPassive) Name() string { return "goldPassive" }
func (a *GoldPassive) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}
func (a *GoldPassive) OnTick(t *tower.Tower, ctx *tower.TickContext) *tower.TickResult {
	// 使用塔的唯一标识（Key + 网格位置）作为累加器 key
	accKey := towerAccKey(t)
	goldPassiveAccumulators[accKey] += ctx.DT
	gold := 0
	for goldPassiveAccumulators[accKey] >= goldPassiveInterval {
		goldPassiveAccumulators[accKey] -= goldPassiveInterval
		gold++
	}
	if gold > 0 {
		return &tower.TickResult{GoldEarned: gold}
	}
	return nil
}

// Interest 利息：波次结算时根据当前金币获得额外收入。
// 实际逻辑在经济系统 (economy.InterestGold) 中，此处为 no-op。
type Interest struct{}

func (a *Interest) Name() string { return "interest" }
func (a *Interest) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}

// towerAccKey 生成塔在累加器 map 中的唯一键（类型+网格位置）。
func towerAccKey(t *tower.Tower) string {
	// 使用简单拼接避免 fmt 导入开销
	return t.Key + "_" + itoa(t.Row) + "_" + itoa(t.Col)
}

// itoa 简单整数转字符串（避免引入 strconv）。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
