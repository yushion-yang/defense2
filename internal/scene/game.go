// game.go — 顶层游戏管理器。
// 实现 ebiten.Game 接口，管理场景切换（排队到下一帧生效）。
package scene

import (
	"defense2/internal/core/game"

	"github.com/hajimehoshi/ebiten/v2"
)

// Game 顶层游戏对象，实现 ebiten.Game 接口。
type Game struct {
	current Scene // 当前活跃场景
	next    Scene // 待切换的下一个场景（下帧生效）
	width   int   // 逻辑宽度
	height  int   // 逻辑高度
}

// NewGame 创建游戏实例，初始场景为标题画面。
func NewGame() *Game {
	g := &Game{
		width:  game.ScreenWidth,
		height: game.ScreenHeight,
	}
	g.current = NewTitleScene(g)
	return g
}

// SwitchScene 排队场景切换（下一帧生效，避免帧内切换导致状态不一致）。
func (g *Game) SwitchScene(next Scene) {
	g.next = next
}

// Update 每帧逻辑更新：先处理场景切换，再更新当前场景。
func (g *Game) Update() error {
	if g.next != nil {
		g.current = g.next
		g.next = nil
	}
	return g.current.Update()
}

// Draw 每帧渲染：委托给当前场景。
func (g *Game) Draw(screen *ebiten.Image) {
	g.current.Draw(screen)
}

// Layout 返回逻辑分辨率（1200x540）。
func (g *Game) Layout(_, _ int) (int, int) {
	return g.width, g.height
}
