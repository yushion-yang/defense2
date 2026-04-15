//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/aiplayer"
)

func TestActionQueueDelay(t *testing.T) {
	q := aiplayer.NewActionQueue()

	executed := false
	q.Enqueue(aiplayer.DelayedAction{
		Delay:   1.0,
		Execute: func() { executed = true },
		Label:   "test_build",
	})

	// 0.5s: 尚未执行
	q.Tick(0.5)
	if executed {
		t.Fatal("action executed too early")
	}
	if q.Pending() != 1 {
		t.Errorf("pending = %d, want 1", q.Pending())
	}

	// 再 0.6s: 应执行
	q.Tick(0.6)
	if !executed {
		t.Fatal("action not executed after delay")
	}
	if q.Pending() != 0 {
		t.Errorf("pending = %d, want 0", q.Pending())
	}
}

func TestActionQueueFIFO(t *testing.T) {
	q := aiplayer.NewActionQueue()
	var order []int
	q.Enqueue(aiplayer.DelayedAction{
		Delay:   0.1,
		Execute: func() { order = append(order, 1) },
		Label:   "first",
	})
	q.Enqueue(aiplayer.DelayedAction{
		Delay:   0.1,
		Execute: func() { order = append(order, 2) },
		Label:   "second",
	})

	q.Tick(0.15) // 执行 first
	q.Tick(0.15) // 执行 second

	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("order = %v, want [1,2]", order)
	}
}

func TestActionQueuePeekLabel(t *testing.T) {
	q := aiplayer.NewActionQueue()
	if q.PeekLabel() != "" {
		t.Errorf("empty queue PeekLabel = %q, want empty", q.PeekLabel())
	}
	q.Enqueue(aiplayer.DelayedAction{
		Delay:   0.5,
		Execute: func() {},
		Label:   "build",
	})
	if q.PeekLabel() != "build" {
		t.Errorf("PeekLabel = %q, want %q", q.PeekLabel(), "build")
	}
}
