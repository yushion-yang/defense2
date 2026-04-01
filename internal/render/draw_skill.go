// draw_skill.go — 技能视觉效果 stub。
// TODO: 完整实现技能 VFX 和冷却进度条。
package render

import (
	"defense2/internal/core/skill"

	"github.com/hajimehoshi/ebiten/v2"
)

// UpdateVFXTick 每帧更新全局 VFX 状态（stub）。
func UpdateVFXTick(dt float64) {
	// TODO: update skill/ability VFX timers
}

// DrawSkillVFX 绘制技能视觉特效（stub）。
func DrawSkillVFX(screen *ebiten.Image, state *skill.SkillState) {
	// TODO: implement skill visual effects
}

// DrawSkillBar 绘制技能冷却进度条（stub）。
func DrawSkillBar(screen *ebiten.Image, x, y float64, state *skill.SkillState, owner interface{}) {
	// TODO: implement skill cooldown bar
}
