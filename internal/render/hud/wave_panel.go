// wave_panel.go — 左侧抽屉式波次预览面板。
// 收起时只露出 tab handle（波次号+箭头），展开时滑出完整面板。
package hud

import (
	"image/color"
	"strconv"

	"defense2/internal/i18n"
	"defense2/internal/render"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

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
	NextWaveTypes []WaveTypeEntry // 原型名x数量
	AllDone       bool            // 所有波次已出完

	// 波次状态（新增）
	WaveActive      bool    // 当前波是否正在出怪
	WaveTimer       float64 // 波间倒计时（秒）
	WaitingForClear bool    // 等待清场
	SpawnProgress   int     // 当前波已出怪数
	SpawnTotal      int     // 当前波总怪数（0=未知/不适用）
}

// WaveTypeEntry 下一波中的一种敌人类型。
type WaveTypeEntry struct {
	Label string
	Count int
}

// 面板布局常量。
const (
	wpHandleH = float32(32) // tab handle 高度
	wpRadius  = float32(10)
	wpPad     = float32(10)
	wpLineH   = float32(16)
	wpMarginB = float32(20) // 底部边距（与 ActionBar 留间距）
	wpMarginL = float32(0)  // 左侧贴边
)

// WavePanelState 波次面板动画状态。
type WavePanelState struct {
	SlideT     float64 // 0=收起, 1=展开
	Open       bool
	LastPanelH float32 // 上次 Draw 计算的面板高度（hit-test 用）
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

	panelW := float32(theme.WavePanelW)
	handleW := float32(theme.WaveHandleW)

	// 计算面板内容高度
	lines := 2 // 波次号 + 场上敌人
	// 波次状态行（倒计时/出怪进度/等待清场）
	if !d.AllDone {
		lines++ // 状态行
	}
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

	// 缓存 panelH 给 hit-test 使用
	state.LastPanelH = panelH

	// 面板位置：左侧滑出
	screenH := float32(theme.CanvasH)
	handleY := screenH - panelH - wpMarginB
	panelX := wpMarginL - panelW + float32(t)*panelW // -panelW ~ 0
	handleX := panelX + panelW                        // handle 在面板右侧

	// 绘制面板主体（slideT > 0 时）
	if t > 0.01 {
		ui.Panel(screen, panelX, handleY, panelW, panelH, ui.PanelStyle{
			BgColor: theme.WavePanelBg, Radius: wpRadius,
		})

		x := float64(panelX) + float64(wpPad)
		y := float64(handleY) + float64(wpPad)
		contentW := float64(panelW) - float64(wpPad)*2

		// 波次号
		waveTxt := i18n.TF("hud.wavepanel.wave_status", d.WaveNum, d.MaxWaves)
		ui.Label(screen, waveTxt, x, y, contentW, ui.LabelStyle{
			Font: theme.FontH2, Color: theme.ResWaves, Bold: true,
		})
		y += float64(wpLineH)

		// 场上敌人
		enemyTxt := i18n.TF("hud.wavepanel.on_field", d.EnemyCount)
		ui.Label(screen, enemyTxt, x, y, contentW, ui.LabelStyle{
			Font: theme.FontCaption, Color: theme.TextBody,
		})
		y += float64(wpLineH)

		// 波次状态行
		if !d.AllDone {
			statusTxt, statusClr := waveStatusText(d)
			ui.Label(screen, statusTxt, x, y, contentW, ui.LabelStyle{
				Font: theme.FontCaption, Color: statusClr,
			})
			y += float64(wpLineH)
		}

		// 下一波预览
		if d.NextWaveCount > 0 && !d.AllDone {
			nextTxt := i18n.TF("hud.wavepanel.next_wave", d.NextWaveCount)
			if d.NextWaveBoss {
				nextTxt += " " + i18n.T("hud.wavepanel.boss_tag")
			}
			clr := theme.TextMuted
			if d.NextWaveBoss {
				clr = color.RGBA{R: 255, G: 100, B: 80, A: 255}
			}
			ui.Label(screen, nextTxt, x, y, contentW, ui.LabelStyle{
				Font: theme.FontCaption, Color: clr,
			})
			y += float64(wpLineH)

			// 原型列表
			for i, entry := range d.NextWaveTypes {
				if i >= 4 {
					break
				}
				entryTxt := "  " + entry.Label + " x" + strconv.Itoa(entry.Count)
				ui.Label(screen, entryTxt, x, y, contentW, ui.LabelStyle{
					Font: theme.FontCaption, Color: theme.TextMuted,
				})
				y += float64(wpLineH)
			}
		} else if d.AllDone {
			ui.Label(screen, i18n.T("hud.wavepanel.final_wave"), x, y, contentW, ui.LabelStyle{
				Font: theme.FontCaption, Color: color.RGBA{R: 255, G: 215, B: 0, A: 255},
			})
		}
	}

	// Tab handle（始终可见）
	ui.Panel(screen, handleX, handleY, handleW, wpHandleH, ui.PanelStyle{
		BgColor: theme.WavePanelBg, Radius: wpRadius,
	})

	hx := float64(handleX) + 8
	hy := float64(handleY) + 8
	handleContentW := float64(handleW) - 16
	waveLbl := i18n.TF("hud.wavepanel.wave_short", d.WaveNum)
	ui.Label(screen, waveLbl, hx, hy, handleContentW, ui.LabelStyle{
		Font: theme.FontBody, Color: theme.ResWaves, Bold: true,
	})

	// 箭头
	arrow := ">"
	if state.Open {
		arrow = "<"
	}
	ui.Label(screen, arrow, hx, hy, handleContentW, ui.LabelStyle{
		Font: theme.FontBody, Color: theme.TextMuted, Align: ui.AlignRight,
	})
}

// waveStatusText 根据波次状态返回状态文本和颜色。
func waveStatusText(d WavePanelData) (string, color.RGBA) {
	if d.WaveActive {
		// 正在出怪
		if d.SpawnTotal > 0 {
			return i18n.TF("hud.wavepanel.spawning", d.SpawnProgress, d.SpawnTotal),
				color.RGBA{R: 100, G: 200, B: 100, A: 255}
		}
		return i18n.T("hud.wavepanel.spawning_simple"), color.RGBA{R: 100, G: 200, B: 100, A: 255}
	}
	if d.WaitingForClear {
		return i18n.T("hud.wavepanel.waiting_clear"), color.RGBA{R: 200, G: 180, B: 100, A: 255}
	}
	if d.WaveTimer > 0 {
		return i18n.TF("hud.wavepanel.next_timer", d.WaveTimer), color.RGBA{R: 180, G: 180, B: 220, A: 255}
	}
	return "", color.RGBA{}
}

// wavePanelHandleRect 返回 tab handle 的屏幕矩形（用于点击检测）。
// 使用 LastPanelH 确保 hit-test 位置与渲染位置一致。
func wavePanelHandleRect(state *WavePanelState) (x, y, w, h float32) {
	t := easeOut(state.SlideT)
	panelW := float32(theme.WavePanelW)
	handleW := float32(theme.WaveHandleW)
	screenH := float32(theme.CanvasH)
	panelX := wpMarginL - panelW + float32(t)*panelW
	handleX := panelX + panelW
	// 使用缓存的面板高度（与 DrawWavePanel 一致）
	panelH := state.LastPanelH
	if panelH < wpHandleH {
		panelH = wpHandleH
	}
	handleY := screenH - panelH - wpMarginB
	return handleX, handleY, handleW, wpHandleH
}

// WavePanelHandleHitTest 检查点击是否命中波次面板的 tab handle。
func WavePanelHandleHitTest(px, py float32, state *WavePanelState) bool {
	x, y, w, h := wavePanelHandleRect(state)
	return px >= x && px <= x+w && py >= y && py <= y+h
}
