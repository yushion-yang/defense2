// draw_tower.go — tower rendering.
// Uses PNG sprites (64px), fallback to geometric shapes. Themed selection ring,
// range indicator, name label, and buff dots for the selected tower.
package render

import (
	"fmt"
	"image/color"
	"math"

	"defense2/internal/config"
	"defense2/internal/core/tower"
	"defense2/internal/render/anim"
	"defense2/internal/render/draw"
	"defense2/internal/render/sprite"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// TowerRenderer manages tower PNG sprite rendering with optional frame animation.
type TowerRenderer struct {
	cache     *sprite.Cache
	assetFS   AssetReader
	animators map[string]*anim.Animator // per tower key, lazy initialized
}

// AssetReader reads embedded asset files.
type AssetReader interface {
	ReadFile(name string) ([]byte, error)
}

// NewTowerRenderer creates a tower renderer.
func NewTowerRenderer(assetFS AssetReader) *TowerRenderer {
	return &TowerRenderer{
		cache:     sprite.NewCache(),
		assetFS:   assetFS,
		animators: make(map[string]*anim.Animator),
	}
}

const towerSpriteSize = 64 // display size in pixels, matching theme.TowerBaseSize

// overshootEase returns an easing that overshoots to ~1.1 then settles at 1.0.
func overshootEase(t float64) float64 {
	if t < 0.7 {
		return (t / 0.7) * 1.1
	}
	return 1.1 - 0.1*((t-0.7)/0.3)
}

// towerAnimScaleAlpha computes the scale multiplier and alpha for build/sell animations.
// Returns (scaleMul, alpha) where scaleMul=1 and alpha=1 mean no animation active.
func towerAnimScaleAlpha(t *tower.Tower) (float64, float64) {
	// Sell animation: expand slightly then shrink to 0
	if t.Selling && t.SellAnim > 0 {
		progress := 1.0 - t.SellAnim/0.25 // 0→1
		if progress > 1 {
			progress = 1
		}
		var scale float64
		if progress < 0.2 {
			scale = 1.0 + progress*1.0 // 1.0 → 1.2
		} else {
			scale = 1.2 * (1.0 - (progress-0.2)/0.8) // 1.2 → 0
		}
		alpha := 1.0 - progress
		return scale, alpha
	}
	// Build animation: overshoot bounce in
	if t.BuildAnim > 0 {
		progress := 1.0 - t.BuildAnim/0.3 // 0→1
		scale := overshootEase(progress)
		alpha := progress // fade in
		return scale, alpha
	}
	return 1.0, 1.0
}

// DrawTowers renders all placed towers.
func (tr *TowerRenderer) DrawTowers(screen *ebiten.Image, pool *tower.Pool, selectedTower *tower.Tower, animTime float64) {
	pool.Each(func(t *tower.Tower) {
		cx := float32(t.X)
		cy := float32(t.Y)
		selected := selectedTower != nil && t == selectedTower

		// Compute animation scale and alpha
		animScale, animAlpha := towerAnimScaleAlpha(t)

		// --- Build ripple effect (expanding white ring) ---
		if t.BuildAnim > 0 {
			progress := 1.0 - t.BuildAnim/0.3
			ringR := float32(10 + 20*progress) // 10 → 30
			ringA := uint8(float64(150) * (1 - progress))
			draw.CircleOutline(screen, cx, cy, ringR, 2,
				color.RGBA{255, 255, 255, ringA})
		}

		// --- Selection ring & range indicator (selected tower only, skip during sell) ---
		if selected && !t.Selling {
			draw.CircleOutline(screen, cx, cy,
				theme.TowerSelectionRingR, theme.TowerSelectionWidth, theme.TowerSelectionRing)
			draw.CircleOutline(screen, cx, cy,
				float32(t.Range), theme.TowerRangeStrokeWidth, theme.TowerRangeStroke)
		}

		// --- Ground shadow (dark ellipse below tower) ---
		shadowAlpha := uint8(float64(30) * animAlpha)
		draw.FilledCircle(screen, cx, cy+float32(towerSpriteSize*0.35),
			float32(float64(towerSpriteSize*0.35)*animScale), color.RGBA{0, 0, 0, shadowAlpha})

		// --- Tower body (animated or static, rotated toward target) ---
		// spin_aoe 不旋转朝向目标
		rotation := t.Angle + math.Pi/2
		if t.AttackStyleID == tower.StyleSpinAoE {
			rotation = 0
		}

		img := tr.getTowerFrame(t, 1.0/60.0)
		if img != nil {
			logicalScale := float64(towerSpriteSize) / float64(img.Bounds().Dx())
			// 射击缩放脉冲：射击瞬间放大 8%，快速恢复（skip during build/sell anim）
			if t.FireAnim > 0 && t.BuildAnim <= 0 && !t.Selling {
				pulse := 1.0 + 0.08*(t.FireAnim/0.15)
				logicalScale *= pulse
			}
			// Apply build/sell animation scale
			logicalScale *= animScale
			if animAlpha < 1.0 {
				draw.SpriteScaledRotatedAlpha(screen, img, float64(cx), float64(cy), logicalScale, rotation, animAlpha)
			} else {
				draw.SpriteScaledRotated(screen, img, float64(cx), float64(cy), logicalScale, rotation)
			}
		} else {
			// Fallback: circle body + barrel rectangle
			bodyClr := theme.TowerFallbackDef
			if selected {
				bodyClr = theme.TowerFallbackSel
			}
			// Apply alpha to fallback colors
			if animAlpha < 1.0 {
				bodyClr.A = uint8(float64(bodyClr.A) * animAlpha)
			}
			draw.FilledCircle(screen, cx, cy, float32(float64(theme.TowerFallbackRadius)*animScale), bodyClr)
			barrelClr := theme.TowerBarrel
			if animAlpha < 1.0 {
				barrelClr.A = uint8(float64(barrelClr.A) * animAlpha)
			}
			barrelLen := 14.0 * animScale
			bx2 := float64(cx) + math.Cos(t.Angle)*barrelLen
			by2 := float64(cy) + math.Sin(t.Angle)*barrelLen
			draw.ThickLine(screen, float32(cx), float32(cy), float32(bx2), float32(by2), float32(8*animScale), barrelClr)
		}

		// 射击反馈已由 shoot pulse（精灵放大 15%）+ muzzle flash 粒子提供，
		// 不再叠加白色圆——高攻速塔会导致持续白圈。

		// --- Charge visual: red glow at muzzle position (follows aim angle) ---
		if !t.Selling && t.BuildAnim <= 0 && t.AttackStyleID == tower.StyleCharge && (t.ChargeProgress > 0 || t.ChargeReady) {
			progress := t.ChargeProgress
			if progress > 1 {
				progress = 1
			}
			// 炮口位置（沿瞄准方向偏移 24px）
			const muzzleDist = 24.0
			mx := float64(cx) + math.Cos(t.Angle)*muzzleDist
			my := float64(cy) + math.Sin(t.Angle)*muzzleDist
			fmx, fmy := float32(mx), float32(my)

			// Layer 1: 聚能光晕（红色半透明填充圆，随进度增大）
			glowR := float32(16 * progress)
			glowA := uint8(38 + 64*progress) // alpha 0.15~0.40
			draw.Glow(screen, fmx, fmy, glowR*0.2, glowR,
				color.RGBA{R: 239, G: 68, B: 68, A: glowA})

			// Layer 2: 脉冲环（浅红色描边圆，快速脉动）
			pulse := 1.0 + math.Sin(animTime*8)*0.2
			ringR := float32(float64(glowR) * pulse)
			ringA := uint8(76 + 128*progress) // alpha 0.3~0.8
			draw.CircleOutline(screen, fmx, fmy, ringR, 1.5,
				color.RGBA{R: 248, G: 113, B: 113, A: ringA})

			// Layer 3: 中心亮点（>50% 进度时出现，近白色）
			if progress > 0.5 {
				dotR := float32(2 + (progress-0.5)*6)
				dotA := uint8((progress - 0.5) * 1.5 * 255)
				draw.FilledCircle(screen, fmx, fmy, dotR,
					color.RGBA{R: 254, G: 242, B: 242, A: dotA})
			}
		}

		// --- Spin AoE visual: rotating blade arcs (no fill) ---
		if !t.Selling && t.BuildAnim <= 0 && t.AttackStyleID == tower.StyleSpinAoE && t.SpinActive > 0 {
			a := t.SpinActive / 0.3
			if a > 1 {
				a = 1
			}
			outerR := float32(t.Range)

			// 4 条旋转弧线
			for i := 0; i < 4; i++ {
				ang := t.SpinAngle + float64(i)*math.Pi/2
				clr := color.RGBA{R: 163, G: 230, B: 53, A: uint8(80 * a)}
				draw.Arc(screen, cx, cy, outerR, float32(ang-0.3), float32(ang+0.3), 2, clr)
			}
		}

		// --- Strength-based visual tiers (glow / rings) ---
		if !t.Selling && t.BuildAnim <= 0 && t.Strength != nil {
			str := t.Strength.Overflow() // strength above baseline (100)

			if str >= 50 {
				// Tier 2 (50+): warm ring outline
				ringAlpha := uint8(30 + min(30, int((str-50)*0.6)))
				draw.CircleOutline(screen, cx, cy, float32(towerSpriteSize*0.4), 1,
					color.RGBA{255, 230, 150, ringAlpha})
			}
			if str >= 100 {
				// Tier 3 (100+): brighter outer ring
				draw.CircleOutline(screen, cx, cy, float32(towerSpriteSize*0.48), 1,
					color.RGBA{255, 220, 100, 50})
			}
			if str >= 150 {
				// Tier 4 (150+): pulsing outer ring
				pulse := 0.5 + 0.5*math.Sin(animTime*3)
				ringAlpha := uint8(30 + 25*pulse)
				draw.CircleOutline(screen, cx, cy, float32(towerSpriteSize*0.55), 1,
					color.RGBA{255, 200, 50, ringAlpha})
			}
		}

		// --- Aura radius circle (for towers with aura abilities) ---
		if !t.Selling && t.BuildAnim <= 0 {
			if auraR, auraClr := towerAuraVisual(t, animTime); auraR > 0 {
				pulse := float32(0.7 + 0.3*math.Sin(animTime*2))
				a := uint8(float64(25) * float64(pulse))
				clr := color.RGBA{auraClr.R, auraClr.G, auraClr.B, a}
				draw.DashedCircle(screen, cx, cy, float32(auraR), 1, 6, 4, clr)
			}
		}

		// --- Buff indicator dots (above tower name) ---
		if !t.Selling && len(t.Buffs) > 0 {
			dotY := cy - float32(towerSpriteSize*0.5) - 6
			dotSpacing := float32(6)
			dotR := float32(2.5)
			n := len(t.Buffs)
			if n > 5 {
				n = 5
			}
			startX := cx - float32(n-1)*dotSpacing/2
			for i := 0; i < n; i++ {
				dx := startX + float32(i)*dotSpacing
				draw.FilledCircle(screen, dx, dotY, dotR,
					color.RGBA{R: 180, G: 140, B: 255, A: 180})
			}
		}

		// --- Name label (skip during sell animation) ---
		if !t.Selling {
			if fm := GlobalFont(); fm != nil {
				fm.DrawCenteredText(screen, t.Label,
					float64(cx), float64(cy)+theme.TowerNameLabelY,
					theme.FontTowerName, theme.TowerNameLabel)
			}
		}
	})
}

// loadTowerImage loads a tower's PNG sprite.
// Convention: assets/towers/{key}/tower-{key}.png
func (tr *TowerRenderer) loadTowerImage(t *tower.Tower) *ebiten.Image {
	if tr.assetFS == nil {
		return nil
	}
	key := t.SpriteKey
	if key == "" {
		key = t.Key
	}
	path := fmt.Sprintf("assets/towers/%s/tower-%s.png", key, key)
	cached := tr.cache.Get(path, towerSpriteSize, towerSpriteSize)
	if cached != nil {
		return cached
	}
	data, err := tr.assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, err := tr.cache.GetOrParse(path, data, towerSpriteSize, towerSpriteSize)
	if err != nil {
		return nil
	}
	return img
}

// GetSprite returns the tower sprite by key for icon/thumbnail use.
// Prefers idle-0 animation frame (matches in-game rendering), falls back to static PNG.
func (tr *TowerRenderer) GetSprite(key string) *ebiten.Image {
	if tr.assetFS == nil {
		return nil
	}
	// Try idle-0 first (matches map rendering)
	for _, path := range []string{
		fmt.Sprintf("assets/towers/%s/tower-%s-idle-0.png", key, key),
		fmt.Sprintf("assets/towers/%s/tower-%s.png", key, key),
	} {
		if img := tr.loadPNG(path); img != nil {
			return img
		}
	}
	return nil
}

func (tr *TowerRenderer) loadPNG(path string) *ebiten.Image {
	if cached := tr.cache.Get(path, towerSpriteSize, towerSpriteSize); cached != nil {
		return cached
	}
	data, err := tr.assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, _ := tr.cache.GetOrParse(path, data, towerSpriteSize, towerSpriteSize)
	return img
}

// getTowerFrame returns the current animation frame for a tower, falling back to static sprite.
func (tr *TowerRenderer) getTowerFrame(t *tower.Tower, dt float64) *ebiten.Image {
	// Lazy-load animator (keyed by SpriteKey for visual swap)
	sprKey := t.SpriteKey
	if sprKey == "" {
		sprKey = t.Key
	}
	animKey := t.InstanceKey
	if animKey == "" {
		animKey = sprKey
	}
	a, ok := tr.animators[animKey]
	if !ok {
		a = anim.LoadTowerAnimator(tr.assetFS, sprKey)
		tr.animators[animKey] = a
	}

	// Choose animation state
	if t.FireAnim > 0 && a.HasAnim("attack") {
		a.Play("attack")
	} else if a.HasAnim("idle") {
		a.Play("idle")
	}
	a.Update(dt)

	img := a.CurrentImage()
	if img != nil {
		return img
	}
	// Fallback to static sprite cache
	return tr.loadTowerImage(t)
}

// DrawTowerRangePreview draws a placement preview (range circle + tower shadow).
func DrawTowerRangePreview(screen *ebiten.Image, cx, cy float32, r float64, valid bool) {
	fr := float32(r)

	var strokeClr color.RGBA
	if valid {
		strokeClr = color.RGBA{R: 34, G: 197, B: 94, A: 153} // green 0.6 alpha
	} else {
		strokeClr = color.RGBA{R: 239, G: 68, B: 68, A: 153} // red 0.6 alpha
	}

	// Range circle (outline only)
	draw.CircleOutline(screen, cx, cy, fr, 1.5, strokeClr)
}

// aura 类型 → 视觉颜色
var auraColors = map[string]color.RGBA{
	"damageUpAura":   {R: 255, G: 160, B: 60, A: 255},  // 橙
	"attackSpeedAura": {R: 100, G: 220, B: 100, A: 255}, // 绿
	"rangeAura":      {R: 100, G: 160, B: 255, A: 255},  // 蓝
	"critAura":       {R: 255, G: 220, B: 60, A: 255},   // 黄
}

// towerAuraVisual 检查塔是否有光环能力，返回半径和颜色。
func towerAuraVisual(t *tower.Tower, _ float64) (radius float64, clr color.RGBA) {
	table := config.GlobalAbilityTable()
	if table == nil {
		return 0, color.RGBA{}
	}
	for _, abName := range t.Abilities {
		if c, ok := auraColors[abName]; ok {
			if def, exists := table[abName]; exists && def.Param > 0 {
				return def.Param, c
			}
		}
	}
	return 0, color.RGBA{}
}
