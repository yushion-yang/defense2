// scroll_panel.go — 可滚动面板组件。
// 在固定高度的可视区域内渲染超高内容，支持鼠标滚轮和触摸拖动。
// 用于 info_panel 的能力/buff 列表等内容可变的 HUD 区域。
package ui

import (
	"image/color"

	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// ScrollState 滚动状态（由调用方持有，跨帧保持）。
type ScrollState struct {
	OffsetY    float32 // 当前滚动偏移（向下为正）
	ContentH   float32 // 内容总高度（由渲染回调设定）
	ViewH      float32 // 可视区域高度
	isDragging bool
	dragStartY float32
	dragStartO float32
}

// Scroll 接收鼠标滚轮增量，更新偏移。deltaY > 0 表示向下滚。
func (s *ScrollState) Scroll(deltaY float64) {
	s.OffsetY -= float32(deltaY) * 20
	s.clamp()
}

// BeginDrag 开始触摸拖动。
func (s *ScrollState) BeginDrag(y float32) {
	s.isDragging = true
	s.dragStartY = y
	s.dragStartO = s.OffsetY
}

// UpdateDrag 更新触摸拖动。
func (s *ScrollState) UpdateDrag(y float32) {
	if !s.isDragging {
		return
	}
	s.OffsetY = s.dragStartO - (y - s.dragStartY)
	s.clamp()
}

// EndDrag 结束拖动。
func (s *ScrollState) EndDrag() {
	s.isDragging = false
}

// CanScroll 返回是否需要滚动（内容超出可视区域）。
func (s *ScrollState) CanScroll() bool {
	return s.ContentH > s.ViewH
}

func (s *ScrollState) clamp() {
	if s.OffsetY < 0 {
		s.OffsetY = 0
	}
	maxOff := s.ContentH - s.ViewH
	if maxOff < 0 {
		maxOff = 0
	}
	if s.OffsetY > maxOff {
		s.OffsetY = maxOff
	}
}

// DrawScrollRegion 在指定区域内渲染可滚动内容。
// renderContent 回调接收 (screen, x, y, w)，其中 y 已减去滚动偏移。
// 回调渲染全部内容，本函数负责裁剪到可视区域。
// 返回是否显示了滚动条（用于调用方判断是否拦截滚轮事件）。
func DrawScrollRegion(screen *ebiten.Image, x, y, w, viewH float32, state *ScrollState,
	renderContent func(screen *ebiten.Image, x, y float64, w float64)) bool {

	state.ViewH = viewH

	// 通过偏移 Y 坐标 + 只渲染可见项实现裁剪
	// 渲染内容，y 坐标减去滚动偏移
	contentY := float64(y) - float64(state.OffsetY)
	renderContent(screen, float64(x), contentY, float64(w))

	// 滚动条指示器
	if !state.CanScroll() {
		return false
	}
	trackH := viewH - 4
	thumbRatio := viewH / state.ContentH
	thumbH := trackH * thumbRatio
	if thumbH < 10 {
		thumbH = 10
	}
	scrollRatio := state.OffsetY / (state.ContentH - viewH)
	thumbY := y + 2 + (trackH-thumbH)*scrollRatio

	draw.FilledRect(screen, x+w-3, thumbY, 2, thumbH, //nolint:hud
		color.RGBA{R: 80, G: 100, B: 140, A: 120}, false)
	return true
}

// ScrollRegionContains 检查点是否在滚动区域内。
func ScrollRegionContains(px, py, x, y, w, h float32) bool {
	return px >= x && px <= x+w && py >= y && py <= y+h
}
