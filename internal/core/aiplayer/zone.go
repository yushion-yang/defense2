// zone.go — 区域划分逻辑。
//
// Phase 1: 按列号中点硬分（左=玩家，右=AI）。
// Phase 4 将改用地图 JSON zones 字段。
package aiplayer

// ZoneOwner 区域所有者。
type ZoneOwner int

const (
	ZoneHuman   ZoneOwner = 0
	ZoneAI      ZoneOwner = 1
	ZoneNeutral ZoneOwner = 2
)

// GridCell 网格单元格坐标。
type GridCell struct {
	Row, Col int
}

// Zone 区域划分器。
type Zone struct {
	cols     int
	rows     int
	SplitCol int // AI 区域起始列（导出供 AIPlayer 读取）
}

// NewZone 创建区域划分器，按列中点分割。
func NewZone(cols, rows int) *Zone {
	return &Zone{
		cols:     cols,
		rows:     rows,
		SplitCol: cols / 2,
	}
}

// Rows 返回地图行数。
func (z *Zone) Rows() int { return z.rows }

// OwnerOf 返回指定格子的所有者 ID（实现 ZoneProvider 接口）。
func (z *Zone) OwnerOf(row, col int) int {
	if col < z.SplitCol {
		return int(ZoneHuman)
	}
	return int(ZoneAI)
}

// BuildableCells 返回指定所有者区域内的所有可建造格子。
// grid 值: 2=CellBuildable
func (z *Zone) BuildableCells(grid [][]int, owner int) []GridCell {
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
