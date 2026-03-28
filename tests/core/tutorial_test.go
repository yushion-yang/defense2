package core_test

import (
	"testing"

	"defense2/internal/core/tutorial"
)

func TestTutorialProgression(t *testing.T) {
	tut := tutorial.DefaultTutorial()

	// 初始状态
	if tut.CurrentMessage() == "" {
		t.Fatal("初始应有教程消息")
	}
	if tut.IsComplete() {
		t.Fatal("初始教程不应已完成")
	}

	// 错误事件不推进
	changed := tut.OnEvent("wrongEvent")
	if changed {
		t.Fatal("错误事件不应推进教程")
	}

	// 正确事件逐步推进
	events := []string{"gameStart", "towerBuilt", "waveStarted", "enemyKilled", "waveCleared"}
	for i, ev := range events {
		changed = tut.OnEvent(ev)
		if !changed {
			t.Fatalf("第 %d 步应推进，事件 %s", i+1, ev)
		}
	}

	if !tut.IsComplete() {
		t.Fatal("全部事件触发后教程应完成")
	}
	if tut.CurrentMessage() != "" {
		t.Fatal("完成后不应有消息")
	}
}

func TestTutorialSkip(t *testing.T) {
	tut := tutorial.DefaultTutorial()
	tut.Skip()
	if !tut.IsComplete() {
		t.Fatal("跳过后应标记完成")
	}
}
