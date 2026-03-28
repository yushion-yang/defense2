// top_bar.go — Centered pill-shaped top status bar.
// Displays resources on the left, action buttons on the right.
package hud

import (
	"fmt"
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TopBarData holds the runtime data the top bar needs to render.
type TopBarData struct {
	Gold      int  // current gold
	Lives     int  // remaining lives
	Wave      int  // current wave number
	MaxWaves  int  // total waves
	Kills     int  // cumulative kills
	Enemies   int  // alive enemies on field
	Speed     int  // game speed multiplier (1 or 2)
	BuildMode bool // whether build mode is active
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

	// ── Divider ──
	divX := pillX + dividerOff
	divY1 := pillY + 6
	divY2 := pillY + pillH - 6
	vector.StrokeLine(screen, divX, divY1, divX, divY2, 1, theme.HUDTopBarDivider, false)

	// ── Right section: buttons ──
	buildTone := theme.ToneSecondary
	if d.BuildMode {
		buildTone = theme.TonePrimary
	}
	speedLabel := "x1"
	if d.Speed == 2 {
		speedLabel = "x2"
	}

	buttons := []topBarBtn{
		{"造塔", theme.BtnBuildW, buildTone},
		{"开波", theme.BtnStartW, theme.TonePrimary},
		{speedLabel, theme.BtnSpeedW, theme.ToneAccent},
		{"菜单", theme.BtnMenuW, theme.ToneSecondary},
	}

	// Calculate total button width to right-align within the pill.
	var totalBtnW float32
	for i, b := range buttons {
		totalBtnW += b.w
		if i > 0 {
			totalBtnW += btnGap
		}
	}

	btnX := pillX + pillW - 12 - totalBtnW
	btnY := pillY + (pillH-btnH)/2

	for _, b := range buttons {
		draw.RoundRect(screen, btnX, btnY, b.w, btnH, btnR, b.tone)
		// Centered label
		cx := float64(btnX) + float64(b.w)/2
		cy := float64(btnY) + float64(btnH)/2 - 6
		fm.DrawCenteredText(screen, b.label, cx, cy, theme.FontMD, color.White)
		btnX += b.w + btnGap
	}
}

// TopBarHitTest returns the button name hit by (px, py), or "" if none.
func TopBarHitTest(px, py float32) string {
	const (
		pillY  = float32(theme.TopBarY)
		pillW  = float32(theme.TopBarW)
		pillH  = float32(theme.TopBarH)
		btnH   = float32(theme.TopBarBtnH)
		btnGap = float32(theme.TopBarBtnGap)
	)
	pillX := topBarX

	// Quick bounds check on pill.
	if px < pillX || px > pillX+pillW || py < pillY || py > pillY+pillH {
		return ""
	}

	type btnDef struct {
		name string
		w    float32
	}
	buttons := []btnDef{
		{"build", theme.BtnBuildW},
		{"start", theme.BtnStartW},
		{"speed", theme.BtnSpeedW},
		{"menu", theme.BtnMenuW},
	}

	var totalBtnW float32
	for i, b := range buttons {
		totalBtnW += b.w
		if i > 0 {
			totalBtnW += btnGap
		}
	}

	btnX := pillX + pillW - 12 - totalBtnW
	btnY := pillY + (pillH-btnH)/2

	for _, b := range buttons {
		if px >= btnX && px <= btnX+b.w && py >= btnY && py <= btnY+btnH {
			return b.name
		}
		btnX += b.w + btnGap
	}
	return ""
}
