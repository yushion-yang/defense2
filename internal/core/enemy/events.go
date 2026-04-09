// events.go — 波次敌人事件系统。
// 处理波次配置中的敌人增益事件（护盾、血量、回血、速度、奖励等），
// 以及精英晋升逻辑。
package enemy

// ── 速度上限 ──

// maxSpeedScale 速度百分比加成的上限倍率（防止敌人过快）。
const maxSpeedScale = 2.4

// ApplyEnemyEvent 对敌人施加波次事件增益。
// kind 支持 4 种类型：
//   - "hpPercent": 按最大血量百分比增加血量/最大血量
//   - "periodicHealPercent": 设置每秒回血量（按最大血量百分比）
//   - "speedPercent": 按百分比提升基础速度（上限 2.4x）
//   - "rewardPercent": 按百分比提升击杀奖励（最少 1 金币）
func ApplyEnemyEvent(e *Enemy, kind string, value float64) {
	switch kind {
	case "hpPercent":
		// 增加血量和最大血量
		bonus := e.MaxHP * value
		e.MaxHP += bonus
		e.HP += bonus

	case "periodicHealPercent":
		// 设置每秒回血量（按最大血量百分比）
		e.RegenPerSec = e.MaxHP * value

	case "speedPercent":
		// 按百分比提升基础速度，上限 2.4 倍原始速度
		e.BaseSpeed *= 1 + value
		cap := e.BaseSpeed // 已乘后的值就是当前值
		origSpeed := e.BaseSpeed / (1 + value)
		maxSpeed := origSpeed * maxSpeedScale
		if cap > maxSpeed {
			e.BaseSpeed = maxSpeed
		}
		// 同步当前速度（如果未被减速）
		if e.SlowTimer <= 0 {
			e.Speed = e.BaseSpeed
		}

	case "rewardPercent":
		// 按百分比提升奖励，最少 1 金币
		e.Reward = int(float64(e.Reward) * (1 + value))
		if e.Reward < 1 {
			e.Reward = 1
		}
	}
}

// ApplyElitePromotion 已移除（精英概念取消）。
