// grid.go — 空间网格索引，用于高效碰撞检测。
//
// 将世界空间划分为 64px 的网格单元，每帧重建一次，支持圆形区域查询。
// 设计为零分配热路径：所有缓冲区预分配，Rebuild/Query 不产生堆分配。
//
// 性能收益（以弹幕碰撞为例）：
//   - 无网格：每弹遍历所有敌人 O(P*E)，P=256弹 * E=50敌 = 12,800 次
//   - 有网格：每弹只查 1-4 个 cell O(P*k)，k≈4，= 1,024 次
//   - 实测 barrage 模式从 262K 次碰撞检查降至 4K 次
//
// Rebuild 采用两遍算法（counting sort 变体）：
//
//	第一遍计数每个 cell 有多少实体 → 前缀和得偏移 → 第二遍填充
//	这避免了每个 cell 用独立 slice 带来的分配。
//
// 关联文件：
//   - tick_combat.go: 穿透弹碰撞检测使用 Query
//   - tick_tower.go: 塔索敌使用 Query
//   - stage.go: 每帧调用 Rebuild 更新网格
package physics

// CellSize 网格单元大小（逻辑像素）。
const CellSize = 64

// SpatialGrid 基于固定网格的空间索引。
// 所有实体按位置分配到网格单元，查询时只检查覆盖区域内的单元。
//
// 存储布局（紧凑数组，非每 cell 一个 slice）：
//
//	offsets[ci] 指向 pool 中该 cell 的起始位置
//	counts[ci] 记录该 cell 中的实体数量
//	pool[offsets[ci]..offsets[ci]+counts[ci]] 存储实体索引
type SpatialGrid struct {
	cols, rows int
	offsets    []int // len = cols*rows, cell 在 pool 中的起始偏移（前缀和）
	counts     []int // len = cols*rows, cell 中的实体数量（Rebuild 中复用为写入游标）
	pool       []int // 所有 cell 共享的后备存储（紧凑排列）
	queryBuf   []int // 查询结果缓冲（避免查询时分配，下次 Query 覆盖）
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

// Query 返回以 (x,y) 为圆心、radius 为半径的区域内所有实体索引（AABB 粗筛，无精确圆检测）。
// 返回的 slice 是内部缓冲，下次 Query 调用会覆盖——调用方需立即消费或拷贝。
// 注意：返回的是 AABB 覆盖的 cell 中的所有实体，调用方需自行做精确距离检查。
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
