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
	"defense2/internal/render/vfx"

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

// towerAnimScaleAlpha computes the scale multiplier and alpha for build/sell animations.
// Returns (scaleMul, alpha) where scaleMul=1 and alpha=1 mean no animation active.
func towerAnimScaleAlpha(t *tower.Tower) (float64, float64) {
	if t.Selling && t.SellAnim > 0 {
		return vfx.SellAnimParams(t.SellAnim)
	}
	return vfx.BuildAnimParams(t.BuildAnim)
}

// DrawTowers renders all placed towers.
func (tr *TowerRenderer) DrawTowers(screen *ebiten.Image, pool *tower.Pool, selectedTower *tower.Tower, animTime float64) {
	pool.Each(func(t *tower.Tower) {
		cx := float32(t.X)
		cy := float32(t.Y)
		selected := selectedTower != nil && t == selectedTower

		// Compute animation scale and alpha
		animScale, animAlpha := towerAnimScaleAlpha(t)

		// --- Build ripple effect ---
		if t.BuildAnim > 0 {
			vfx.DrawBuildRipple(screen, cx, cy, 1.0-t.BuildAnim/0.3)
		}

		// --- Selection ring & range indicator (selected tower only, skip during sell) ---
		if selected && !t.Selling {
			vfx.DrawSelectionRing(screen, cx, cy,
				theme.TowerSelectionRingR, theme.TowerSelectionWidth, theme.TowerSelectionRing,
				float32(t.Range), theme.TowerRangeStrokeWidth, theme.TowerRangeStroke)
		}

		// --- Ground shadow (dark ellipse below tower) ---
		shadowAlpha := uint8(float64(30) * animAlpha)
		draw.FilledCircle(screen, cx, cy+float32(towerSpriteSize*0.35),
			float32(float64(towerSpriteSize*0.35)*animScale), color.RGBA{0, 0, 0, shadowAlpha})

		// --- Under-body VFX (drawn BEFORE sprite so they don't obscure it) ---
		if !t.Selling && t.BuildAnim <= 0 && t.Strength != nil {
			vfx.DrawStrengthGlow(screen, cx, cy, t.Strength.Overflow(), animTime)
		}
		if !t.Selling && t.BuildAnim <= 0 {
			for _, av := range towerAuraVisuals(t) {
				vfx.DrawAuraPulse(screen, cx, cy, av.Radius, av.Color, animTime)
			}
		}

		// --- Tower body (animated or static, rotated toward target) ---
		// spin_aoe 不旋转朝向目标
		rotation := t.Angle + math.Pi/2
		if t.AttackStyleID == tower.StyleSpinAoE {
			rotation = 0
		}

		img := tr.getTowerFrame(t, 1.0/60.0)
		if img != nil {
			logicalScale := float64(towerSpriteSize) / float64(img.Bounds().Dx())
			// 射击缩放脉冲（skip during build/sell anim）
			if t.FireAnim > 0 && t.BuildAnim <= 0 && !t.Selling {
				logicalScale *= vfx.FirePulseScale(t.FireAnim)
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

		// --- Spin AoE visual: rotating blade arcs ---
		if !t.Selling && t.BuildAnim <= 0 && t.AttackStyleID == tower.StyleSpinAoE && t.SpinActive > 0 {
			vfx.DrawSpinBlades(screen, cx, cy, float32(t.Range), t.SpinAngle, t.SpinActive/0.3)
		}

		// --- Buff indicator dots ---
		if !t.Selling && t.Buffs.Count() > 0 {
			vfx.DrawBuffDots(screen, cx, cy, t.Buffs.Count())
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
	// animKey 包含 sprKey 以确保 SpriteKey 变更（如选择能力后）重新加载动画
	animKey := t.InstanceKey + ":" + sprKey
	if t.InstanceKey == "" {
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

// auraVisualDef 定义一个有半径的 buff/zone 能力的视觉效果。
// 新增 buff 类型只需在此注册表添加一条即可自动显示光圈。
type auraVisualDef struct {
	Color       color.RGBA
	UseTowerRange bool // true: 用塔射程作为半径（zone 类），false: 用 ability param
}

// auraRegistry buff/zone 能力 → 视觉定义。
var auraRegistry = map[string]auraVisualDef{
	// 增益光环（buff 类，param=半径）
	"damageUpAura":    {Color: color.RGBA{R: 255, G: 160, B: 60, A: 255}},  // 橙 — 增伤
	"attackSpeedAura": {Color: color.RGBA{R: 100, G: 220, B: 100, A: 255}}, // 绿 — 攻速
	"rangeAura":       {Color: color.RGBA{R: 100, G: 160, B: 255, A: 255}}, // 蓝 — 射程
	"critAura":        {Color: color.RGBA{R: 255, G: 220, B: 60, A: 255}},  // 金 — 暴击
	"soloBoost":       {Color: color.RGBA{R: 200, G: 120, B: 255, A: 255}}, // 紫 — 独行
	// 区域效果（zone 类，用塔射程）
	"poisonZone":  {Color: color.RGBA{R: 120, G: 200, B: 60, A: 255}, UseTowerRange: true},  // 毒绿
	"silenceZone": {Color: color.RGBA{R: 180, G: 80, B: 220, A: 255}, UseTowerRange: true},  // 沉默紫
	"curseZone":   {Color: color.RGBA{R: 160, G: 50, B: 50, A: 255}, UseTowerRange: true},   // 诅咒红
	"weakenZone":  {Color: color.RGBA{R: 220, G: 140, B: 60, A: 255}, UseTowerRange: true},  // 脆弱橙
}

// auraVisual 一个光环的渲染参数。
type auraVisual struct {
	Radius float64
	Color  color.RGBA
}

// towerAuraVisuals 返回塔上所有 buff/zone 能力的视觉参数列表。
// 支持多光环叠加渲染。新增能力只需在 auraRegistry 中注册。
func towerAuraVisuals(t *tower.Tower) []auraVisual {
	table := config.GlobalAbilityTable()
	if table == nil {
		return nil
	}
	var result []auraVisual
	for _, abName := range t.Abilities {
		reg, ok := auraRegistry[abName]
		if !ok {
			continue
		}
		var radius float64
		if reg.UseTowerRange {
			radius = t.Range // zone 类用塔射程
		} else if def, exists := table[abName]; exists && def.Param > 0 {
			radius = def.Param // buff 类用 ability param
		}
		if radius <= 0 {
			continue
		}
		result = append(result, auraVisual{Radius: radius, Color: reg.Color})
	}
	return result
}
