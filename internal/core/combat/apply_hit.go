// apply_hit.go — 统一命中处理。
// 所有攻击方式（弹射物/即时光束/范围AoE等）共用此函数处理能力触发+扣血+击杀。
package combat

import (
	"math"
	"math/rand"

	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	tel "defense2/internal/core/telemetry"
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
	OnCC        CCCallback         // CC 效果命中回调（可为 nil）
}

// HitOutput 命中结果。
type HitOutput struct {
	TotalDamage       float64
	IsCrit            bool
	Killed            bool
	ExtraKills        int  // deathMark 等额外击杀
	ProjectileBlocked bool // 弹幕盾：阻止弹射物继续传播（弹射/穿透停止）
}

// ApplyHit 统一命中处理：遍历能力 → 计算最终伤害 → 扣血 → 击杀检查。
func ApplyHit(input HitInput, onHit HitCallback) HitOutput {
	e := input.Target

	// ── 怪物闪避（完全回避，不触发任何 OnHit）──
	if e.EvasionChance > 0 && !e.AbilitySilenced {
		if rand.Float64() < e.EvasionChance {
			e.DodgeFlash = 0.3
			e.SetFloatText("MISS", 255, 255, 255)
			return HitOutput{}
		}
	}

	// 弹幕盾标记：命中正常受伤，但阻止弹射物继续飞行/弹射
	hasShield := e.ProjectileBlockChance > 0 && !e.AbilitySilenced

	totalDmg := input.BaseDamage
	isCrit := false

	// ── 装甲固定减免 ──
	if e.ArmorFlat > 0 && !e.AbilitySilenced {
		totalDmg -= e.ArmorFlat
		if totalDmg < 1 {
			totalDmg = 1
		}
		e.ArmorSpark = 0.15 // 触发装甲火花视觉
	}

	// ── 受击冲刺触发 ──
	if e.DashSpeedBoost > 0 && e.DashCooldownT <= 0 && !e.AbilitySilenced {
		e.DashActiveT = e.DashDuration
		e.DashCooldownT = e.DashCooldown
	}

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

	// 遍历塔能力，触发 OnHit（weaken 等 debuff 在此设置，立即对本次命中生效）
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
			applyHitEffectsUnified(result, input.Target, synth, input.Tower, input.Enemies, input.Projectiles, onHit, input.OnCC)
		}
	}

	// CritBonus 暴击（无 crit 能力时 critAura 仍可独立触发，统一 2x）
	if !isCrit && input.Tower != nil && input.Tower.CritBonus > 0 {
		if rand.Float64() < input.Tower.CritBonus {
			totalDmg *= 2 // 固定 2 倍
			isCrit = true
		}
	}

	// DamageAmp 全伤害增幅（damageUpAura 提供，所有伤害类型统一乘算）
	if input.Tower != nil && input.Tower.DamageAmp > 0 {
		totalDmg *= 1 + input.Tower.DamageAmp
	}

	// 扣血 — 走伤害管线（免疫/减免/阈值/遥测统一处理）
	pipeResult := ProcessDamage(DamageInput{
		Target:     input.Target,
		RawDamage:  totalDmg,
		DamageType: DmgPhysical, // 塔弹射物默认物理伤害
	})

	killed := pipeResult.Killed
	finalDmg := pipeResult.FinalDamage
	if pipeResult.Blocked {
		finalDmg = 0
	}

	// 命中回调（飘字、音效等）
	if onHit != nil {
		onHit(input.Target, finalDmg, killed, input.Style, isCrit)
	}

	// 击杀处理
	extraKills := 0
	if killed && input.Tower != nil {
		input.Tower.Kills++
		if input.Enemies != nil {
			extraKills = applyDeathExplosionUnified(input.Tower, input.Target, input.Enemies, onHit)
			input.Tower.Kills += extraKills
			input.Enemies.Kill(input.Target)
		}
	}

	// 弹幕盾：正常受伤后阻止弹射物继续传播
	blocked := false
	if hasShield && !killed {
		e.BlockFlash = 0.25
		blocked = true
	}

	return HitOutput{
		TotalDamage:       finalDmg,
		IsCrit:            isCrit,
		Killed:            killed,
		ExtraKills:        extraKills,
		ProjectileBlocked: blocked,
	}
}

// applyHitEffectsUnified 施加能力效果（减速、眩晕、流血、灼烧、溅射、弹射）。
func applyHitEffectsUnified(r *tower.HitResult, target *enemy.Enemy, p *projectile.Projectile, srcTower *tower.Tower, enemies *enemy.Pool, projectiles *projectile.Pool, onHit HitCallback, onCC CCCallback) {
	if r.Slow != nil {
		if ApplySlow(target, r.Slow.Factor, r.Slow.Duration, p.SourceTowerKey) && onCC != nil {
			if r.Slow.Factor < 0.4 {
				onCC(target.X, target.Y, "freeze")
			} else {
				onCC(target.X, target.Y, "slow")
			}
		}
		tel.T.Record("ability", "slow")
	}
	if r.Stun != nil {
		if ApplyStun(target, r.Stun.Duration, p.SourceTowerKey) && onCC != nil {
			onCC(target.X, target.Y, "stun")
		}
		tel.T.Record("ability", "stun")
	}
	if r.Bleed != nil {
		target.BleedTimer = r.Bleed.Duration
		target.BleedDPS = r.Bleed.DPS
		tel.T.Record("ability", "bleed")
	}
	if r.Burn != nil {
		wasBurning := target.BurnTimer > 0
		target.BurnTimer = r.Burn.Duration
		target.BurnDPS = r.Burn.DPS
		tel.T.Record("ability", "burn")
		if !wasBurning && onCC != nil {
			onCC(target.X, target.Y, "burn")
		}
	}
	if r.Splash != nil && enemies != nil {
		tel.T.Record("ability", "splash")
		splashDamage := p.Damage * r.Splash.Ratio
		splashTower := srcTower
		enemies.Each(func(e *enemy.Enemy) {
			if e == target || e.IsDying() {
				return
			}
			if math.Hypot(e.X-target.X, e.Y-target.Y) <= r.Splash.Radius {
				// 溅射走完整 ApplyHit（含 OnHit 能力触发），但 Enemies=nil 防止递归溅射
				splashResult := ApplyHit(HitInput{
					Tower:      splashTower,
					Target:     e,
					BaseDamage: splashDamage,
					Style:      "splash",
					Enemies:    nil, // 阻断递归溅射
					Projectiles: projectiles,
					Projectile: p,
					OnCC:       onCC,
				}, onHit)
				if e.HitFlash < 0.06 && e.Age > 0.1 {
					e.HitFlash = 0.08
				}
				if splashResult.Killed {
					enemies.Kill(e)
				}
			}
		})
	}
	// 弹幕盾阻止弹射继续链接
	if r.Bounce != nil && p.BounceCount < r.Bounce.MaxBounces && projectiles != nil && !(target.ProjectileBlockChance > 0 && !target.AbilitySilenced) {
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
			tel.T.Record("ability", "bounce")
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
	str := 100.0
	if t.Strength != nil {
		str = t.Strength.Effective()
	}
	explodeDmg := 10 + 15*(str/100.0) // 默认值
	explodeR := 50.0
	if abTable := config.GlobalAbilityTable(); abTable != nil {
		if def := abTable["deathMark"]; def != nil {
			explodeDmg = def.CalcScale(str)
			if def.Param > 0 {
				explodeR = def.Param
			}
		}
	}

	extraKills := 0
	enemies.Each(func(e2 *enemy.Enemy) {
		if e2 == killed || e2.IsDying() {
			return
		}
		if math.Hypot(e2.X-killed.X, e2.Y-killed.Y) <= explodeR {
			er := ProcessDamage(DamageInput{
				Target:     e2,
				RawDamage:  explodeDmg,
				DamageType: DmgPhysical,
			})
			finalDmg := er.FinalDamage
			if er.Blocked {
				finalDmg = 0
			}
			if onHit != nil {
				onHit(e2, finalDmg, er.Killed, "explosion", false)
			}
			if er.Killed {
				enemies.Kill(e2)
				extraKills++
			}
		}
	})
	return extraKills
}
