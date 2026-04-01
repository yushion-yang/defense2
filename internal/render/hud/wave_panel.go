// wave_panel.go — Left-side wave preview panel.
// Uses FlexPanel + AnchoredRect for adaptive layout.
package hud

import (
	"fmt"

	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// WavePanelData holds data for the wave preview panel.
type WavePanelData struct {
	WaveNum    int
	MaxWaves   int
	EnemyCount int
	WaveLabel  string
}

// DrawWavePanel renders the left-side wave information panel using FlexPanel.
func DrawWavePanel(screen *ebiten.Image, d WavePanelData) {
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	const (
		panelW = float32(theme.InfoPanelW)
		pad    = float32(12)
		lineH  = float32(20)
	)

	// 计算内容高度
	contentLines := 2 // wave + enemy count
	if d.WaveLabel != "" {
		contentLines++
	}
	panelH := pad*2 + lineH*float32(contentLines)

	// 锚定到左下角
	rect := ui.AnchoredRect(ui.AnchorBottomLeft, panelW, panelH,
		0, 0, float32(theme.BottomMargin), float32(theme.InfoPanelX))

	// 面板
	p := ui.NewFlexPanel(rect.X, rect.Y, rect.W, pad)
	p.Radius = float32(theme.WavePanelRadius)
	p.BgColor = theme.WavePanelBg

	// 波次号
	p.AddRow(lineH, func(screen *ebiten.Image, x, y float64, w float64) {
		waveTxt := fmt.Sprintf("%d波预览", d.WaveNum)
		fm.DrawText(screen, waveTxt, x, y, theme.FontLG, theme.ResWaves)
	})

	// 可选标签
	if d.WaveLabel != "" {
		p.AddRow(lineH, func(screen *ebiten.Image, x, y float64, w float64) {
			fm.DrawText(screen, d.WaveLabel, x, y, theme.FontSM, theme.TextMuted)
		})
	}

	// 敌人数量
	p.AddRow(lineH, func(screen *ebiten.Image, x, y float64, w float64) {
		if im := render.GlobalIcons(); im != nil {
			if img := im.Get("stat-target"); img != nil {
				draw.Sprite(screen, img, x+5, y+5, 10)
				fm.DrawText(screen, fmt.Sprintf("%d", d.EnemyCount), x+16, y, theme.FontSM, theme.TextBody)
				return
			}
		}
		fm.DrawText(screen, fmt.Sprintf("场上: %d", d.EnemyCount), x, y, theme.FontSM, theme.TextBody)
	})

	p.Draw(screen)
}
