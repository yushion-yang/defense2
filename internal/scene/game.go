// game.go — 顶层游戏管理器。
// 实现 ebiten.Game 接口，管理场景切换（带淡入淡出过渡）。
package scene

import (
	"image/color"
	"log"
	"strconv"

	gameAudio "defense2/internal/audio"
	"defense2/internal/config"
	"defense2/internal/core/event"
	"defense2/internal/core/game"
	"defense2/internal/core/tower/abilities"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/postprocess"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// transitionState 场景过渡状态。
type transitionState int

const (
	transIdle    transitionState = iota // 无过渡
	transFadeOut                        // 当前场景淡出到黑色
	transFadeIn                         // 新场景从黑色淡入
)

// transSpeed 过渡速度：alpha 每帧变化量（0.3s = 18 帧 @ 60fps）。
const transSpeed = 1.0 / 18.0

// Game 顶层游戏对象，实现 ebiten.Game 接口。
type Game struct {
	current  Scene              // 当前活跃场景
	next     Scene              // 待切换的下一个场景（下帧生效，无过渡时用）
	width    int                // 逻辑宽度
	height   int                // 逻辑高度
	audioMgr *gameAudio.Manager // 全局音效管理器（跨场景复用）
	bus      *event.Bus         // 全局事件总线（跨场景复用）

	// 场景过渡
	transState  transitionState // 当前过渡状态
	transAlpha  float64         // 0.0（透明）→ 1.0（全黑）
	pendingNext Scene           // 淡出完成后切换到的场景
}

// HeadlessMode 自动对局模式开关: 跳过音效加载 + 直接切场景 + turbo tick。
// 在 ebiten.RunGame 之前由 cmd/autoplay 设置。
var HeadlessMode bool

// turboTicksPerFrame turbo 模式下每帧推进的 tick 数上限。
// 遇到截图请求时提前 break 让 Draw 渲染一帧。
const turboTicksPerFrame = 5000

// NewGame 创建游戏实例，初始场景为标题画面。
func NewGame() *Game {
	// 字体/图标/着色器始终加载（截图渲染需要）
	initFont()
	render.InitGlobalIcons(config.GetAssetFS())
	abilities.InitConfigAbilities()
	if err := postprocess.InitShaders(); err != nil {
		log.Printf("后处理着色器编译失败（bloom 禁用）: %v", err)
	}
	// 音效: HeadlessMode 跳过加载（自动对局不需要声音）
	var am *gameAudio.Manager
	if !HeadlessMode {
		am = initAudio()
	} else {
		am = gameAudio.NewManager()
	}
	g := &Game{
		width:    game.ScreenWidth,
		height:   game.ScreenHeight,
		audioMgr: am,
		bus:      event.NewBus(),
	}
	if !HeadlessMode {
		g.current = NewSelectScene(g)
	}
	return g
}

// AudioManager 返回全局音效管理器（实现 Switcher 接口）。
func (g *Game) AudioManager() *gameAudio.Manager {
	return g.audioMgr
}

// EventBus 返回全局事件总线（实现 Switcher 接口）。
func (g *Game) EventBus() *event.Bus {
	return g.bus
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

// SwitchScene 触发带淡入淡出过渡的场景切换。
// 如果过渡已在进行中，新请求被忽略。
func (g *Game) SwitchScene(next Scene) {
	if HeadlessMode {
		g.current = next // 无头模式: 直接切换，不走过渡动画
		return
	}
	if g.transState != transIdle {
		return // 过渡进行中，忽略
	}
	g.pendingNext = next
	g.transState = transFadeOut
	g.transAlpha = 0
}

// Update 每帧逻辑更新：处理过渡动画 + 更新当前场景。
func (g *Game) Update() error {
	// HeadlessMode turbo: 每帧跑数千 tick，截图时 break 让 Draw 渲染
	if HeadlessMode && g.current != nil {
		for range turboTicksPerFrame {
			if err := g.current.Update(); err != nil {
				return err
			}
			// 有截图请求 → break 让 Draw 渲染一帧
			type screenshotChecker interface{ HasPendingScreenshot() bool }
			if c, ok := g.current.(screenshotChecker); ok && c.HasPendingScreenshot() {
				break
			}
		}
		return nil
	}

	// 兼容旧的 next 直接切换（无过渡）
	if g.next != nil {
		g.current = g.next
		g.next = nil
	}

	switch g.transState {
	case transFadeOut:
		g.transAlpha += transSpeed
		if g.transAlpha >= 1.0 {
			g.transAlpha = 1.0
			// 淡出完成：清除旧场景订阅，切换场景，开始淡入
			g.bus.Clear()
			g.current = g.pendingNext
			g.pendingNext = nil
			g.transState = transFadeIn
		}
	case transFadeIn:
		g.transAlpha -= transSpeed
		if g.transAlpha <= 0 {
			g.transAlpha = 0
			g.transState = transIdle
		}
	}

	return g.current.Update()
}

// Draw 每帧渲染：委托给当前场景，叠加过渡遮罩。
func (g *Game) Draw(screen *ebiten.Image) {
	g.current.Draw(screen)

	// 过渡遮罩（全屏半透明黑色）
	if g.transAlpha > 0 {
		a := uint8(g.transAlpha * 255)
		w := float32(screen.Bounds().Dx())
		h := float32(screen.Bounds().Dy())
		draw.FilledRect(screen, 0, 0, w, h, color.RGBA{0, 0, 0, a}, false)
	}

	// 常驻 FPS 显示（左下角）
	if fm := render.GlobalFont(); fm != nil {
		fps := int(ebiten.ActualFPS() + 0.5)
		var buf [16]byte
		b := buf[:0]
		b = append(b, "FPS: "...)
		b = strconv.AppendInt(b, int64(fps), 10)
		fm.DrawText(screen, string(b), 4, float64(game.ScreenHeight)-4-float64(theme.FontCaption), theme.FontCaption, theme.TextMuted)
	}
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
