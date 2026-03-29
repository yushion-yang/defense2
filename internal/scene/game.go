// game.go — 顶层游戏管理器。
// 实现 ebiten.Game 接口，管理场景切换（排队到下一帧生效）。
package scene

import (
	"log"

	gameAudio "defense2/internal/audio"
	"defense2/internal/config"
	"defense2/internal/core/game"
	"defense2/internal/core/tower/abilities"
	"defense2/internal/render"
	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// Game 顶层游戏对象，实现 ebiten.Game 接口。
type Game struct {
	current  Scene              // 当前活跃场景
	next     Scene              // 待切换的下一个场景（下帧生效）
	width    int                // 逻辑宽度
	height   int                // 逻辑高度
	audioMgr *gameAudio.Manager // 全局音效管理器（跨场景复用）
}

// NewGame 创建游戏实例，初始场景为标题画面。
func NewGame() *Game {
	initFont()
	render.InitGlobalIcons(config.GetAssetFS())
	abilities.InitConfigAbilities() // 从 abilities.json 注册数据驱动能力
	g := &Game{
		width:    game.ScreenWidth,
		height:   game.ScreenHeight,
		audioMgr: initAudio(),
	}
	g.current = NewSelectScene(g)
	return g
}

// AudioManager 返回全局音效管理器（实现 Switcher 接口）。
func (g *Game) AudioManager() *gameAudio.Manager {
	return g.audioMgr
}

// initFont 初始化全局双字体管理器（JetBrains Mono + Noto Sans SC 回退）。
func initFont() {
	assetFS := config.GetAssetFS()
	if assetFS == nil {
		return
	}
	fontData, err := assetFS.ReadFile("assets/fonts/NotoSansSC-Medium.otf")
	if err != nil {
		log.Printf("字体加载失败: %v", err)
		return
	}
	if err := render.InitGlobalFont(fontData); err != nil {
		log.Printf("全局字体初始化失败: %v", err)
	}
}

// initAudio 创建音效管理器并从嵌入式文件系统预加载所有 WAV。
func initAudio() *gameAudio.Manager {
	mgr := gameAudio.NewManager()
	assetFS := config.GetAssetFS()
	if assetFS != nil {
		mgr.LoadAllFromFS(assetFS)
	}
	return mgr
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

// Layout 返回逻辑分辨率（仅在 LayoutF 不可用时调用）。
func (g *Game) Layout(_, _ int) (int, int) {
	return g.width, g.height
}

// LayoutF 返回原生物理分辨率，实现 HiDPI 清晰渲染。
// draw 包内部自动处理所有坐标缩放（等同于浏览器 Canvas 的行为）。
func (g *Game) LayoutF(_, _ float64) (float64, float64) {
	draw.Scale = ebiten.Monitor().DeviceScaleFactor()
	return float64(g.width) * draw.Scale, float64(g.height) * draw.Scale
}
