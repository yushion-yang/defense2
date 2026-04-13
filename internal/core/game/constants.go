// constants.go — 全局常量。
// 定义屏幕分辨率、对象池容量、帧率等核心常量。
package game

// Version 游戏版本号（集中管理，各场景引用此常量）。
const Version = "v0.1.0"

// 屏幕逻辑分辨率（横屏塔防布局）
const (
	ScreenWidth  = 1200
	ScreenHeight = 540
)

// 设计分辨率（用于缩放参考）
const (
	DesignWidth  = 1200
	DesignHeight = 540
)

// 对象池容量上限
const (
	MaxTowers      = 64   // 最大塔数
	MaxEnemies     = 256  // 最大敌人数
	MaxProjectiles = 1024 // 最大弹射物数
)

// 帧率
const (
	TargetTPS = 60 // 目标每秒逻辑帧数
)

// devModeStr 通过 ldflags 注入：go build -ldflags="-X 'defense2/internal/core/game.devModeStr=false'"
// 默认 "true"，release 构建设为 "false"。
var devModeStr = "true"

// DevMode 开发/测试模式开关。
// 开启后：Select 界面显示"测试"入口，可进入 TestSelect 场景。
// 发布时通过 ldflags 设为 false 隐藏测试功能。
var DevMode = true

func init() {
	DevMode = devModeStr != "false"
}
