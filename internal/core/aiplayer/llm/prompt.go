// prompt.go -- 局势摘要 -> LLM prompt 构建。
//
// 将游戏状态快照转化为自然语言描述，作为 LLM 的输入。
// 包含短期记忆系统（最近 3 次交互），让 LLM 产生连贯对话。
// 关联：由 aiplayer.go 的 buildSituation() 填充 Situation，
//       connector.go 的 callAPI() 使用 BuildPrompt() 生成 prompt。
package llm

import (
	"fmt"
	"strings"
)

// Situation 游戏局势摘要（纯值结构体）。
type Situation struct {
	Wave, MaxWaves    int
	AIGold, HumanGold int
	Lives, MaxLives   int
	AITowerCount      int
	HumanTowerCount   int
	LastAction        string // "built tower" / "upgraded" / "idle"
	ThreatLevel       string // "low" / "medium" / "high"

	// ── Phase 2-3 新增字段 ──
	PersonalityDesc string // 个性描述（如 "你性格激进，喜欢DPS塔"）
	Mood            string // 当前情绪（如 "excited" / "worried" / "calm"）
}

// ── 短期记忆系统 ──

const memoryCapacity = 3 // 最多保留 3 条记忆

// MemoryEntry 记忆条目：一次事件和对应的 LLM 输出。
type MemoryEntry struct {
	Event    string // "built tower at row 5" / "player leaked" 等
	Response string // 上次 LLM 的输出
}

// Memory 短期记忆（环形缓冲区，最多 3 条）。
type Memory struct {
	entries []MemoryEntry
}

// NewMemory 创建空记忆。
func NewMemory() *Memory {
	return &Memory{
		entries: make([]MemoryEntry, 0, memoryCapacity),
	}
}

// Add 添加一条记忆。超过容量时丢弃最旧的。
func (m *Memory) Add(event, response string) {
	if len(m.entries) >= memoryCapacity {
		// 移除最旧的条目
		m.entries = m.entries[1:]
	}
	m.entries = append(m.entries, MemoryEntry{Event: event, Response: response})
}

// Entries 返回所有记忆条目（只读副本）。
func (m *Memory) Entries() []MemoryEntry {
	copied := make([]MemoryEntry, len(m.entries))
	copy(copied, m.entries)
	return copied
}

// Len 返回当前记忆条目数量。
func (m *Memory) Len() int {
	return len(m.entries)
}

// Clear 清空记忆。
func (m *Memory) Clear() {
	m.entries = m.entries[:0]
}

// BuildPrompt 根据局势生成 LLM prompt。
// 包含个性描述、情绪状态、短期记忆和具体输出要求。
func BuildPrompt(s Situation, memory *Memory) string {
	var sb strings.Builder

	// ── 角色设定 ──
	sb.WriteString("你是塔防游戏中的 AI 玩家。你正在和一个人类队友一起守关。\n")

	// 个性描述
	if s.PersonalityDesc != "" {
		sb.WriteString(s.PersonalityDesc)
		sb.WriteString("\n")
	}

	// 情绪状态
	if s.Mood != "" {
		sb.WriteString(fmt.Sprintf("你现在的心情: %s\n", s.Mood))
	}

	// ── 短期记忆 ──
	if memory != nil && memory.Len() > 0 {
		sb.WriteString("\n你之前说过:\n")
		for _, e := range memory.Entries() {
			if e.Response != "" {
				sb.WriteString(fmt.Sprintf("- [%s时] \"%s\"\n", e.Event, e.Response))
			}
		}
	}

	// ── 当前局势 ──
	sb.WriteString(fmt.Sprintf(`
当前局势:
- 第 %d/%d 波
- 你的金币: %d, 队友金币: %d
- 共享生命: %d/%d
- 你有 %d 座塔, 队友有 %d 座塔
- 你刚刚: %s
- 威胁等级: %s
`,
		s.Wave, s.MaxWaves,
		s.AIGold, s.HumanGold,
		s.Lives, s.MaxLives,
		s.AITowerCount, s.HumanTowerCount,
		s.LastAction,
		s.ThreatLevel,
	))

	// ── 输出要求 ──
	sb.WriteString(`用一句话（15字以内）表达你当前的想法或对局势的反应。
要求：口语化中文、有个性、偶尔吐槽。不要用emoji。只输出这一句话，不要任何解释。`)

	return sb.String()
}
