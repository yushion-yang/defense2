// debug_overlay.go — 调试覆盖层（F2 切换）。
// 显示射程圈、瞄准线、实体统计等调试可视化信息。
package hud

import (
	"fmt"
	"image/color"

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
	fm.DrawText(screen, text, 10, float64(game.ScreenHeight)-30, theme.FontXS,
		color.RGBA{R: 100, G: 200, B: 255, A: 180})
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

	// 半透明背景条
	barH := float32(20)
	bgClr := color.RGBA{R: 0, G: 0, B: 0, A: 150}
	draw.FilledRect(screen, 0, 0, float32(game.ScreenWidth), barH, bgClr, false)

	// 统计文本
	stats := fmt.Sprintf("塔:%d  敌:%d  光束:%d  弹:%d", towerCount, enemyCount, beamCount, projCount)
	textClr := color.RGBA{R: 180, G: 220, B: 255, A: 230}
	fm.DrawCenteredText(screen, stats, float64(game.ScreenWidth)/2, 2, theme.FontXS, textClr)
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
