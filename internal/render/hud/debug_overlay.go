// debug_overlay.go — 调试覆盖层（F2 切换）。
// 显示射程圈、瞄准线、实体统计等调试可视化信息。
package hud

import (
	"fmt"
	"strconv"

	"defense2/internal/core/debug"
	"defense2/internal/core/game"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// DebugOverlay 调试覆盖层（F2 切换）。
type DebugOverlay struct {
	Enabled bool // 是否启用
}

// NewDebugOverlay 创建调试覆盖层（默认关闭）。
func NewDebugOverlay() *DebugOverlay {
	return &DebugOverlay{Enabled: false}
}

// Toggle 切换覆盖层显示状态。
func (o *DebugOverlay) Toggle() {
	o.Enabled = !o.Enabled
}

// DrawWorld 绘制世界空间调试信息（射程圈、瞄准线等）。
// towers/enemies 使用 interface{} 避免 render 包导入 core 包。
// 调用方应传入切片长度，具体的范围/瞄准渲染由调用方在 stage 层处理。
func (o *DebugOverlay) DrawWorld(screen *ebiten.Image, towers, enemies interface{}) {
	if !o.Enabled {
		return
	}
	// 占位实现：绘制实体数量提示文本
	// 实际射程圈和瞄准线应由 stage 层使用 draw.CircleOutline 等绘制
	fm := render.GlobalFont()
	if fm == nil {
		return
	}
	towerCount := countSlice(towers)
	enemyCount := countSlice(enemies)
	text := fmt.Sprintf("调试: %d塔 / %d敌", towerCount, enemyCount)
	fm.DrawText(screen, text, 10, float64(game.ScreenHeight)-30, theme.FontCaption, theme.DebugTextClr)
}

// DrawHUD 绘制 HUD 空间调试统计栏（屏幕顶部）。
func (o *DebugOverlay) DrawHUD(screen *ebiten.Image, towerCount, enemyCount, beamCount, projCount int) {
	if !o.Enabled {
		return
	}
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	// 半透明背景条（紧贴 TopBar 下方）
	barY := float32(theme.TopBarY+theme.TopBarH) + 2
	barH := float32(18)
	bgClr := theme.OverlayHeavy
	draw.FilledRect(screen, 0, barY, float32(game.ScreenWidth), barH, bgClr, false)

	// 统计文本
	stats := fmt.Sprintf("塔:%d  敌:%d  光束:%d  弹:%d", towerCount, enemyCount, beamCount, projCount)
	textClr := theme.DebugStatsClr
	fm.DrawCenteredText(screen, stats, float64(game.ScreenWidth)/2, float64(barY)+2, theme.FontCaption, textClr)
}

// DrawPerf 绘制性能统计栏（紧贴 DrawHUD 实体栏下方）。
// 使用 strconv + stack buffer 避免 fmt.Sprintf 分配。
// Format: "FPS:60 | U:2.1ms D:4.3ms | P99:3.2/6.1 | GC:2/s | Heap:12M"
func (o *DebugOverlay) DrawPerf(screen *ebiten.Image, pt *debug.PerfTracker) {
	if !o.Enabled || pt == nil {
		return
	}
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	// Position: second bar below entity stats bar
	barY := float32(theme.TopBarY+theme.TopBarH) + 2 + 18 + 1 // entity bar height + gap
	barH := float32(18)
	draw.FilledRect(screen, 0, barY, float32(game.ScreenWidth), barH, theme.OverlayHeavy, false)

	// Build string with strconv (zero-alloc pattern using stack buffer)
	var buf [256]byte
	b := buf[:0]

	// FPS:60
	b = append(b, "FPS:"...)
	b = strconv.AppendInt(b, int64(pt.FPS+0.5), 10)

	// | U:2.1ms D:4.3ms
	b = append(b, " | U:"...)
	b = strconv.AppendFloat(b, pt.AvgUpdateMs, 'f', 1, 64)
	b = append(b, "ms D:"...)
	b = strconv.AppendFloat(b, pt.AvgDrawMs, 'f', 1, 64)
	b = append(b, "ms"...)

	// | P99:3.2/6.1
	b = append(b, " | P99:"...)
	b = strconv.AppendFloat(b, pt.P99UpdateMs, 'f', 1, 64)
	b = append(b, '/')
	b = strconv.AppendFloat(b, pt.P99DrawMs, 'f', 1, 64)

	// | GC:2/s
	b = append(b, " | GC:"...)
	b = strconv.AppendUint(b, uint64(pt.GCCount), 10)
	b = append(b, "/s"...)

	// | Heap:12M
	b = append(b, " | Heap:"...)
	b = strconv.AppendInt(b, int64(pt.HeapMB+0.5), 10)
	b = append(b, 'M')

	text := string(b) // single allocation for the final string
	fm.DrawCenteredText(screen, text, float64(game.ScreenWidth)/2, float64(barY)+2, theme.FontCaption, theme.DebugStatsClr)
}

// countSlice 尝试获取切片长度（辅助函数）。
func countSlice(v interface{}) int {
	if v == nil {
		return 0
	}
	// 尝试常见的切片接口
	type hasLen interface {
		Len() int
	}
	if s, ok := v.(hasLen); ok {
		return s.Len()
	}
	return 0
}
