// debug_overlay.go — 调试覆盖层（F2 切换）。
// 显示射程圈、瞄准线、实体统计等调试可视化信息。
package hud

import (
	"strconv"

	"defense2/internal/core/game"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// PerfVM 性能统计的展示数据（纯值类型，无核心依赖）。
type PerfVM struct {
	FPS         float64
	AvgUpdateMs float64
	AvgDrawMs   float64
	P99UpdateMs float64
	P99DrawMs   float64
	GCCount     uint32
	HeapMB      float64
}

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

	// 统计文本（strconv + stack buffer 避免 fmt.Sprintf 分配）
	var buf [64]byte
	b := buf[:0]
	b = append(b, "塔:"...)
	b = strconv.AppendInt(b, int64(towerCount), 10)
	b = append(b, "  敌:"...)
	b = strconv.AppendInt(b, int64(enemyCount), 10)
	b = append(b, "  光束:"...)
	b = strconv.AppendInt(b, int64(beamCount), 10)
	b = append(b, "  弹:"...)
	b = strconv.AppendInt(b, int64(projCount), 10)
	stats := string(b)
	textClr := theme.DebugStatsClr
	fm.DrawCenteredText(screen, stats, float64(game.ScreenWidth)/2, float64(barY)+2, theme.FontCaption, textClr)
}

// DrawPerf 绘制性能统计栏（紧贴 DrawHUD 实体栏下方）。
// 使用 strconv + stack buffer 避免 fmt.Sprintf 分配。
// Format: "FPS:60 | U:2.1ms D:4.3ms | P99:3.2/6.1 | GC:2/s | Heap:12M"
func (o *DebugOverlay) DrawPerf(screen *ebiten.Image, pt PerfVM) {
	if !o.Enabled {
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

