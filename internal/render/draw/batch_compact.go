// batch_compact.go — 线段批量缓冲区压缩工具。
//
// 策略同 render/batch_compact.go：容量超过使用量 4 倍且超过初始预分配时重新分配。
package draw

import "github.com/hajimehoshi/ebiten/v2"

func compactVs(s []ebiten.Vertex, initCap int) []ebiten.Vertex {
	used := len(s)
	capacity := cap(s)
	if capacity > initCap && capacity > used*4 {
		newCap := used * 2
		if newCap < initCap {
			newCap = initCap
		}
		return make([]ebiten.Vertex, 0, newCap)
	}
	return s[:0]
}

func compactIs(s []uint16, initCap int) []uint16 {
	used := len(s)
	capacity := cap(s)
	if capacity > initCap && capacity > used*4 {
		newCap := used * 2
		if newCap < initCap {
			newCap = initCap
		}
		return make([]uint16, 0, newCap)
	}
	return s[:0]
}
