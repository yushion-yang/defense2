// ai_overlay.go — AI 玩家精灵和思维气泡渲染。
//
// 遵循 HUD ViewModel 模式：接受纯值 VM，用 draw/ui 组件绘制。
// 在世界空间渲染（受相机偏移），精灵正上方显示气泡。
package hud

import (
	"image/color"

	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// AIOverlayVM AI 覆盖层 ViewModel。
type AIOverlayVM struct {
	// 精灵
	SpriteX, SpriteY float64       // 世界坐标
	SpriteImg        *ebiten.Image // 精灵图片（nil 时画默认光球）
	SpriteAlpha      float64       // 0-1

	// 气泡
	BubbleVisible bool
	BubbleText    string
	BubbleAlpha   float64

	// 区域
	ZoneSplitX float64 // 区域分界线 X（世界坐标）
	ShowZone   bool
	MapHeight  float64 // 地图像素高度（画分界线用）
	OwnerIndex int     // AI 所有者编号（用于区分多个 AI 的颜色/位置）
}

const (
	aiSpriteR       float32 = 14  // 精灵半径
	aiBubbleOffsetY         = -32 // 气泡距精灵中心的 Y 偏移
	aiBubbleMaxW            = 140 // 气泡最大宽度
	aiBubblePadH            = 8   // 水平内边距
	aiBubblePadV            = 4   // 垂直内边距
)

// DrawAIOverlay 渲染 AI 精灵和气泡。在 drawScene 的世界空间阶段调用。
func DrawAIOverlay(screen *ebiten.Image, vm AIOverlayVM) {
	if vm.SpriteAlpha <= 0 {
		return
	}

	// ── 区域分界虚线 ──
	if vm.ShowZone && vm.ZoneSplitX > 0 {
		h := float32(vm.MapHeight)
		if h <= 0 {
			h = 780
		}
		lineClr := color.NRGBA{R: 100, G: 180, B: 255, A: 35}
		draw.DashedLine(screen, float32(vm.ZoneSplitX), 0, float32(vm.ZoneSplitX), h, 1, 8, 6, lineClr) //nolint:hud
	}

	cx, cy := vm.SpriteX, vm.SpriteY

	// ── 精灵 ──
	if vm.SpriteImg != nil {
		scale := float64(aiSpriteR) * 2 / float64(vm.SpriteImg.Bounds().Dx())
		draw.SpriteScaledRotatedAlpha(screen, vm.SpriteImg, cx, cy, scale, 0, vm.SpriteAlpha) //nolint:hud
	} else {
		// 默认光球：颜色由 OwnerIndex 决定
		baseClr := theme.AIOwnerColors[0] // fallback 蓝色
		if vm.OwnerIndex >= 1 && vm.OwnerIndex <= len(theme.AIOwnerColors) {
			baseClr = theme.AIOwnerColors[vm.OwnerIndex-1]
		}
		a := uint8(float64(baseClr.A) * vm.SpriteAlpha)
		fcx, fcy := float32(cx), float32(cy)
		draw.FilledCircle(screen, fcx, fcy, aiSpriteR, color.NRGBA{R: baseClr.R, G: baseClr.G, B: baseClr.B, A: a}) //nolint:hud
		outA := uint8(float64(a) * 0.5)
		draw.CircleOutline(screen, fcx, fcy, aiSpriteR+3, 1, color.NRGBA{R: baseClr.R, G: baseClr.G, B: baseClr.B, A: outA}) //nolint:hud
	}

	// ── 思维气泡 ──
	if !vm.BubbleVisible || vm.BubbleText == "" || vm.BubbleAlpha <= 0 {
		return
	}
	drawAIBubble(screen, cx, cy+aiBubbleOffsetY, vm.BubbleText, vm.BubbleAlpha)
}

// drawAIBubble 渲染思维气泡。
func drawAIBubble(screen *ebiten.Image, cx, cy float64, text string, alpha float64) {
	fontSize := theme.FontXS
	bw := float64(aiBubbleMaxW)
	bh := float64(fontSize) + aiBubblePadV*2
	bx := cx - bw/2
	by := cy - bh

	bgA := uint8(200 * alpha)
	draw.RoundRect(screen, float32(bx), float32(by), float32(bw), float32(bh), 4, //nolint:hud
		color.NRGBA{R: 30, G: 30, B: 40, A: bgA})

	textA := uint8(255 * alpha)
	ui.LabelV(screen, text, cx, by+bh/2, bw-aiBubblePadH*2, ui.LabelStyle{
		Font:  float64(fontSize),
		Color: color.NRGBA{R: 255, G: 255, B: 255, A: textA},
		Align: ui.AlignCenter,
	})
}
