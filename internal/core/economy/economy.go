// economy.go — 经济系统。
// 管理金币收入来源：击杀奖励、波次通过奖励。
package economy

// Config 经济系统配置。
type Config struct {
	KillReward      int     // 每击杀一个敌人的基础金币
	SellRefundRatio float64 // 卖塔返还比例
}

// DefaultConfig 返回默认经济配置。
func DefaultConfig() Config {
	return Config{
		KillReward:      15,
		SellRefundRatio: 0.7,
	}
}

// KillGold 计算击杀奖励。
func (c Config) KillGold() int {
	return c.KillReward
}

// SellRefund 计算卖塔返还金额。
func (c Config) SellRefund(cost int) int {
	return int(float64(cost) * c.SellRefundRatio)
}
