// draw_projectile.go — 弹道体渲染模块。
//
// 弹道渲染是本游戏 draw call 最密集的部分（高攻速时同屏 100+ 弹丸），
// 因此采用全量批量化策略，通过 DrawTriangles 一次性提交 GPU：
//
//	尾迹（trail）: 圆形纹理四边形（点）+ 白像素线段四边形（连接线），见 trail_batch.go
//	弹体（body）:  速度尾线（tailVs）+ 弹体圆点（bodyVs）+ 辉光光环（glowVs，画到 GlowTarget）
//
// 唯一例外：wind 类弹丸的弧线效果难以批量化，延迟到最后逐个渲染（通常仅 1-2 塔）。
//
// 渲染顺序（由远到近）：
//
//	flushTrailBatch: 尾迹线 → 尾迹点
//	flushBodyBatch:  速度尾线 → 弹体圆 → 辉光（到 GlowTarget 做加法混合）→ wind 弧线
//
// 单次 pool.Each 遍历同时收集 trail + body 几何数据，最终仅 3-5 次 DrawTriangles。
package render

import (
	"image/color"
	"math"

	"defense2/internal/core/projectile"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/vfx"

	"github.com/hajimehoshi/ebiten/v2"
)

// DrawProjectiles 渲染所有存活弹丸，采用全批量化策略。
// 单次 pool.Each 遍历同时收集尾迹和弹体几何数据，最终以 3-5 次 DrawTriangles 完成。
func DrawProjectiles(screen *ebiten.Image, pool *projectile.Pool) {
	beginTrailBatch()
	beginBodyBatch()

	pool.Each(func(p *projectile.Projectile) {
		if !IsInView(p.X, p.Y) {
			return
		}
		// Trail (dots + connectors)
		collectTrail(p)
		// Body (velocity tail + type-specific shape)
		collectBody(p)
	})

	// Flush trails: lines behind, dots on top
	flushTrailBatch(screen)
	// Flush body: tails first, then body circles, then glow halos to glow target
	flushBodyBatch(screen)
}

// ── 弹体批量缓冲区 ────────────────────────────────────────────────
// 三个独立顶点缓冲区分别绘制到不同目标：
// - tailVs/tailIs: 速度尾线 → 画到 screen（白像素纹理）
// - bodyVs/bodyIs: 弹体圆点 → 画到 screen（圆形纹理）
// - glowVs/glowIs: 辉光外圈 → 画到 GlowTarget（加法混合，叠加发光效果）
// - deferred:      wind 弧线 → 最后逐个渲染（无法批量化）

var bodyBuf struct {
	tailVs   []ebiten.Vertex // 速度尾线顶点（线段四边形）
	tailIs   []uint16        // 速度尾线索引
	bodyVs   []ebiten.Vertex // 弹体圆点顶点（圆形纹理四边形）
	bodyIs   []uint16        // 弹体圆点索引
	glowVs   []ebiten.Vertex // 辉光外圈顶点（画到 GlowTarget）
	glowIs   []uint16        // 辉光外圈索引
	deferred []deferredBody  // 延迟渲染的特殊弹丸（wind 弧线）
}

type deferredBody struct {
	cx, cy float32
	angle  float64
	style  string
}

// 弹体缓冲区初始容量和安全上限。
const (
	bodyInitCapVs = 1024 * 4 // 4096
	bodyInitCapIs = 1024 * 6 // 6144
	bodyMaxVerts  = 60000    // uint16 安全上限（<65536）
)

// init 预分配弹体缓冲区，每种 1024 弹丸容量（4 顶点/弹 + 6 索引/弹）。
// 预分配避免运行时 append 触发频繁扩容和 GC。
func init() {
	bodyBuf.tailVs = make([]ebiten.Vertex, 0, bodyInitCapVs)
	bodyBuf.tailIs = make([]uint16, 0, bodyInitCapIs)
	bodyBuf.bodyVs = make([]ebiten.Vertex, 0, bodyInitCapVs)
	bodyBuf.bodyIs = make([]uint16, 0, bodyInitCapIs)
	bodyBuf.glowVs = make([]ebiten.Vertex, 0, bodyInitCapVs)
	bodyBuf.glowIs = make([]uint16, 0, bodyInitCapIs)
	bodyBuf.deferred = make([]deferredBody, 0, 32)
}

func beginBodyBatch() {
	bodyBuf.tailVs = compactVertices(bodyBuf.tailVs, bodyInitCapVs)
	bodyBuf.tailIs = compactIndices(bodyBuf.tailIs, bodyInitCapIs)
	bodyBuf.bodyVs = compactVertices(bodyBuf.bodyVs, bodyInitCapVs)
	bodyBuf.bodyIs = compactIndices(bodyBuf.bodyIs, bodyInitCapIs)
	bodyBuf.glowVs = compactVertices(bodyBuf.glowVs, bodyInitCapVs)
	bodyBuf.glowIs = compactIndices(bodyBuf.glowIs, bodyInitCapIs)
	bodyBuf.deferred = bodyBuf.deferred[:0]
}

// collectBody 将单个弹丸的弹体几何数据加入批量缓冲区。
// 所有弹丸共有速度尾线（白色半透明线段），之后按类型分支：
//   - Penetrate（穿透弹）: 紫色辉光 + 白芯双层，辉光画到 GlowTarget
//   - ScatterVisual（散射弹）: 蓝色小圆 + 短尾线
//   - sniper/rapid/freeze: 各有专属颜色和形状（冰弹为旋转菱形）
//   - wind: 弹体批量化 + 弧线延迟到 deferred 逐个渲染
//   - default: 通用辉光 + 小圆
func collectBody(p *projectile.Projectile) {
	cx := float32(p.X)
	cy := float32(p.Y)
	angle := math.Atan2(p.VY, p.VX)

	// Velocity tail (all projectiles)
	const tailLen = 8.0
	tailX := float64(cx) - math.Cos(angle)*tailLen
	tailY := float64(cy) - math.Sin(angle)*tailLen
	addBodyLine(cx, cy, float32(tailX), float32(tailY), 1.5,
		color.RGBA{R: 255, G: 255, B: 255, A: 120})

	switch {
	case p.Penetrate:
		// Outer glow (to glow target)
		addGlowCircle(cx, cy, 12, color.RGBA{R: 180, G: 100, B: 255, A: 50})
		// Inner bright core
		addBodyCircle(cx, cy, 5, color.RGBA{R: 180, G: 100, B: 255, A: 200})
		addBodyCircle(cx, cy, 3, color.RGBA{R: 220, G: 180, B: 255, A: 230})

	case p.ScatterVisual:
		addBodyCircle(cx, cy, 3, color.RGBA{R: 100, G: 180, B: 255, A: 200})
		scTailX := float64(cx) - math.Cos(angle)*5
		scTailY := float64(cy) - math.Sin(angle)*5
		addBodyLine(cx, cy, float32(scTailX), float32(scTailY), 1,
			color.RGBA{R: 100, G: 180, B: 255, A: 140})

	case isSniper(p.SourceTowerKey):
		addGlowCircle(cx, cy, theme.ProjSniperGlow, theme.ProjSniper)
		addBodyCircle(cx, cy, theme.ProjSniperR, theme.ProjSniper)

	case isRapid(p.SourceTowerKey):
		addBodyCircle(cx, cy, theme.ProjDefaultR, theme.ProjRapid)
		trailCx := cx - float32(math.Cos(angle)*3)
		trailCy := cy - float32(math.Sin(angle)*3)
		addBodyCircle(trailCx, trailCy, theme.ProjDefaultR*0.6,
			color.RGBA{R: theme.ProjRapid.R, G: theme.ProjRapid.G, B: theme.ProjRapid.B, A: 160})
		addGlowCircle(cx, cy, theme.ProjDefaultR+4,
			color.RGBA{R: theme.ProjRapid.R, G: theme.ProjRapid.G, B: theme.ProjRapid.B, A: 60})

	case isFreeze(p.SourceTowerKey):
		// Diamond as rotated square quad
		addBodyDiamond(cx, cy, theme.ProjDefaultR+1, float32(angle), theme.ProjFreeze)

	case isWind(p.SourceTowerKey):
		// Wind has arcs that are hard to batch — defer to individual render
		addBodyCircle(cx, cy, theme.ProjDefaultR, theme.ProjWind)
		bodyBuf.deferred = append(bodyBuf.deferred, deferredBody{cx, cy, angle, p.SourceTowerKey})

	default:
		addGlowCircle(cx, cy, 10, theme.ProjDefault)
		addBodyCircle(cx, cy, 3, theme.ProjDefault)
	}
}

// flushBodyBatch 提交所有弹体几何数据到 GPU，按 4 层顺序绘制：
//  1. 速度尾线（白像素纹理，画到 screen）
//  2. 弹体圆点（圆形纹理，画到 screen）
//  3. 辉光外圈（圆形纹理，画到 GlowTarget 做加法混合；无 GlowTarget 时降级到 BlendLighter）
//  4. 延迟渲染（wind 弧线，逐个 vector 绘制）
func flushBodyBatch(screen *ebiten.Image) {
	wp := trailWhitePixel()                // 3x3 白像素纹理，用于线段渲染
	ct := trailCircleTex()                 // 32x32 软边圆形纹理，用于圆点渲染
	noAA := &ebiten.DrawTrianglesOptions{} // 无需 AA：圆形纹理自带软边

	// 1. Velocity tails (line quads on screen)
	if len(bodyBuf.tailVs) > 0 {
		screen.DrawTriangles(bodyBuf.tailVs, bodyBuf.tailIs, wp, noAA)
	}

	// 2. Body circles (on screen)
	if len(bodyBuf.bodyVs) > 0 {
		screen.DrawTriangles(bodyBuf.bodyVs, bodyBuf.bodyIs, ct, noAA)
	}

	// 3. Glow circles (on glow target for additive compositing)
	if len(bodyBuf.glowVs) > 0 {
		if gt := draw.GlowTarget(); gt != nil {
			gt.DrawTriangles(bodyBuf.glowVs, bodyBuf.glowIs, ct, noAA)
		} else {
			// No glow pass active: draw directly with lighter blend
			screen.DrawTriangles(bodyBuf.glowVs, bodyBuf.glowIs, ct,
				&ebiten.DrawTrianglesOptions{Blend: ebiten.BlendLighter})
		}
	}

	// 4. Deferred individual renders (wind arcs — rare, ~1-2 towers worth)
	for _, d := range bodyBuf.deferred {
		vfx.DrawWindArcs(screen, d.cx, d.cy, d.angle)
	}
}

// ── 批量辅助函数 ─────────────────────────────────────────────────
// 以下函数将单个几何图元（圆/线/菱形）转为 4 顶点 + 6 索引的四边形，
// 追加到对应的缓冲区。draw.S32() 将逻辑坐标转为物理像素坐标。
// premulColor() 将 RGBA 转为预乘 alpha 格式（GPU 混合必须）。

// addBodyCircle 向弹体缓冲区追加一个圆形四边形（采样圆形纹理）。
func addBodyCircle(cx, cy, radius float32, clr color.RGBA) {
	if clr.A == 0 || len(bodyBuf.bodyVs) >= bodyMaxVerts {
		return
	}
	sx := draw.S32(cx)
	sy := draw.S32(cy)
	r := draw.S32(radius)
	cr, cg, cb, ca := premulColor(clr)
	idx := uint16(len(bodyBuf.bodyVs))
	bodyBuf.bodyVs = append(bodyBuf.bodyVs,
		ebiten.Vertex{DstX: sx - r, DstY: sy - r, SrcX: circU0, SrcY: circV0, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx + r, DstY: sy - r, SrcX: circU1, SrcY: circV0, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx + r, DstY: sy + r, SrcX: circU1, SrcY: circV1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx - r, DstY: sy + r, SrcX: circU0, SrcY: circV1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	bodyBuf.bodyIs = append(bodyBuf.bodyIs, idx, idx+1, idx+2, idx, idx+2, idx+3)
}

// addGlowCircle 向辉光缓冲区追加一个圆形四边形。
// alpha 自动降至 1/4 产生柔和光晕效果，最终画到 GlowTarget 做加法混合。
func addGlowCircle(cx, cy, radius float32, clr color.RGBA) {
	if clr.A == 0 || len(bodyBuf.glowVs) >= bodyMaxVerts {
		return
	}
	sx := draw.S32(cx)
	sy := draw.S32(cy)
	r := draw.S32(radius)
	// Glow outer circle: reduce alpha for soft halo
	a := clr.A / 4
	if a == 0 {
		a = 1
	}
	cr, cg, cb, ca := premulColor(color.RGBA{R: clr.R, G: clr.G, B: clr.B, A: a})
	idx := uint16(len(bodyBuf.glowVs))
	bodyBuf.glowVs = append(bodyBuf.glowVs,
		ebiten.Vertex{DstX: sx - r, DstY: sy - r, SrcX: circU0, SrcY: circV0, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx + r, DstY: sy - r, SrcX: circU1, SrcY: circV0, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx + r, DstY: sy + r, SrcX: circU1, SrcY: circV1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx - r, DstY: sy + r, SrcX: circU0, SrcY: circV1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	bodyBuf.glowIs = append(bodyBuf.glowIs, idx, idx+1, idx+2, idx, idx+2, idx+3)
}

// addBodyLine 向尾线缓冲区追加一个线段四边形（沿法线方向扩展宽度）。
func addBodyLine(x1, y1, x2, y2, width float32, clr color.RGBA) {
	if clr.A == 0 || len(bodyBuf.tailVs) >= bodyMaxVerts {
		return
	}
	sx1, sy1 := draw.S32(x1), draw.S32(y1)
	sx2, sy2 := draw.S32(x2), draw.S32(y2)
	w := draw.S32(width) * 0.5
	dx, dy := float64(sx2-sx1), float64(sy2-sy1)
	length := math.Sqrt(dx*dx + dy*dy)
	if length < 0.001 {
		return
	}
	nx := float32(-dy / length * float64(w))
	ny := float32(dx / length * float64(w))
	cr, cg, cb, ca := premulColor(clr)
	idx := uint16(len(bodyBuf.tailVs))
	bodyBuf.tailVs = append(bodyBuf.tailVs,
		ebiten.Vertex{DstX: sx1 - nx, DstY: sy1 - ny, SrcX: 1, SrcY: 1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx1 + nx, DstY: sy1 + ny, SrcX: 2, SrcY: 1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx2 + nx, DstY: sy2 + ny, SrcX: 2, SrcY: 2, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx2 - nx, DstY: sy2 - ny, SrcX: 1, SrcY: 2, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	bodyBuf.tailIs = append(bodyBuf.tailIs, idx, idx+1, idx+2, idx, idx+2, idx+3)
}

// addBodyDiamond 向弹体缓冲区追加一个菱形四边形（冰弹专用）。
// 通过旋转 4 个顶点实现菱形，采样圆形纹理的边缘区域产生锐利菱形轮廓。
func addBodyDiamond(cx, cy, size, angle float32, clr color.RGBA) {
	if len(bodyBuf.bodyVs) >= bodyMaxVerts {
		return
	}
	sx := draw.S32(cx)
	sy := draw.S32(cy)
	r := draw.S32(size)
	cr, cg, cb, ca := premulColor(clr)
	cos := float32(math.Cos(float64(angle)))
	sin := float32(math.Sin(float64(angle)))
	// Diamond: 4 vertices at 45° rotated by angle
	idx := uint16(len(bodyBuf.bodyVs))
	bodyBuf.bodyVs = append(bodyBuf.bodyVs,
		ebiten.Vertex{DstX: sx + r*cos, DstY: sy + r*sin, SrcX: circU1/2 + circU1/4, SrcY: circV0 + 1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx - r*sin, DstY: sy + r*cos, SrcX: circU1 - 1, SrcY: circV1/2 + circV1/4, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx - r*cos, DstY: sy - r*sin, SrcX: circU1/2 + circU1/4, SrcY: circV1 - 1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx + r*sin, DstY: sy - r*cos, SrcX: circU0 + 1, SrcY: circV1/2 + circV1/4, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	bodyBuf.bodyIs = append(bodyBuf.bodyIs, idx, idx+1, idx+2, idx, idx+2, idx+3)
}

// premulColor 将 RGBA 颜色转为预乘 alpha 格式（Ebitengine DrawTriangles 要求）。
func premulColor(clr color.RGBA) (float32, float32, float32, float32) {
	a := float32(clr.A) / 255
	return float32(clr.R) / 255 * a, float32(clr.G) / 255 * a, float32(clr.B) / 255 * a, a
}

// ── 塔类型匹配器（避免每弹丸每帧调用 strings.Contains 产生分配） ──
// 先尝试前缀快速匹配，失败时回退到手写子串搜索（零分配）。

func isSniper(key string) bool {
	return len(key) >= 6 && key[:6] == "sniper" || containsStr(key, "sniper")
}
func isRapid(key string) bool {
	return len(key) >= 5 && key[:5] == "rapid" || containsStr(key, "rapid")
}
func isFreeze(key string) bool {
	return len(key) >= 6 && key[:6] == "freeze" || containsStr(key, "freeze")
}
func isWind(key string) bool { return len(key) >= 4 && key[:4] == "wind" || containsStr(key, "wind") }

func containsStr(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
