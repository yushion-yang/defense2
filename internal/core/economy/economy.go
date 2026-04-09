// economy.go — 经济系统。
// 管理金币收入来源：击杀奖励、波次通过奖励。
//
// 此包虽然只有 ~30 行，但提供击杀/卖塔经济计算的单一入口，
// 被 stage.go 和多个测试文件引用。保留独立包可避免 scene 层
// 直接耦合 config.GlobalBalance() 的内部结构。
package economy

import "defense2/internal/config"

// Config 经济系统配置。
type Config struct {
	KillReward      int     // 每击杀一个敌人的基础金币
	SellRefundRatio float64 // 卖塔返还比例
}

// DefaultConfig 返回默认经济配置（从 balance.json 读取）。
func DefaultConfig() Config {
	bal := config.GlobalBalance()
	return Config{
		KillReward:      int(bal.Economy.KillReward),
		SellRefundRatio: bal.Economy.SellRefundRatio,
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
