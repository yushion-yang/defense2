//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/aiplayer/llm"
)

// TestLLMMemoryAdd 验证短期记忆添加和容量限制。
func TestAIPlayerLLMMemoryAdd(t *testing.T) {
	mem := llm.NewMemory()

	if mem.Len() != 0 {
		t.Errorf("new memory len = %d, want 0", mem.Len())
	}

	mem.Add("built tower", "搞定了")
	mem.Add("upgraded", "升级完成")
	mem.Add("started wave", "来了来了")

	if mem.Len() != 3 {
		t.Errorf("after 3 adds, len = %d, want 3", mem.Len())
	}

	// 第 4 条应该挤掉第 1 条
	mem.Add("idle", "等等")
	if mem.Len() != 3 {
		t.Errorf("after 4 adds, len = %d, want 3 (capacity limit)", mem.Len())
	}

	entries := mem.Entries()
	if entries[0].Event != "upgraded" {
		t.Errorf("oldest entry event = %q, want 'upgraded' (first should be evicted)", entries[0].Event)
	}
	if entries[2].Event != "idle" {
		t.Errorf("newest entry event = %q, want 'idle'", entries[2].Event)
	}
}

// TestLLMMemoryClear 验证记忆清空。
func TestAIPlayerLLMMemoryClear(t *testing.T) {
	mem := llm.NewMemory()
	mem.Add("a", "b")
	mem.Add("c", "d")
	mem.Clear()

	if mem.Len() != 0 {
		t.Errorf("after clear, len = %d, want 0", mem.Len())
	}
}

// TestLLMMemoryEntries 验证 Entries 返回副本。
func TestAIPlayerLLMMemoryEntries(t *testing.T) {
	mem := llm.NewMemory()
	mem.Add("event1", "response1")

	entries := mem.Entries()
	entries[0].Event = "modified"

	// 原始数据不应受影响
	original := mem.Entries()
	if original[0].Event != "event1" {
		t.Errorf("entries returned reference, not copy")
	}
}

// TestLLMBuildPromptWithMemory 验证带记忆的 prompt 生成。
func TestAIPlayerLLMBuildPromptWithMemory(t *testing.T) {
	mem := llm.NewMemory()
	mem.Add("built tower", "搞定了")
	mem.Add("started wave", "来了来了")

	situation := llm.Situation{
		Wave:            3,
		MaxWaves:        12,
		AIGold:          150,
		HumanGold:       100,
		Lives:           18,
		MaxLives:        20,
		AITowerCount:    3,
		HumanTowerCount: 2,
		LastAction:      "upgraded",
		ThreatLevel:     "medium",
		PersonalityDesc: "你的性格: 激进，喜欢DPS塔。",
		Mood:            "excited",
	}

	prompt := llm.BuildPrompt(situation, mem)

	// 验证 prompt 包含关键内容
	checks := []struct {
		name    string
		content string
	}{
		{"wave info", "第 3/12 波"},
		{"AI gold", "你的金币: 150"},
		{"human gold", "队友金币: 100"},
		{"lives", "共享生命: 18/20"},
		{"personality", "激进"},
		{"mood", "excited"},
		{"memory", "你之前说过"},
		{"memory entry", "搞定了"},
		{"no emoji", "不要用emoji"},
	}

	for _, check := range checks {
		if !containsStr(prompt, check.content) {
			t.Errorf("prompt missing %s: should contain %q\nprompt:\n%s", check.name, check.content, prompt)
		}
	}
}

// TestLLMBuildPromptWithoutMemory 验证无记忆时 prompt 不含 "你之前说过"。
func TestAIPlayerLLMBuildPromptWithoutMemory(t *testing.T) {
	situation := llm.Situation{
		Wave: 1, MaxWaves: 12,
		AIGold: 100, Lives: 20, MaxLives: 20,
		LastAction: "idle", ThreatLevel: "low",
	}
	prompt := llm.BuildPrompt(situation, nil)
	if containsStr(prompt, "你之前说过") {
		t.Error("prompt should not contain memory section when memory is nil")
	}

	emptyMem := llm.NewMemory()
	prompt2 := llm.BuildPrompt(situation, emptyMem)
	if containsStr(prompt2, "你之前说过") {
		t.Error("prompt should not contain memory section when memory is empty")
	}
}

// containsStr 简单子串检查。
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
