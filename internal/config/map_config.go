// map_config.go — 地图配置数据结构。
// 定义从 JSON 加载的地图配置，包括网格布局、路径点、多入口信息。
package config

// MapConfig 地图配置，从 JSON 文件反序列化。
type MapConfig struct {
	ID         string     `json:"id"`                   // 地图唯一标识（如 "map_01"）
	Name       string     `json:"name"`                 // 地图显示名称
	Cols       int        `json:"cols"`                 // 网格列数
	Rows       int        `json:"rows"`                 // 网格行数
	CellSize   int        `json:"cellSize"`             // 单元格边长（像素）
	Theme      string     `json:"theme"`                // 主题风格
	Waves      int        `json:"waves"`                // 总波次数
	Difficulty string     `json:"difficulty"`            // 难度标签
	Grid       [][]int    `json:"grid"`                 // 二维网格，值为 CellXxx 常量
	PathOrder  [][2]int   `json:"pathOrder"`            // 敌人行进路径（[row, col] 序列）
	Entries    []MapEntry `json:"entries,omitempty"`     // 多入口配置（可选）
	PathOrders map[string][][2]int `json:"pathOrders,omitempty"` // 多路径配置（可选，按入口 ID 索引）
}

// MapEntry 出怪入口，用于多路径地图。
type MapEntry struct {
	ID     string  `json:"id"`     // 入口唯一标识
	Cell   [2]int  `json:"cell"`   // 入口所在格子 [row, col]
	Weight float64 `json:"weight"` // 出怪权重（越大越频繁）
}

// 网格单元格类型常量。
const (
	CellEmpty    = 0 // 空地（不可通行、不可建造）
	CellPath     = 1 // 敌人行进路径
	CellBuildable = 2 // 可建造塔的位置
	CellSpawn    = 4 // 出怪点
	CellBase     = 5 // 基地（敌人终点）
	CellHeroBase = 6 // 英雄放置点
)
