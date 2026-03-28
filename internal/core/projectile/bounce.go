// bounce.go — 弹射物链弹逻辑。
// 弹射物命中目标后搜索附近敌人跳跃，每次跳跃伤害衰减。
package projectile

import (
	"math"

	"defense2/internal/core/enemy"
)

// BounceConfig 弹射配置（附加到弹射物上）。
type BounceConfig struct {
	MaxBounces  int     // 最大弹射次数
	Range       float64 // 弹射搜索范围（像素）
	DamageDecay float64 // 每次弹射伤害衰减比例（0.7 = 保留 70%）
	Bounced     int     // 已弹射次数
	LastHitX    float64 // 上次命中位置 X
	LastHitY    float64 // 上次命中位置 Y
}

// TryBounce 尝试从命中位置向附近敌人弹射。
// 返回 true 表示成功弹射（弹射物继续飞行），false 表示弹射结束（应释放弹射物）。
func TryBounce(p *Projectile, bc *BounceConfig, hitX, hitY float64, enemies *enemy.Pool, excludeEnemy *enemy.Enemy) bool {
	if bc.Bounced >= bc.MaxBounces {
		return false
	}

	// 搜索范围内最近的未命中敌人
	var best *enemy.Enemy
	bestDist := math.MaxFloat64

	enemies.Each(func(e *enemy.Enemy) {
		if e == excludeEnemy {
			return
		}
		dx := e.X - hitX
		dy := e.Y - hitY
		d := math.Hypot(dx, dy)
		if d <= bc.Range && d < bestDist {
			bestDist = d
			best = e
		}
	})

	if best == nil {
		return false
	}

	// 重定向弹射物
	dx := best.X - hitX
	dy := best.Y - hitY
	dist := math.Hypot(dx, dy)
	if dist < 1 {
		dist = 1
	}
	p.X = hitX
	p.Y = hitY
	p.VX = (dx / dist) * p.Speed
	p.VY = (dy / dist) * p.Speed
	p.Damage *= bc.DamageDecay
	p.Life = p.MaxLife // 重置存活时间

	bc.Bounced++
	bc.LastHitX = hitX
	bc.LastHitY = hitY

	return true
}
