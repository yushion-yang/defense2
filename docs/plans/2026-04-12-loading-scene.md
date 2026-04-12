# Loading Scene Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a splash/loading screen as the first scene, showing game branding + progress bar while deferring heavy init (audio/shaders/configs) into the game loop.

**Architecture:** Split `NewGame()` into two phases: (1) minimal sync init (font only) + create LoadingScene, (2) LoadingScene.Update() runs one init step per frame with progress reporting. After 100%, auto-fade to SelectScene.

**Tech Stack:** Ebitengine Scene interface, existing draw/font/theme packages, audio batch loading API.

---

### Task 1: Add batch loading API to audio Manager

**Files:**
- Modify: `internal/audio/loader.go`

**Step 1: Add `ScanWAVEntries` and `LoadWAVEntry` methods**

在 `LoadAllFromFS` 之后新增两个方法，支持分批加载：

```go
// WAVEntry 单个 WAV 文件的加载条目。
type WAVEntry struct {
	FileName string // 原始文件名 (e.g. "tower-build.wav")
	SFXName  string // camelCase 音效名 (e.g. "towerBuild")
}

// ScanWAVEntries 扫描嵌入 FS 中的 WAV 文件列表（不加载数据）。
// 同时保存 FS 引用供后续 LoadWAVEntry / BGM 使用。
func (m *Manager) ScanWAVEntries(fs *embed.FS) []WAVEntry {
	if fs == nil {
		return nil
	}
	m.assetFS = fs
	entries, err := fs.ReadDir("assets/audio")
	if err != nil {
		log.Printf("读取音效目录失败: %v", err)
		return nil
	}
	var result []WAVEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".wav") {
			continue
		}
		sfxName := strings.TrimSuffix(name, filepath.Ext(name))
		sfxName = kebabToCamel(sfxName)
		result = append(result, WAVEntry{FileName: name, SFXName: sfxName})
	}
	return result
}

// LoadWAVEntry 加载单个 WAV 条目。调用前必须先调用 ScanWAVEntries 设置 FS。
func (m *Manager) LoadWAVEntry(entry WAVEntry) error {
	data, err := m.assetFS.ReadFile("assets/audio/" + entry.FileName)
	if err != nil {
		return err
	}
	return m.LoadWAV(entry.SFXName, data)
}
```

**Step 2: Verify build**

Run: `go build ./internal/audio/...`
Expected: OK

**Step 3: Commit**

```
feat: add batch WAV loading API to audio Manager
```

---

### Task 2: Create LoadingScene

**Files:**
- Create: `internal/scene/loading.go`

**Step 1: Implement LoadingScene**

```go
// loading.go — 加载场景。
// 游戏启动首屏，显示品牌标题和加载进度条，分帧初始化重型资源。
package scene

import (
	"fmt"
	"image/color"
	"log"
	"math"

	gameAudio "defense2/internal/audio"
	"defense2/internal/config"
	"defense2/internal/core/game"
	"defense2/internal/core/mascot"
	"defense2/internal/core/tower/abilities"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/postprocess"

	"github.com/hajimehoshi/ebiten/v2"
)

// loadPhase 加载阶段枚举。
type loadPhase int

const (
	phaseConfigs    loadPhase = iota // 配置加载
	phaseShaders                     // 着色器编译
	phaseAudioScan                   // 音频扫描
	phaseAudioLoad                   // 音频解码（多帧）
	phaseMascot                      // 吉祥物
	phaseSettings                    // 设置
	phaseFinalize                    // 创建首场景
	phaseDone                        // 完成
)

// audioLoadBatchSize 每帧解码的 WAV 数量。
const audioLoadBatchSize = 10

// loadingHoldFrames 加载完成后停留帧数（~0.5s）。
const loadingHoldFrames = 30

// LoadingScene 加载场景：显示品牌和进度条，分帧初始化资源。
type LoadingScene struct {
	g       *Game
	fontMgr *render.FontManager

	phase      loadPhase
	progress   float64 // 0.0 ~ 1.0
	statusText string

	// 音频分批加载状态
	wavEntries []gameAudio.WAVEntry
	wavIndex   int

	holdFrames int // 完成后停留计数

	animTime float64 // 动画计时
}

// NewLoadingScene 创建加载场景。
func NewLoadingScene(g *Game) *LoadingScene {
	return &LoadingScene{
		g:          g,
		fontMgr:    render.GlobalFont(),
		statusText: "Initializing...",
	}
}

func (s *LoadingScene) Update() error {
	s.animTime += 1.0 / 60.0

	switch s.phase {
	case phaseConfigs:
		s.statusText = "Loading configs..."
		render.InitGlobalIcons(config.GetAssetFS())
		abilities.InitConfigAbilities()
		config.LoadBalance()
		config.LoadTierPresets()
		config.LoadAndCacheWardenConfigs()
		config.LoadBuffRules()
		config.LoadSpawnerConfig()
		s.progress = 0.10
		s.phase = phaseShaders

	case phaseShaders:
		s.statusText = "Compiling shaders..."
		if err := postprocess.InitShaders(); err != nil {
			log.Printf("后处理着色器编译失败: %v", err)
		}
		s.progress = 0.20
		s.phase = phaseAudioScan

	case phaseAudioScan:
		s.statusText = "Scanning audio..."
		s.g.audioMgr = gameAudio.NewManager()
		assetFS := config.GetAssetFS()
		if assetFS != nil {
			s.wavEntries = s.g.audioMgr.ScanWAVEntries(assetFS)
		}
		s.wavIndex = 0
		s.progress = 0.22
		s.phase = phaseAudioLoad

	case phaseAudioLoad:
		if s.wavIndex >= len(s.wavEntries) {
			log.Printf("已加载 %d 个音效", s.g.audioMgr.Count())
			s.progress = 0.82
			s.phase = phaseMascot
			break
		}
		// 每帧加载一批
		end := s.wavIndex + audioLoadBatchSize
		if end > len(s.wavEntries) {
			end = len(s.wavEntries)
		}
		for i := s.wavIndex; i < end; i++ {
			if err := s.g.audioMgr.LoadWAVEntry(s.wavEntries[i]); err != nil {
				log.Printf("解码音效 %s 失败: %v", s.wavEntries[i].FileName, err)
			}
		}
		s.wavIndex = end
		// 进度: 0.22 ~ 0.82 之间线性
		if len(s.wavEntries) > 0 {
			s.progress = 0.22 + 0.60*float64(s.wavIndex)/float64(len(s.wavEntries))
		}
		loaded := s.wavIndex
		total := len(s.wavEntries)
		s.statusText = fmt.Sprintf("Loading audio... %d/%d", loaded, total)

	case phaseMascot:
		s.statusText = "Loading mascot..."
		mascotDialogs, err := mascot.LoadAllDialogs(config.GetDataFS())
		if err != nil {
			log.Printf("[mascot] dialog load error: %v", err)
		}
		s.g.mascot = mascot.NewGuide(mascotDialogs, nil)
		s.g.mascot.InitConditions(mascot.DefaultConditions())
		s.g.mascotAnim = render.LoadMascotSprites(config.GetAssetFS())
		s.progress = 0.90
		s.phase = phaseSettings

	case phaseSettings:
		s.statusText = "Loading settings..."
		sd := LoadSettings()
		log.Printf("[settings] sfxEnabled=%v sfxVol=%.2f bgmVol=%.2f", sd.SFXEnabled, sd.SFXVolume, sd.BGMVolume)
		s.g.audioMgr.SetSFXEnabled(sd.SFXEnabled)
		s.g.audioMgr.SetVolume(sd.SFXVolume)
		s.g.audioMgr.SetBGMVolume(sd.BGMVolume)
		if sd.Quality >= 0 && sd.Quality <= 2 {
			game.CurrentQuality = game.QualityLevel(sd.Quality)
		}
		s.progress = 0.95
		s.phase = phaseFinalize

	case phaseFinalize:
		s.statusText = "Ready"
		s.progress = 1.0
		s.phase = phaseDone
		s.holdFrames = 0

	case phaseDone:
		s.holdFrames++
		if s.holdFrames >= loadingHoldFrames {
			s.g.SwitchScene(NewSelectScene(s.g))
		}
	}

	return nil
}

// --- 渲染常量 ---
var (
	loadingBg       = color.RGBA{R: 18, G: 25, B: 45, A: 255}  // 与 bgColor 一致
	loadingBarBg    = color.RGBA{R: 40, G: 48, B: 70, A: 255}  // 进度条背景
	loadingBarFill  = color.RGBA{R: 76, G: 175, B: 80, A: 255} // 进度条填充（绿色）
	loadingBarGlow  = color.RGBA{R: 76, G: 175, B: 80, A: 60}  // 进度条辉光
	loadingTextMain = color.RGBA{R: 230, G: 230, B: 235, A: 255}
	loadingTextSub  = color.RGBA{R: 140, G: 145, B: 160, A: 255}
	loadingTextDim  = color.RGBA{R: 90, G: 95, B: 110, A: 255}
)

const (
	barWidth  float32 = 400
	barHeight float32 = 6
	barRadius float32 = 3
)

func (s *LoadingScene) Draw(screen *ebiten.Image) {
	screen.Fill(loadingBg)

	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)
	cx := float32(sw / 2)

	// 装饰线（与 Title 场景一致风格）
	lineClr := color.RGBA{R: 50, G: 60, B: 90, A: 100}
	halfW := float32(220)
	topLineY := float32(sh/2 - 80)
	botLineY := float32(sh/2 + 30)
	draw.Line(screen, cx-halfW, topLineY, cx+halfW, topLineY, 1, lineClr, false)
	draw.Line(screen, cx-halfW, botLineY, cx+halfW, botLineY, 1, lineClr, false)

	// 角落菱形
	dClr := color.RGBA{R: 76, G: 175, B: 80, A: 60}
	dr := float32(4)
	for _, pos := range [][2]float32{
		{cx - halfW, topLineY}, {cx + halfW, topLineY},
		{cx - halfW, botLineY}, {cx + halfW, botLineY},
	} {
		px, py := pos[0], pos[1]
		draw.Line(screen, px, py-dr, px+dr, py, 1, dClr, false)
		draw.Line(screen, px+dr, py, px, py+dr, 1, dClr, false)
		draw.Line(screen, px, py+dr, px-dr, py, 1, dClr, false)
		draw.Line(screen, px-dr, py, px, py-dr, 1, dClr, false)
	}

	fm := s.fontMgr
	if fm == nil {
		return
	}

	// 标题
	fm.DrawCenteredBoldText(screen, "Mini Tower Defense", sw/2, sh/2-50, 28, loadingTextMain)
	// 副标题
	fm.DrawCenteredText(screen, "迷你塔防", sw/2, sh/2-18, 14, loadingTextSub)

	// --- 进度条 ---
	barX := cx - barWidth/2
	barY := float32(sh/2 + 50)

	// 背景
	draw.RoundRect(screen, barX, barY, barWidth, barHeight, barRadius, loadingBarBg)

	// 填充
	fillW := barWidth * float32(s.progress)
	if fillW > 0 {
		draw.RoundRect(screen, barX, barY, fillW, barHeight, barRadius, loadingBarFill)
		// 顶部辉光
		draw.RoundRect(screen, barX, barY, fillW, barHeight/2, barRadius, loadingBarGlow)
	}

	// 百分比
	pct := int(s.progress * 100)
	if pct > 100 {
		pct = 100
	}
	pctText := fmt.Sprintf("%d%%", pct)
	fm.DrawText(screen, pctText, float64(barX+barWidth+8), float64(barY+barHeight/2+2), 10, loadingTextSub)

	// 状态文字
	fm.DrawCenteredText(screen, s.statusText, sw/2, float64(barY+barHeight+16), 10, loadingTextDim)

	// 底部脉冲版本号
	pulse := 0.5 + 0.5*math.Sin(s.animTime*2)
	alpha := uint8(60 + 40*pulse)
	fm.DrawCenteredText(screen, "v1.0", sw/2, sh-16, 10, color.RGBA{R: 90, G: 95, B: 110, A: alpha})
}
```

**Step 2: Verify build**

Run: `go build ./internal/scene/...`
Expected: OK

**Step 3: Commit**

```
feat: add LoadingScene with progress bar and branding
```

---

### Task 3: Refactor NewGame() to use LoadingScene

**Files:**
- Modify: `internal/scene/game.go`

**Step 1: Simplify NewGame() — defer heavy init to LoadingScene**

修改 `NewGame()` 仅保留字体初始化（LoadingScene 需要画文字），其余全部延迟：

```go
func NewGame() *Game {
	// 字体必须同步初始化（LoadingScene 需要画文字）
	initFont()

	g := &Game{
		width:  game.ScreenWidth,
		height: game.ScreenHeight,
		bus:    event.NewBus(),
	}

	if !HeadlessMode {
		g.current = NewLoadingScene(g)
	} else {
		// HeadlessMode: 同步加载所有资源（无 UI）
		render.InitGlobalIcons(config.GetAssetFS())
		abilities.InitConfigAbilities()
		config.LoadBalance()
		config.LoadTierPresets()
		config.LoadAndCacheWardenConfigs()
		config.LoadBuffRules()
		config.LoadSpawnerConfig()
		if err := postprocess.InitShaders(); err != nil {
			log.Printf("后处理着色器编译失败: %v", err)
		}
		g.audioMgr = gameAudio.NewManager()
	}

	g.sessionStart = time.Now()
	return g
}
```

**Step 2: Update `currentSceneName` type switch**

添加 LoadingScene 分支：

```go
case *LoadingScene:
    return "loading"
```

**Step 3: Verify build**

Run: `go build ./...`
Expected: OK

**Step 4: Run game to verify loading screen works**

Run: `make run`
Expected: 看到品牌页 + 进度条从 0 到 100%，然后自动过渡到 SelectScene

**Step 5: Commit**

```
feat: refactor NewGame to use LoadingScene for deferred init
```

---

### Task 4: Verify autoplay (HeadlessMode) still works

**Step 1: Run quick autoplay**

Run: `go run cmd/autoplay/main.go --scenario attack-style-coverage`
Expected: 正常完成，无 panic

**Step 2: Commit (if any fixes needed)**

---

### Implementation Notes

- `LoadingScene` 不走 `SwitchScene` 创建（直接赋值 `g.current`），因为 fade 过渡需要 scene 已存在才能 Draw
- `LoadingScene.Update()` 每帧执行一个 phase，音频 phase 内部再分批（每帧 10 个 WAV）
- 完成后 0.5s 停留，然后用 `g.SwitchScene()` 触发标准 fade 过渡到 SelectScene
- HeadlessMode 保持同步加载（autoplay 不需要 UI）
- `LoadAllFromFS` 保留不动（其他地方可能还在用），新增的是 `ScanWAVEntries` + `LoadWAVEntry`
