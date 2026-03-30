// strategy.go — AutoPlay 核心类型与策略接口定义。
// 所有策略基于 GameState 快照做出决策，不直接接触游戏内部状态。
package autoplay

import (
	"fmt"
	"time"
)

// GameState 游戏状态快照，由 Controller 每帧构建。
type GameState struct {
	Tick         int
	Gold         int
	Lives        int
	Wave         int
	MaxWaves     int
	WaveActive   bool
	GameOver     bool
	Victory      bool
	WardenReady  bool
	InteractMode int // scene.interactMode 的值
	Enemies      []EnemyInfo
	Towers       []TowerInfo
	BuildCells   []Cell
	TowerDefs    []TowerDefInfo
}

// EnemyInfo 敌人快照。
type EnemyInfo struct {
	ID        int
	X, Y      float64
	HP, MaxHP float64
	Speed     float64
	Archetype string
	Boss      bool
	Active    bool
	Dying     bool
}

// TowerInfo 已建塔快照。
type TowerInfo struct {
	Key      string
	Row, Col int
	X, Y     float64
	Damage   float64
	Range    float64
	Cost     int
	Strength int
}

// TowerDefInfo 可用塔类型定义。
type TowerDefInfo struct {
	Key    string
	Cost   int
	Range  float64
	Damage float64
	Index  int // 在 towerDefs 列表中的索引
}

// Cell 可用建造位置。
type Cell struct {
	Row, Col int
	X, Y     float64 // 像素中心坐标
}

// ActionType 自动操作类型。
type ActionType int

const (
	ActionBuild        ActionType = iota // 建造塔
	ActionUpgrade                        // 升级塔
	ActionSell                           // 出售塔
	ActionStartWave                      // 开始下一波
	ActionSelectWarden                   // 选择战灵
	ActionChooseEvent                    // 选择事件
	ActionNoop                           // 空操作
)

// Action 自动操作指令。
type Action struct {
	Type       ActionType
	TowerKey   string // Build: 要建造的塔类型
	Cell       Cell   // Build: 建造位置
	Row, Col   int    // Upgrade/Sell: 目标塔网格坐标
	WardenKey  string // SelectWarden: 战灵类型
	EventIndex int    // ChooseEvent: 事件选项索引
}

// Strategy 自动对局策略接口。
type Strategy interface {
	// Name 返回策略名称。
	Name() string
	// Init 在游戏开始时调用一次，可做预计算。
	Init(state *GameState)
	// Decide 每帧调用，返回 0~N 个操作指令。
	Decide(state *GameState) []Action
}

// FormatSessionID 生成格式化的会话 ID。
func FormatSessionID(strategy, mapID, difficulty string, run int) string {
	date := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s_%s_%s_%s_%03d", date, strategy, mapID, difficulty, run)
}
