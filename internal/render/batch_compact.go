// batch_compact.go — 批量渲染缓冲区压缩工具。
//
// 解决缓冲区只增不缩问题：Go slice 通过 append 扩容后，[:0] 重置只清长度不释放
// backing array。高峰期（大量弹丸）扩容的内存会永久驻留，增加 GC 扫描负担。
//
// 策略：每帧 begin 时检查——如果容量超过上帧使用量 4 倍且超过初始预分配，
// 则以 2 倍使用量重新分配，释放峰值内存。正常帧完全零分配。
package render

import "github.com/hajimehoshi/ebiten/v2"

// compactVertices 压缩顶点 slice：如果容量远超使用量，重新分配释放内存。
// initCap 是初始预分配容量，不会缩到比它更小。
func compactVertices(s []ebiten.Vertex, initCap int) []ebiten.Vertex {
	used := len(s)
	capacity := cap(s)
	// 容量超过使用量 4 倍且超过初始预分配 → 压缩
	if capacity > initCap && capacity > used*4 {
		newCap := used * 2
		if newCap < initCap {
			newCap = initCap
		}
		return make([]ebiten.Vertex, 0, newCap)
	}
	return s[:0]
}

// compactIndices 压缩索引 slice，逻辑同 compactVertices。
func compactIndices(s []uint16, initCap int) []uint16 {
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
