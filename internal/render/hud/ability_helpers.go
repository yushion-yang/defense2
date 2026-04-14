// ability_helpers.go — 能力显示相关的共享工具。
// 提供 AbilitySegment → ui.TextSegment 转换，供 choice_panel / build_menu / info_panel 使用。
package hud

import (
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"
)

// abilitySegsToTextSegs converts AbilitySegment slice to ui.TextSegment slice,
// mapping Kind to the appropriate theme color.
func abilitySegsToTextSegs(segs []AbilitySegment) []ui.TextSegment {
	out := make([]ui.TextSegment, len(segs))
	for i, seg := range segs {
		var clr color.Color
		switch seg.Kind {
		case "text":
			clr = theme.TextMuted
		case "base", "total":
			clr = theme.TextBody
		case "scaled":
			clr = seg.Color
			if clr == nil {
				clr = theme.TextBody
			}
		default:
			clr = theme.TextBody
		}
		out[i] = ui.TextSegment{Text: seg.Text, Color: clr}
	}
	return out
}

// measureAbilityRowHeight 计算一个能力行渲染后的实际高度（含段落换行）。
// iconW: 图标宽度（含间距），labelW: 标签文本宽度（含间距）。
func measureAbilityRowHeight(fm *render.FontManager, ab AbilityVM, w float64) float64 {
	if fm == nil {
		return 20
	}
	const iconLabelW = 19.0 + 6.0 // icon(16) + gap(3) + label gap(6)
	labelW := fm.MeasureText(ab.Label, theme.FontSM) + 4

	if len(ab.Segments) > 0 {
		segs := abilitySegsToTextSegs(ab.Segments)
		remainW := w - iconLabelW - labelW
		if remainW < 40 {
			remainW = 40
		}
		h := ui.MeasureSegmentsWrappedHeight(fm, segs, remainW, theme.FontSM)
		if h < 20 {
			return 20
		}
		return h
	}
	// Fallback 单行
	return 20
}
