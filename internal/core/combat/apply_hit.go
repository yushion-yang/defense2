// apply_hit.go — 统一命中处理（战斗系统的核心枢纽）。
// 所有攻击方式（弹射物/即时光束/范围AoE等）共用此函数处理能力触发+扣血+击杀。
//
// 本文件连接三大子系统：
//   - attack.go 的各种 handler → 命中时调用 ApplyHit
//   - tower 的能力系统 → OnHit 回调产生 debuff/弹射/溅射等效果
//   - damage_pipeline.go → 最终伤害计算和扣血
//
// 关键设计决策：
//   - splash 递归调用 ApplyHit 时传 Enemies=nil，阻断无限溅射链
//   - separateDmg 走独立的伤害管线，让固伤能力有自己的 damageCap 额度
//   - 弹幕盾（shield）只阻止传播（弹射/穿透停止），不减少当前命中的伤害
//   - 能力 OnHit 在伤害计算前执行，weaken 等 debuff 立即对本次命中生效
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
	"defense2/internal/i18n"
)

const (
	dodgeFlashDuration  = 0.3  // 闪避残影时长(秒)
	armorSparkDuration  = 0.15 // 装甲火花时长(秒)
	blockFlashDuration  = 0.25 // 弹幕盾脉冲时长(秒)
	freezeSlowThreshold = 0.4  // 减速达此值时触发冻结CC回调
)

// blockableStyles 定义弹幕盾可拦截的攻击方式。
// 设计意图：盾牌阻止传播（弹射链中断/穿透弹停止）但不减免当前命中伤害。
// 这让盾牌成为"反传播"机制而非"减伤"机制，避免与 damageReduce 功能重叠。
var blockableStyles = map[string]bool{
	tower.AbilityBounce:  true,
	tower.AbilityScatter: true,
	tower.AbilityRadial:  true,
	"fireball":           true, // 战灵火球（warden 系统的弹射物）
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
	ProjectileBlocked bool // 弹幕盾：阻止弹射物继续传播（弹射/穿透停止）
}

// ApplyHit 统一命中处理入口。
//
// 完整处理流程：
//  1. 闪避判定 — EvasionChance 触发则直接返回 Dodged，不进入后续步骤
//  2. 弹幕盾预判 — 记录 shieldBlocked 标记，不影响伤害，仅在末尾阻止传播
//  3. 装甲固定减免 — ArmorFlat 扣减基础伤害，最低保底 1 点
//  4. 受击冲刺触发 — DashSpeedBoost 让敌人被打后短暂加速脱离
//  5. 能力 OnHit 遍历 — 遍历塔的所有能力，触发 debuff/弹射/溅射/暴击等效果
//     - weaken 等 debuff 在此设置，立即对本次命中的伤害管线生效
//     - separateDmg 累加到独立伤害池（对抗 damageCap 的关键设计）
//     - splash 递归调用 ApplyHit(Enemies=nil) 阻断无限溅射
//  6. 暴击判定 — 能力未触发暴击时，CritBonus 独立再判一次
//  7. 全伤害增幅 — DamageAmp（来自 damageUpAura）统一乘算
//  8. 主伤害管线 — 调用 ApplyDamage() 走 8 步管线扣血
//  9. 独立二段伤害 — separateDmg 单独走一次 ApplyDamage（独立 damageCap）
//
// 10. 命中回调 — 触发飘字/音效
// 11. 击杀处理 — 递增 Kills 计数，触发 deathMark 爆炸，调用 pool.Kill
// 12. 弹幕盾视觉 — shieldBlocked 时触发 BlockFlash 脉冲动画
func ApplyHit(input HitInput, onHit HitCallback) HitOutput {
	e := input.Target

	// ── 怪物闪避（完全回避，不触发任何 OnHit）──
	if e.EvasionChance > 0 && !e.AbilitySilenced {
		if rand.Float64() < e.EvasionChance {
			e.DodgeFlash = dodgeFlashDuration
			e.SetFloatText(i18n.T("combat.dodge"), 255, 255, 255)
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

	// 构建合成弹射物用于能力 OnHit 调用。
	// 即时伤害（如 spin_aoe、wideBeam）没有真实弹射物，需要构造一个携带来源信息的
	// 合成对象，让能力的 OnHit 回调能正确读取 SourceTowerKey 和 Damage。
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

	// 遍历塔能力，触发 OnHit（weaken 等 debuff 在此设置，立即对本次命中生效）。
	// separateDmg 是独立伤害池：某些能力（如固伤类）将伤害累加到此变量，
	// 后续单独走一次 ApplyDamage，使其有独立的 damageCap 额度。
	// 这是对抗高 damageCap 敌人的核心机制——主伤害被 cap 住，但 separateDmg 不受影响。
	var separateDmg float64
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

	// CritBonus 暴击：能力层未触发暴击时，塔自身的 CritBonus 再独立判定一次。
	// CritBonus 来源于 critAura 光环，即使塔没有暴击能力也能通过光环获得暴击机会。
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

	// 独立二段伤害：单独走一次伤害管线，拥有独立的 damageCap 额度。
	// 必须检查目标仍存活（!killed && Active && !IsDying），避免对已死敌人重复扣血。
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
	if killed && input.Tower != nil {
		input.Tower.Kills++
		if input.Enemies != nil {
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
		ProjectileBlocked: shieldBlocked,
	}
}

// applyHitEffectsUnified 施加能力效果（减速、眩晕、流血、灼烧、溅射、弹射）。
//
// 处理顺序：slow → stun → bleed → burn → splash → bounce
// 这个顺序很重要：debuff（slow/stun/bleed/burn）先施加，确保溅射/弹射命中时
// 目标已经带有 debuff。splash 在 bounce 之前，因为溅射是以当前目标为中心的 AoE，
// 而弹射是向下一个目标发射新弹射物。
//
// splash 递归调用 ApplyHit 时传 Enemies=nil，这是防止无限溅射链的关键设计：
// 溅射命中的敌人不会再触发新的溅射，但仍会触发其他能力效果（slow/stun 等）。
func applyHitEffectsUnified(r *tower.HitResult, target *enemy.Enemy, p *projectile.Projectile, srcTower *tower.Tower, enemies *enemy.Pool, projectiles *projectile.Pool, onHit HitCallback, onCC CCCallback, onSplashVFX func(x, y, radius float64)) {
	// 减速效果：factor < 0.4 时视觉上显示为"冻结"（更强的减速用不同音效/特效）
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
			target.SetFloatText(i18n.T("combat.bleed"), 220, 60, 60)
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
			target.SetFloatText(i18n.T("combat.burn"), 255, 140, 40)
		}
		tel.T.Record("ability", "burn")
		if !wasBurning && onCC != nil {
			onCC(target.X, target.Y, CCBurn)
		}
	}
	// 溅射 AoE：以命中目标为圆心，半径内所有敌人受到比例伤害。
	// enemies != nil 是递归防护——splash 命中的敌人再次走 ApplyHit 时 Enemies=nil，
	// 到这里 enemies==nil 就不会再触发新的溅射。
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
		enemies.EachActive(func(e *enemy.Enemy) {
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
				e.TriggerHitFlash()
				if splashResult.Killed {
					enemies.Kill(e)
				}
			}
		})
	}
	// 弹射链：向范围内最近的未命中敌人发射新弹射物。
	// 弹幕盾检查：如果当前目标有盾，弹射链在此中断（不发射新弹）。
	// BounceCount 追踪已弹射次数，超过 MaxBounces 自然终止。
	if r.Bounce != nil && p.BounceCount < r.Bounce.MaxBounces && projectiles != nil && !ShouldShieldBlock(target, tower.AbilityBounce) {
		// 栈数组避免堆分配：弹射记录使用固定大小数组而非 slice，
		// 因为 MaxBounces 通常 <= 5，8 元素栈数组足够且零 GC 压力。
		var hitBuf [8]int
		n := copy(hitBuf[:], p.BounceHitIDs)
		hitBuf[n] = target.ID
		hitIDs := hitBuf[:n+1]

		var best *enemy.Enemy
		bestDist := r.Bounce.Range
		if enemies != nil {
			enemies.EachActive(func(e2 *enemy.Enemy) {
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

