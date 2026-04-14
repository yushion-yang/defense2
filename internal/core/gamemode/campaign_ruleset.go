// campaign_ruleset.go — 战役模式塔建造规则。
//
// 嵌入 baseTowerRuleset，所有方法使用默认值（= campaign 行为）。
// 显式定义此类型是为了语义明确：campaign 的规则是有意识的设计决策，不是"恰好等于默认"。
package gamemode

// CampaignRuleset 战役模式（含娱乐战役）的塔建造规则。
// 特征：花钱解锁能力 3 选 1、无强度上限、概率道具掉落。
type CampaignRuleset struct{ baseTowerRuleset }
