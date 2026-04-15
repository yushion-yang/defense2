// prompt.go — 局势摘要 -> LLM prompt 构建。
//
// 将游戏状态快照转化为自然语言描述，作为 LLM 的输入。
// 关联：由 aiplayer.go 的 buildSituation() 填充 Situation，
//       connector.go 的 callAPI() 使用 BuildPrompt() 生成 prompt。
package llm

import "fmt"

// Situation 游戏局势摘要（纯值结构体）。
type Situation struct {
	Wave, MaxWaves    int
	AIGold, HumanGold int
	Lives, MaxLives   int
	AITowerCount      int
	HumanTowerCount   int
	LastAction        string // "built tower" / "upgraded" / "idle"
	ThreatLevel       string // "low" / "medium" / "high"
}

// BuildPrompt 根据局势生成 LLM prompt。
// 要求模型输出一句 15 字以内的口语化评论。
func BuildPrompt(s Situation) string {
	return fmt.Sprintf(`你是塔防游戏中的 AI 玩家。你正在和一个人类队友一起守关。

当前局势:
- 第 %d/%d 波
- 你的金币: %d, 队友金币: %d
- 共享生命: %d/%d
- 你有 %d 座塔, 队友有 %d 座塔
- 你刚刚: %s
- 威胁等级: %s

用一句话（15字以内）表达你当前的想法或对局势的反应。
要求：口语化、有个性、偶尔吐槽。只输出这一句话，不要任何解释。`,
		s.Wave, s.MaxWaves,
		s.AIGold, s.HumanGold,
		s.Lives, s.MaxLives,
		s.AITowerCount, s.HumanTowerCount,
		s.LastAction,
		s.ThreatLevel,
	)
}
