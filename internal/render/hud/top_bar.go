// top_bar.go — Centered pill-shaped top status bar.
// Displays resources on the left, action buttons on the right.
package hud

import (
	"fmt"
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// TopBarData holds the runtime data the top bar needs to render.
type TopBarData struct {
	Gold      int  // current gold
	Lives     int  // remaining lives
	Wave      int  // current wave number
	MaxWaves  int  // total waves
	Kills     int  // cumulative kills
	Enemies   int  // alive enemies on field
	Speed     int  // game speed multiplier (1, 2, 3, 10)
	BuildMode bool // whether build mode is active
	TestMode  bool // 测试模式（显示额外按钮）
	SpawnMode bool // 造怪模式激活
	DebugOpen bool // 调试面板打开
}

// topBarBtn describes a button inside the top bar.
type topBarBtn struct {
	label string
	w     float32
	tone  color.RGBA
}

// DrawTopBar renders the centered pill-shaped top bar.
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

	// ── Left section: resources ──
	resX := float64(pillX) + 16
	resY := float64(pillY) + 12 // vertically centered baseline

	// Heart icon + lives
	draw.FilledCircle(screen, float32(resX)+6, float32(resY)+2, 6, theme.ResHearts)
	resX += 16
	livesTxt := fmt.Sprintf("%d", d.Lives)
	fm.DrawText(screen, livesTxt, resX, resY-5, theme.FontTopBar, color.White)
	resX += fm.MeasureText(livesTxt, theme.FontTopBar) + 10

	// Coin icon + gold
	draw.FilledCircle(screen, float32(resX)+6, float32(resY)+2, 6, theme.ResGold)
	resX += 16
	goldTxt := fmt.Sprintf("%d", d.Gold)
	fm.DrawText(screen, goldTxt, resX, resY-5, theme.FontTopBar, color.White)
	resX += fm.MeasureText(goldTxt, theme.FontTopBar) + 10

	// Wave icon + wave/maxWaves
	draw.FilledCircle(screen, float32(resX)+5, float32(resY)+2, 5, theme.ResWaves)
	resX += 14
	waveTxt := fmt.Sprintf("%d/%d", d.Wave, d.MaxWaves)
	fm.DrawText(screen, waveTxt, resX, resY-5, theme.FontTopBar, color.White)
	resX += fm.MeasureText(waveTxt, theme.FontTopBar) + 10

	// Kills icon (stat-target) + kill count
	if im := render.GlobalIcons(); im != nil {
		if img := im.Get("stat-target"); img != nil {
			draw.Sprite(screen, img, resX+5, float64(resY)+1, 10)
			resX += 14
			killsTxt := fmt.Sprintf("%d", d.Kills)
			fm.DrawText(screen, killsTxt, resX, resY-5, theme.FontTopBar, color.White)
		}
	}

	// ── Divider ──
	divX := pillX + dividerOff
	divY1 := pillY + 6
	divY2 := pillY + pillH - 6
	draw.Line(screen, divX, divY1, divX, divY2, 1, theme.HUDTopBarDivider, false)

	// ── Right section: buttons (using ButtonRow) ──
	buildTone := theme.ToneSecondary
	if d.BuildMode {
		buildTone = theme.TonePrimary
	}
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
	var btns []btnDef

	btns = append(btns, btnDef{"build", "造塔", buildTone})

	// 测试模式：造怪 + 调试按钮
	if d.TestMode {
		spawnClr := theme.TonePrimary
		if d.SpawnMode {
			spawnClr = color.RGBA{R: 220, G: 60, B: 60, A: 255} // 红色表示激活
		}
		btns = append(btns, btnDef{"spawn", "造怪", spawnClr})
	}

	btns = append(btns, btnDef{"start", "开波", theme.TonePrimary})
	btns = append(btns, btnDef{"speed", speedLabel, theme.ToneAccent})
	btns = append(btns, btnDef{"menu", "菜单", theme.ToneSecondary})

	if d.TestMode {
		debugClr := theme.ToneSecondary
		if d.DebugOpen {
			debugClr = color.RGBA{R: 80, G: 120, B: 180, A: 255}
		}
		btns = append(btns, btnDef{"debug", "调试", debugClr})
	}

	items := make([]ui.ButtonRowItem, len(btns))
	names := make([]string, len(btns))
	for i, b := range btns {
		items[i] = ui.ButtonRowItem{Label: b.label, Color: b.clr}
		names[i] = b.name
	}

	// 计算按钮区域（右对齐）
	totalBtnW := float32(len(items))*theme.BtnBuildW + float32(len(items)-1)*btnGap
	btnArea := ui.Rect{
		X: pillX + pillW - 12 - totalBtnW,
		Y: pillY + (pillH-btnH)/2,
		W: totalBtnW,
		H: btnH,
	}

	result := ui.DrawButtonRow(screen, btnArea, items, ui.ButtonRowStyle{
		Height:   btnH,
		Gap:      btnGap,
		Radius:   btnR,
		FontSize: theme.FontMD,
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
