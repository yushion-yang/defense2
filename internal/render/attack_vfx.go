// attack_vfx.go — 攻击特效系统。
// 管理枪口闪光、蓄力发光等塔攻击视觉效果。
package render

// AttackVFXState 攻击特效状态（挂载在塔上）。
type AttackVFXState struct {
	EffectScale float64 // 效果缩放（0.8~1.2，基于战力）
	MuzzleFlash float64 // 枪口闪光剩余时间
	ChargeGlow  float64 // 蓄力发光强度(0~1)
}

// CalcEffectScale 根据战力计算特效缩放系数。
// 战力 0 → 0.8, 战力 200+ → 1.2，线性插值。
func CalcEffectScale(strength float64) float64 {
	t := strength / 200.0
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return 0.8 + 0.4*t
}

// UpdateAttackVFX 每帧衰减攻击特效状态。
func UpdateAttackVFX(state *AttackVFXState, dt float64) {
	if state == nil {
		return
	}
	// 枪口闪光衰减
	if state.MuzzleFlash > 0 {
		state.MuzzleFlash -= dt
		if state.MuzzleFlash < 0 {
			state.MuzzleFlash = 0
		}
	}
	// 蓄力发光衰减（每秒衰减 2.0）
	if state.ChargeGlow > 0 {
		state.ChargeGlow -= dt * 2.0
		if state.ChargeGlow < 0 {
			state.ChargeGlow = 0
		}
	}
}

// TriggerMuzzleFlash 触发枪口闪光效果。
func TriggerMuzzleFlash(state *AttackVFXState, duration float64) {
	if state == nil {
		return
	}
	state.MuzzleFlash = duration
}
