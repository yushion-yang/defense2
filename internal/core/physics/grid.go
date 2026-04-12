// grid.go — 空间网格索引，用于高效碰撞检测。
// 将世界空间划分为固定大小的网格单元，每帧重建一次，支持区域查询。
// 设计为零分配热路径：所有缓冲区预分配，Rebuild/Query 不产生堆分配。
package physics

// CellSize 网格单元大小（逻辑像素）。
const CellSize = 64

// SpatialGrid 基于固定网格的空间索引。
// 所有实体按位置分配到网格单元，查询时只检查覆盖区域内的单元。
type SpatialGrid struct {
	cols, rows int
	// 每个 cell 在 pool 中的起始偏移和数量
	offsets []int // len = cols*rows, cell[r*cols+c] 的元素从 pool[offsets[i]] 开始
	counts  []int // len = cols*rows, cell[r*cols+c] 的元素数量
	// 所有 cell 共享的后备存储
	pool []int
	// 查询结果缓冲（避免查询时分配）
	queryBuf []int
}

// NewSpatialGrid 创建覆盖指定世界尺寸的空间网格。
func NewSpatialGrid(worldW, worldH float64) *SpatialGrid {
	cols := int(worldW/CellSize) + 1
	rows := int(worldH/CellSize) + 1
	n := cols * rows
	return &SpatialGrid{
		cols:     cols,
		rows:     rows,
		offsets:  make([]int, n),
		counts:   make([]int, n),
		pool:     make([]int, 0, 512),
		queryBuf: make([]int, 0, 64),
	}
}

// EntityPos 实体位置接口，由 Rebuild 的调用者提供。
type EntityPos struct {
	Index  int
	X, Y   float64
	Active bool
}

// Rebuild 从实体列表重建网格。每帧调用一次。
// 两遍算法：第一遍计数，第二遍填充。零分配（复用 pool）。
func (g *SpatialGrid) Rebuild(entities []EntityPos) {
	n := g.cols * g.rows

	// 清零计数
	for i := 0; i < n; i++ {
		g.counts[i] = 0
	}

	// 第一遍：计数
	for i := range entities {
		e := &entities[i]
		if !e.Active {
			continue
		}
		ci := g.cellIndex(e.X, e.Y)
		if ci >= 0 && ci < n {
			g.counts[ci]++
		}
	}

	// 计算偏移（前缀和）
	total := 0
	for i := 0; i < n; i++ {
		g.offsets[i] = total
		total += g.counts[i]
	}

	// 确保 pool 容量足够
	if cap(g.pool) < total {
		g.pool = make([]int, total)
	}
	g.pool = g.pool[:total]

	// 临时复用 counts 作为写入游标（重置为 0）
	for i := 0; i < n; i++ {
		g.counts[i] = 0
	}

	// 第二遍：填充
	for i := range entities {
		e := &entities[i]
		if !e.Active {
			continue
		}
		ci := g.cellIndex(e.X, e.Y)
		if ci >= 0 && ci < n {
			pos := g.offsets[ci] + g.counts[ci]
			g.pool[pos] = e.Index
			g.counts[ci]++
		}
	}
}

// Query 返回以 (x,y) 为圆心、radius 为半径的区域内所有实体索引。
// 返回的 slice 是内部缓冲，下次 Query 调用会覆盖。
func (g *SpatialGrid) Query(x, y, radius float64) []int {
	g.queryBuf = g.queryBuf[:0]

	minCol := int((x - radius) / CellSize)
	maxCol := int((x + radius) / CellSize)
	minRow := int((y - radius) / CellSize)
	maxRow := int((y + radius) / CellSize)

	if minCol < 0 {
		minCol = 0
	}
	if minRow < 0 {
		minRow = 0
	}
	if maxCol >= g.cols {
		maxCol = g.cols - 1
	}
	if maxRow >= g.rows {
		maxRow = g.rows - 1
	}

	for r := minRow; r <= maxRow; r++ {
		for c := minCol; c <= maxCol; c++ {
			ci := r*g.cols + c
			off := g.offsets[ci]
			cnt := g.counts[ci]
			for j := 0; j < cnt; j++ {
				g.queryBuf = append(g.queryBuf, g.pool[off+j])
			}
		}
	}

	return g.queryBuf
}

// cellIndex 返回坐标所在的 cell 线性索引。
func (g *SpatialGrid) cellIndex(x, y float64) int {
	c := int(x / CellSize)
	r := int(y / CellSize)
	if c < 0 || c >= g.cols || r < 0 || r >= g.rows {
		return -1
	}
	return r*g.cols + c
}
