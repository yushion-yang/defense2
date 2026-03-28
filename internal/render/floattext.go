// floattext.go — 浮动文本系统。
// 管理屏幕上的临时飘字（伤害数字、金币获取等），自动向上漂浮并淡出。
package render

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// FloatText 单条浮动文本。
type FloatText struct {
	X, Y    float64    // 当前位置
	StartY  float64    // 起始 Y（用于计算偏移）
	Text    string     // 显示文本
	Color   color.RGBA // 基础颜色（alpha 会随时间衰减）
	Size    float64    // 字体大小
	Life    float64    // 剩余存活时间（秒）
	MaxLife float64    // 最大存活时间（秒）
	Active  bool       // 是否存活
}

const maxFloatTexts = 64

var floatTexts [maxFloatTexts]FloatText
var ftCursor int

// SpawnDamageText 在指定位置弹出伤害数字。
func SpawnDamageText(x, y, damage float64, crit bool) {
	clr := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	size := 11.0
	if crit {
		clr = color.RGBA{R: 255, G: 80, B: 60, A: 255}
		size = 14.0
	}
	text := fmt.Sprintf("%.0f", damage)
	if damage >= 100 {
		text = fmt.Sprintf("%.0f", damage)
	}
	spawnFloatText(x, y, text, clr, size, 0.8)
}

// SpawnGoldText 在指定位置弹出金币获取文字。
func SpawnGoldText(x, y float64, amount int) {
	clr := color.RGBA{R: 255, G: 215, B: 0, A: 255}
	spawnFloatText(x, y, fmt.Sprintf("+%d", amount), clr, 11, 1.0)
}

// SpawnKillText 在指定位置弹出击杀文字。
func SpawnKillText(x, y float64) {
	clr := color.RGBA{R: 200, G: 60, B: 60, A: 255}
	spawnFloatText(x, y-10, "KILL", clr, 10, 0.6)
}

func spawnFloatText(x, y float64, text string, clr color.RGBA, size, life float64) {
	ft := &floatTexts[ftCursor]
	ft.X = x
	ft.Y = y
	ft.StartY = y
	ft.Text = text
	ft.Color = clr
	ft.Size = size
	ft.Life = life
	ft.MaxLife = life
	ft.Active = true
	ftCursor = (ftCursor + 1) % maxFloatTexts
}

// UpdateFloatTexts 每帧更新所有浮动文本（上移 + 衰减）。
func UpdateFloatTexts(dt float64) {
	for i := range floatTexts {
		ft := &floatTexts[i]
		if !ft.Active {
			continue
		}
		ft.Life -= dt
		if ft.Life <= 0 {
			ft.Active = false
			continue
		}
		// 向上漂浮（先快后慢，使用 ease-out）
		progress := 1 - ft.Life/ft.MaxLife // 0→1
		ft.Y = ft.StartY - 30*easeOutQuad(progress)
	}
}

// DrawFloatTexts 渲染所有存活的浮动文本。
func DrawFloatTexts(screen *ebiten.Image) {
	fm := GlobalFont()
	if fm == nil {
		return
	}
	for i := range floatTexts {
		ft := &floatTexts[i]
		if !ft.Active {
			continue
		}
		// Alpha 衰减
		progress := 1 - ft.Life/ft.MaxLife
		alpha := uint8(float64(ft.Color.A) * (1 - progress))
		clr := color.RGBA{R: ft.Color.R, G: ft.Color.G, B: ft.Color.B, A: alpha}

		// 微缩放效果：出生时稍大，然后恢复
		scale := 1.0
		if progress < 0.15 {
			scale = 1.0 + 0.3*(1-progress/0.15) // 1.3→1.0
		}
		size := ft.Size * scale

		fm.DrawCenteredText(screen, ft.Text, ft.X, ft.Y, size, clr)
	}
}

func easeOutQuad(t float64) float64 {
	return 1 - math.Pow(1-t, 2)
}
