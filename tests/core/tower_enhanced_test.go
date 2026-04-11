package core_test

import (
	"testing"

	"defense2/internal/core/tower"
)

// ============================================================
// 塔生命周期钩子测试
// ============================================================

func TestLifecycle_CreateHook(t *testing.T) {
	tower.ResetHooks()
	called := false
	tower.OnTowerCreate(func(tw *tower.Tower) { called = true })

	tw := &tower.Tower{Key: "test"}
	tower.ExecuteCreate(tw)
	if !called {
		t.Error("Create钩子应被调用")
	}
}

func TestLifecycle_DestroyHook(t *testing.T) {
	tower.ResetHooks()
	called := false
	tower.OnTowerDestroy(func(tw *tower.Tower) { called = true })

	tw := &tower.Tower{Key: "test"}
	tower.ExecuteDestroy(tw)
	if !called {
		t.Error("Destroy钩子应被调用")
	}
}

func TestLifecycle_UpgradeHook(t *testing.T) {
	tower.ResetHooks()
	called := false
	tower.OnTowerUpgrade(func(tw *tower.Tower) { called = true })

	tw := &tower.Tower{Key: "test"}
	tower.ExecuteUpgrade(tw)
	if !called {
		t.Error("Upgrade钩子应被调用")
	}
}

func TestLifecycle_PanicRecovery(t *testing.T) {
	tower.ResetHooks()
	tower.OnTowerCreate(func(tw *tower.Tower) { panic("boom") })

	secondCalled := false
	tower.OnTowerCreate(func(tw *tower.Tower) { secondCalled = true })

	tw := &tower.Tower{Key: "test"}
	// 不应panic
	tower.ExecuteCreate(tw)
	if !secondCalled {
		t.Error("panic不应阻止后续钩子执行")
	}
}

