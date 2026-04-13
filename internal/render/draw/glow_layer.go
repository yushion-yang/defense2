// glow_layer.go — 辉光离屏合成层。
//
// 将所有 Glow() 绘制重定向到一张离屏缓冲图像，最终通过加法混合(BlendLighter)
// 合成到主画面，实现类似 bloom 的柔和发光效果，且无需 shader 支持。
//
// 使用流程（每帧由 stage 的 Draw 驱动）：
//  1. BeginGlowPass(screen) — 开启离屏捕获
//  2. 正常调用 Glow() — 自动写入离屏缓冲而非 screen
//  3. EndGlowPass(screen)  — 将缓冲以加法混合叠加到 screen
//
// 设计选择：加法混合使辉光叠加后变亮而非变不透明，
// 符合"干净轮廓 > 堆叠辉光"的 VFX 设计原则。
package draw

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	glowTarget *ebiten.Image // 离屏辉光缓冲图像，尺寸与屏幕一致
	glowActive bool          // 当前是否处于辉光捕获阶段
)

// BeginGlowPass 开启辉光捕获阶段，后续 Glow() 调用将绘制到离屏缓冲。
// 如果离屏缓冲尚未创建或尺寸不匹配（如窗口大小变化），会重新分配。
// FilledCircle、Line 等非辉光绘制不受影响，仍然直接画到 screen。
func BeginGlowPass(screen *ebiten.Image) {
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	if glowTarget == nil || glowTarget.Bounds() != image.Rect(0, 0, w, h) {
		glowTarget = ebiten.NewImage(w, h)
	}
	glowTarget.Clear()
	glowActive = true
}

// EndGlowPass 将辉光缓冲以加法混合(BlendLighter)合成到主画面，并重置状态。
// BlendLighter 使得辉光区域颜色值相加，产生自然的"发光变亮"视觉效果。
// 安全调用：即使未调用过 BeginGlowPass，也不会 panic。
func EndGlowPass(screen *ebiten.Image) {
	if !glowActive || glowTarget == nil {
		glowActive = false
		return
	}
	glowActive = false

	op := &ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendLighter
	screen.DrawImage(glowTarget, op)
}

// GlowTarget 返回当前辉光离屏渲染目标。
// 辉光通道未激活时返回 nil，供 Glow() 内部判断重定向。
func GlowTarget() *ebiten.Image {
	if glowActive {
		return glowTarget
	}
	return nil
}

// GlowPassActive 返回辉光通道是否正在进行中。
func GlowPassActive() bool {
	return glowActive
}
