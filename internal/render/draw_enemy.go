// draw_enemy.go — enemy rendering.
// Uses PNG sprites with fallback circles. Themed HP bar, boss pulsing rings,
// runner/tank/flying visuals, and status effect dots.
package render

import (
	"image/color"
	"math"
	"slices"

	"defense2/internal/core/enemy"
	"defense2/internal/render/anim"
	"defense2/internal/render/draw"
	"defense2/internal/render/sprite"
	"defense2/internal/render/theme"
	"defense2/internal/render/vfx"

	"github.com/hajimehoshi/ebiten/v2"
)

// EnemyRenderer manages enemy PNG sprite rendering with optional frame animation.
type EnemyRenderer struct {
	cache    *sprite.Cache
	assetFS  AssetReader
	animLibs map[string]*anim.AnimLib // per archetype shared frame data
}

// NewEnemyRenderer creates an enemy renderer.
func NewEnemyRenderer(assetFS AssetReader) *EnemyRenderer {
	return &EnemyRenderer{
		cache:    sprite.NewCache(),
		assetFS:  assetFS,
		animLibs: make(map[string]*anim.AnimLib),
	}
}

const enemySpriteSize = 32

// hpBarEntry holds data for deferred HP bar rendering (Pass 3).
// Collected during Pass 2, then sorted and repulsed to avoid overlap.
type hpBarEntry struct {
	cx, cy     float32 // enemy center
	barW, barH float32
	barOffY    float32 // base Y offset above enemy center
	adjustY    float32 // additional Y offset from repulsion (negative = higher)
	hp, maxHP  float64
	displayHP  float64
	boss       bool
	// status dots
	slowed, stunned, rooted, bleeding, burning, poisoned, weakened bool
}

// spritePathCache caches fmt.Sprintf results to avoid per-frame allocations.
var spritePathCache = map[string]string{}

func cachedEnemySpritePath(dir string) string {
	if p, ok := spritePathCache[dir]; ok {
		return p
	}
	p := "assets/enemies/sprites/" + dir + "/" + dir + ".png"
	spritePathCache[dir] = p
	return p
}

func cachedShieldPath(name string) string {
	key := "shield:" + name
	if p, ok := spritePathCache[key]; ok {
		return p
	}
	p := "assets/enemies/shields/" + name + ".png"
	spritePathCache[key] = p
	return p
}

// DrawEnemies renders all alive enemies.
// Two-pass rendering: dying enemies first (behind), then active enemies on top.
func (er *EnemyRenderer) DrawEnemies(screen *ebiten.Image, pool *enemy.Pool, animTime float64) {
	// Pass 1: dying enemies (rendered behind active ones)
	pool.Each(func(e *enemy.Enemy) {
		if !e.IsDying() {
			return
		}
		if e.DyingDuration <= 0 {
			return
		}
		// Viewport culling: skip dying enemies outside camera view
		if !IsInView(e.X, e.Y) {
			return
		}
		cx := float32(e.X)
		cy := float32(e.Y)
		scale, alphaF, offsetY := vfx.DyingAnimParams(e.DyingTimer, e.DyingDuration)
		alpha := float32(alphaF)

		img := er.getEnemyFrame(e, 1.0/60.0)
		if img != nil {
			displaySize := float64(enemySpriteSize) * scale
			if displaySize < 0.5 {
				return // too small to see
			}
			w := float64(img.Bounds().Dx())
			h := float64(img.Bounds().Dy())
			s := displaySize / w * draw.Scale
			var op ebiten.DrawImageOptions
			op.GeoM.Translate(-w/2, -h/2)
			op.GeoM.Scale(s, s)
			op.GeoM.Translate(float64(cx)*draw.Scale, (float64(cy)+offsetY)*draw.Scale)
			op.ColorScale.ScaleAlpha(alpha)
			screen.DrawImage(img, &op)
		}
	})

	// HP bar entries collected during Pass 2, drawn in Pass 3 with repulsion.
	var hpBarsArr [256]hpBarEntry
	hpBars := hpBarsArr[:0]

	// Pass 2: active (non-dying) enemies
	pool.Each(func(e *enemy.Enemy) {
		if e.IsDying() {
			return
		}

		// Viewport culling: skip rendering for off-screen enemies
		if !IsInView(e.X, e.Y) {
			return
		}

		cx := float32(e.X)
		cy := float32(e.Y)

		r := float32(e.Radius)

		// --- Boss pulsing rings ---
		if e.Boss {
			vfx.DrawBossPulse(screen, cx, cy, r, animTime)
		}

		// --- Runner pulsing ring ---
		if e.Archetype == "runner" {
			vfx.DrawRunnerRing(screen, cx, cy, r, animTime)
		}

		// --- Root ground effect (drawn UNDER enemy body) ---
		if e.IsRooted() {
			vfx.DrawRootGround(screen, cx, cy, r)
		}

		// --- Buffer aura ring (drawn UNDER body, hidden when silenced) ---
		if e.Behavior == "buffer" && e.HasBufferAura() && !e.AbilitySilenced {
			if ba, ok := e.Buffs.Get("bufferAura"); ok {
				vfx.DrawBufferAura(screen, cx, cy, ba.Value2, animTime)
			}
		}

		// (旧 healer aura ring 已移到能力 VFX 系统)

		// --- Spawn animation modifiers ---
		spawnScale, spawnAlpha := 1.0, 1.0
		if e.IsSpawning() && e.SpawnDuration > 0 {
			spawnScale, spawnAlpha = vfx.SpawnAnimParams(e.SpawnTimer, e.SpawnDuration)
		}

		// --- Enemy body (animated or static) ---
		img := er.getEnemyFrame(e, 1.0/60.0)
		if img != nil {
			// 行走摆动：用 X 位置作为相位，产生微小的上下浮动和旋转
			wobblePhase := e.X*0.05 + animTime*4
			wobbleY := math.Sin(wobblePhase) * 1.5
			wobbleRot := math.Sin(wobblePhase) * 0.05 // ~3 degrees
			if e.IsStunned() || e.IsRooted() {
				wobbleY = 0
				wobbleRot = 0
			}
			// Boss/Elite 呼吸缩放：体型周期性脉冲
			displaySize := float64(enemySpriteSize)
			if e.Boss {
				displaySize *= 1.0 + 0.04*math.Sin(animTime*1.8)
			}
			// Apply spawn scale
			displaySize *= spawnScale

			// 计算 alpha（隐身/相位）
			bodyAlpha := 1.0
			if e.IsStealthed() {
				bodyAlpha = 0.15
			} else if e.PhaseActive {
				bodyAlpha = 0.35
			}
			// Apply spawn alpha
			bodyAlpha *= spawnAlpha

			if bodyAlpha < 1.0 || spawnScale != 1.0 {
				logicalScale := displaySize / float64(img.Bounds().Dx())
				draw.SpriteScaledRotatedAlpha(screen, img, float64(cx), float64(cy)+wobbleY,
					logicalScale, wobbleRot, bodyAlpha)
			} else {
				draw.SpriteRotated(screen, img, float64(cx), float64(cy), displaySize, wobbleRot, wobbleY)
			}
		} else {
			bodyColor := theme.EnemyFallback
			if e.Boss {
				bodyColor = theme.EnemyFallbackBoss
			}
			if e.IsStealthed() {
				bodyColor.A = 38
			} else if e.PhaseActive {
				bodyColor.A = 90
			}
			// Apply spawn alpha to fallback circle
			bodyColor.A = uint8(float64(bodyColor.A) * spawnAlpha)
			draw.FilledCircle(screen, cx, cy, r*float32(spawnScale), bodyColor)
		}

		// --- Buff behavior VFX (drawn over body) ---
		if e.GetDamageReduce() > 0 {
			vfx.DrawDamageReduceShield(screen, cx, cy, float32(e.Radius), animTime)
		}
		if e.HasBerserk() && e.BerserkTriggered {
			vfx.DrawBerserkFlare(screen, cx, cy, float32(e.Radius), animTime)
		}
		if e.HasRegen() {
			vfx.DrawRegenAura(screen, cx, cy, float32(e.Radius), animTime)
		}

		// --- Status effect body overlays (subtle, sprite-sized) ---
		spriteR := float32(enemySpriteSize) / 2
		if e.IsSlowed() {
			vfx.DrawSlowOverlay(screen, cx, cy, spriteR, animTime)
		}
		if e.IsBurning() {
			vfx.DrawBurnOverlay(screen, cx, cy, spriteR, animTime)
		}
		if e.Buffs != nil && e.Buffs.Has("poison") {
			vfx.DrawPoisonOverlay(screen, cx, cy, spriteR, animTime)
		}

		// --- Stun rotating stars ---
		if e.IsStunned() {
			vfx.DrawStunStars(screen, cx, cy, r, animTime)
		}

		// --- Hit flash overlay ---
		if e.HitFlash > 0 && !e.IsDying() {
			vfx.DrawHitFlash(screen, cx, cy, spriteR, e.HitFlash)
		}

		// (tank overlay removed — was debug placeholder)

		// Stealthed enemies: skip HP bar and status dots (nearly invisible)
		if e.IsStealthed() {
			return
		}

		// Collect HP bar entries for Pass 3 (deferred drawing with repulsion).
		// Skip full-HP enemies unless they have status effects (C: hide-when-full).
		hasStatus := e.IsSlowed() || e.IsStunned() || e.IsRooted() || e.IsBleeding() || e.IsBurning() || e.IsPoisoned() || e.IsWeakened()
		if e.HP < e.MaxHP || e.Boss || hasStatus {
			var barW, barH, barOffY float32
			if e.Boss {
				barW = theme.EnemyBossHPBarW
				barH = theme.EnemyBossHPBarH
				barOffY = theme.EnemyBossHPOffsetY
			} else {
				barW = theme.EnemyHPBarW
				barH = theme.EnemyHPBarH
				barOffY = theme.EnemyHPBarOffsetY
			}
			hpBars = append(hpBars, hpBarEntry{
				cx: cx, cy: cy,
				barW: barW, barH: barH, barOffY: barOffY,
				hp: e.HP, maxHP: e.MaxHP, displayHP: e.DisplayHP,
				boss:     e.Boss,
				slowed:   e.IsSlowed(),
				stunned:  e.IsStunned(),
				rooted:   e.IsRooted(),
				bleeding: e.IsBleeding(),
				burning:  e.IsBurning(),
			poisoned: e.IsPoisoned(),
			weakened: e.IsWeakened(),
			})
		}

		// --- 能力常驻视觉（被沉默时全部隐藏）---
		if !e.AbilitySilenced {
			// 免疫脚环（只显示天生能力，净化临时免疫用白色微光）
			footR := float32(e.Radius) + 2
			if hasAbility(e, "ccImmune") {
				vfx.DrawImmunityRing(screen, cx, cy, footR, theme.EnemyImmuneCC, animTime)
			} else if hasAbility(e, "slowImmune") {
				vfx.DrawImmunityRing(screen, cx, cy, footR, theme.EnemyImmuneSlow, animTime)
			}

			// 盾牌叠加（能力对应颜色盾牌）
			shieldOffset := float32(e.Radius) * 0.6
			if e.ProjectileBlockChance > 0 {
				if img := er.loadShield("shield-white"); img != nil {
					draw.Sprite(screen, img, float64(cx+shieldOffset), float64(cy), 14)
				}
			}
			if e.ArmorFlat > 0 {
				if img := er.loadShield("shield-blue"); img != nil {
					draw.Sprite(screen, img, float64(cx+shieldOffset), float64(cy), 14)
				}
			}
			if e.DamageCap > 0 || e.DamageCapPercent > 0 {
				if img := er.loadShield("shield-orange"); img != nil {
					draw.Sprite(screen, img, float64(cx+shieldOffset), float64(cy), 14)
				}
			}

			// 净化免疫期白色微光
			if e.PurgeInterval > 0 && e.HasControlImmunity() {
				vfx.DrawPurgeGlow(screen, cx, cy, float32(e.Radius), animTime)
			}
		}

		// --- 飘字渲染 ---
		if e.FloatText != "" && e.FloatTextTimer > 0 {
			fm := GlobalFont()
			if fm != nil {
				progress := 1 - e.FloatTextTimer/0.6
				floatY := float64(cy) - float64(e.Radius) - 10 - progress*12
				alpha := uint8(255 * (1 - progress))
				fm.DrawCenteredText(screen, e.FloatText, float64(cx), floatY, 10,
					color.RGBA{R: e.FloatTextR, G: e.FloatTextG, B: e.FloatTextB, A: alpha})
			}
		}
	})

	// Pass 3: draw HP bars with Y-axis repulsion to reduce overlap.
	if len(hpBars) > 0 {
		repulseHPBars(hpBars)
		drawHPBars(screen, hpBars, animTime)
	}
}

// repulseHPBars applies simple Y-axis repulsion so overlapping HP bars spread apart.
// Sort by bar-center Y, then push overlapping bars upward.
func repulseHPBars(bars []hpBarEntry) {
	if len(bars) < 2 {
		return
	}
	// Sort by the bar's screen Y position (enemy cy - barOffY).
	slices.SortFunc(bars, func(a, b hpBarEntry) int {
		ay := a.cy - a.barOffY
		by := b.cy - b.barOffY
		if ay < by {
			return -1
		}
		if ay > by {
			return 1
		}
		return 0
	})

	const minGap float32 = 2 // minimum vertical gap between bars

	for i := 1; i < len(bars); i++ {
		prev := &bars[i-1]
		cur := &bars[i]

		// Check horizontal overlap first — bars far apart in X don't need repulsion.
		maxW := prev.barW
		if cur.barW > maxW {
			maxW = cur.barW
		}
		dx := cur.cx - prev.cx
		if dx < 0 {
			dx = -dx
		}
		if dx > maxW {
			continue // no horizontal overlap
		}

		prevBottom := prev.cy - prev.barOffY + prev.adjustY + prev.barH
		curTop := cur.cy - cur.barOffY + cur.adjustY

		overlap := prevBottom + minGap - curTop
		if overlap > 0 {
			// Push current bar down, previous bar up (split evenly).
			half := overlap / 2
			prev.adjustY -= half
			cur.adjustY += half
		}
	}
}

// drawHPBars renders all collected HP bar entries.
func drawHPBars(screen *ebiten.Image, bars []hpBarEntry, animTime float64) {
	for i := range bars {
		b := &bars[i]
		barX := b.cx - b.barW/2
		barY := b.cy - b.barOffY + b.adjustY

		// Only draw HP bar if not full HP (status-only entries skip the bar).
		if b.hp < b.maxHP {
			// Background (1px pseudo-border)
			draw.FilledRect(screen, barX-1, barY-1, b.barW+2, b.barH+2,
				theme.EnemyHPBarBg, true)

			// Damage trail (orange)
			if b.displayHP > b.hp && b.displayHP > 0 {
				trailRatio := float32(b.displayHP / b.maxHP)
				if trailRatio > 1 {
					trailRatio = 1
				}
				draw.FilledRect(screen, barX, barY, b.barW*trailRatio, b.barH,
					theme.EnemyHPBarTrail, true)
			}

			// HP fill
			ratio := float32(b.hp / b.maxHP)
			if ratio < 0 {
				ratio = 0
			} else if ratio > 1 {
				ratio = 1
			}
			fillW := b.barW * ratio

			var fillClr color.RGBA
			switch {
			case ratio > 0.6:
				fillClr = theme.EnemyHPFillHigh
			case ratio > 0.3:
				fillClr = theme.EnemyHPFillMid
			default:
				fillClr = theme.EnemyHPFillLow
			}
			draw.FilledRect(screen, barX, barY, fillW, b.barH, fillClr, true)

			// Boss segment dividers
			if b.boss {
				for s := 1; s < 5; s++ {
					divX := barX + b.barW*float32(s)*0.2
					draw.FilledRect(screen, divX, barY, 1, b.barH, theme.EnemyHPSegDiv, true)
				}
			}
		}

		// Status effect dots (above the bar)
		dotY := barY - 4
		var dotsArr [7]vfx.StatusDot
		dots := dotsArr[:0]
		if b.slowed {
			dots = append(dots, vfx.StatusDot{Color: theme.EnemyDotSlowed})
		}
		if b.stunned {
			dots = append(dots, vfx.StatusDot{Color: theme.EnemyDotStunned})
		}
		if b.rooted {
			dots = append(dots, vfx.StatusDot{Color: theme.EnemyDotRooted})
		}
		if b.bleeding {
			dots = append(dots, vfx.StatusDot{Color: theme.EnemyDotBleeding})
		}
		if b.burning {
			dots = append(dots, vfx.StatusDot{Color: theme.EnemyDotBurning})
		}
		if b.poisoned {
			dots = append(dots, vfx.StatusDot{Color: theme.EnemyDotPoison})
		}
		if b.weakened {
			dots = append(dots, vfx.StatusDot{Color: theme.EnemyDotWeaken})
		}
		if len(dots) > 0 {
			vfx.DrawStatusDots(screen, b.cx, dotY, dots, animTime)
		}
	}
}

// hasAbility 检查敌人是否装配了指定能力。
func hasAbility(e *enemy.Enemy, abilityType string) bool {
	for _, id := range e.AbilityIDs {
		if id == abilityType {
			return true
		}
	}
	return false
}

// loadShield 加载盾牌 PNG（缓存）。
func (er *EnemyRenderer) loadShield(name string) *ebiten.Image {
	path := cachedShieldPath(name)
	if cached := er.cache.Get(path, 12, 16); cached != nil {
		return cached
	}
	if er.assetFS == nil {
		return nil
	}
	data, err := er.assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, _ := er.cache.GetOrParse(path, data, 12, 16)
	return img
}

// loadEnemyImage loads an enemy's PNG sprite.
// Convention: assets/enemies/sprites/{spriteDir}/{spriteDir}.png
func (er *EnemyRenderer) loadEnemyImage(e *enemy.Enemy) *ebiten.Image {
	spriteDir := e.SpriteDir
	if spriteDir == "" {
		spriteDir = e.Archetype
	}
	if er.assetFS == nil || spriteDir == "" {
		return nil
	}
	path := cachedEnemySpritePath(spriteDir)
	cached := er.cache.Get(path, enemySpriteSize, enemySpriteSize)
	if cached != nil {
		return cached
	}
	data, err := er.assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, err := er.cache.GetOrParse(path, data, enemySpriteSize, enemySpriteSize)
	if err != nil {
		return nil
	}
	return img
}

// GetSprite 按原型名加载敌人静态精灵（供造怪菜单等外部使用）。
func (er *EnemyRenderer) GetSprite(archetype string) *ebiten.Image {
	if er.assetFS == nil || archetype == "" {
		return nil
	}
	path := cachedEnemySpritePath(archetype)
	if cached := er.cache.Get(path, enemySpriteSize, enemySpriteSize); cached != nil {
		return cached
	}
	data, err := er.assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, _ := er.cache.GetOrParse(path, data, enemySpriteSize, enemySpriteSize)
	return img
}

// getEnemyFrame returns the current animation frame for an enemy, falling back to static sprite.
// Uses per-enemy animation state (AnimCur/AnimFrame/AnimTimer/AnimDone) with shared AnimLib.
func (er *EnemyRenderer) getEnemyFrame(e *enemy.Enemy, dt float64) *ebiten.Image {
	if e.Archetype == "" {
		return nil
	}

	sprKey := e.SpriteDir
	if sprKey == "" {
		sprKey = e.Archetype
	}
	lib, ok := er.animLibs[sprKey]
	if !ok {
		lib = anim.LoadEnemyAnimLib(er.assetFS, sprKey)
		er.animLibs[sprKey] = lib
	}

	// Determine target animation
	target := "walk"
	if e.HitFlash > 0 && lib.HasAnim("hit") {
		target = "hit"
	}

	// Switch animation if needed (reset on change or when finished non-loop replays)
	if e.AnimCur != target || (e.AnimDone && target != e.AnimCur) {
		e.AnimCur = target
		e.AnimFrame = 0
		e.AnimTimer = 0
		e.AnimDone = false
	}

	// Advance per-enemy timer
	a, exists := lib.Anims[e.AnimCur]
	if exists && !e.AnimDone && len(a.Frames) > 0 {
		e.AnimTimer += dt
		frameDur := 1.0 / a.FPS
		if e.AnimTimer >= frameDur {
			e.AnimTimer -= frameDur
			e.AnimFrame++
			if e.AnimFrame >= len(a.Frames) {
				if a.Loop {
					e.AnimFrame = 0
				} else {
					e.AnimFrame = len(a.Frames) - 1
					e.AnimDone = true
				}
			}
		}
	}

	img := lib.Frame(e.AnimCur, e.AnimFrame)
	if img != nil {
		return img
	}
	return er.loadEnemyImage(e)
}
