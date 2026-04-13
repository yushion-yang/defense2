// game.go — 顶层游戏管理器。
// 实现 ebiten.Game 接口，管理场景切换（带淡入淡出过渡）。
package scene

import (
	"image/color"
	"log"
	"runtime/debug"
	"strconv"
	"time"

	gameAudio "defense2/internal/audio"
	"defense2/internal/config"
	"defense2/internal/core/event"
	"defense2/internal/core/game"
	"defense2/internal/core/mascot"
	"defense2/internal/core/tower/abilities"
	"defense2/internal/render"
	"defense2/internal/render/anim"
	"defense2/internal/render/draw"
	"defense2/internal/render/hud"
	"defense2/internal/render/postprocess"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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

	// 吉祥物向导系统
	mascot        *mascot.Guide  // 对话状态机
	mascotAnim    *anim.Animator // 表情帧动画
	mascotTime    float64        // 全局动画计时（idle bob 用）
	prevSceneName string         // 上一帧场景名（避免每帧重复 SetScene）
	sessionStart  time.Time      // 游戏启动时间（吉祥物 SessionSecs 用）
}

// HeadlessMode 自动对局模式开关: 跳过音效加载 + 直接切场景 + turbo tick。
// 在 ebiten.RunGame 之前由 cmd/autoplay 设置。
var HeadlessMode bool

// turboTicksPerFrame turbo 模式下每帧推进的 tick 数上限。
const turboTicksPerFrame = 5000

// NewGame 创建游戏实例。
// 正常模式：仅初始化字体，其余资源由 LoadingScene 分帧加载。
// HeadlessMode：同步加载所有资源（无 UI）。
func NewGame() *Game {
	// 字体必须同步初始化（LoadingScene 需要画文字）
	initFont()

	g := &Game{
		width:        game.ScreenWidth,
		height:       game.ScreenHeight,
		bus:          event.NewBus(),
		sessionStart: time.Now(),
	}

	if !HeadlessMode {
		// 正常模式：进入加载场景，由 LoadingScene 分帧完成剩余初始化
		g.current = NewLoadingScene(g)
	} else {
		// HeadlessMode：同步加载全部资源（autoplay 不需要 UI）
		render.InitGlobalIcons(config.GetAssetFS())
		abilities.InitConfigAbilities()
		config.LoadBalance()
		config.LoadTierPresets()
		config.LoadAndCacheWardenConfigs()
		config.LoadBuffRules()
		config.LoadSpawnerConfig()
		if err := postprocess.InitShaders(); err != nil {
			log.Printf("后处理着色器编译失败（bloom 禁用）: %v", err)
		}
		g.audioMgr = gameAudio.NewManager()
	}

	return g
}

// NewGameLite 创建轻量游戏实例，跳过全局资源重复初始化。
// 适用于 autoplay 批量运行：字体/图标/能力表/着色器已由首次 NewGame 初始化。
func NewGameLite() *Game {
	var am *gameAudio.Manager
	if !HeadlessMode {
		am = initAudio()
	} else {
		am = gameAudio.NewManager()
	}
	return &Game{
		audioMgr: am,
		bus:      event.NewBus(),
	}
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
	// HeadlessMode turbo: 每帧跑数千 tick
	if HeadlessMode && g.current != nil {
		for range turboTicksPerFrame {
			if err := g.current.Update(); err != nil {
				return err
			}
		}
		return nil
	}

	// 更新全局长按悬浮追踪器（触摸设备上长按=悬浮）
	draw.TickHover()

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

	err := g.safeSceneUpdate()

	// 吉祥物向导更新（HeadlessMode 已在上方 turbo 路径返回，此处必为正常模式）
	if g.mascot != nil {
		const dt = 1.0 / 60.0
		g.mascotTime += dt

		// 仅在场景变化时通知 Guide（避免每帧重复触发 scene_enter）
		if name := g.currentSceneName(); name != g.prevSceneName {
			g.mascot.SetScene(name)
			g.prevSceneName = name
		}
		g.mascot.Tick(dt)

		// 更新吉祥物帧动画
		if g.mascotAnim != nil {
			g.mascotAnim.Update(dt)
		}

		// Build game context for condition evaluation
		ctx := mascot.GameContext{
			SceneName:   g.currentSceneName(),
			HourOfDay:   time.Now().Hour(),
			SessionSecs: time.Since(g.sessionStart).Seconds(),
		}
		if provider, ok := g.current.(MascotSnapshotProvider); ok {
			ctx.InStage = true
			ctx.StageSnapshot = provider.MascotSnapshot()
		}
		g.mascot.UpdateContext(ctx)

		// Route pending mascot actions to the active scene
		if action := g.mascot.ConsumeAction(); action != nil {
			if executor, ok := g.current.(MascotActionExecutor); ok {
				if executor.ExecuteMascotAction(action) {
					g.mascot.NotifyActionComplete(action.Type)
				}
			}
		}

		// 处理吉祥物点击（JustPressed 单次触发，适合 click-to-advance）
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			mx, my := draw.CursorPos()
			if hud.MascotHitTest(mx, my) {
				if g.mascot.IsAbilityHintActive() {
					// Hint 气泡显示时点击 → 触发技能
					g.mascot.RequestHelp()
				} else if g.mascot.HasActiveDialog() {
					g.mascot.ClickAdvance()
				} else {
					g.mascot.Trigger("mascot_tap")
				}
			}
		}
	}

	return err
}

// Draw 每帧渲染：委托给当前场景，叠加过渡遮罩。
func (g *Game) Draw(screen *ebiten.Image) {
	g.safeSceneDraw(screen)

	// 吉祥物覆盖层（场景之上、过渡遮罩之下）
	if g.mascot != nil {
		coreVM := g.mascot.VM()
		overlayVM := hud.MascotOverlayVM{
			Visible:      coreVM.Visible,
			HasDialog:    coreVM.HasDialog,
			Text:         coreVM.Text,
			Expression:   coreVM.Expression,
			CanClick:     coreVM.CanClick,
			AnimTime:     g.mascotTime,
			AbilityReady: coreVM.AbilityReady,
			CooldownPct:  coreVM.CooldownPct,
		}
		if g.mascotAnim != nil {
			if g.mascotAnim.HasAnim(coreVM.Expression) {
				g.mascotAnim.Play(coreVM.Expression)
			}
			overlayVM.Sprite = g.mascotAnim.CurrentImage()
		}
		hud.DrawMascotOverlay(screen, overlayVM)
	}

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

// safeSceneUpdate 包装场景 Update，捕获 panic 并触发萌妹彩蛋。
// 返回 nil 让 Ebitengine 继续运行（非 nil error 会导致游戏退出）。
func (g *Game) safeSceneUpdate() error {
	var sceneErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[panic-recover] scene Update panic: %v\n%s", r, debug.Stack())
				if g.mascot != nil {
					g.mascot.ForceTrigger("panic_recover")
				}
			}
		}()
		sceneErr = g.current.Update()
	}()
	return sceneErr
}

// safeSceneDraw 包装场景 Draw，捕获 panic 并触发萌妹彩蛋。
func (g *Game) safeSceneDraw(screen *ebiten.Image) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[panic-recover] scene Draw panic: %v\n%s", r, debug.Stack())
			if g.mascot != nil {
				g.mascot.ForceTrigger("panic_recover")
			}
		}
	}()
	g.current.Draw(screen)
}

// currentSceneName 返回当前场景的字符串标识（供吉祥物向导匹配对话用）。
func (g *Game) currentSceneName() string {
	switch g.current.(type) {
	case *LoadingScene:
		return "loading"
	case *TitleScene:
		return "title"
	case *SelectScene:
		return "select"
	case *CampaignSelectScene:
		return "campaign_select"
	case *TestSelectScene:
		return "test_select"
	case *StageScene:
		return "stage"
	case *ResultScene:
		return "result"
	case *SettingsScene:
		return "settings"
	case *WardenSelectScene:
		return "warden_select"
	case *LangSelectScene:
		return "lang_select"
	case *BestiaryScene:
		return "bestiary"
	default:
		return "other"
	}
}
