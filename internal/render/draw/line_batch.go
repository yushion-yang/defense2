// line_batch.go — 透明批量化 vector 线段。
// 开启 BeginLineBatch 后，StrokeLine 调用会收集到顶点缓冲而非立即渲染。
// FlushLineBatch 一次性 DrawTriangles 输出所有线段。
// 对调用方完全透明——不改 VFX 代码，只需在外层包裹 begin/flush。
package draw

import (
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

var lineBatch struct {
	active bool
	target *ebiten.Image
	vs     []ebiten.Vertex
	is     []uint16
}

var (
	lbWhiteOnce  sync.Once
	lbWhiteImage *ebiten.Image
)

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

func init() {
	lineBatch.vs = make([]ebiten.Vertex, 0, 4096)
	lineBatch.is = make([]uint16, 0, 6144)
}

// BeginLineBatch 开启线段批量化模式。之后 batchStrokeLine 会收集到缓冲。
func BeginLineBatch(target *ebiten.Image) {
	lineBatch.active = true
	lineBatch.target = target
	lineBatch.vs = lineBatch.vs[:0]
	lineBatch.is = lineBatch.is[:0]
}

// FlushLineBatch 输出所有收集的线段。
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

// LineBatchActive 返回线段批量化是否激活。
func LineBatchActive() bool { return lineBatch.active }

// batchStrokeLine 将一条线段收集到缓冲。
func batchStrokeLine(x1, y1, x2, y2, width float32, clr color.Color) {
	r, g, b, a := clr.RGBA()
	if a == 0 {
		return
	}
	af := float32(a) / 0xffff
	rf := float32(r) / 0xffff
	gf := float32(g) / 0xffff
	bf := float32(b) / 0xffff

	w := width * 0.5
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	length := math.Sqrt(dx*dx + dy*dy)
	if length < 0.001 {
		return
	}
	nx := float32(-dy / length * float64(w))
	ny := float32(dx / length * float64(w))

	idx := uint16(len(lineBatch.vs))
	lineBatch.vs = append(lineBatch.vs,
		ebiten.Vertex{DstX: x1 - nx, DstY: y1 - ny, SrcX: 1, SrcY: 1, ColorR: rf, ColorG: gf, ColorB: bf, ColorA: af},
		ebiten.Vertex{DstX: x1 + nx, DstY: y1 + ny, SrcX: 2, SrcY: 1, ColorR: rf, ColorG: gf, ColorB: bf, ColorA: af},
		ebiten.Vertex{DstX: x2 + nx, DstY: y2 + ny, SrcX: 2, SrcY: 2, ColorR: rf, ColorG: gf, ColorB: bf, ColorA: af},
		ebiten.Vertex{DstX: x2 - nx, DstY: y2 - ny, SrcX: 1, SrcY: 2, ColorR: rf, ColorG: gf, ColorB: bf, ColorA: af},
	)
	lineBatch.is = append(lineBatch.is, idx, idx+1, idx+2, idx, idx+2, idx+3)
}
