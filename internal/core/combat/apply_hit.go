// apply_hit.go — 统一命中处理。
// 所有攻击方式（弹射物/即时光束/范围AoE等）共用此函数处理能力触发+扣血+击杀。
package combat

import (
	"math"
	"math/rand"

	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	tel "defense2/internal/core/telemetry"
	"defense2/internal/core/tower"
)

const (
	dodgeFlashDuration  = 0.3   // 闪避残影时长(秒)
	armorSparkDuration  = 0.15  // 装甲火花时长(秒)
	blockFlashDuration  = 0.25  // 弹幕盾脉冲时长(秒)
	freezeSlowThreshold = 0.4   // 减速达此值时触发冻结CC回调
	defaultDeathExpStr  = 100.0 // 死亡爆炸默认强度
	defaultDeathExpDmg  = 10.0  // 死亡爆炸基础伤害
	defaultDeathExpCoef = 15.0  // 死亡爆炸强度系数
	defaultDeathExpR    = 50.0  // 死亡爆炸默认半径
)

// blockableStyles 定义弹幕盾可拦截的攻击方式。
// 盾牌阻止传播（穿透/弹射）但不阻止伤害。
var blockableStyles = map[string]bool{
	tower.AbilityBounce:  true,
	tower.AbilityScatter: true,
	tower.AbilityRadial:  true,
	tower.AbilityBarrage: true,
	"fireball":           true, // 战灵火球
}

// ShouldShieldBlock 返回弹幕盾是否应拦截指定攻击方式的传播。
func ShouldShieldBlock(e *enemy.Enemy, style string) bool {
	return e.ProjectileBlockChance > 0 && !e.AbilitySilenced && blockableStyles[style]
}

// HitInput 描述一次命中事件。
type HitInput struct {
	Tower       *tower.Tower               // 来源塔（可为 nil，如战灵弹射物）
	Target      *enemy.Enemy               // 命中目标
	BaseDamage  float64                    // 基础伤害
	Style       string                     // 攻击方式标识
	Enemies     *enemy.Pool                // 用于 splash/bounce
	Projectiles *projectile.Pool           // 用于 bounce
	Projectile  *projectile.Projectile     // 原始弹射物（弹射物路径传入，即时伤害传 nil）
	OnCC        CCCallback                 // CC 效果命中回调（可为 nil）
	OnSplashVFX func(x, y, radius float64) // 溅射 VFX 回调（可为 nil）
}

// HitOutput 命中结果。
type HitOutput struct {
	TotalDamage       float64
	IsCrit            bool
	Killed            bool
	Dodged            bool // 目标闪避了攻击
	ExtraKills        int  // deathMark 等额外击杀
	ProjectileBlocked bool // 弹幕盾：阻止弹射物继续传播（弹射/穿透停止）
}

// ApplyHit 统一命中处理：遍历能力 → 计算最终伤害 → 扣血 → 击杀检查。
func ApplyHit(input HitInput, onHit HitCallback) HitOutput {
	e := input.Target

	// ── 怪物闪避（完全回避，不触发任何 OnHit）──
	if e.EvasionChance > 0 && !e.AbilitySilenced {
		if rand.Float64() < e.EvasionChance {
			e.DodgeFlash = dodgeFlashDuration
			e.SetFloatText("闪避", 255, 255, 255)
			if input.OnCC != nil {
				input.OnCC(e.X, e.Y, CCDodge)
			}
			return HitOutput{Dodged: true}
		}
	}

	// 弹幕盾标记（正常受伤，末尾标记阻止弹射物继续传播）
	shieldBlocked := ShouldShieldBlock(e, input.Style)

	totalDmg := input.BaseDamage
	isCrit := false

	// ── 装甲固定减免 ──
	if e.ArmorFlat > 0 && !e.AbilitySilenced {
		totalDmg -= e.ArmorFlat
		if totalDmg < 1 {
			totalDmg = 1
		}
		e.ArmorSpark = armorSparkDuration // 触发装甲火花视觉
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
	var separateDmg float64 // 独立二段伤害（对抗 damageCap）
	if input.Tower != nil {
		for _, aName := range input.Tower.Abilities {
			ab, ok := tower.Lookup(aName)
			if !ok {
				continue
			}
			result := ab.OnHit(input.Tower, synth, input.Target)
			if result == nil {
				continue
			}
			totalDmg += result.BonusDamage
			separateDmg += result.SeparateDamage
			if result.IsCrit {
				isCrit = true
			}
			applyHitEffectsUnified(result, input.Target, synth, input.Tower, input.Enemies, input.Projectiles, onHit, input.OnCC, input.OnSplashVFX)
		}
	}

	// CritBonus 暴击（无 crit 能力时 critAura 仍可独立触发）
	if !isCrit && input.Tower != nil && input.Tower.CritBonus > 0 {
		if rand.Float64() < input.Tower.CritBonus {
			totalDmg *= config.GlobalBalance().Combat.CritMultiplier
			isCrit = true
		}
	}

	// DamageAmp 全伤害增幅（damageUpAura 提供，所有伤害类型统一乘算）
	if input.Tower != nil && input.Tower.DamageAmp > 0 {
		totalDmg *= 1 + input.Tower.DamageAmp
	}

	// 扣血 — 走伤害管线（免疫/减免/阈值/遥测统一处理）
	pipeResult := ApplyDamage(DamageInput{
		Target:     input.Target,
		RawDamage:  totalDmg,
		DamageType: DmgPhysical, // 塔弹射物默认物理伤害
	})

	killed := pipeResult.Killed
	finalDmg := pipeResult.FinalDamage
	if pipeResult.Blocked {
		finalDmg = 0
	}

	// 独立二段伤害（固伤能力，单独走伤害管线以独立受 damageCap 限制）
	if separateDmg > 0 && !killed && e.Active && !e.IsDying() {
		sep := ApplyDamage(DamageInput{
			Target:     e,
			RawDamage:  separateDmg,
			DamageType: DmgPhysical,
		})
		finalDmg += sep.FinalDamage
		if sep.Killed {
			killed = true
		}
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

	// 弹幕盾：正常受伤后标记阻止传播 + 触发视觉
	if shieldBlocked {
		e.BlockFlash = blockFlashDuration
	}

	return HitOutput{
		TotalDamage:       finalDmg,
		IsCrit:            isCrit,
		Killed:            killed,
		ExtraKills:        extraKills,
		ProjectileBlocked: shieldBlocked,
	}
}

// applyHitEffectsUnified 施加能力效果（减速、眩晕、流血、灼烧、溅射、弹射）。
func applyHitEffectsUnified(r *tower.HitResult, target *enemy.Enemy, p *projectile.Projectile, srcTower *tower.Tower, enemies *enemy.Pool, projectiles *projectile.Pool, onHit HitCallback, onCC CCCallback, onSplashVFX func(x, y, radius float64)) {
	if r.Slow != nil {
		if ApplySlow(target, r.Slow.Factor, r.Slow.Duration, p.SourceTowerKey) && onCC != nil {
			if r.Slow.Factor < freezeSlowThreshold {
				onCC(target.X, target.Y, CCFreeze)
			} else {
				onCC(target.X, target.Y, CCSlow)
			}
		}
		tel.T.Record("ability", "slow")
	}
	if r.Stun != nil {
		if ApplyStun(target, r.Stun.Duration, p.SourceTowerKey) && onCC != nil {
			onCC(target.X, target.Y, CCStun)
		}
		tel.T.Record("ability", "stun")
	}
	if r.Bleed != nil {
		wasBleeding := target.Buffs.Has(buff.IDBleed)
		target.Buffs.Add(buff.Buff{
			ID:        buff.IDBleed,
			Category:  buff.CatDoT,
			Source:    p.SourceTowerKey,
			Value:     r.Bleed.DPS,
			Duration:  r.Bleed.Duration,
			Remaining: r.Bleed.Duration,
		})
		if !wasBleeding {
			target.SetFloatText("流血", 220, 60, 60)
		}
		tel.T.Record("ability", "bleed")
	}
	if r.Burn != nil {
		wasBurning := target.Buffs.Has(buff.IDBurn)
		target.Buffs.Add(buff.Buff{
			ID:        buff.IDBurn,
			Category:  buff.CatDoT,
			Source:    p.SourceTowerKey,
			Value:     r.Burn.DPS,
			Duration:  r.Burn.Duration,
			Remaining: r.Burn.Duration,
		})
		if !wasBurning {
			target.SetFloatText("灼烧", 255, 140, 40)
		}
		tel.T.Record("ability", "burn")
		if !wasBurning && onCC != nil {
			onCC(target.X, target.Y, CCBurn)
		}
	}
	if r.Splash != nil && enemies != nil {
		tel.T.Record("ability", "splash")
		if onCC != nil {
			onCC(target.X, target.Y, CCSplash)
		}
		if onSplashVFX != nil {
			onSplashVFX(target.X, target.Y, r.Splash.Radius)
		}
		splashDamage := p.Damage * r.Splash.Ratio
		splashTower := srcTower
		enemies.Each(func(e *enemy.Enemy) {
			if e == target || e.IsDying() || e.IsSpawning() {
				return
			}
			if math.Hypot(e.X-target.X, e.Y-target.Y) <= r.Splash.Radius {
				// 溅射走完整 ApplyHit（含 OnHit 能力触发），但 Enemies=nil 防止递归溅射
				splashResult := ApplyHit(HitInput{
					Tower:       splashTower,
					Target:      e,
					BaseDamage:  splashDamage,
					Style:       "splash",
					Enemies:     nil, // 阻断递归溅射
					Projectiles: projectiles,
					Projectile:  p,
					OnCC:        onCC,
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
	if r.Bounce != nil && p.BounceCount < r.Bounce.MaxBounces && projectiles != nil && !ShouldShieldBlock(target, tower.AbilityBounce) {
		hitIDs := append([]int{}, p.BounceHitIDs...)
		hitIDs = append(hitIDs, target.ID)

		var best *enemy.Enemy
		bestDist := r.Bounce.Range
		if enemies != nil {
			enemies.Each(func(e2 *enemy.Enemy) {
				if e2.IsDying() || e2.IsSpawning() {
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
		if aName != tower.AbilityDeathMark {
			continue
		}
		ab, ok := tower.Lookup(aName)
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
	str := defaultDeathExpStr
	if t.Strength != nil {
		str = t.Strength.Effective()
	}
	explodeDmg := defaultDeathExpDmg + defaultDeathExpCoef*(str/defaultDeathExpStr)
	explodeR := defaultDeathExpR
	if abTable := config.GlobalAbilityTable(); abTable != nil {
		if def := abTable[tower.AbilityDeathMark]; def != nil {
			explodeDmg = def.CalcScale(str)
			if def.Param > 0 {
				explodeR = def.Param
			}
		}
	}

	extraKills := 0
	enemies.Each(func(e2 *enemy.Enemy) {
		if e2 == killed || e2.IsDying() || e2.IsSpawning() {
			return
		}
		if math.Hypot(e2.X-killed.X, e2.Y-killed.Y) <= explodeR {
			er := ApplyDamage(DamageInput{
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
