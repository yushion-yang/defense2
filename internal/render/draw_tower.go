// draw_tower.go — tower rendering.
// Uses PNG sprites (64px), fallback to geometric shapes. Themed selection ring,
// range indicator, name label, and buff dots for the selected tower.
package render

import (
	"fmt"
	"image/color"
	"math"

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

// DrawTowers renders all placed towers.
func (tr *TowerRenderer) DrawTowers(screen *ebiten.Image, pool *tower.Pool, selectedTower *tower.Tower, animTime float64) {
	pool.Each(func(t *tower.Tower) {
		cx := float32(t.X)
		cy := float32(t.Y)
		selected := selectedTower != nil && t == selectedTower

		// --- Selection ring & range indicator (selected tower only) ---
		if selected {
			draw.CircleOutline(screen, cx, cy,
				theme.TowerSelectionRingR, theme.TowerSelectionWidth, theme.TowerSelectionRing)
			draw.CircleOutline(screen, cx, cy,
				float32(t.Range), theme.TowerRangeStrokeWidth, theme.TowerRangeStroke)
		}

		// --- Ground shadow (dark ellipse below tower) ---
		draw.FilledCircle(screen, cx, cy+float32(towerSpriteSize*0.35),
			float32(towerSpriteSize*0.35), color.RGBA{0, 0, 0, 30})

		// --- Tower body (animated or static, rotated toward target) ---
		// spin_aoe / summon 不旋转朝向目标
		rotation := t.Angle + math.Pi/2
		if t.AttackStyleID == tower.StyleSpinAoE {
			rotation = 0
		}

		img := tr.getTowerFrame(t, 1.0/60.0)
		if img != nil {
			logicalScale := float64(towerSpriteSize) / float64(img.Bounds().Dx())
			// 射击缩放脉冲：射击瞬间放大 15%，快速恢复
			if t.FireAnim > 0 {
				pulse := 1.0 + 0.15*(t.FireAnim/0.15)
				logicalScale *= pulse
			}
			draw.SpriteScaledRotated(screen, img, float64(cx), float64(cy), logicalScale, rotation)
		} else {
			// Fallback: circle body + barrel rectangle
			bodyClr := theme.TowerFallbackDef
			if selected {
				bodyClr = theme.TowerFallbackSel
			}
			draw.FilledCircle(screen, cx, cy, theme.TowerFallbackRadius, bodyClr)
			draw.FilledRect(screen, cx-4, cy-14, 8, 14, theme.TowerBarrel, true)
		}

		// --- Charge visual: red glow at muzzle position (follows aim angle) ---
		if t.AttackStyleID == tower.StyleCharge && (t.ChargeProgress > 0 || t.ChargeReady) {
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

		// --- Spin AoE visual: rotating blade arcs + inner zone highlight ---
		if t.AttackStyleID == tower.StyleSpinAoE && t.SpinActive > 0 {
			alpha := t.SpinActive / 0.3
			if alpha > 1 {
				alpha = 1
			}
			outerR := float32(t.Range)
			innerR := float32(t.Range * 0.5) // spin_aoe 内圈半径比例

			// 4 条旋转弧线
			for i := 0; i < 4; i++ {
				a := t.SpinAngle + float64(i)*math.Pi/2
				arcR := outerR
				clr := color.RGBA{R: 163, G: 230, B: 53, A: uint8(100 * alpha)}
				draw.Arc(screen, cx, cy, arcR, float32(a-0.3), float32(a+0.3), 3, clr)
			}

			// 内圈半透明填充
			innerClr := color.RGBA{R: 134, G: 239, B: 172, A: uint8(20 * alpha)}
			draw.FilledCircle(screen, cx, cy, innerR, innerClr)
		}

		// --- Name label ---
		if fm := GlobalFont(); fm != nil {
			fm.DrawCenteredText(screen, t.Label,
				float64(cx), float64(cy)+theme.TowerNameLabelY,
				theme.FontCaption, theme.TowerNameLabel)
		}

		// Tower struct has no Buffs field — buff dots rendering skipped.
		// When the Buffs field is added, draw colored dots above the tower:
		// offset = -theme.TowerBuffDotBaseY, spacing = theme.TowerBuffDotSpacing, r = theme.TowerBuffDotRadius
	})
}

// loadTowerImage loads a tower's PNG sprite.
// Convention: assets/towers/{key}/tower-{key}.png
func (tr *TowerRenderer) loadTowerImage(t *tower.Tower) *ebiten.Image {
	if tr.assetFS == nil {
		return nil
	}
	path := fmt.Sprintf("assets/towers/%s/tower-%s.png", t.Key, t.Key)
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
	// Lazy-load animator
	a, ok := tr.animators[t.Key]
	if !ok {
		a = anim.LoadTowerAnimator(tr.assetFS, t.Key)
		tr.animators[t.Key] = a
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
