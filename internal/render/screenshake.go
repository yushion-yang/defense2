// screenshake.go — 屏幕震动效果。
// 全局单例，在 Draw 阶段对画面施加短暂的随机偏移。
package render

import "math/rand"

// ScreenShake 屏幕震动状态。
type ScreenShake struct {
	Timer    float64 // 剩余时间（秒）
	Duration float64 // 总持续时间（秒）
	Strength float64 // 最大偏移像素
}

var (
	shake        ScreenShake
	shakeEnabled bool // 震动开关，false=关闭
)

// ResetShake 清除残留的屏幕震动状态。
func ResetShake() { shake = ScreenShake{} }

// SetShakeEnabled 设置屏幕震动开关。
func SetShakeEnabled(enabled bool) { shakeEnabled = enabled }

// ShakeEnabled 返回屏幕震动是否启用。
func ShakeEnabled() bool { return shakeEnabled }

// TriggerShake 触发屏幕震动。
// strength: 最大偏移像素（推荐 2-6），duration: 持续时间秒（推荐 0.15-0.4）。
func TriggerShake(strength, duration float64) {
	if !shakeEnabled {
		return
	}
	// 不覆盖更强的震动
	if shake.Timer > 0 && shake.Strength > strength {
		return
	}
	shake.Timer = duration
	shake.Duration = duration
	shake.Strength = strength
}

// UpdateShake 每帧更新震动计时器。
func UpdateShake(dt float64) {
	if shake.Timer > 0 {
		shake.Timer -= dt
		if shake.Timer < 0 {
			shake.Timer = 0
		}
	}
}

// ShakeOffset 返回当前帧的震动偏移量 (dx, dy)。
// 在 Draw 中对整个游戏画面应用此偏移。
func ShakeOffset() (float64, float64) {
	if shake.Timer <= 0 {
		return 0, 0
	}
	ratio := shake.Timer / shake.Duration
	str := shake.Strength * ratio
	dx := (rand.Float64()*2 - 1) * str
	dy := (rand.Float64()*2 - 1) * str
	return dx, dy
}
