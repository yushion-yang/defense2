// cooldown_channel.go — 冷却-通道状态机。
// 用于蓄力/引导类攻击：冷却结束 → 进入通道释放 → 按 tickRate 触发回调 → 通道结束 → 重新冷却。
package tower

// CooldownChannelState 冷却-通道状态机运行时状态。
type CooldownChannelState struct {
	Channeling bool    // 是否正在通道释放
	Timer      float64 // 冷却计时器
	Elapsed    float64 // 通道已用时间
	TickAccum  float64 // tick累加器
}

// CooldownChannelConfig 状态机配置。
type CooldownChannelConfig struct {
	Cooldown    float64              // 冷却时间（秒）
	Duration    float64              // 通道持续时间（秒）
	TickRate    float64              // tick间隔（秒）
	CanActivate func() bool          // 是否可激活判定
	OnActivate  func()               // 激活时回调
	OnTick      func() interface{}   // tick回调，返回命中结果
}

// CooldownChannelResult 单帧 tick 结果。
type CooldownChannelResult struct {
	Active   bool          // 是否正在通道释放
	Ready    bool          // 冷却是否就绪
	Progress float64       // 进度(0~1)
	Ticks    []interface{} // 本帧的 tick 结果
}

// InitCooldownChannel 创建初始冷却-通道状态。
func InitCooldownChannel() *CooldownChannelState {
	return &CooldownChannelState{}
}

// TickCooldownChannel 驱动冷却-通道状态机前进一帧。
func TickCooldownChannel(state *CooldownChannelState, cfg *CooldownChannelConfig, dt float64) CooldownChannelResult {
	result := CooldownChannelResult{}

	if state.Channeling {
		// 通道阶段：累加时间，按 tickRate 触发回调
		state.Elapsed += dt
		state.TickAccum += dt

		tickRate := cfg.TickRate
		if tickRate <= 0 {
			tickRate = 0.1 // 默认 tick 间隔
		}

		for state.TickAccum >= tickRate {
			state.TickAccum -= tickRate
			if cfg.OnTick != nil {
				if r := cfg.OnTick(); r != nil {
					result.Ticks = append(result.Ticks, r)
				}
			}
		}

		// 检查通道是否结束
		if state.Elapsed >= cfg.Duration {
			state.Channeling = false
			state.Elapsed = 0
			state.TickAccum = 0
			state.Timer = 0 // 重新开始冷却
		}

		result.Active = state.Channeling
		if cfg.Duration > 0 {
			result.Progress = state.Elapsed / cfg.Duration
		}
		return result
	}

	// 冷却阶段
	state.Timer += dt

	if cfg.Cooldown > 0 {
		result.Progress = state.Timer / cfg.Cooldown
		if result.Progress > 1 {
			result.Progress = 1
		}
	}

	if state.Timer >= cfg.Cooldown {
		result.Ready = true

		// 检查是否可以激活
		canActivate := true
		if cfg.CanActivate != nil {
			canActivate = cfg.CanActivate()
		}

		if canActivate {
			state.Channeling = true
			state.Elapsed = 0
			state.TickAccum = 0
			state.Timer = 0
			result.Active = true
			result.Ready = false

			if cfg.OnActivate != nil {
				cfg.OnActivate()
			}
		}
	}

	return result
}
