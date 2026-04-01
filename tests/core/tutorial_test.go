package core_test

import (
	"testing"

	"defense2/internal/core/tutorial"
)

func TestTutorialProgression(t *testing.T) {
	tut := tutorial.DefaultTutorial()

	// 初始状态：step 0 ("欢迎！点击任意位置继续")
	if tut.CurrentMessage() == "" {
		t.Fatal("初始应有教程消息")
	}
	if tut.IsComplete() {
		t.Fatal("初始教程不应已完成")
	}
	if tut.StepIndex() != 0 {
		t.Fatalf("初始步骤应为 0，实际 %d", tut.StepIndex())
	}

	// 错误事件不推进 click-advance 步骤
	changed := tut.Trigger("wrongEvent")
	if changed {
		t.Fatal("错误事件不应推进教程")
	}

	// Step 0: 欢迎（点击推进）
	changed = tut.ClickAdvance()
	if !changed {
		t.Fatal("step 0 应可点击推进")
	}
	if tut.StepIndex() != 1 {
		t.Fatalf("推进后应在 step 1，实际 %d", tut.StepIndex())
	}

	// Step 1: 点击「造塔」(event="build")
	changed = tut.ClickAdvance()
	if changed {
		t.Fatal("有事件要求的步骤不应点击推进")
	}
	changed = tut.Trigger("build")
	if !changed {
		t.Fatal("step 1 应被 build 事件推进")
	}

	// Step 2: 选择塔型放置 (event="tower_placed")
	changed = tut.Trigger("tower_placed")
	if !changed {
		t.Fatal("step 2 应被 tower_placed 事件推进")
	}

	// Step 3: 点击「开波」(event="wave_start")
	changed = tut.Trigger("wave_start")
	if !changed {
		t.Fatal("step 3 应被 wave_start 事件推进")
	}

	// Step 4: 选塔查看详情 (event="tower_select", autoAdvance=8)
	changed = tut.Trigger("tower_select")
	if !changed {
		t.Fatal("step 4 应被 tower_select 事件推进")
	}

	// Step 5: 升级 (event="upgrade", autoAdvance=8)
	changed = tut.Trigger("upgrade")
	if !changed {
		t.Fatal("step 5 应被 upgrade 事件推进")
	}

	// Step 6: 道具 (event="item_use", autoAdvance=10)
	changed = tut.Trigger("item_use")
	if !changed {
		t.Fatal("step 6 应被 item_use 事件推进")
	}

	// Step 7: 完成（autoAdvance=3，也可点击推进）
	if tut.IsComplete() {
		t.Fatal("step 7 是最后一步但教程不应提前完成")
	}
	changed = tut.ClickAdvance()
	if !changed {
		t.Fatal("step 7 应可点击推进")
	}

	if !tut.IsComplete() {
		t.Fatal("全部步骤完成后教程应标记完成")
	}
	if tut.CurrentMessage() != "" {
		t.Fatal("完成后不应有消息")
	}
	if tut.CurrentStep() != nil {
		t.Fatal("完成后 CurrentStep 应返回 nil")
	}
}

func TestTutorialAutoAdvance(t *testing.T) {
	tut := tutorial.DefaultTutorial()

	// 跳到 step 4 (tower_select, autoAdvance=8)
	tut.ClickAdvance()                // step 0 → 1
	tut.Trigger("build")             // step 1 → 2
	tut.Trigger("tower_placed")      // step 2 → 3
	tut.Trigger("wave_start")        // step 3 → 4

	if tut.StepIndex() != 4 {
		t.Fatalf("应在 step 4，实际 %d", tut.StepIndex())
	}

	// 模拟 7 秒（不够 8 秒自动推进）
	for i := 0; i < 420; i++ { // 420 * 1/60 ≈ 7s
		tut.Update(1.0 / 60.0)
	}
	if tut.StepIndex() != 4 {
		t.Fatal("7 秒不应自动推进")
	}

	// 再过 2 秒（总 9 秒，超过 8 秒限制）
	for i := 0; i < 120; i++ {
		tut.Update(1.0 / 60.0)
	}
	if tut.StepIndex() != 5 {
		t.Fatalf("9 秒后应自动推进到 step 5，实际 %d", tut.StepIndex())
	}
}

func TestTutorialOnEventCompatibility(t *testing.T) {
	tut := tutorial.DefaultTutorial()

	// Skip welcome step
	tut.ClickAdvance()

	// 旧事件名应通过 OnEvent 兼容映射推进
	// step 1: event="build" — OnEvent("build") 直接映射
	// 但 step 1 的 event 是 "build"，而旧代码没有这个事件
	// 所以新步骤需要新事件名

	// 跳到 step 2 先
	tut.Trigger("build")

	// step 2: event="tower_placed" ← OnEvent("towerBuilt") 应映射
	changed := tut.OnEvent("towerBuilt")
	if !changed {
		t.Fatal("OnEvent towerBuilt 应推进 tower_placed 步骤")
	}

	// step 3: event="wave_start" ← OnEvent("waveStarted") 应映射
	changed = tut.OnEvent("waveStarted")
	if !changed {
		t.Fatal("OnEvent waveStarted 应推进 wave_start 步骤")
	}
}

func TestTutorialSkip(t *testing.T) {
	tut := tutorial.DefaultTutorial()
	tut.Skip()
	if !tut.IsComplete() {
		t.Fatal("跳过后应标记完成")
	}
	if tut.CurrentStep() != nil {
		t.Fatal("跳过后 CurrentStep 应返回 nil")
	}
}

func TestTutorialStepCount(t *testing.T) {
	tut := tutorial.DefaultTutorial()
	if tut.StepCount() != 8 {
		t.Fatalf("应有 8 步，实际 %d", tut.StepCount())
	}
}
