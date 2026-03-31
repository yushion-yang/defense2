// apply_hit.go — 统一命中处理。
// 所有攻击方式（弹射物/即时光束/范围AoE等）共用此函数处理能力触发+扣血+击杀。
package combat

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

// HitInput 描述一次命中事件。
type HitInput struct {
	Tower       *tower.Tower       // 来源塔（可为 nil，如战灵弹射物）
	Target      *enemy.Enemy       // 命中目标
	BaseDamage  float64            // 基础伤害
	Style       string             // 攻击方式标识
	Enemies     *enemy.Pool        // 用于 splash/bounce
	Projectiles *projectile.Pool   // 用于 bounce
	Projectile  *projectile.Projectile // 原始弹射物（弹射物路径传入，即时伤害传 nil）
}

// HitOutput 命中结果。
type HitOutput struct {
	TotalDamage float64
	IsCrit      bool
	Killed      bool
	ExtraKills  int // deathMark 等额外击杀
}

// ApplyHit 统一命中处理：遍历能力 → 计算最终伤害 → 扣血 → 击杀检查。
func ApplyHit(input HitInput, onHit HitCallback) HitOutput {
	totalDmg := input.BaseDamage
	isCrit := false

	// 构建合成弹射物用于能力 OnHit 调用
	var synth *projectile.Projectile
	if input.Projectile != nil {
		synth = input.Projectile
	} else {
		sourceKey := ""
		if input.Tower != nil {
			sourceKey = input.Tower.InstanceKey
		}
		synth = &projectile.Projectile{
			Damage:         input.BaseDamage,
			SourceTowerKey: sourceKey,
		}
	}

	// 遍历塔能力，触发 OnHit
	if input.Tower != nil {
		for _, aName := range input.Tower.Abilities {
			ab, ok := tower.Registry[aName]
			if !ok {
				continue
			}
			result := ab.OnHit(input.Tower, synth, input.Target)
			if result == nil {
				continue
			}
			totalDmg += result.BonusDamage
			if result.IsCrit {
				isCrit = true
			}
			applyHitEffectsUnified(result, input.Target, synth, input.Enemies, input.Projectiles)
		}
	}

	// 扣血
	input.Target.HP -= totalDmg
	killed := input.Target.HP <= 0

	// 命中回调（飘字、音效等）
	if onHit != nil {
		onHit(input.Target, totalDmg, killed, input.Style, isCrit)
	}

	// 击杀处理
	extraKills := 0
	if killed && input.Tower != nil {
		extraKills = applyDeathExplosionUnified(input.Tower, input.Target, input.Enemies, onHit)
		input.Enemies.Kill(input.Target)
	}

	return HitOutput{
		TotalDamage: totalDmg,
		IsCrit:      isCrit,
		Killed:      killed,
		ExtraKills:  extraKills,
	}
}

// applyHitEffectsUnified 施加能力效果（减速、眩晕、流血、灼烧、溅射、弹射）。
func applyHitEffectsUnified(r *tower.HitResult, target *enemy.Enemy, p *projectile.Projectile, enemies *enemy.Pool, projectiles *projectile.Pool) {
	if r.Slow != nil {
		ApplySlow(target, r.Slow.Factor, r.Slow.Duration, p.SourceTowerKey)
	}
	if r.Stun != nil {
		ApplyStun(target, r.Stun.Duration, p.SourceTowerKey)
	}
	if r.Bleed != nil {
		target.BleedTimer = r.Bleed.Duration
		target.BleedDPS = r.Bleed.DPS
	}
	if r.Burn != nil {
		target.BurnTimer = r.Burn.Duration
		target.BurnDPS = r.Burn.DPS
	}
	if r.Splash != nil && enemies != nil {
		splashDamage := p.Damage * r.Splash.Ratio
		enemies.Each(func(e *enemy.Enemy) {
			if e == target || e.IsDying() {
				return
			}
			if math.Hypot(e.X-target.X, e.Y-target.Y) <= r.Splash.Radius {
				e.HP -= splashDamage
				if e.HP <= 0 {
					enemies.Kill(e)
				}
			}
		})
	}
	if r.Bounce != nil && p.BounceCount < r.Bounce.MaxBounces && projectiles != nil {
		hitIDs := append([]int{}, p.BounceHitIDs...)
		hitIDs = append(hitIDs, target.ID)

		var best *enemy.Enemy
		bestDist := r.Bounce.Range
		if enemies != nil {
			enemies.Each(func(e2 *enemy.Enemy) {
				if e2.IsDying() {
					return
				}
				for _, id := range hitIDs {
					if e2.ID == id {
						return
					}
				}
				d := math.Hypot(e2.X-target.X, e2.Y-target.Y)
				if d < bestDist {
					bestDist = d
					best = e2
				}
			})
		}
		if best != nil {
			bounceDmg := r.Bounce.SrcDamage * r.Bounce.DamageRatio
			projectiles.FireBounce(target.X, target.Y, best, bounceDmg, p.Speed, p.Radius, p.SourceTowerKey, p.BounceCount+1, hitIDs)
		}
	}
}

// applyDeathExplosionUnified 击杀后检查 deathMark 能力并触发 AoE 爆炸。
func applyDeathExplosionUnified(t *tower.Tower, killed *enemy.Enemy, enemies *enemy.Pool, onHit HitCallback) int {
	if enemies == nil {
		return 0
	}
	for _, aName := range t.Abilities {
		if aName != "deathMark" {
			continue
		}
		ab, ok := tower.Registry[aName]
		if !ok {
			return 0
		}
		// 通过能力的 OnHit 获取爆炸参数（deathMark OnHit 返回 nil，但我们需要读取配置）
		_ = ab
		// 直接从配置读取
		return deathExplosion(t, killed, enemies, onHit)
	}
	return 0
}

// deathExplosion 执行死亡爆炸 AoE。
func deathExplosion(t *tower.Tower, killed *enemy.Enemy, enemies *enemy.Pool, onHit HitCallback) int {
	// 从全局能力表读取 deathMark 参数
	// 这里需要避免循环依赖，所以用简化版本
	str := 100.0
	if t.Strength != nil {
		str = t.Strength.Effective()
	}
	// deathMark: base=10, potential=15, param=50(radius)
	// CalcScale = 10 + 15 * (str/100)
	explodeDmg := 10 + 15*(str/100.0)
	explodeR := 50.0

	extraKills := 0
	enemies.Each(func(e2 *enemy.Enemy) {
		if e2 == killed || e2.IsDying() {
			return
		}
		if math.Hypot(e2.X-killed.X, e2.Y-killed.Y) <= explodeR {
			e2.HP -= explodeDmg
			if onHit != nil {
				onHit(e2, explodeDmg, e2.HP <= 0, "explosion", false)
			}
			if e2.HP <= 0 {
				enemies.Kill(e2)
				extraKills++
			}
		}
	})
	return extraKills
}
