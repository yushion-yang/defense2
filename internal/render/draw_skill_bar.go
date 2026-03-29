// draw_skill_bar.go — 技能 CD 进度条渲染。
package render

import (
	"image/color"

	"defense2/internal/core/skill"
	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	skillBarW    float32 = 30
	skillBarH    float32 = 4
	skillBarOffY float64 = -20
)

var (
	skillBarBg    = color.RGBA{R: 20, G: 20, B: 30, A: 200}
	skillBarCD    = color.RGBA{R: 70, G: 100, B: 160, A: 220}
	skillBarReady = color.RGBA{R: 50, G: 210, B: 140, A: 240}
	skillBarBdr   = color.RGBA{R: 80, G: 100, B: 140, A: 150}
)

// DrawSkillBar 在 (cx, cy) 上方绘制技能 CD 进度条。
func DrawSkillBar(screen *ebiten.Image, cx, cy float64, state *skill.SkillState, owner interface{}) {
	if state == nil || state.Skill == nil {
		return
	}
	ratio, ready := state.Skill.GetProgress(owner)

	x := float32(cx) - skillBarW/2
	y := float32(cy+skillBarOffY) - skillBarH/2

	// 边框 + 背景
	draw.FilledRect(screen, x-1, y-1, skillBarW+2, skillBarH+2, skillBarBdr, false)
	draw.FilledRect(screen, x, y, skillBarW, skillBarH, skillBarBg, false)

	// 进度填充
	fillW := skillBarW * float32(ratio)
	if fillW < 0 {
		fillW = 0
	}
	if fillW > skillBarW {
		fillW = skillBarW
	}
	clr := skillBarCD
	if ready {
		clr = skillBarReady
		fillW = skillBarW
	}
	if fillW > 0 {
		draw.FilledRect(screen, x, y, fillW, skillBarH, clr, false)
	}
}
