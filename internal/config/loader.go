// loader.go — 配置加载器。
// 从嵌入式文件系统读取 JSON 配置并反序列化为 Go 结构体。
package config

import (
	"embed"
	"encoding/json"
	"fmt"
)

// dataFS 持有嵌入式配置文件系统的引用。
var dataFS *embed.FS

// assetFS 持有嵌入式资源文件系统的引用（SVG、音频等）。
var assetFS *embed.FS

// SetDataFS 注入配置文件系统。
func SetDataFS(fs *embed.FS) {
	dataFS = fs
}

// SetAssetFS 注入资源文件系统。
func SetAssetFS(fs *embed.FS) {
	assetFS = fs
}

// GetAssetFS 返回资源文件系统引用。
func GetAssetFS() *embed.FS {
	return assetFS
}

// LevelEntry 关卡列表条目（来自 level-list.json）。
type LevelEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Waves       int    `json:"waves"`
	Difficulty  string `json:"difficulty"`
}

// LoadLevelList 加载关卡列表。
func LoadLevelList() ([]LevelEntry, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load level list: dataFS not initialized")
	}
	data, err := dataFS.ReadFile("config/level-list.json")
	if err != nil {
		return nil, fmt.Errorf("load level list: %w", err)
	}
	var wrap struct {
		Levels []LevelEntry `json:"levels"`
	}
	if err := json.Unmarshal(data, &wrap); err != nil {
		return nil, fmt.Errorf("parse level list: %w", err)
	}
	return wrap.Levels, nil
}

// ── 地图配置 ──────────────────────────────────────

// MapConfig 地图配置，从 JSON 文件反序列化。
type MapConfig struct {
	ID          string              `json:"id"`                   // 地图唯一标识（如 "map_01"）
	Name        string              `json:"name"`                 // 地图显示名称
	Description string              `json:"description"`          // 地图描述
	Cols        int                 `json:"cols"`                 // 网格列数
	Rows        int                 `json:"rows"`                 // 网格行数
	CellSize    int                 `json:"cellSize"`             // 单元格边长（像素）
	Theme       string              `json:"theme"`                // 主题风格
	Waves       int                 `json:"waves"`                // 总波次数
	Difficulty  string              `json:"difficulty"`           // 难度标签
	Grid        [][]int             `json:"grid"`                 // 二维网格，值为 CellXxx 常量
	PathOrder   [][2]int            `json:"pathOrder"`            // 敌人行进路径（[row, col] 序列）
	Entries     []MapEntry          `json:"entries,omitempty"`    // 多入口配置（可选）
	PathOrders  map[string][][2]int `json:"pathOrders,omitempty"` // 多路径配置（可选，按入口 ID 索引）
}

// MapEntry 出怪入口，用于多路径地图。
type MapEntry struct {
	ID     string  `json:"id"`     // 入口唯一标识
	Cell   [2]int  `json:"cell"`   // 入口所在格子 [row, col]
	Weight float64 `json:"weight"` // 出怪权重（越大越频繁）
}

// 网格单元格类型常量。
const (
	CellEmpty     = 0 // 空地（不可通行、不可建造）
	CellPath      = 1 // 敌人行进路径
	CellBuildable = 2 // 可建造塔的位置
	CellSpawn     = 4 // 出怪点
	CellBase      = 5 // 基地（敌人终点）
)

// LoadMap 按地图 ID（如 "map_01"）加载地图配置。
func LoadMap(id string) (*MapConfig, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load map %s: dataFS not initialized", id)
	}
	path := fmt.Sprintf("config/levels/%s.json", id)
	data, err := dataFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load map %s: %w", id, err)
	}
	var m MapConfig
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse map %s: %w", id, err)
	}
	return &m, nil
}
