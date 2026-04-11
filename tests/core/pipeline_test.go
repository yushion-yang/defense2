package core_test

import (
	"testing"

	"defense2/internal/core/pipeline"
)

// ─── Orchestrator ───────────────────────────────────────────────

func TestOrchestratorRunsInOrder(t *testing.T) {
	var order []int
	o := pipeline.NewOrchestrator()
	o.Add("first", pipeline.Func(func(ctx *pipeline.TickCtx) bool {
		order = append(order, 1)
		return false
	}))
	o.Add("second", pipeline.Func(func(ctx *pipeline.TickCtx) bool {
		order = append(order, 2)
		return false
	}))
	o.Add("third", pipeline.Func(func(ctx *pipeline.TickCtx) bool {
		order = append(order, 3)
		return false
	}))
	o.Tick(&pipeline.TickCtx{})
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Errorf("order = %v, want [1, 2, 3]", order)
	}
}

func TestOrchestratorAbortOnTrue(t *testing.T) {
	var ran []string
	o := pipeline.NewOrchestrator()
	o.Add("first", pipeline.Func(func(ctx *pipeline.TickCtx) bool {
		ran = append(ran, "first")
		return false
	}))
	o.Add("abort", pipeline.Func(func(ctx *pipeline.TickCtx) bool {
		ran = append(ran, "abort")
		return true // should abort remaining
	}))
	o.Add("skipped", pipeline.Func(func(ctx *pipeline.TickCtx) bool {
		ran = append(ran, "skipped")
		return false
	}))
	o.Tick(&pipeline.TickCtx{})
	if len(ran) != 2 || ran[0] != "first" || ran[1] != "abort" {
		t.Errorf("ran = %v, want [first, abort]", ran)
	}
}

func TestOrchestratorAllReturnFalse(t *testing.T) {
	count := 0
	o := pipeline.NewOrchestrator()
	for i := 0; i < 5; i++ {
		o.Add("sys", pipeline.Func(func(ctx *pipeline.TickCtx) bool {
			count++
			return false
		}))
	}
	o.Tick(&pipeline.TickCtx{})
	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}
}

func TestOrchestratorEmpty(t *testing.T) {
	o := pipeline.NewOrchestrator()
	// Should not panic on empty orchestrator
	o.Tick(&pipeline.TickCtx{})
}

func TestOrchestratorSystemCount(t *testing.T) {
	o := pipeline.NewOrchestrator()
	if o.SystemCount() != 0 {
		t.Errorf("empty orchestrator SystemCount = %d, want 0", o.SystemCount())
	}
	o.Add("a", pipeline.Func(func(ctx *pipeline.TickCtx) bool { return false }))
	o.Add("b", pipeline.Func(func(ctx *pipeline.TickCtx) bool { return false }))
	if o.SystemCount() != 2 {
		t.Errorf("SystemCount = %d, want 2", o.SystemCount())
	}
}

// ─── Func adapter ───────────────────────────────────────────────

func TestFuncAdapter(t *testing.T) {
	called := false
	sys := pipeline.Func(func(ctx *pipeline.TickCtx) bool {
		called = true
		return true
	})
	result := sys.Tick(&pipeline.TickCtx{})
	if !called {
		t.Error("Func adapter did not call wrapped function")
	}
	if !result {
		t.Error("Func adapter should return true")
	}
}

func TestFuncAdapterReturnsFalse(t *testing.T) {
	sys := pipeline.Func(func(ctx *pipeline.TickCtx) bool {
		return false
	})
	if sys.Tick(&pipeline.TickCtx{}) {
		t.Error("Func adapter should return false")
	}
}

// ─── TickCtx field access ───────────────────────────────────────

func TestTickCtxMutableStatePointers(t *testing.T) {
	// Verify that systems can modify Gold/Lives/Kills through pointers
	gold, lives, kills := 100, 20, 0
	ctx := &pipeline.TickCtx{
		Gold:  &gold,
		Lives: &lives,
		Kills: &kills,
		DT:    1.0 / 60.0,
	}

	o := pipeline.NewOrchestrator()
	o.Add("earn_gold", pipeline.Func(func(c *pipeline.TickCtx) bool {
		*c.Gold += 50
		return false
	}))
	o.Add("take_damage", pipeline.Func(func(c *pipeline.TickCtx) bool {
		*c.Lives -= 1
		return false
	}))
	o.Add("score_kill", pipeline.Func(func(c *pipeline.TickCtx) bool {
		*c.Kills += 3
		return false
	}))

	o.Tick(ctx)

	if gold != 150 {
		t.Errorf("gold = %d, want 150", gold)
	}
	if lives != 19 {
		t.Errorf("lives = %d, want 19", lives)
	}
	if kills != 3 {
		t.Errorf("kills = %d, want 3", kills)
	}
}

// ─── SysWardenGate ──────────────────────────────────────────────

func TestSysWardenGateAborts(t *testing.T) {
	shown := false
	ctx := &pipeline.TickCtx{
		CB: pipeline.TickCallbacks{
			ShouldShowWardenSelect: func() bool { return true },
			ShowWardenSelect:       func() { shown = true },
		},
	}
	sys := pipeline.SysWardenGate{}
	result := sys.Tick(ctx)
	if !result {
		t.Error("SysWardenGate should return true when warden select should show")
	}
	if !shown {
		t.Error("SysWardenGate should call ShowWardenSelect")
	}
}

func TestSysWardenGateNoAbort(t *testing.T) {
	ctx := &pipeline.TickCtx{
		CB: pipeline.TickCallbacks{
			ShouldShowWardenSelect: func() bool { return false },
		},
	}
	sys := pipeline.SysWardenGate{}
	result := sys.Tick(ctx)
	if result {
		t.Error("SysWardenGate should return false when warden select not needed")
	}
}

func TestSysWardenGateNilCallbacks(t *testing.T) {
	// ShouldShowWardenSelect is nil — should not panic, return false
	ctx := &pipeline.TickCtx{}
	sys := pipeline.SysWardenGate{}
	result := sys.Tick(ctx)
	if result {
		t.Error("SysWardenGate should return false when callbacks are nil")
	}
}

// ─── SysWardenTick ──────────────────────────────────────────────

func TestSysWardenTickSkipsWhenNotReady(t *testing.T) {
	ctx := &pipeline.TickCtx{
		WardenReady: false,
		WardenUnit:  nil,
	}
	sys := pipeline.SysWardenTick{}
	result := sys.Tick(ctx)
	if result {
		t.Error("SysWardenTick should return false when warden not ready")
	}
}

// ─── SysEndCondition ────────────────────────────────────────────

func TestSysEndConditionCallbacks(t *testing.T) {
	// SysEndCondition delegates to Session.TickEndConditions — testing callback wiring
	// with nil Session would panic, so we verify that the struct can be constructed
	// with callbacks set
	victoryFired := false
	defeatFired := false
	_ = pipeline.SysEndCondition{
		OnVictory: func() { victoryFired = true },
		OnDefeat:  func() { defeatFired = true },
	}
	// Cannot fully test without gamemode.Session, but verify no compile error
	_ = victoryFired
	_ = defeatFired
}

// ─── First system abort skips all subsequent ────────────────────

func TestOrchestratorFirstSystemAbort(t *testing.T) {
	var ran []string
	o := pipeline.NewOrchestrator()
	o.Add("abort", pipeline.Func(func(ctx *pipeline.TickCtx) bool {
		ran = append(ran, "abort")
		return true
	}))
	o.Add("never1", pipeline.Func(func(ctx *pipeline.TickCtx) bool {
		ran = append(ran, "never1")
		return false
	}))
	o.Add("never2", pipeline.Func(func(ctx *pipeline.TickCtx) bool {
		ran = append(ran, "never2")
		return false
	}))
	o.Tick(&pipeline.TickCtx{})
	if len(ran) != 1 || ran[0] != "abort" {
		t.Errorf("ran = %v, want [abort]", ran)
	}
}

// ─── Systems can read DT from context ───────────────────────────

func TestSystemReadsDT(t *testing.T) {
	var gotDT float64
	o := pipeline.NewOrchestrator()
	o.Add("reader", pipeline.Func(func(ctx *pipeline.TickCtx) bool {
		gotDT = ctx.DT
		return false
	}))
	o.Tick(&pipeline.TickCtx{DT: 0.016})
	if gotDT != 0.016 {
		t.Errorf("DT = %f, want 0.016", gotDT)
	}
}

// ─── Callback wiring in TickCallbacks ───────────────────────────

func TestTickCallbacksWiring(t *testing.T) {
	killCalled := false
	ctx := &pipeline.TickCtx{
		CB: pipeline.TickCallbacks{
			OnKill: func(isBoss bool, source string) {
				killCalled = true
			},
		},
	}
	// Simulate a system calling the kill callback
	o := pipeline.NewOrchestrator()
	o.Add("trigger_kill", pipeline.Func(func(c *pipeline.TickCtx) bool {
		if c.CB.OnKill != nil {
			c.CB.OnKill(false, "test")
		}
		return false
	}))
	o.Tick(ctx)
	if !killCalled {
		t.Error("OnKill callback was not called")
	}
}
