// mode.go — 游戏模式框架。
// 定义游戏模式接口和注册表，支持战役、Boss Rush、无尽等模式切换。
// 新模式只需实现 Mode 接口并注册，无需修改主循环。
package gamemode

// Mode 游戏模式接口。
type Mode interface {
	// Name 返回模式标识（用于注册和序列化）。
	Name() string
	// OnInit 模式初始化（关卡开始时调用）。
	OnInit(ctx *ModeContext)
	// OnWaveStart 波次开始时回调。
	OnWaveStart(wave int, ctx *ModeContext)
	// OnWaveEnd 波次结束时回调。
	OnWaveEnd(wave int, ctx *ModeContext)
	// OnUpdate 每帧更新（用于模式特有逻辑）。
	OnUpdate(dt float64, ctx *ModeContext)
	// IsWon 判断是否满足胜利条件。
	IsWon(ctx *ModeContext) bool
	// IsLost 判断是否满足失败条件。
	IsLost(ctx *ModeContext) bool
}

// ModeContext 模式回调时传入的游戏状态。
type ModeContext struct {
	Wave     int // 当前波次
	MaxWaves int // 总波次数
	Lives    int // 剩余生命
	Kills    int // 累计击杀
	Gold     int // 当前金币
}

// 全局模式注册表。
var registry = map[string]Mode{}

// Register 注册一种游戏模式。
func Register(m Mode) {
	registry[m.Name()] = m
}

// Get 获取指定名称的模式，未找到返回 nil。
func Get(name string) Mode {
	return registry[name]
}

// List 返回所有已注册模式的名称。
func List() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
