// economy.go — 经济系统。
// 管理金币收入来源：击杀奖励、波次通过奖励、利息。
package economy

// Config 经济系统配置。
type Config struct {
	KillReward      int     // 每击杀一个敌人的基础金币
	WaveBonus       int     // 每通过一波的基础奖励
	WaveBonusScale  int     // 每波递增的奖励
	InterestRate    float64 // 利息比例（每波结算时）
	InterestCap     int     // 利息上限
	SellRefundRatio float64 // 卖塔返还比例
}

// DefaultConfig 返回默认经济配置。
func DefaultConfig() Config {
	return Config{
		KillReward:      15,
		WaveBonus:       30,
		WaveBonusScale:  5,
		InterestRate:    0.05,
		InterestCap:     50,
		SellRefundRatio: 0.5,
	}
}

// KillGold 计算击杀奖励。
func (c Config) KillGold() int {
	return c.KillReward
}

// WaveCompleteGold 计算波次通过奖励（波数越高奖励越多）。
func (c Config) WaveCompleteGold(wave int) int {
	return c.WaveBonus + wave*c.WaveBonusScale
}

// InterestGold 计算利息收入。
func (c Config) InterestGold(currentGold int) int {
	interest := int(float64(currentGold) * c.InterestRate)
	if interest > c.InterestCap {
		return c.InterestCap
	}
	return interest
}

// SellRefund 计算卖塔返还金额。
func (c Config) SellRefund(cost int) int {
	return int(float64(cost) * c.SellRefundRatio)
}
