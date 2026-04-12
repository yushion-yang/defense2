// wave_panel.go — 左侧抽屉式波次预览面板。
// 收起时只露出 tab handle（波次号+箭头），展开时滑出完整面板。
package hud

import (
	"image/color"
	"strconv"

	"defense2/internal/core/game"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// WavePanelData holds data for the wave preview panel.
type WavePanelData struct {
	WaveNum    int
	MaxWaves   int
	EnemyCount int // 场上敌人数
	// 下一波预览
	NextWaveCount int
	NextWaveBoss  bool
	NextWaveTypes []WaveTypeEntry // 原型名×数量
	AllDone       bool            // 所有波次已出完
}

// WaveTypeEntry 下一波中的一种敌人类型。
type WaveTypeEntry struct {
	Label string
	Count int
}

// 面板布局常量。
const (
	wpHandleW = float32(76)  // tab handle 宽度
	wpHandleH = float32(32)  // tab handle 高度
	wpPanelW  = float32(190) // 展开面板宽度（不含 handle）
	wpRadius  = float32(10)
	wpPad     = float32(10)
	wpLineH   = float32(16)
	wpMarginB = float32(14) // 底部边距
	wpMarginL = float32(0)  // 左侧贴边
)

// WavePanelState 波次面板动画状态。
type WavePanelState struct {
	SlideT float64 // 0=收起, 1=展开
	Open   bool
}

// Update 驱动抽屉滑动动画。
func (s *WavePanelState) Update(dt float64, open bool) {
	s.Open = open
	speed := 6.0 // ~0.17s
	if open {
		s.SlideT += dt * speed
		if s.SlideT > 1 {
			s.SlideT = 1
		}
	} else {
		s.SlideT -= dt * speed
		if s.SlideT < 0 {
			s.SlideT = 0
		}
	}
}

// easeOut 简单的 ease-out 缓动。
func easeOut(t float64) float64 {
	return 1 - (1-t)*(1-t)
}

// DrawWavePanel 绘制抽屉式波次面板。
func DrawWavePanel(screen *ebiten.Image, d WavePanelData, state *WavePanelState) {
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	t := easeOut(state.SlideT)

	// 计算面板内容高度
	lines := 2 // 波次号 + 场上敌人
	if d.NextWaveCount > 0 && !d.AllDone {
		lines++ // "下一波: N怪"
		typeLines := len(d.NextWaveTypes)
		if typeLines > 4 {
			typeLines = 4 // 最多显示4种
		}
		lines += typeLines
	} else if d.AllDone {
		lines++ // "最终波!"
	}
	panelH := wpPad*2 + wpLineH*float32(lines)
	if panelH < wpHandleH {
		panelH = wpHandleH
	}

	// 面板位置：左侧滑出
	screenH := float32(game.ScreenHeight)
	handleY := screenH - panelH - wpMarginB
	panelX := wpMarginL - wpPanelW + float32(t)*wpPanelW // -panelW ~ 0
	handleX := panelX + wpPanelW                         // handle 在面板右侧

	// 绘制面板主体（slideT > 0 时）
	if t > 0.01 {
		draw.RoundRect(screen, panelX, handleY, wpPanelW, panelH, wpRadius,
			theme.WavePanelBg)

		x := float64(panelX) + float64(wpPad)
		y := float64(handleY) + float64(wpPad)

		// 波次号
		waveTxt := strconv.Itoa(d.WaveNum) + "/" + strconv.Itoa(d.MaxWaves) + "波"
		fm.DrawBoldText(screen, waveTxt, x, y, theme.FontH2, theme.ResWaves)
		y += float64(wpLineH)

		// 场上敌人
		enemyTxt := "场上: " + strconv.Itoa(d.EnemyCount)
		fm.DrawText(screen, enemyTxt, x, y, theme.FontCaption, theme.TextBody)
		y += float64(wpLineH)

		// 下一波预览
		if d.NextWaveCount > 0 && !d.AllDone {
			nextTxt := "下一波: " + strconv.Itoa(d.NextWaveCount) + "怪"
			if d.NextWaveBoss {
				nextTxt += " [首领]"
			}
			clr := theme.TextMuted
			if d.NextWaveBoss {
				clr = color.RGBA{R: 255, G: 100, B: 80, A: 255}
			}
			fm.DrawText(screen, nextTxt, x, y, theme.FontCaption, clr)
			y += float64(wpLineH)

			// 原型列表
			for i, entry := range d.NextWaveTypes {
				if i >= 4 {
					break
				}
				entryTxt := "  " + entry.Label + " ×" + strconv.Itoa(entry.Count)
				fm.DrawText(screen, entryTxt, x, y, theme.FontCaption, theme.TextMuted)
				y += float64(wpLineH)
			}
		} else if d.AllDone {
			fm.DrawText(screen, "最终波!", x, y, theme.FontCaption,
				color.RGBA{R: 255, G: 215, B: 0, A: 255})
		}
	}

	// Tab handle（始终可见）
	draw.RoundRect(screen, handleX, handleY, wpHandleW, wpHandleH, wpRadius,
		theme.WavePanelBg)

	hx := float64(handleX) + 8
	hy := float64(handleY) + 8
	waveLbl := strconv.Itoa(d.WaveNum) + "波"
	fm.DrawBoldText(screen, waveLbl, hx, hy, theme.FontBody, theme.ResWaves)

	// 箭头
	arrow := ">"
	if state.Open {
		arrow = "<"
	}
	arrowX := float64(handleX) + float64(wpHandleW) - 16
	fm.DrawText(screen, arrow, arrowX, hy, theme.FontBody, theme.TextMuted)
}

// wavePanelHandleRect 返回 tab handle 的屏幕矩形（用于点击检测）。
func wavePanelHandleRect(state *WavePanelState) (x, y, w, h float32) {
	// 计算与 Draw 一致的 handle 位置
	t := easeOut(state.SlideT)
	screenH := float32(game.ScreenHeight)
	panelX := wpMarginL - wpPanelW + float32(t)*wpPanelW
	handleX := panelX + wpPanelW
	handleY := screenH - wpHandleH - wpMarginB
	return handleX, handleY, wpHandleW, wpHandleH
}

// WavePanelHandleHitTest 检查点击是否命中波次面板的 tab handle。
func WavePanelHandleHitTest(px, py float32, state *WavePanelState) bool {
	x, y, w, h := wavePanelHandleRect(state)
	return px >= x && px <= x+w && py >= y && py <= y+h
}
