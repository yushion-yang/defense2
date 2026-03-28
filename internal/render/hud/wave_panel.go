// wave_panel.go — Left-side wave preview panel.
// Shows current wave number, optional label, and alive enemy count.
package hud

import (
	"fmt"

	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// WavePanelData holds data for the wave preview panel.
type WavePanelData struct {
	WaveNum    int    // current wave number
	MaxWaves   int    // total waves
	EnemyCount int    // enemies alive on field
	WaveLabel  string // optional label like "发育波"
}

// DrawWavePanel renders the left-side wave information panel.
func DrawWavePanel(screen *ebiten.Image, d WavePanelData) {
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	const (
		panelX = float32(theme.InfoPanelX)
		panelW = float32(theme.InfoPanelW)
		panelR = float32(theme.WavePanelRadius)
		pad    = float32(12)
	)

	// Calculate height based on content.
	lineH := float32(20)
	panelH := pad*2 + lineH // wave number line
	if d.WaveLabel != "" {
		panelH += lineH // optional label line
	}
	panelH += lineH // enemy count line

	panelY := float32(theme.CanvasH) - panelH - float32(theme.BottomMargin)

	// Background
	draw.RoundRect(screen, panelX, panelY, panelW, panelH, panelR, theme.WavePanelBg)

	// Content
	ix := float64(panelX) + float64(pad)
	iy := float64(panelY) + float64(pad)

	// Wave number
	waveTxt := fmt.Sprintf("%d波预览", d.WaveNum)
	fm.DrawText(screen, waveTxt, ix, iy, theme.FontLG, theme.ResWaves)
	iy += float64(lineH)

	// Optional wave label
	if d.WaveLabel != "" {
		fm.DrawText(screen, d.WaveLabel, ix, iy, theme.FontSM, theme.TextMuted)
		iy += float64(lineH)
	}

	// Enemy count
	enemyTxt := fmt.Sprintf("场上: %d", d.EnemyCount)
	fm.DrawText(screen, enemyTxt, ix, iy, theme.FontSM, theme.TextBody)
}
