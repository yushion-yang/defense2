// trail_batch.go — 弹道尾迹批量渲染器。
//
// 将所有弹丸的尾迹点和连接线收集到顶点缓冲区，最终以 1-2 次 DrawTriangles
// 完成渲染，替代原先 12,000+ 次逐个 draw.FilledCircle/ThickLine 调用。
//
// 核心纹理资源（懒初始化，全局唯一）：
//   - circTexImage: 32x32 软边圆形纹理（径向渐变），用于尾迹点的圆形渲染
//     纹理自带边缘 1.5px 淡出过渡，无需开启 AntiAlias 标志即可获得平滑边缘
//   - trailWhiteImage: 3x3 纯白像素纹理，用于线段四边形的颜色采样
//
// 这两个纹理也被 draw_projectile.go 的弹体渲染复用（circU0/V0/U1/V1 是共享 UV 坐标）。
package render

import (
	"image/color"
	"math"
	"sync"

	"defense2/internal/core/projectile"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/vfx"

	"github.com/hajimehoshi/ebiten/v2"
)

// trailBatch 收集尾迹几何数据供批量渲染。
// circVs/circIs: 圆形点的顶点/索引（使用圆形纹理）
// lineVs/lineIs: 连接线的顶点/索引（使用白像素纹理）
var trailBatch struct {
	circVs []ebiten.Vertex
	circIs []uint16
	lineVs []ebiten.Vertex
	lineIs []uint16
}

// ── 圆形纹理（软边径向渐变）──────────────────────────────────────
// 在 GPU 上用纹理采样模拟圆形，比 vector 绘制圆快一个数量级。
// 纹理中心为实心白色，边缘 1.5px 线性淡出到透明，实现天然抗锯齿。
// 预乘 alpha 格式：R=G=B=A（白色 * alpha），通过 ColorScale 着色。

const circTexSize = 32 // 32x32 预渲染圆形纹理（含软边过渡）

var (
	circTexOnce  sync.Once
	circTexImage *ebiten.Image
	// UV coords for sampling the circle center region
	circU0, circV0, circU1, circV1 float32
)

func trailCircleTex() *ebiten.Image {
	circTexOnce.Do(func() {
		const s = circTexSize
		circTexImage = ebiten.NewImage(s, s)
		pix := make([]byte, s*s*4)
		center := float64(s) / 2
		maxR := center - 0.5 // leave half-pixel border

		for y := 0; y < s; y++ {
			for x := 0; x < s; x++ {
				dx := float64(x) + 0.5 - center
				dy := float64(y) + 0.5 - center
				dist := math.Sqrt(dx*dx + dy*dy)

				var alpha float64
				if dist <= maxR-1.5 {
					alpha = 1.0 // solid core
				} else if dist <= maxR {
					alpha = 1.0 - (dist-(maxR-1.5))/1.5 // smooth fade at edge
				}
				// else alpha = 0 (outside circle)

				a := byte(alpha * 255)
				off := (y*s + x) * 4
				pix[off+0] = a // pre-multiplied white: R=A, G=A, B=A, A=A
				pix[off+1] = a
				pix[off+2] = a
				pix[off+3] = a
			}
		}
		circTexImage.WritePixels(pix)

		// UV: sample full texture (0..circTexSize maps to SrcX/SrcY)
		circU0 = 0
		circV0 = 0
		circU1 = float32(s)
		circV1 = float32(s)
	})
	return circTexImage
}

// ── 白像素纹理（用于线段渲染）─────────────────────────────────────
// 3x3 全白纹理（非 1x1，避免 GPU 采样边缘问题），线段四边形从中采样纯白色，
// 实际颜色通过顶点 ColorR/G/B/A 着色。

var (
	trailWhiteOnce  sync.Once
	trailWhiteImage *ebiten.Image
)

func trailWhitePixel() *ebiten.Image {
	trailWhiteOnce.Do(func() {
		trailWhiteImage = ebiten.NewImage(3, 3)
		pix := make([]byte, 3*3*4)
		for i := range pix {
			pix[i] = 0xff
		}
		trailWhiteImage.WritePixels(pix)
	})
	return trailWhiteImage
}

// ── 初始化 ───────────────────────────────────────────────────────────

// init 预分配尾迹缓冲区。
// maxDots=7168: 按最大 1024 弹丸、每弹 7 个尾迹点估算
// maxLines=5120: 每弹 5 条连接线段估算
// 每个图元占 4 顶点 + 6 索引，预分配后运行时几乎不触发扩容。
func init() {
	const maxDots = 1024 * 7
	const maxLines = 1024 * 5
	trailBatch.circVs = make([]ebiten.Vertex, 0, maxDots*4)
	trailBatch.circIs = make([]uint16, 0, maxDots*6)
	trailBatch.lineVs = make([]ebiten.Vertex, 0, maxLines*4)
	trailBatch.lineIs = make([]uint16, 0, maxLines*6)
}

// ── Batch API ────────────────────────────────────────────────────────

// trailInitCapVs / trailInitCapIs 是 init() 中预分配的初始容量。
// 压缩逻辑不会缩到比初始值更小，避免正常场景下反复分配。
const (
	trailInitCapCircVs = 1024 * 7 * 4 // 28672
	trailInitCapCircIs = 1024 * 7 * 6 // 43008
	trailInitCapLineVs = 1024 * 5 * 4 // 20480
	trailInitCapLineIs = 1024 * 5 * 6 // 30720
	trailMaxVerts      = 60000        // uint16 安全上限（<65536），超过则丢弃新几何
)

func beginTrailBatch() {
	// 压缩：如果容量超过上帧使用量 4 倍且超过初始预分配，重新分配释放峰值内存
	trailBatch.circVs = compactVertices(trailBatch.circVs, trailInitCapCircVs)
	trailBatch.circIs = compactIndices(trailBatch.circIs, trailInitCapCircIs)
	trailBatch.lineVs = compactVertices(trailBatch.lineVs, trailInitCapLineVs)
	trailBatch.lineIs = compactIndices(trailBatch.lineIs, trailInitCapLineIs)
}

// addTrailDot 向批量缓冲区追加一个软边圆点（采样圆形纹理）。
// 颜色预乘 alpha 处理后设置到四个顶点的 ColorR/G/B/A 上。
func addTrailDot(cx, cy, radius float32, clr color.RGBA) {
	if clr.A == 0 || radius <= 0 {
		return
	}
	// uint16 索引安全上限，超过则丢弃（防止索引溢出导致渲染垃圾）
	if len(trailBatch.circVs) >= trailMaxVerts {
		return
	}
	sx := draw.S32(cx)
	sy := draw.S32(cy)
	r := draw.S32(radius)

	cr := float32(clr.R) / 255 * float32(clr.A) / 255
	cg := float32(clr.G) / 255 * float32(clr.A) / 255
	cb := float32(clr.B) / 255 * float32(clr.A) / 255
	ca := float32(clr.A) / 255

	idx := uint16(len(trailBatch.circVs))
	trailBatch.circVs = append(trailBatch.circVs,
		ebiten.Vertex{DstX: sx - r, DstY: sy - r, SrcX: circU0, SrcY: circV0, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx + r, DstY: sy - r, SrcX: circU1, SrcY: circV0, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx + r, DstY: sy + r, SrcX: circU1, SrcY: circV1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx - r, DstY: sy + r, SrcX: circU0, SrcY: circV1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	trailBatch.circIs = append(trailBatch.circIs, idx, idx+1, idx+2, idx, idx+2, idx+3)
}

// addTrailLine 向批量缓冲区追加一条粗线段（扩展为旋转矩形四边形）。
// 计算线段法线方向并沿法线扩展半宽度，生成 4 个顶点构成矩形。
func addTrailLine(x1, y1, x2, y2, width float32, clr color.RGBA) {
	if clr.A == 0 || width <= 0 {
		return
	}
	if len(trailBatch.lineVs) >= trailMaxVerts {
		return
	}
	sx1 := draw.S32(x1)
	sy1 := draw.S32(y1)
	sx2 := draw.S32(x2)
	sy2 := draw.S32(y2)
	w := draw.S32(width) * 0.5

	dx := float64(sx2 - sx1)
	dy := float64(sy2 - sy1)
	length := math.Sqrt(dx*dx + dy*dy)
	if length < 0.001 {
		return
	}
	nx := float32(-dy / length * float64(w))
	ny := float32(dx / length * float64(w))

	cr := float32(clr.R) / 255 * float32(clr.A) / 255
	cg := float32(clr.G) / 255 * float32(clr.A) / 255
	cb := float32(clr.B) / 255 * float32(clr.A) / 255
	ca := float32(clr.A) / 255

	idx := uint16(len(trailBatch.lineVs))
	trailBatch.lineVs = append(trailBatch.lineVs,
		ebiten.Vertex{DstX: sx1 - nx, DstY: sy1 - ny, SrcX: 1, SrcY: 1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx1 + nx, DstY: sy1 + ny, SrcX: 2, SrcY: 1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx2 + nx, DstY: sy2 + ny, SrcX: 2, SrcY: 2, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx2 - nx, DstY: sy2 - ny, SrcX: 1, SrcY: 2, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	trailBatch.lineIs = append(trailBatch.lineIs, idx, idx+1, idx+2, idx, idx+2, idx+3)
}

// collectTrail 将单个弹丸的尾迹数据加入批量缓冲区。
// 尾迹是固定长度的环形缓冲区（TrailLen 个点），从最旧到最新遍历。
// 每个点绘制一个圆点，相邻活跃点之间绘制一条连接线。
// alpha 和半径随 frac（0→1 从旧到新）递增，产生渐隐拖尾效果。
// 最新点额外叠加一个 1.5x 大小的半透明光晕作为弹头标记。
func collectTrail(p *projectile.Projectile) {
	baseClr := vfx.ProjectileTrailColor(p.SourceTowerKey)
	n := projectile.TrailLen

	var prevX, prevY float32
	var prevActive bool
	var prevAlpha uint8
	var prevR float32

	for i := 0; i < n; i++ {
		idx := (p.TrailCursor + i) % n
		pt := p.Trail[idx]
		if !pt.Active {
			prevActive = false
			continue
		}
		frac := float64(i+1) / float64(n)
		alpha := uint8(140 * frac)
		r := float32(theme.ProjDefaultR) * float32(0.3+0.7*frac)
		clr := color.RGBA{R: baseClr.R, G: baseClr.G, B: baseClr.B, A: alpha}

		curX := float32(pt.X)
		curY := float32(pt.Y)

		if prevActive {
			lineAlpha := prevAlpha
			if alpha < lineAlpha {
				lineAlpha = alpha
			}
			lineClr := color.RGBA{R: baseClr.R, G: baseClr.G, B: baseClr.B, A: lineAlpha}
			lineW := prevR * 0.8
			if lineW < 0.5 {
				lineW = 0.5
			}
			addTrailLine(prevX, prevY, curX, curY, lineW, lineClr)
		}

		addTrailDot(curX, curY, r, clr)

		if i == n-1 {
			addTrailDot(curX, curY, r*1.5, color.RGBA{R: clr.R, G: clr.G, B: clr.B, A: clr.A / 3})
		}

		prevX, prevY = curX, curY
		prevAlpha = alpha
		prevR = r
		prevActive = true
	}
}

// flushTrailBatch 提交所有收集到的尾迹几何数据到 GPU。
// 渲染顺序：先连接线（白像素纹理），再圆点（圆形纹理）。
// 线在下、点在上，确保尾迹点不被连接线遮挡。
// 无需 AntiAlias 标志——圆形纹理的软边已提供天然抗锯齿。
func flushTrailBatch(screen *ebiten.Image) {
	// Lines: use white pixel, no AA needed for thin connectors
	if len(trailBatch.lineVs) > 0 {
		screen.DrawTriangles(trailBatch.lineVs, trailBatch.lineIs, trailWhitePixel(),
			&ebiten.DrawTrianglesOptions{})
	}
	// Dots: use circle texture for smooth round shapes
	if len(trailBatch.circVs) > 0 {
		screen.DrawTriangles(trailBatch.circVs, trailBatch.circIs, trailCircleTex(),
			&ebiten.DrawTrianglesOptions{})
	}
}
