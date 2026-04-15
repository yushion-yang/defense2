// loader.go — 配置加载核心，是整个配置系统的基础层。
//
// 职责：
//  1. 管理两个 embed.FS 引用（dataFS=JSON 配置, assetFS=SVG/音频/字体等资源）
//  2. 提供地图配置（MapConfig）的加载与网格常量定义
//  3. 提供关卡列表（LevelEntry）的动态扫描
//
// 设计决策：
//   - 使用全局变量 + Set/Get 函数而非构造器注入，因为 embed.FS 在 main 包中声明，
//     而 config 包被 20+ 个包依赖，构造器注入会导致依赖穿透过深
//   - dataFS 和 assetFS 分离是因为 WASM 构建时资源文件可能走不同的加载路径
//   - 所有 Load* 函数都要求 SetDataFS() 已被调用，否则返回明确错误
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

// GetDataFS 返回配置文件系统引用。
func GetDataFS() *embed.FS {
	return dataFS
}

// SetAssetFS 注入资源文件系统。
func SetAssetFS(fs *embed.FS) {
	assetFS = fs
}

// GetAssetFS 返回资源文件系统引用。
func GetAssetFS() *embed.FS {
	return assetFS
}

// LevelEntry 关卡列表条目（从 levels/map_*.json 动态构建）。
type LevelEntry struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Waves           int    `json:"waves"`
	Difficulty      string `json:"difficulty"`
	CoopPlayerCount int    `json:"coopPlayerCount,omitempty"` // 合作模式人数（0=非合作）
}

// LoadLevelList 从 config/levels/ 目录扫描地图构建关卡列表，按模式过滤。
//
// 过滤规则：
//   - modeID == "classic": 只返回 map_cXX 前缀的经典模式专属地图
//   - modeID == "test" 或 "": 返回所有正式地图（campaign + classic）
//   - 其他（campaign 模式如 casual/hard/extreme）: 只返回 map_XX（纯数字编号）
//
// 所有模式都跳过 map_test/map_dummy 等测试地图。
// 按 ID 字母序排列（embed.FS.ReadDir 保证排序，无需额外 sort）。
func LoadLevelList(modeID string) ([]LevelEntry, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load level list: dataFS not initialized")
	}
	entries, err := dataFS.ReadDir("config/levels")
	if err != nil {
		return nil, fmt.Errorf("load level list: %w", err)
	}

	var levels []LevelEntry
	for _, entry := range entries {
		name := entry.Name()
		// 基础格式校验：至少 "map_X.json"（10 字符），前缀 map_，后缀 .json
		if len(name) < 10 || name[:4] != "map_" || name[len(name)-5:] != ".json" {
			continue
		}
		id := name[:len(name)-5] // 如 "map_01" 或 "map_c01"
		numPart := name[4 : len(name)-5]

		// 判断地图类型：经典地图(map_cXX) vs 战役地图(map_XX)
		isClassicMap := len(numPart) >= 2 && numPart[0] == 'c' && isDigits(numPart[1:])
		isCampaignMap := isDigits(numPart)
		isCoopMap := len(numPart) >= 3 && numPart[:2] == "co" && isDigits(numPart[2:])

		// 跳过非正式地图（map_test, map_dummy 等）
		if !isClassicMap && !isCampaignMap && !isCoopMap {
			continue
		}

		// 按模式过滤
		switch modeID {
		case "classic":
			if !isClassicMap {
				continue
			}
		case "coop":
			if !isCoopMap {
				continue
			}
		case "test", "":
			// 返回所有正式地图
		default:
			// campaign 模式：只返回战役地图
			if !isCampaignMap {
				continue
			}
		}

		m, err := LoadMap(id)
		if err != nil {
			continue // 跳过无法加载的文件
		}
		entry := LevelEntry{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			Waves:       m.Waves,
			Difficulty:  m.Difficulty,
		}
		if m.Coop != nil {
			entry.CoopPlayerCount = m.Coop.PlayerCount
		}
		levels = append(levels, entry)
	}

	return levels, nil
}

// isDigits 检查字符串是否全为数字且非空。
func isDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
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
	Coop        *CoopConfig         `json:"coop,omitempty"`       // 合作模式分区配置（可选）
}

// CoopConfig 合作模式地图配置。
type CoopConfig struct {
	PlayerCount int           `json:"playerCount"` // 玩家人数（2/4/6）
	Sections    []CoopSection `json:"sections"`    // 分区列表
}

// CoopSection 合作模式分区定义。
type CoopSection struct {
	ID       string `json:"id"`       // 分区标识（"A"/"B"/...）
	ColStart int    `json:"colStart"` // 列范围起始（含）
	ColEnd   int    `json:"colEnd"`   // 列范围结束（不含）
	RowStart int    `json:"rowStart"` // 行范围起始（含），0=从第0行开始
	RowEnd   int    `json:"rowEnd"`   // 行范围结束（不含），0=到最后一行
	Theme    string `json:"theme"`    // 分区主题（可与地图主题不同）
	Owner    int    `json:"owner"`    // 所有者：0=human, 1..N-1=AI
}

// MapEntry 出怪入口，用于多路径地图。
type MapEntry struct {
	ID     string  `json:"id"`     // 入口唯一标识
	Cell   [2]int  `json:"cell"`   // 入口所在格子 [row, col]
	Weight float64 `json:"weight"` // 出怪权重（越大越频繁）
}

// 网格单元格类型常量。
// 这些值与 JSON 地图文件的 grid[][] 数值一一对应，修改需同步更新所有地图文件。
// 注意：值 3 未使用（历史遗留，JS 版本中为装饰物）。
const (
	CellEmpty     = 0 // 空地（不可通行、不可建造）
	CellPath      = 1 // 敌人行进路径
	CellBuildable = 2 // 可建造塔的位置
	CellSpawn     = 4 // 出怪点（敌人从此格进入）
	CellBase      = 5 // 基地（敌人到达此格扣血）
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
