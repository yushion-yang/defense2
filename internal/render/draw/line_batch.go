// line_batch.go — 透明批量化 vector 线段，draw 包的核心性能优化之一。
//
// 问题：塔 VFX（Diamond/Arc/DashedCircle/ThickLine）每帧产生大量细碎线段，
// 逐条调用 vector.StrokeLine 导致 GPU draw call 爆炸。
//
// 方案：开启 BeginLineBatch 后，所有线段调用不再立即渲染，而是收集到顶点缓冲。
// FlushLineBatch 一次性通过 DrawTriangles 输出全部线段，将数百次 draw call 合为 1 次。
//
// 关键设计——对调用方完全透明：
// Diamond/Arc/ThickLine/DashedLine/DashedCircle 内部会检查 lineBatch.active，
// 如果批量化已开启则自动走 batchStrokeLine 路径，VFX 代码零改动。
// 只需在外层（draw_tower_vfx.go）包裹 BeginLineBatch/FlushLineBatch 即可。
package draw

import (
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// lineBatch 全局批量化状态。Ebitengine 单线程模型，无需加锁。
var lineBatch struct {
	active bool            // 批量化是否激活
	target *ebiten.Image   // 最终输出目标（screen 或 glow buffer）
	vs     []ebiten.Vertex // 顶点缓冲（每条线段 4 个顶点构成矩形）
	is     []uint16        // 索引缓冲（每条线段 6 个索引 = 2 个三角形）
}

var (
	lbWhiteOnce  sync.Once
	lbWhiteImage *ebiten.Image
)

// lbWhitePixel 返回 3x3 纯白纹理（DrawTriangles 必须有纹理源）。
// 用顶点颜色着色，纹理只是白色占位。3x3 而非 1x1 是为了避免边缘采样问题。
func lbWhitePixel() *ebiten.Image {
	lbWhiteOnce.Do(func() {
		lbWhiteImage = ebiten.NewImage(3, 3)
		pix := make([]byte, 3*3*4)
		for i := range pix {
			pix[i] = 0xff
		}
		lbWhiteImage.WritePixels(pix)
	})
	return lbWhiteImage
}

// init 预分配缓冲容量，避免运行时频繁扩容。
// 4096 顶点 = 1024 条线段，6144 索引 = 1024×6，足够覆盖单帧峰值。
func init() {
	lineBatch.vs = make([]ebiten.Vertex, 0, 4096)
	lineBatch.is = make([]uint16, 0, 6144)
}

// BeginLineBatch 开启线段批量化模式。
// 之后 Diamond/Arc/ThickLine 等函数检测到 active=true 会自动走批量路径。
// target 通常是 screen，也可以是 glow buffer。
func BeginLineBatch(target *ebiten.Image) {
	lineBatch.active = true
	lineBatch.target = target
	lineBatch.vs = lineBatch.vs[:0] // 复用底层数组，零分配
	lineBatch.is = lineBatch.is[:0]
}

// FlushLineBatch 将缓冲中所有线段通过一次 DrawTriangles 输出到 target。
// 这是批量化的核心收益——数百条线段只需 1 次 GPU draw call。
func FlushLineBatch() {
	if !lineBatch.active {
		return
	}
	lineBatch.active = false
	if len(lineBatch.vs) == 0 {
		return
	}
	lineBatch.target.DrawTriangles(lineBatch.vs, lineBatch.is, lbWhitePixel(),
		&ebiten.DrawTrianglesOptions{})
	lineBatch.target = nil
}

// LineBatchActive 返回线段批量化是否激活（供外部查询，如 dashed.go）。
func LineBatchActive() bool { return lineBatch.active }

// batchStrokeLine 将一条线段转化为矩形(4 顶点 + 6 索引)并追加到缓冲。
// 所有坐标已是物理像素（调用方负责 S32 缩放），颜色通过顶点着色实现。
//
// 几何原理：线段(x1,y1)→(x2,y2)沿法线方向各扩展 width/2，形成矩形。
// 法线方向 = (-dy, dx) / length，即线段方向旋转 90°。
func batchStrokeLine(x1, y1, x2, y2, width float32, clr color.Color) {
	// 解析颜色为 0~1 浮点（DrawTriangles 顶点颜色格式）
	r, g, b, a := clr.RGBA()
	if a == 0 {
		return
	}
	af := float32(a) / 0xffff
	rf := float32(r) / 0xffff
	gf := float32(g) / 0xffff
	bf := float32(b) / 0xffff

	// 计算线段法线方向，向两侧各扩展半宽
	w := width * 0.5
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	length := math.Sqrt(dx*dx + dy*dy)
	if length < 0.001 {
		return // 退化线段（两端重合），跳过
	}
	nx := float32(-dy / length * float64(w))
	ny := float32(dx / length * float64(w))

	// 追加 4 个顶点（矩形的 4 个角）和 6 个索引（2 个三角形）
	// SrcX/SrcY 指向白色纹理内部(1,1)~(2,2)，避免边缘采样
	idx := uint16(len(lineBatch.vs))
	lineBatch.vs = append(lineBatch.vs,
		ebiten.Vertex{DstX: x1 - nx, DstY: y1 - ny, SrcX: 1, SrcY: 1, ColorR: rf, ColorG: gf, ColorB: bf, ColorA: af},
		ebiten.Vertex{DstX: x1 + nx, DstY: y1 + ny, SrcX: 2, SrcY: 1, ColorR: rf, ColorG: gf, ColorB: bf, ColorA: af},
		ebiten.Vertex{DstX: x2 + nx, DstY: y2 + ny, SrcX: 2, SrcY: 2, ColorR: rf, ColorG: gf, ColorB: bf, ColorA: af},
		ebiten.Vertex{DstX: x2 - nx, DstY: y2 - ny, SrcX: 1, SrcY: 2, ColorR: rf, ColorG: gf, ColorB: bf, ColorA: af},
	)
	lineBatch.is = append(lineBatch.is, idx, idx+1, idx+2, idx, idx+2, idx+3)
}
