// draw_warden.go — warden rendering.
// Uses PNG sprites for warden bodies, keeps dynamic effects (shoot lines,
// fireballs, flame trails, AoE strikes, buff connections) as code-rendered.
package render

import (
	"fmt"
	"image/color"
	"math"

	"defense2/internal/core/warden"
	wardenTypes "defense2/internal/core/warden/types"
	"defense2/internal/render/draw"
	"defense2/internal/render/sprite"

	"github.com/hajimehoshi/ebiten/v2"
)

// WardenRenderer manages warden PNG sprite rendering.
type WardenRenderer struct {
	cache   *sprite.Cache
	assetFS AssetReader
}

// NewWardenRenderer creates a warden renderer.
func NewWardenRenderer(assetFS AssetReader) *WardenRenderer {
	return &WardenRenderer{
		cache:   sprite.NewCache(),
		assetFS: assetFS,
	}
}

const wardenSpriteSize = 24 // display size in logical pixels

// loadSprite loads and caches a warden's PNG sprite.
// Convention: assets/wardens/warden-{type}.png
func (wr *WardenRenderer) loadSprite(typ string) *ebiten.Image {
	if wr.assetFS == nil || typ == "" {
		return nil
	}
	path := fmt.Sprintf("assets/wardens/warden-%s.png", typ)
	if cached := wr.cache.Get(path, wardenSpriteSize, wardenSpriteSize); cached != nil {
		return cached
	}
	data, err := wr.assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, _ := wr.cache.GetOrParse(path, data, wardenSpriteSize, wardenSpriteSize)
	return img
}

// GetSprite returns the cached sprite image for a warden type (for UI preview).
func (wr *WardenRenderer) GetSprite(typ string) *ebiten.Image {
	return wr.loadSprite(typ)
}

// DrawWarden renders a single warden entity.
func (wr *WardenRenderer) DrawWarden(screen *ebiten.Image, w *warden.Warden) {
	if w == nil || !w.Active {
		return
	}

	base := w.BaseState()
	if base == nil {
		return
	}
	if base.X == 0 && base.Y == 0 {
		return
	}

	// Type-specific effects (drawn BEFORE body so body renders on top)
	switch s := w.State.(type) {
	case *wardenTypes.PrinceState:
		drawPrinceEffects(screen, s)
	case *wardenTypes.CoreState:
		drawCoreEffects(screen, &s.WardenState)
	case *wardenTypes.ChainState:
		drawChainEffects(screen, s)
	case *wardenTypes.SkystrikeState:
		drawSkystrikeEffects(screen, s)
	case *wardenTypes.EnvoyState:
		drawEnvoyEffects(screen, s)
	}

	// Body sprite（根据精灵原始朝向校正旋转角度）
	img := wr.loadSprite(w.Type)
	if img != nil {
		rotation := wardenSpriteRotation(w.Type, base.FacingAngle)
		draw.SpriteRotated(screen, img, float64(base.X), float64(base.Y),
			wardenSpriteSize, rotation, 0)
	} else {
		// Fallback: simple colored circle
		draw.FilledCircle(screen, float32(base.X), float32(base.Y), 8,
			wardenShootColor(w.Type))
	}
}

// wardenSpriteRotation 根据精灵原始朝向计算实际旋转角度。
// rotation = facingAngle - spriteNativeAngle
// 对称精灵（chain, envoy）不旋转。
func wardenSpriteRotation(typ string, facingAngle float64) float64 {
	switch typ {
	case "prince", "core":
		// 精灵朝上（-π/2），移动方向 facingAngle=0 时需旋转 +π/2
		return facingAngle + math.Pi/2
	case "skystrike":
		// 精灵朝下（+π/2），移动方向 facingAngle=0 时需旋转 -π/2
		return facingAngle - math.Pi/2
	default:
		// chain, envoy: 对称精灵不旋转
		return 0
	}
}

// wardenShootColor returns the shoot-line color for each warden type.
func wardenShootColor(typ string) color.RGBA {
	switch typ {
	case "prince":
		return color.RGBA{R: 255, G: 140, B: 30, A: 200}
	case "core":
		return color.RGBA{R: 100, G: 180, B: 255, A: 200}
	case "chain":
		return color.RGBA{R: 160, G: 80, B: 255, A: 200}
	case "skystrike":
		return color.RGBA{R: 80, G: 200, B: 255, A: 200}
	case "envoy":
		return color.RGBA{R: 255, G: 200, B: 100, A: 200}
	default:
		return color.RGBA{R: 200, G: 200, B: 200, A: 200}
	}
}

// ── 火灵特效：火球飞行 + 地面燃烧区 ──

func drawPrinceEffects(screen *ebiten.Image, s *wardenTypes.PrinceState) {
	// 地面燃烧区（多层渐变）
	for _, t := range s.Trails {
		p := t.Life / t.MaxLife
		r := float32(t.Radius)
		// 外圈暗红光晕
		draw.FilledCircle(screen, float32(t.X), float32(t.Y), r*1.3,
			color.RGBA{R: 180, G: 40, B: 0, A: uint8(40 * p)})
		// 中圈橙色火焰
		draw.FilledCircle(screen, float32(t.X), float32(t.Y), r,
			color.RGBA{R: 255, G: 100, B: 30, A: uint8(100 * p)})
		// 内圈亮黄
		draw.FilledCircle(screen, float32(t.X), float32(t.Y), r*0.5,
			color.RGBA{R: 255, G: 200, B: 60, A: uint8(80 * p)})
	}

	// 飞行火球（拖尾 + 核心）
	for _, fb := range s.Fireballs {
		fx, fy := float32(fb.X), float32(fb.Y)
		r := float32(fb.Radius)
		// 拖尾：沿飞行方向画 3 个渐隐圆
		if fb.Progress > 0.05 {
			dx := fb.EndX - fb.StartX
			dy := fb.EndY - fb.StartY
			dist := math.Hypot(dx, dy)
			if dist > 1 {
				nx, ny := float32(dx/dist), float32(dy/dist)
				for i := 1; i <= 3; i++ {
					off := float32(i) * r * 0.6
					ta := uint8(60 - i*15)
					draw.FilledCircle(screen, fx-nx*off, fy-ny*off, r*0.4,
						color.RGBA{R: 255, G: 120, B: 20, A: ta})
				}
			}
		}
		// 外层光晕
		draw.FilledCircle(screen, fx, fy, r*0.9,
			color.RGBA{R: 255, G: 100, B: 0, A: 50})
		// 核心
		draw.FilledCircle(screen, fx, fy, r*0.45,
			color.RGBA{R: 255, G: 200, B: 60, A: 240})
	}
}

// ── 机甲特效：射击闪光 ──

func drawCoreEffects(screen *ebiten.Image, s *warden.WardenState) {
	if s.ShootTimer <= 0 {
		return
	}
	// 枪口闪光
	p := s.ShootTimer / 0.15
	alpha := uint8(200 * p)
	r := float32(6 + 4*p)
	draw.FilledCircle(screen, float32(s.X), float32(s.Y), r,
		color.RGBA{R: 100, G: 200, B: 255, A: alpha / 3})
	draw.FilledCircle(screen, float32(s.X), float32(s.Y), r*0.4,
		color.RGBA{R: 200, G: 230, B: 255, A: alpha})
}

// ── 聚能特效：串联电弧 ──

func drawChainEffects(screen *ebiten.Image, s *wardenTypes.ChainState) {
	// 射击闪光
	if s.ShootTimer > 0 {
		p := s.ShootTimer / 0.15
		alpha := uint8(180 * p)
		draw.FilledCircle(screen, float32(s.X), float32(s.Y), float32(5+3*p),
			color.RGBA{R: 160, G: 80, B: 255, A: alpha / 2})
	}
}

// ── 水灵特效：天降冰柱/水花 ──

func drawSkystrikeEffects(screen *ebiten.Image, s *wardenTypes.SkystrikeState) {
	if s.StrikeTimer <= 0 {
		return
	}
	p := s.StrikeTimer / 0.5 // 1→0 衰减
	sx, sy := float32(s.StrikeX), float32(s.StrikeY)

	// 根据模式选择颜色
	var baseClr, coreClr color.RGBA
	switch s.LastMode {
	case 1: // 散射 — 天蓝色
		baseClr = color.RGBA{R: 80, G: 200, B: 255, A: uint8(120 * p)}
		coreClr = color.RGBA{R: 160, G: 230, B: 255, A: uint8(200 * p)}
	case 2: // 连击 — 深蓝色
		baseClr = color.RGBA{R: 40, G: 120, B: 255, A: uint8(130 * p)}
		coreClr = color.RGBA{R: 120, G: 180, B: 255, A: uint8(220 * p)}
	case 3: // 收割 — 冰白色
		baseClr = color.RGBA{R: 180, G: 220, B: 255, A: uint8(100 * p)}
		coreClr = color.RGBA{R: 220, G: 240, B: 255, A: uint8(200 * p)}
	default:
		baseClr = color.RGBA{R: 80, G: 200, B: 255, A: uint8(120 * p)}
		coreClr = color.RGBA{R: 160, G: 230, B: 255, A: uint8(200 * p)}
	}

	// 天降冰柱：从上方到打击点的垂直光柱
	beamTop := sy - 80*float32(p) // 光柱从上方降下
	draw.Line(screen, sx-3, beamTop, sx-1, sy, 3, baseClr, true)
	draw.Line(screen, sx+3, beamTop, sx+1, sy, 3, baseClr, true)
	draw.Line(screen, sx, beamTop-10, sx, sy, 2, coreClr, true)

	// 落地冲击波纹（向外扩散）
	waveR := float32(s.AoERadius) * float32(1.5-0.5*p)
	draw.CircleOutline(screen, sx, sy, waveR, 1.5,
		color.RGBA{R: baseClr.R, G: baseClr.G, B: baseClr.B, A: uint8(80 * p)})
	draw.CircleOutline(screen, sx, sy, waveR*0.6, 1,
		color.RGBA{R: coreClr.R, G: coreClr.G, B: coreClr.B, A: uint8(60 * p)})

	// 落点冰花
	draw.FilledCircle(screen, sx, sy, float32(8*p),
		color.RGBA{R: coreClr.R, G: coreClr.G, B: coreClr.B, A: uint8(150 * p)})
	// 碎冰粒子（4个方向）
	spread := float32(20 * (1 - p))
	pAlpha := uint8(100 * p)
	draw.FilledCircle(screen, sx-spread, sy-spread*0.5, 2, color.RGBA{R: 200, G: 230, B: 255, A: pAlpha})
	draw.FilledCircle(screen, sx+spread, sy-spread*0.3, 2, color.RGBA{R: 200, G: 230, B: 255, A: pAlpha})
	draw.FilledCircle(screen, sx-spread*0.7, sy+spread*0.4, 2, color.RGBA{R: 200, G: 230, B: 255, A: pAlpha})
	draw.FilledCircle(screen, sx+spread*0.5, sy+spread*0.6, 2, color.RGBA{R: 200, G: 230, B: 255, A: pAlpha})
}

// ── 金灵特效：射击闪光（连线和塔顶特效改为五星芒阵在 draw_tower_buff.go 中处理）──

func drawEnvoyEffects(screen *ebiten.Image, s *wardenTypes.EnvoyState) {
	if s.ShootTimer <= 0 {
		return
	}
	p := s.ShootTimer / 0.15
	alpha := uint8(180 * p)
	draw.FilledCircle(screen, float32(s.X), float32(s.Y), float32(5+3*p),
		color.RGBA{R: 255, G: 210, B: 80, A: alpha / 2})
}
