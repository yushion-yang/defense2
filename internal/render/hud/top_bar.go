// top_bar.go — Centered pill-shaped top status bar.
// Displays resources on the left, action buttons on the right.
//
// Draws directly to the screen each frame (no offscreen cache) so that
// draw.* auto-scaling works correctly on HiDPI displays.
// Button hit detection uses rects computed during the last render.
package hud

import (
	"image/color"
	"math"
	"strconv"

	"defense2/internal/i18n"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// TopBarData holds the runtime data the top bar needs to render.
type TopBarData struct {
	Gold          int     // current gold
	Lives         int     // remaining lives
	Wave          int     // current wave number
	MaxWaves      int     // total waves
	Kills         int     // cumulative kills
	Enemies       int     // alive enemies on field
	Speed         int     // game speed multiplier (1, 2, 3, 10)
	WaveCountdown float64 // 波间倒计时剩余秒数（0 表示无倒计时）
	TestMode      bool    // 测试模式（显示额外按钮）
	SpawnMode     bool    // 造怪模式激活
	DebugOpen     bool    // 调试面板打开
}

// topBarBtn describes a button inside the top bar.
type topBarBtn struct {
	label string
	w     float32
	tone  color.RGBA
}

// DrawTopBar renders the centered pill-shaped top bar directly to screen.
func DrawTopBar(screen *ebiten.Image, d TopBarData) {
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	const (
		pillY      = float32(theme.TopBarY)
		pillW      = float32(theme.TopBarW)
		pillH      = float32(theme.TopBarH)
		pillR      = float32(theme.TopBarRadius)
		btnH       = float32(theme.TopBarBtnH)
		btnGap     = float32(theme.TopBarBtnGap)
		btnR       = float32(theme.BtnRadius)
		dividerOff = float32(theme.TopBarDividerOffset)
	)
	pillX := topBarX

	// ── Pill background + border ──
	draw.RoundRect(screen, pillX, pillY, pillW, pillH, pillR, theme.HUDTopBarBg)
	draw.StrokeRoundRect(screen, pillX, pillY, pillW, pillH, pillR, 1, theme.HUDTopBarBorder)

	// ── Left section: resources (clamped to divider boundary) ──
	resX := float64(pillX) + 16
	resY := float64(pillY) + 12 // vertically centered baseline
	maxResX := float64(pillX) + float64(dividerOff) - 8 // 不超过分隔线
	const topFS = theme.FontH1                           // 顶栏使用 H1 字号

	// Heart icon + lives
	draw.FilledCircle(screen, float32(resX)+6, float32(resY)+2, 6, theme.ResHearts)
	resX += 16
	livesTxt := strconv.Itoa(d.Lives)
	fm.DrawText(screen, livesTxt, resX, resY-5, topFS, color.White)
	resX += fm.MeasureText(livesTxt, topFS) + 10

	// Coin icon + gold
	if resX+30 < maxResX {
		draw.FilledCircle(screen, float32(resX)+6, float32(resY)+2, 6, theme.ResGold)
		resX += 16
		goldTxt := strconv.Itoa(d.Gold)
		fm.DrawText(screen, goldTxt, resX, resY-5, topFS, color.White)
		resX += fm.MeasureText(goldTxt, topFS) + 10
	}

	// Wave icon + wave/maxWaves
	if resX+30 < maxResX {
		draw.FilledCircle(screen, float32(resX)+5, float32(resY)+2, 5, theme.ResWaves)
		resX += 14
		waveTxt := strconv.Itoa(d.Wave) + "/" + strconv.Itoa(d.MaxWaves)
		fm.DrawText(screen, waveTxt, resX, resY-5, topFS, color.White)
		resX += fm.MeasureText(waveTxt, topFS) + 10
	}

	// Kills icon (stat-target) + kill count
	if resX+30 < maxResX {
		if im := render.GlobalIcons(); im != nil {
			if img := im.Get("stat-target"); img != nil {
				draw.Sprite(screen, img, resX+5, float64(resY)+1, 10)
				resX += 14
				killsTxt := strconv.Itoa(d.Kills)
				fm.DrawText(screen, killsTxt, resX, resY-5, topFS, color.White)
			}
		}
	}

	// ── Divider ──
	divX := pillX + dividerOff
	divY1 := pillY + 6
	divY2 := pillY + pillH - 6
	draw.Line(screen, divX, divY1, divX, divY2, 1, theme.HUDTopBarDivider, false)

	// ── Right section: buttons (using ButtonRow) ──
	speedLabel := "x1"
	switch d.Speed {
	case 2:
		speedLabel = "x2"
	case 3:
		speedLabel = "x3"
	case 10:
		speedLabel = "T"
	default:
		speedLabel = "x1"
	}

	// 构建按钮列表 + 名称映射
	type btnDef struct {
		name  string
		label string
		clr   color.RGBA
	}
	var btnsBuf [8]btnDef
	btns := btnsBuf[:0]

	// 测试模式：造怪 + 调试按钮
	if d.TestMode {
		spawnClr := theme.TonePrimary
		if d.SpawnMode {
			spawnClr = theme.BtnDanger // 红色表示激活
		}
		btns = append(btns, btnDef{"spawn", i18n.T("hud.topbar.spawn"), spawnClr})
	}

	startLabel := i18n.T("hud.topbar.start_wave")
	if d.WaveCountdown > 0 {
		startLabel = i18n.TF("hud.topbar.start_wave_countdown", int(math.Ceil(d.WaveCountdown)))
	}
	btns = append(btns, btnDef{"start", startLabel, theme.TonePrimary})
	btns = append(btns, btnDef{"speed", speedLabel, theme.ToneAccent})
	btns = append(btns, btnDef{"menu", i18n.T("hud.topbar.menu"), theme.ToneSecondary})

	if d.TestMode {
		debugClr := theme.ToneSecondary
		if d.DebugOpen {
			debugClr = theme.ToneAccent
		}
		btns = append(btns, btnDef{"debug", i18n.T("hud.topbar.debug"), debugClr})
	}

	// 截图按钮（所有模式可用）
	btns = append(btns, btnDef{"screenshot", i18n.T("hud.topbar.screenshot"), theme.ToneSecondary})

	// Hover detection using last frame's rects
	mx, my := draw.CursorPos()
	tbHoverIdx := ui.HitTestButtonRow(lastTopBarBtnRects, float64(mx), float64(my))

	var itemBuf [8]ui.ButtonRowItem
	var nameBuf [8]string
	items := itemBuf[:len(btns)]
	names := nameBuf[:len(btns)]
	for i, b := range btns {
		items[i] = ui.ButtonRowItem{Label: b.label, Color: b.clr, Hovered: i == tbHoverIdx}
		names[i] = b.name
	}

	// 计算按钮区域（右对齐，按文本自适应宽度）
	// pad 必须与 DrawButtonRowAutoWidth 内部的 pad=16 一致
	const btnPadX float32 = 16
	totalBtnW := float32(0)
	for _, item := range items {
		tw := float32(fm.MeasureText(item.Label, theme.FontH2))
		w := tw + btnPadX*2
		if w < 40 {
			w = 40
		}
		totalBtnW += w
	}
	totalBtnW += float32(len(items)-1) * btnGap

	btnArea := ui.Rect{
		X: pillX + pillW - 12 - totalBtnW,
		Y: pillY + (pillH-btnH)/2,
		W: totalBtnW,
		H: btnH,
	}

	result := ui.DrawButtonRowAutoWidth(screen, btnArea, items, ui.ButtonRowStyle{
		Height:   btnH,
		Gap:      btnGap,
		Radius:   btnR,
		FontSize: theme.FontH2,
	})
	lastTopBarBtnRects = result.Rects
	lastTopBarBtnNames = names
}

// ── 按钮碰撞检测（复用 DrawButtonRow 的 Rects） ──

var lastTopBarBtnRects []ui.Rect
var lastTopBarBtnNames []string

// TopBarHitTest returns the button name hit by (px, py), or "" if none.
// Uses the Rects computed by the last DrawTopBar call.
func TopBarHitTest(px, py float32) string {
	idx := ui.HitTestButtonRow(lastTopBarBtnRects, float64(px), float64(py))
	if idx >= 0 && idx < len(lastTopBarBtnNames) {
		return lastTopBarBtnNames[idx]
	}
	return ""
}
