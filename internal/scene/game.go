// game.go — 顶层游戏管理器，实现 Ebitengine 的 ebiten.Game 接口。
//
// 核心职责：
//  1. Ebitengine 生命周期: 实现 Update()（逻辑 60TPS）、Draw()（渲染按刷新率）、LayoutF()（HiDPI）
//  2. 场景管理: 持有当前场景，通过 SwitchScene() 执行带淡入淡出过渡的场景切换
//  3. 全局资源: 持有 audioMgr（音效管理器）和 bus（事件总线），通过 Switcher 接口注入各场景
//  4. 吉祥物向导: 每帧更新 Guide 状态机，路由点击和动作到当前场景
//  5. HeadlessMode: 为 autoplay 自动对局提供无 UI 的 turbo 模式（每帧跑数千 tick）
//
// 渲染分层（Draw 中从下到上）：
//
//	场景内容 → 吉祥物覆盖层 → 过渡遮罩（全屏半透明黑色）→ FPS 显示
package scene

import (
	"fmt"
	"image/color"
	"log"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"time"

	gameAudio "defense2/internal/audio"
	"defense2/internal/config"
	"defense2/internal/core/event"
	"defense2/internal/core/game"
	"defense2/internal/core/mascot"
	"defense2/internal/core/tower/abilities"
	"defense2/internal/core/tower/descriptor"
	"defense2/internal/i18n"
	"defense2/internal/render"
	"defense2/internal/render/anim"
	"defense2/internal/render/draw"
	"defense2/internal/render/hud"
	"defense2/internal/render/postprocess"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ─── 场景过渡系统 ───────────────────────────────────────────────────
// 过渡流程: transIdle → transFadeOut（当前场景渐暗） → transFadeIn（新场景渐亮） → transIdle
// 整个过渡约 0.6s（淡出 0.3s + 淡入 0.3s）。过渡期间新的 SwitchScene 请求被忽略。
// 淡出完成时执行 bus.Clear() 清理旧场景的事件订阅，防止跨场景事件泄漏。

// transitionState 场景过渡状态。
type transitionState int

const (
	transIdle    transitionState = iota // 无过渡，正常运行
	transFadeOut                        // 淡出阶段：当前场景逐渐变暗至全黑
	transFadeIn                         // 淡入阶段：新场景从全黑逐渐变亮
)

// transSpeed 过渡速度：alpha 每帧变化量。
// 1.0/18.0 ≈ 0.056，即 18 帧（0.3s @ 60fps）完成一个方向的渐变。
const transSpeed = 1.0 / 18.0

// Game 顶层游戏对象，实现 ebiten.Game 接口。
// 生命周期: NewGame() 创建 → ebiten.RunGame(g) 启动主循环 → 每帧调用 Update/Draw/LayoutF。
// Game 同时实现 Switcher 接口，各场景通过构造函数接收 *Game 来访问全局资源和请求场景切换。
type Game struct {
	// ── 场景管理 ──
	current Scene // 当前活跃场景（Update/Draw 委托目标）
	next    Scene // 待切换的下一个场景（无过渡的直接切换，兼容旧代码路径）
	width   int   // 逻辑宽度（固定 1200）
	height  int   // 逻辑高度（固定 540）

	// ── 全局资源（跨场景共享） ──
	audioMgr *gameAudio.Manager // 音效管理器：持有所有 WAV 的解码缓存，避免每个场景重复加载
	bus      *event.Bus         // 事件总线：场景间低频通信（成就/解锁/统计），切换时 Clear

	// ── 场景过渡动画 ──
	transState  transitionState // 当前过渡阶段（idle/fadeOut/fadeIn）
	transAlpha  float64         // 遮罩透明度，0.0=完全透明，1.0=完全黑色
	pendingNext Scene           // 淡出完成后要切换到的新场景

	// ── 吉祥物向导系统 ──
	// 吉祥物是全局 UI 覆盖层，显示在场景之上、过渡遮罩之下。
	// Guide 是对话状态机，根据场景名称和游戏状态触发条件对话。
	mascot        *mascot.Guide  // 对话状态机（nil 表示未加载吉祥物资源）
	mascotAnim    *anim.Animator // 表情帧动画播放器
	mascotTime    float64        // 累计时间（秒），用于 idle 浮动动画
	prevSceneName string         // 上一帧的场景名，避免每帧重复调用 SetScene
	sessionStart  time.Time      // 游戏启动时间戳，用于计算 SessionSecs（吉祥物条件判断用）

	// ── 全局调试工具 ──
	screenshotPending bool // F12 截图请求标志（全局可用，不限场景）
}

// HeadlessMode 自动对局（autoplay）模式开关。
// 由 cmd/autoplay/main.go 在 ebiten.RunGame 之前设置为 true。
// 开启后的行为差异：
//   - NewGame: 同步加载所有资源（无 LoadingScene 分帧加载）
//   - SwitchScene: 直接切换，跳过淡入淡出过渡动画
//   - Update: 每帧执行 turboTicksPerFrame 次 tick（极速推进游戏逻辑）
//   - 跳过吉祥物、音效、悬浮追踪等仅 UI 相关的逻辑
var HeadlessMode bool

// ── 功能开关（默认关闭，发布前隐藏未完成/不想暴露的内容） ──

// MascotEnabled 控制萌妹向导系统。关闭时不加载资源、不渲染、不响应交互。
var MascotEnabled = true

// WardenEnabled 控制战灵系统。关闭时跳过战灵选择，直接无战灵开波。
var WardenEnabled = true

// turboTicksPerFrame HeadlessMode 下每帧推进的 tick 数上限。
// 5000 tick ≈ 83 秒游戏时间（@ 60 TPS），配合 Ebitengine 的帧循环可在数秒内跑完一局。
const turboTicksPerFrame = 5000

// NewGame 创建游戏实例。
//
// 初始化策略分两种模式：
//   - 正常模式：仅同步初始化字体（LoadingScene 需要画文字），其余资源
//     （图标、能力表、着色器、音效等）由 LoadingScene 分帧异步加载，避免启动黑屏。
//   - HeadlessMode：同步加载全部资源（autoplay 不需要 UI 反馈，追求确定性）。
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
		// 描述符能力覆盖 ConfigAbility（失败时 ConfigAbility 仍作为 fallback）
		if err := descriptor.InitDescriptorAbilities(config.GetDataFS()); err != nil {
			log.Printf("[descriptor] warning: %v, falling back to ConfigAbility", err)
		}
		config.LoadBalance()
		config.LoadPlatform()
		config.LoadTierPresets()
		config.LoadClassicPresets()
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

// SwitchScene 触发带淡入淡出过渡的场景切换（实现 Switcher 接口）。
// 流程: 记录 pendingNext → 进入 transFadeOut → 淡出完成时替换 current → 进入 transFadeIn。
// 安全保证: 过渡进行中（transState != transIdle）时忽略新请求，防止快速连续点击导致状态混乱。
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

// Update 每帧逻辑更新（Ebitengine 以固定 60 TPS 调用）。
//
// 执行顺序：
//  1. HeadlessMode turbo 快进（如果开启，直接跑数千 tick 后返回）
//  2. 长按悬浮追踪器更新（触摸设备上 500ms 长按 = 鼠标悬浮）
//  3. 旧式直接切换（兼容 g.next 赋值路径）
//  4. 过渡动画状态机推进（fadeOut → 切换场景 → fadeIn）
//  5. 安全包装的场景 Update（panic 不会崩溃，会触发吉祥物彩蛋）
//  6. 吉祥物向导更新（状态机 tick + 场景感知 + 点击处理）
func (g *Game) Update() error {
	// HeadlessMode turbo: 每帧跑数千 tick，快速推进游戏逻辑
	if HeadlessMode && g.current != nil {
		for range turboTicksPerFrame {
			if err := g.current.Update(); err != nil {
				return err
			}
		}
		return nil
	}

	// 每帧重置点击消费标记（供 tapConsumed 机制使用）
	resetTapConsumed()

	// 更新全局长按悬浮追踪器（触摸设备上长按=悬浮）
	draw.TickHover()

	// F12 全局截图（调试功能，任意场景可用）
	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		g.screenshotPending = true
	}

	// 兼容旧的 next 直接切换（无过渡）
	if g.next != nil {
		g.current = g.next
		g.next = nil
	}

	// 过渡动画状态机：通过 transAlpha 控制全屏黑色遮罩的透明度
	switch g.transState {
	case transFadeOut:
		// 淡出阶段：遮罩逐渐变暗
		g.transAlpha += transSpeed
		if g.transAlpha >= 1.0 {
			g.transAlpha = 1.0
			// 淡出完成（全黑）：这是安全切换场景的时机
			// bus.Clear() 清除旧场景的所有事件订阅，防止已销毁场景的回调被触发
			g.bus.Clear()
			g.current = g.pendingNext
			g.pendingNext = nil
			g.transState = transFadeIn
		}
	case transFadeIn:
		// 淡入阶段：遮罩逐渐变透明，新场景逐渐可见
		g.transAlpha -= transSpeed
		if g.transAlpha <= 0 {
			g.transAlpha = 0
			g.transState = transIdle // 过渡完成，恢复正常
		}
	}

	// ── 吉祥物点击检测（必须在 safeSceneUpdate 之前，防止点击穿透到场景） ──
	if MascotEnabled && g.mascot != nil && isTapJustPressed() {
		mx, my := draw.CursorPos()
		if hud.MascotHitTest(mx, my) {
			if g.mascot.AbilityReady() {
				// 技能就绪时点击 → 触发技能（不管提示气泡是否显示）
				g.mascot.RequestHelp()
			} else if g.mascot.HasActiveDialog() {
				g.mascot.ClickAdvance()
			} else {
				g.mascot.Trigger("mascot_tap")
			}
			consumeTap() // 标记点击已消费，场景的 isTapJustPressed 将返回 false
		}
	}

	err := g.safeSceneUpdate()

	// ── 吉祥物向导更新 ──
	// （HeadlessMode 已在上方 turbo 路径返回，此处必为正常模式）
	// 吉祥物向导是全局 UI 覆盖层，独立于场景运行，但能感知当前场景状态。
	if MascotEnabled && g.mascot != nil {
		const dt = 1.0 / 60.0 // 固定 60 TPS 的帧间隔
		g.mascotTime += dt

		// 仅在场景变化时通知 Guide（避免每帧重复触发 scene_enter 事件）
		if name := g.currentSceneName(); name != g.prevSceneName {
			g.mascot.SetScene(name)
			g.prevSceneName = name
		}
		g.mascot.Tick(dt) // 推进对话状态机（超时关闭、冷却计时等）

		// 更新吉祥物表情帧动画（idle/happy/surprise 等）
		if g.mascotAnim != nil {
			g.mascotAnim.Update(dt)
		}

		// 构建游戏上下文供吉祥物条件判断（如"波次 > 5 时触发提示"）
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

		// 将吉祥物的待处理动作路由到当前场景执行
		if action := g.mascot.ConsumeAction(); action != nil {
			if executor, ok := g.current.(MascotActionExecutor); ok {
				if executor.ExecuteMascotAction(action) {
					g.mascot.NotifyActionComplete(action.Type)
				}
			}
		}
	}

	return err
}

// Draw 每帧渲染（Ebitengine 按显示器刷新率调用，可能与 Update 不同频率）。
//
// 渲染分层（从下到上）：
//  1. 场景内容（safeSceneDraw，panic 安全）
//  2. 吉祥物覆盖层（对话气泡 + 角色精灵）
//  3. 过渡遮罩（全屏半透明黑色，alpha 由 transAlpha 控制）
//  4. FPS 显示（左下角常驻，零分配 strconv）
func (g *Game) Draw(screen *ebiten.Image) {
	g.safeSceneDraw(screen)

	// 吉祥物覆盖层（场景之上、过渡遮罩之下）
	if MascotEnabled && g.mascot != nil {
		coreVM := g.mascot.VM()
		overlayVM := hud.MascotOverlayVM{
			Visible:         coreVM.Visible,
			HasDialog:       coreVM.HasDialog,
			Text:            coreVM.Text,
			Expression:      coreVM.Expression,
			CanClick:        coreVM.CanClick,
			AnimTime:        g.mascotTime,
			AbilityReady:    coreVM.AbilityReady,
			AbilityHintText: coreVM.AbilityHintText,
			CooldownPct:     coreVM.CooldownPct,
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

	// F12 截图：全部渲染完成后读取像素并异步保存
	if g.screenshotPending {
		g.screenshotPending = false
		if img := readScreenPixels(screen); img != nil {
			fname := filepath.Join("docs", "bug", "pic",
				fmt.Sprintf("screenshot_%s.png", time.Now().Format("20060102_150405")))
			saveImageAsync(img, fname)
			hud.ShowToast(i18n.T("game.stage.screenshot_saved"))
			log.Printf("screenshot saved: %s", fname)
		}
	}
}

// Layout 返回逻辑分辨率（Ebitengine 在 LayoutF 不可用时的回退调用）。
// 本项目优先使用 LayoutF，此方法仅为满足接口要求。
func (g *Game) Layout(_, _ int) (int, int) {
	return g.width, g.height
}

// LayoutF 返回物理分辨率，实现 HiDPI 清晰渲染。
//
// Ebitengine 每帧调用此方法来确定内部渲染目标的大小。
// 本项目策略：返回 逻辑分辨率 × 设备缩放因子 = 物理分辨率。
// 例如 Retina 2x 显示器上：1200×540 × 2 = 2400×1080 物理像素。
//
// draw.Scale 是全局缩放因子，draw 包的所有绘制函数内部乘以 Scale 将逻辑坐标映射到物理坐标。
// 对上层代码完全透明——所有业务代码只使用逻辑坐标（1200×540），无需关心 HiDPI。
func (g *Game) LayoutF(_, _ float64) (float64, float64) {
	draw.Scale = ebiten.Monitor().DeviceScaleFactor()
	return float64(g.width) * draw.Scale, float64(g.height) * draw.Scale
}

// safeSceneUpdate 安全包装场景 Update，捕获 panic 防止游戏崩溃。
//
// 设计意图：游戏场景中的 bug 不应导致整个程序退出。
// panic 被捕获后：记录堆栈日志 + 触发吉祥物 "panic_recover" 彩蛋 + 返回 nil 继续运行。
// 注意：使用闭包 + defer 是因为 defer 只能在当前函数栈帧中 recover。
func (g *Game) safeSceneUpdate() error {
	var sceneErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[panic-recover] scene Update panic: %v\n%s", r, debug.Stack())
				if MascotEnabled && g.mascot != nil {
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
			if MascotEnabled && g.mascot != nil {
				g.mascot.ForceTrigger("panic_recover")
			}
		}
	}()
	g.current.Draw(screen)
}

// currentSceneName 返回当前场景的字符串标识。
// 吉祥物向导系统通过此名称匹配对话脚本中的 scene_enter 条件。
// 新增场景时需同步在此添加 case，否则吉祥物无法感知该场景。
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
	case *BlueprintEditScene:
		return "blueprint_edit"
	case *AbilityEditScene:
		return "ability_edit"
	default:
		return "other"
	}
}
