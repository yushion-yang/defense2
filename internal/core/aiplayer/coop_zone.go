// coop_zone.go — N 分区 Zone 系统。
//
// 替代 Phase 1 的二分 Zone（按列中点硬分），支持任意数量的分区。
// 每个分区由列/行范围定义，支持 2 人（左右）、4 人（田字）、6 人（六角）等布局。
// 分区信息从地图 JSON 的 coop.sections 字段加载。
package aiplayer

// ZoneSection 合作地图分区定义（从 config.CoopSection 转换而来）。
type ZoneSection struct {
	ID       string
	ColStart int // 列范围 [ColStart, ColEnd)
	ColEnd   int
	RowStart int // 行范围 [RowStart, RowEnd)
	RowEnd   int
	Owner    int // 0=human, 1..N-1=AI
	Theme    string
}

// CoopZone N 分区 Zone 系统。
type CoopZone struct {
	cols, rows int
	sections   []ZoneSection
}

// NewCoopZone 创建 N 分区 Zone。
// RowEnd=0 的分区自动补全为全行。
func NewCoopZone(cols, rows int, sections []ZoneSection) *CoopZone {
	copied := make([]ZoneSection, len(sections))
	copy(copied, sections)
	for i := range copied {
		if copied[i].RowEnd == 0 {
			copied[i].RowEnd = rows
		}
	}
	return &CoopZone{cols: cols, rows: rows, sections: copied}
}

// OwnerOf 返回指定格子的所有者 ID。
// 多个分区重叠时返回第一个匹配的。无匹配返回 0（human）。
func (z *CoopZone) OwnerOf(row, col int) int {
	for _, s := range z.sections {
		if col >= s.ColStart && col < s.ColEnd &&
			row >= s.RowStart && row < s.RowEnd {
			return s.Owner
		}
	}
	return 0
}

// BuildCellsForOwner 返回指定 owner 区域内的所有可建造格子。
// grid 值: 2=CellBuildable
func (z *CoopZone) BuildCellsForOwner(grid [][]int, owner int) []GridCell {
	var cells []GridCell
	for r, row := range grid {
		for c, v := range row {
			if v == 2 && z.OwnerOf(r, c) == owner {
				cells = append(cells, GridCell{Row: r, Col: c})
			}
		}
	}
	return cells
}

// Sections 返回所有分区。
func (z *CoopZone) Sections() []ZoneSection { return z.sections }

// PlayerCount 返回总玩家数（最大 Owner + 1）。
func (z *CoopZone) PlayerCount() int {
	maxOwner := 0
	for _, s := range z.sections {
		if s.Owner > maxOwner {
			maxOwner = s.Owner
		}
	}
	return maxOwner + 1
}

// SectionBoundaries 返回所有分区的竖直分界线 X 坐标（用于渲染虚线）。
func (z *CoopZone) SectionBoundaries(cellSize int) []float64 {
	seen := make(map[int]bool)
	var xs []float64
	for _, s := range z.sections {
		if s.ColStart > 0 && !seen[s.ColStart] {
			seen[s.ColStart] = true
			xs = append(xs, float64(s.ColStart*cellSize))
		}
		if s.ColEnd < z.cols && !seen[s.ColEnd] {
			seen[s.ColEnd] = true
			xs = append(xs, float64(s.ColEnd*cellSize))
		}
	}
	return xs
}
