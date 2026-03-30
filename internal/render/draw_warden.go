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
	trailClr := wardenShootColor(w.Type)
	cursor := base.TrailCursor
	for i := 0; i < len(base.TrailHistory); i++ {
		idx := (cursor - 1 - i + len(base.TrailHistory)) % len(base.TrailHistory)
		tx, ty := base.TrailHistory[idx][0], base.TrailHistory[idx][1]
		if tx == 0 && ty == 0 {
			break
		}
		age := float64(i+1) / float64(len(base.TrailHistory))
		alpha := uint8(float64(trailClr.A) * 0.3 * (1 - age))
		r := float32(6 * (1 - age))
		if r < 1 {
			break
		}
		draw.FilledCircle(screen, float32(tx), float32(ty), r,
			color.RGBA{R: trailClr.R, G: trailClr.G, B: trailClr.B, A: alpha})
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
	// 悬浮动画：Y 轴上下浮动 + 微缩放脉冲
	bobY := math.Sin(animTime*2.5) * 3.0
	displaySize := float64(wardenSpriteSize) * (1.0 + 0.015*math.Sin(animTime*2.0))

	img := wr.loadSprite(w.Type)
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
	// 能量连线：串联塔之间画紫色脉冲线段
	for _, link := range s.ChainLinks {
		draw.Line(screen, float32(link.X1), float32(link.Y1),
			float32(link.X2), float32(link.Y2), 1.5,
			color.RGBA{R: 160, G: 80, B: 255, A: 50}, true)
	}

	// 射击闪光
	if s.ShootTimer > 0 {
		p := s.ShootTimer / 0.15
		alpha := uint8(180 * p)
		draw.FilledCircle(screen, float32(s.X), float32(s.Y), float32(5+3*p),
			color.RGBA{R: 160, G: 80, B: 255, A: alpha / 2})
	}
}

// ── 水灵特效：每个被选中敌人头顶天降冰柱 + 轻微命中闪光 ──

func drawSkystrikeEffects(screen *ebiten.Image, s *wardenTypes.SkystrikeState) {
	for _, st := range s.Strikes {
		drawSingleStrike(screen, st)
	}
}

func drawSingleStrike(screen *ebiten.Image, st wardenTypes.StrikeVFX) {
	p := float32(st.Timer / 0.8) // 1→0
	sx, sy := float32(st.X), float32(st.Y)

	// 预警瞄准圈：落下前显示，alpha 随 p 衰减
	if p > 0.3 {
		circleA := uint8(120 * (p - 0.3) / 0.7)
		circleR := float32(12 * (2 - p)) // 扩散效果
		draw.DashedCircle(screen, sx, sy, circleR, 1, 4, 3,
			color.RGBA{R: 120, G: 200, B: 255, A: circleA})
	}

	switch st.Mode {
	case 1:
		drawIceCone(screen, sx, sy, p)
	case 2:
		drawWaterDrop(screen, sx, sy, p)
	case 3:
		drawGeyser(screen, sx, sy, p)
	default:
		drawIceCone(screen, sx, sy, p)
	}
}

// ── 散射：冰锥从高处加速坠落砸中 ──
func drawIceCone(screen *ebiten.Image, sx, sy, p float32) {
	a := uint8(200 * p)
	// 冰锥从 80px 上方加速落下（p*p 加速曲线，越接近地面越快）
	drop := 80 * p * p
	tipY := sy - drop
	// 菱形冰锥：上尖下宽
	draw.Line(screen, sx, tipY-8, sx-4, tipY, 1.5, color.RGBA{R: 120, G: 210, B: 255, A: a}, true)
	draw.Line(screen, sx, tipY-8, sx+4, tipY, 1.5, color.RGBA{R: 120, G: 210, B: 255, A: a}, true)
	draw.Line(screen, sx-4, tipY, sx, tipY+5, 1.5, color.RGBA{R: 120, G: 210, B: 255, A: a}, true)
	draw.Line(screen, sx+4, tipY, sx, tipY+5, 1.5, color.RGBA{R: 120, G: 210, B: 255, A: a}, true)
	// 冰锥高光
	draw.FilledCircle(screen, sx, tipY-2, 2, color.RGBA{R: 220, G: 245, B: 255, A: a})

	// 落地冲击（只在接近地面时，p < 0.4）
	if p < 0.4 {
		ip := (0.4 - p) / 0.4 // 0→1
		ia := uint8(180 * ip)
		draw.FilledCircle(screen, sx, sy, float32(6*ip), color.RGBA{R: 160, G: 230, B: 255, A: ia / 2})
		draw.FilledCircle(screen, sx, sy, float32(3*ip), color.RGBA{R: 220, G: 245, B: 255, A: ia})
		// 碎冰
		sp := float32(8 * ip)
		draw.FilledCircle(screen, sx-sp, sy-sp*0.4, 1.5, color.RGBA{R: 200, G: 235, B: 255, A: ia / 2})
		draw.FilledCircle(screen, sx+sp*0.8, sy+sp*0.3, 1.5, color.RGBA{R: 200, G: 235, B: 255, A: ia / 2})
	}
}

// ── 连击：水滴从高处加速坠落溅开 ──
func drawWaterDrop(screen *ebiten.Image, sx, sy, p float32) {
	a := uint8(220 * p)
	// 水滴从 60px 上方加速落下
	drop := 60 * p * p
	dropY := sy - drop
	// 水滴（椭圆形）
	draw.FilledCircle(screen, sx, dropY, 3, color.RGBA{R: 60, G: 140, B: 255, A: a})
	draw.FilledCircle(screen, sx, dropY-2, 1.5, color.RGBA{R: 150, G: 200, B: 255, A: a})

	// 溅射（落地后）
	if p < 0.5 {
		ip := (0.5 - p) / 0.5
		ia := uint8(160 * ip)
		// 水花向两侧溅开
		sp := float32(6 * ip)
		draw.FilledCircle(screen, sx-sp, sy-sp*0.3, 2, color.RGBA{R: 80, G: 160, B: 255, A: ia})
		draw.FilledCircle(screen, sx+sp, sy-sp*0.2, 2, color.RGBA{R: 80, G: 160, B: 255, A: ia})
		draw.FilledCircle(screen, sx, sy, float32(4*ip), color.RGBA{R: 120, G: 190, B: 255, A: ia / 2})
	}
}

// ── 收割：水柱从地面冲击涌起 ──
func drawGeyser(screen *ebiten.Image, sx, sy, p float32) {
	a := uint8(180 * p)
	// 水柱从地面向上涌起（不是从天降下）
	height := float32(35 * p)
	// 底部水花
	draw.FilledCircle(screen, sx, sy, float32(8*p), color.RGBA{R: 140, G: 210, B: 255, A: a / 3})
	// 水柱（多层，从底部往上渐细）
	draw.Line(screen, sx-3, sy, sx-2, sy-height, 3, color.RGBA{R: 100, G: 190, B: 255, A: a}, true)
	draw.Line(screen, sx+3, sy, sx+2, sy-height, 3, color.RGBA{R: 100, G: 190, B: 255, A: a}, true)
	draw.Line(screen, sx, sy, sx, sy-height-5, 2, color.RGBA{R: 200, G: 235, B: 255, A: a}, true)
	// 顶部水花飞溅
	draw.FilledCircle(screen, sx, sy-height-3, float32(3*p), color.RGBA{R: 220, G: 240, B: 255, A: a})
	// 侧面水珠
	if p > 0.3 {
		sp := float32((p - 0.3) / 0.7)
		spr := float32(12 * sp)
		sa := uint8(120 * sp)
		draw.FilledCircle(screen, sx-spr, sy-height*0.6, 2, color.RGBA{R: 160, G: 220, B: 255, A: sa})
		draw.FilledCircle(screen, sx+spr*0.8, sy-height*0.5, 1.5, color.RGBA{R: 160, G: 220, B: 255, A: sa})
		draw.FilledCircle(screen, sx-spr*0.5, sy-height*0.3, 1.5, color.RGBA{R: 160, G: 220, B: 255, A: sa})
	}
}

// ── 金灵特效：射击闪光 + 施 buff 时短暂金色光束 ──

func drawEnvoyEffects(screen *ebiten.Image, s *wardenTypes.EnvoyState) {
	// 射击闪光
	if s.ShootTimer > 0 {
		p := s.ShootTimer / 0.15
		alpha := uint8(180 * p)
		draw.FilledCircle(screen, float32(s.X), float32(s.Y), float32(5+3*p),
			color.RGBA{R: 255, G: 210, B: 80, A: alpha / 2})
	}

	// 施 buff 瞬间：短暂金色光束（仅 buff 刚施加的前 0.6s）
	if s.BuffedTower == nil || s.BuffExpiry <= 0 {
		return
	}
	elapsed := s.BuffDuration - s.BuffExpiry // 已过去的时间
	if elapsed > 0.6 {
		return
	}
	p := 1.0 - elapsed/0.6 // 1→0 衰减
	alpha := uint8(200 * p)
	tx, ty := float32(s.BuffedTower.X), float32(s.BuffedTower.Y)
	wx, wy := float32(s.X), float32(s.Y)

	// 金色光束
	draw.Line(screen, wx, wy, tx, ty, float32(2*p),
		color.RGBA{R: 255, G: 220, B: 80, A: alpha}, true)
	// 塔顶闪光
	draw.FilledCircle(screen, tx, ty, float32(10*p),
		color.RGBA{R: 255, G: 240, B: 130, A: alpha / 2})
}
