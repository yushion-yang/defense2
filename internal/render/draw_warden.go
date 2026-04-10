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
	"defense2/internal/render/anim"
	"defense2/internal/render/draw"
	"defense2/internal/render/sprite"
	"defense2/internal/render/vfx"

	"github.com/hajimehoshi/ebiten/v2"
)

// WardenRenderer manages warden PNG sprite rendering with optional frame animation.
type WardenRenderer struct {
	cache     *sprite.Cache
	assetFS   AssetReader
	animators map[string]*anim.Animator // per warden type, lazy initialized
}

// NewWardenRenderer creates a warden renderer.
func NewWardenRenderer(assetFS AssetReader) *WardenRenderer {
	return &WardenRenderer{
		cache:     sprite.NewCache(),
		assetFS:   assetFS,
		animators: make(map[string]*anim.Animator),
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

// getWardenFrame returns the current animation frame for a warden, falling back to static sprite.
// Mirrors TowerRenderer.getTowerFrame pattern.
func (wr *WardenRenderer) getWardenFrame(w *warden.Warden, dt float64) *ebiten.Image {
	typ := w.Type
	a, ok := wr.animators[typ]
	if !ok {
		a = anim.LoadWardenAnimator(wr.assetFS, typ)
		wr.animators[typ] = a
	}

	// Choose animation state based on warden behavior:
	// ShootTimer > 0 means the warden just fired (attack visual window).
	base := w.BaseState()
	isAttacking := base != nil && base.ShootTimer > 0
	if isAttacking && a.HasAnim("attack") {
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
	return wr.loadSprite(typ)
}

// DrawWarden renders a single warden entity.
func (wr *WardenRenderer) DrawWarden(screen *ebiten.Image, w *warden.Warden, animTime float64) {
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

	// Movement trail (rendered behind everything)
	vfx.DrawMovementTrail(screen, base.TrailHistory[:], base.TrailCursor, wardenShootColor(w.Type))

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
	// 悬浮动画：Y 轴上下浮动 + 微缩放脉冲
	bobY := math.Sin(animTime*2.5) * 3.0
	displaySize := float64(wardenSpriteSize) * (1.0 + 0.015*math.Sin(animTime*2.0))

	const wardenDT = 1.0 / 60.0
	img := wr.getWardenFrame(w, wardenDT)
	if img != nil {
		rotation := wardenSpriteRotation(w.Type, base.FacingAngle)
		draw.SpriteRotated(screen, img, float64(base.X), float64(base.Y),
			displaySize, rotation, bobY)
	} else {
		// Fallback: simple colored circle
		draw.FilledCircle(screen, float32(base.X), float32(base.Y+bobY), 8,
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
	trails := make([]vfx.FireTrailVFX, len(s.Trails))
	for i, t := range s.Trails {
		trails[i] = vfx.FireTrailVFX{X: t.X, Y: t.Y, Life: t.Life, MaxLife: t.MaxLife, Radius: t.Radius}
	}
	vfx.DrawFireTrails(screen, trails)

	fireballs := make([]vfx.FireballVFX, len(s.Fireballs))
	for i, fb := range s.Fireballs {
		fireballs[i] = vfx.FireballVFX{
			X: fb.X, Y: fb.Y, Radius: fb.Radius, Progress: fb.Progress,
			StartX: fb.StartX, StartY: fb.StartY, EndX: fb.EndX, EndY: fb.EndY,
		}
	}
	vfx.DrawFireballs(screen, fireballs)
}

// ── 机甲特效：射击闪光 ──

func drawCoreEffects(screen *ebiten.Image, s *warden.WardenState) {
	vfx.DrawShootFlash(screen, float32(s.X), float32(s.Y), s.ShootTimer,
		color.RGBA{R: 100, G: 200, B: 255, A: 200})
}

// ── 聚能特效：串联电弧 ──

func drawChainEffects(screen *ebiten.Image, s *wardenTypes.ChainState) {
	links := make([][4]float64, len(s.ChainLinks))
	for i, l := range s.ChainLinks {
		links[i] = [4]float64{l.X1, l.Y1, l.X2, l.Y2}
	}
	vfx.DrawChainLinks(screen, links)

	vfx.DrawShootFlash(screen, float32(s.X), float32(s.Y), s.ShootTimer,
		color.RGBA{R: 160, G: 80, B: 255, A: 200})
}

// ── 水灵特效：每个被选中敌人头顶天降冰柱 + 轻微命中闪光 ──

func drawSkystrikeEffects(screen *ebiten.Image, s *wardenTypes.SkystrikeState) {
	for _, st := range s.Strikes {
		vfx.DrawSkyStrike(screen, float32(st.X), float32(st.Y), st.Timer, st.Mode)
	}
}

// ── 金灵特效：射击闪光 + 施 buff 时短暂金色光束 ──

func drawEnvoyEffects(screen *ebiten.Image, s *wardenTypes.EnvoyState) {
	vfx.DrawShootFlash(screen, float32(s.X), float32(s.Y), s.ShootTimer,
		color.RGBA{R: 255, G: 210, B: 80, A: 200})

	if s.BuffedTower == nil || s.BuffExpiry <= 0 {
		return
	}
	elapsed := s.BuffDuration - s.BuffExpiry
	if elapsed > 0.6 {
		return
	}
	vfx.DrawGoldBeam(screen, float32(s.X), float32(s.Y),
		float32(s.BuffedTower.X), float32(s.BuffedTower.Y), 1.0-elapsed/0.6)
}
