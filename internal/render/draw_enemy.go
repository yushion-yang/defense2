// draw_enemy.go — enemy rendering.
// Uses PNG sprites with fallback circles. Themed HP bar, boss pulsing rings,
// runner/tank/flying visuals, and status effect dots.
package render

import (
	"fmt"
	"image/color"
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/render/anim"
	"defense2/internal/render/draw"
	"defense2/internal/render/sprite"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// EnemyRenderer manages enemy PNG sprite rendering with optional frame animation.
type EnemyRenderer struct {
	cache     *sprite.Cache
	assetFS   AssetReader
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

const enemySpriteSize = 24

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
		cx := float32(e.X)
		cy := float32(e.Y)
		progress := 1.0 - e.DyingTimer/e.DyingDuration // 0→1 (0=just died, 1=gone)
		scale := 1.0 - progress                         // shrink from 1 to 0
		alpha := float32(1.0 - progress)                // fade from 1 to 0
		offsetY := -progress * 8                         // float up 8px

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

	// Pass 2: active (non-dying) enemies
	pool.Each(func(e *enemy.Enemy) {
		if e.IsDying() {
			return
		}
		cx := float32(e.X)
		cy := float32(e.Y)

		r := float32(e.Radius)

		// --- Flying enemy ground shadow ---
		if e.Archetype == "flying" {
			draw.FilledCircle(screen, cx+2, cy+8, r*1.3, theme.EnemyFlyShadow)
		}

		// --- Boss pulsing rings ---
		if e.Boss {
			innerAlpha := uint8(clampF(float64(theme.EnemyBossInner.A)*
				(0.5+0.5*math.Sin(animTime*2.5)), 0, 255))
			innerClr := color.RGBA{R: theme.EnemyBossInner.R, G: theme.EnemyBossInner.G,
				B: theme.EnemyBossInner.B, A: innerAlpha}
			draw.CircleOutline(screen, cx, cy, r+6, 2, innerClr)

			outerAlpha := uint8(clampF(float64(theme.EnemyBossOuter.A)*
				(0.5+0.5*math.Sin(animTime*2.5+1.5)), 0, 255))
			outerClr := color.RGBA{R: theme.EnemyBossOuter.R, G: theme.EnemyBossOuter.G,
				B: theme.EnemyBossOuter.B, A: outerAlpha}
			draw.CircleOutline(screen, cx, cy, r+10, 1.5, outerClr)
		}

		// --- Runner pulsing ring ---
		if e.Archetype == "runner" {
			pulseAlpha := uint8(clampF(float64(theme.EnemyRunnerPulse.A)*
				(0.5+0.5*math.Sin(animTime*3)), 0, 255))
			pulseClr := color.RGBA{R: theme.EnemyRunnerPulse.R, G: theme.EnemyRunnerPulse.G,
				B: theme.EnemyRunnerPulse.B, A: pulseAlpha}
			draw.CircleOutline(screen, cx, cy, r+4, 1.5, pulseClr)
		}

		// --- Root ground effect (drawn UNDER enemy body) ---
		if e.RootTimer > 0 {
			draw.FilledCircle(screen, cx, cy+r, r*0.8, color.RGBA{100, 70, 40, 60})
		}

		// --- Buffer aura ring (drawn UNDER body) ---
		if e.Behavior == "buffer" && e.BuffRadius > 0 {
			auraAlpha := uint8(clampF(40+20*math.Sin(animTime*3), 20, 70))
			draw.CircleOutline(screen, cx, cy, float32(e.BuffRadius), 1.5,
				color.RGBA{R: 245, G: 158, B: 11, A: auraAlpha}) // amber/gold
		}

		// (旧 healer aura ring 已移到能力 VFX 系统)

		// --- Spawn animation modifiers ---
		var spawnScale float64 = 1.0
		var spawnAlpha float64 = 1.0
		if e.IsSpawning() && e.SpawnDuration > 0 {
			progress := 1.0 - e.SpawnTimer/e.SpawnDuration // 0 at start -> 1 at end
			// Scale: overshoot from 0 to 1.15 then settle to 1.0
			if progress < 0.7 {
				spawnScale = progress / 0.7 * 1.15
			} else {
				t := (progress - 0.7) / 0.3
				spawnScale = 1.15 - 0.15*t
			}
			// Alpha: ease in (easeInQuad)
			spawnAlpha = progress * progress
		}

		// --- Enemy body (animated or static) ---
		img := er.getEnemyFrame(e, 1.0/60.0)
		if img != nil {
			// 行走摆动：用 X 位置作为相位，产生微小的上下浮动和旋转
			wobblePhase := e.X*0.05 + animTime*4
			wobbleY := math.Sin(wobblePhase) * 1.5
			wobbleRot := math.Sin(wobblePhase) * 0.05 // ~3 degrees
			if e.StunTimer > 0 || e.RootTimer > 0 {
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
			if e.Stealthed {
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
			bodyColor := color.RGBA{R: 200, G: 60, B: 60, A: 255}
			if e.Boss {
				bodyColor = color.RGBA{R: 220, G: 160, B: 40, A: 255}
			}
			if e.Stealthed {
				bodyColor.A = 38
			} else if e.PhaseActive {
				bodyColor.A = 90
			}
			// Apply spawn alpha to fallback circle
			bodyColor.A = uint8(float64(bodyColor.A) * spawnAlpha)
			draw.FilledCircle(screen, cx, cy, r*float32(spawnScale), bodyColor)
		}

		// --- Status effect body overlays (subtle, sprite-sized) ---
		spriteR := float32(enemySpriteSize) / 2
		if e.SlowTimer > 0 {
			draw.CircleOutline(screen, cx, cy, spriteR+1, 1.5, color.RGBA{80, 160, 255, 80})
		}
		if e.BurnTimer > 0 {
			draw.FilledCircle(screen, cx, cy, spriteR*0.5, color.RGBA{255, 120, 30, 35})
		}

		// --- Stun rotating stars (3 yellow circles orbiting above head) ---
		if e.StunTimer > 0 {
			starR := float32(2)
			orbitR := r + 4
			for i := 0; i < 3; i++ {
				angle := animTime*5 + float64(i)*2.094 // 120° apart, rotating
				sx := cx + orbitR*float32(math.Cos(angle))
				sy := cy - r - 4 + orbitR*0.4*float32(math.Sin(angle)) // above head, elliptical
				draw.FilledCircle(screen, sx, sy, starR, color.RGBA{255, 255, 100, 200})
			}
		}

		// --- Hit flash overlay (red tint, sprite-sized, NOT collision-radius) ---
		if e.HitFlash > 0 && !e.IsDying() {
			flashAlpha := uint8(clampF(float64(e.HitFlash)*300, 0, 90))
			draw.FilledCircle(screen, cx, cy, spriteR, color.RGBA{R: 255, G: 80, B: 60, A: flashAlpha})
		}

		// (tank overlay removed — was debug placeholder)

		// Stealthed enemies: skip HP bar and status dots (nearly invisible)
		if e.Stealthed {
			return
		}

		// --- HP bar dimensions ---
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

			// --- HP bar (always visible) ---
		{
			barX := cx - barW/2
			barY := cy - barOffY

			// Background (includes 1px border via darker color, saves 1 draw call)
			draw.FilledRect(screen, barX-1, barY-1, barW+2, barH+2,
				theme.EnemyHPBarBg, true)

			// Damage trail (orange, behind HP fill)
			if e.DisplayHP > e.HP && e.DisplayHP > 0 {
				trailRatio := float32(e.DisplayHP / e.MaxHP)
				if trailRatio > 1 {
					trailRatio = 1
				}
				draw.FilledRect(screen, barX, barY, barW*trailRatio, barH,
					color.RGBA{R: 251, G: 146, B: 60, A: 255}, true) // #fb923c
			}

			// HP fill (red-only color scheme)
			ratio := float32(e.HP / e.MaxHP)
			if ratio < 0 {
				ratio = 0
			}
			fillW := barW * ratio

			var fillClr color.RGBA
			switch {
			case ratio > 0.6:
				fillClr = color.RGBA{R: 239, G: 68, B: 68, A: 255}  // #ef4444 bright red
			case ratio > 0.3:
				fillClr = color.RGBA{R: 220, G: 38, B: 38, A: 255}  // #dc2626 darker red
			default:
				fillClr = color.RGBA{R: 153, G: 27, B: 27, A: 255}  // #991b1b deep red
			}
			draw.FilledRect(screen, barX, barY, fillW, barH, fillClr, true)

			// Boss HP segment dividers (5 segments, 20% each)
			if e.Boss {
				for i := 1; i < 5; i++ {
					divX := barX + barW*float32(i)*0.2
					draw.FilledRect(screen, divX, barY, 1, barH, theme.EnemyHPSegDiv, true)
				}
			}

			// (Elite center tick removed)
		}

		// --- Status effect dots ---
		dotY := cy - barOffY - 4
		dotX := cx - 8.0
		dotR := float32(2.5)

		if e.SlowTimer > 0 {
			draw.FilledCircle(screen, dotX, dotY, dotR,
				color.RGBA{R: 125, G: 211, B: 252, A: 235})
			dotX += 6
		}
		if e.StunTimer > 0 {
			draw.FilledCircle(screen, dotX, dotY, dotR,
				color.RGBA{R: 255, G: 255, B: 100, A: 235}) // yellow (matches rotating stars)
			dotX += 6
		}
		if e.RootTimer > 0 {
			draw.FilledCircle(screen, dotX, dotY, dotR,
				color.RGBA{R: 139, G: 90, B: 43, A: 235}) // brown
			dotX += 6
		}
		if e.BleedTimer > 0 {
			draw.FilledCircle(screen, dotX, dotY, dotR,
				color.RGBA{R: 239, G: 68, B: 68, A: 255})
			dotX += 6
		}
		if e.BurnTimer > 0 {
			draw.FilledCircle(screen, dotX, dotY, dotR,
				color.RGBA{R: 255, G: 140, B: 40, A: 255})
			dotX += 6
		}

		// --- 能力常驻视觉（被沉默时全部隐藏）---
		if !e.AbilitySilenced {
			// 免疫脚环（只显示天生能力，净化临时免疫用白色微光）
			footR := float32(e.Radius) + 2
			if hasAbility(e, "ccImmune") {
				draw.CircleOutline(screen, cx, cy+footR*0.3, footR, 1, color.RGBA{R: 220, G: 60, B: 60, A: 80})
			} else if hasAbility(e, "slowImmune") {
				draw.CircleOutline(screen, cx, cy+footR*0.3, footR, 1, color.RGBA{R: 60, G: 180, B: 200, A: 80})
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
			if e.PurgeInterval > 0 && e.ControlImmuneTimer > 0 {
				glowAlpha := uint8(60 + 30*math.Sin(animTime*6))
				draw.CircleOutline(screen, cx, cy, float32(e.Radius)+3, 1.5, color.RGBA{R: 255, G: 255, B: 255, A: glowAlpha})
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
	path := fmt.Sprintf("assets/enemies/shields/%s.png", name)
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
	path := fmt.Sprintf("assets/enemies/sprites/%s/%s.png", spriteDir, spriteDir)
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
	path := fmt.Sprintf("assets/enemies/sprites/%s/%s.png", archetype, archetype)
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

// clampF clamps a float64 value between min and max.
func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
