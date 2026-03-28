// panel_fsm.go — 面板状态机。
// 管理互斥面板切换，保证同一时刻只有一个面板激活，切换时自动清理旧面板。
package scene

// PanelType 面板类型。
type PanelType int

const (
	PanelIdle      PanelType = iota // 无面板
	PanelBuildMenu                  // 建造菜单
	PanelBuildPlace                 // 放塔模式
	PanelTowerSel                   // 塔选中信息面板
	PanelEvent                      // 事件选择弹窗
	PanelPaused                     // 暂停菜单
	PanelChoice                     // 通用选择面板
	PanelSpawnMenu                  // 造怪菜单（测试模式）
)

// PanelFSM 面板状态机（互斥切换 + 自动清理）。
type PanelFSM struct {
	Current PanelType            // 当前激活面板
	cleanup map[PanelType]func() // 每种面板的清理函数
}

// NewPanelFSM 创建面板状态机，初始状态为 PanelIdle。
func NewPanelFSM() *PanelFSM {
	return &PanelFSM{
		Current: PanelIdle,
		cleanup: make(map[PanelType]func()),
	}
}

// RegisterCleanup 注册指定面板的清理回调。
// 当从该面板切走时自动执行清理。
func (f *PanelFSM) RegisterCleanup(panel PanelType, fn func()) {
	if fn != nil {
		f.cleanup[panel] = fn
	}
}

// Transition 切换到目标面板。
// 会先执行旧面板的清理函数，再设置新状态。
func (f *PanelFSM) Transition(to PanelType) {
	if f.Current == to {
		return
	}
	// 执行旧面板清理
	if fn, ok := f.cleanup[f.Current]; ok {
		fn()
	}
	f.Current = to
}

// CloseAll 关闭所有面板，切换回 PanelIdle。
func (f *PanelFSM) CloseAll() {
	f.Transition(PanelIdle)
}

// Is 检查当前是否为指定面板类型。
func (f *PanelFSM) Is(panel PanelType) bool {
	return f.Current == panel
}
