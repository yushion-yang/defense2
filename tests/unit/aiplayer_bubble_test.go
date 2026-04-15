//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/aiplayer"
)

func TestBubbleLifecycle(t *testing.T) {
	bm := aiplayer.NewBubbleManager()

	bm.Show("测试消息", aiplayer.BubbleThink, 2.0)
	if !bm.Visible() {
		t.Fatal("bubble should be visible after Show")
	}
	if bm.Text() != "测试消息" {
		t.Errorf("text = %q, want %q", bm.Text(), "测试消息")
	}

	// 1 秒后仍可见
	bm.Tick(1.0)
	if !bm.Visible() {
		t.Fatal("bubble should still be visible after 1s")
	}

	// 再 1.5 秒后应消失
	bm.Tick(1.5)
	if bm.Visible() {
		t.Fatal("bubble should be hidden after 2.5s (duration=2.0)")
	}
}

func TestBubbleAlphaFadeout(t *testing.T) {
	bm := aiplayer.NewBubbleManager()
	bm.Show("fade", aiplayer.BubbleAction, 1.0)

	// 刚显示时 alpha=1
	if a := bm.Alpha(); a != 1.0 {
		t.Errorf("alpha at start = %f, want 1.0", a)
	}

	// 0.8s 后 (剩余 0.2s < 0.3s)，alpha 应 < 1
	bm.Tick(0.8)
	if a := bm.Alpha(); a >= 1.0 {
		t.Errorf("alpha at 0.2s remaining = %f, want < 1.0", a)
	}
}

func TestDialogueBank(t *testing.T) {
	bank := aiplayer.DefaultDialogueBank()

	categories := []string{"build_thinking", "build_done", "upgrade_thinking",
		"low_gold", "enemy_leak", "wave_start", "idle"}
	for _, cat := range categories {
		text := bank.Random(cat)
		if text == "" {
			t.Errorf("category %q returned empty text", cat)
		}
	}

	// 不存在的类别返回空
	if text := bank.Random("nonexistent"); text != "" {
		t.Errorf("nonexistent category returned %q, want empty", text)
	}
}
