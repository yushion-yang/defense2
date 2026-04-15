package core_test

import (
	"os"
	"testing"

	"defense2/internal/core/persistence"
)

func TestStorageRoundTrip(t *testing.T) {
	s := newMemStorage()
	err := s.Set("test", map[string]int{"a": 1})
	if err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	var result map[string]int
	err = s.Get("test", &result)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if result["a"] != 1 {
		t.Fatalf("预期 a=1，实际 %d", result["a"])
	}
}

func TestProgressManagerRecordWin(t *testing.T) {
	s := newMemStorage()
	pm := persistence.NewProgressManager(s)

	pm.RecordGameResult("casual", "map_01", 50, true)
	p := pm.Progress()

	if p.TotalWins != 1 {
		t.Fatalf("预期 1 胜，实际 %d", p.TotalWins)
	}
	if p.ModeScores["casual_map_01"] != 50 {
		t.Fatalf("预期最高分 50，实际 %d", p.ModeScores["casual_map_01"])
	}
	// 通关 map_01 应解锁 casual 的 map_02
	if !pm.IsMapUnlocked("casual", "map_02") {
		t.Fatal("通关 map_01 应解锁 casual 的 map_02")
	}
	// classic 模式不应受影响
	if pm.IsMapUnlocked("classic", "map_02") {
		t.Fatal("casual 通关不应解锁 classic 的 map_02")
	}
}

func TestProgressManagerRecordLoss(t *testing.T) {
	s := newMemStorage()
	pm := persistence.NewProgressManager(s)

	pm.RecordGameResult("casual", "map_01", 30, false)
	p := pm.Progress()

	if p.TotalGames != 1 {
		t.Fatalf("预期 1 场，实际 %d", p.TotalGames)
	}
	if p.TotalWins != 0 {
		t.Fatalf("失败不应计入胜场，实际 %d", p.TotalWins)
	}
	// 失败不解锁下一关
	if pm.IsMapUnlocked("casual", "map_02") {
		t.Fatal("失败不应解锁 map_02")
	}
}

func TestProgressManagerTutorial(t *testing.T) {
	s := newMemStorage()
	pm := persistence.NewProgressManager(s)

	if pm.Progress().TutorialDone {
		t.Fatal("初始教程应未完成")
	}
	pm.SetTutorialDone()
	if !pm.Progress().TutorialDone {
		t.Fatal("标记后教程应已完成")
	}
}

func TestClassicModeDefaultUnlock(t *testing.T) {
	s := newMemStorage()
	pm := persistence.NewProgressManager(s)

	// 经典模式首关 map_c01 应默认解锁
	if !pm.IsMapUnlocked("classic", "map_c01") {
		t.Fatal("经典模式 map_c01 应默认解锁")
	}
	// 经典模式 map_c02 应锁定
	if pm.IsMapUnlocked("classic", "map_c02") {
		t.Fatal("经典模式 map_c02 初始应锁定")
	}
	// 战役模式 map_01 仍应默认解锁
	if !pm.IsMapUnlocked("casual", "map_01") {
		t.Fatal("战役模式 map_01 应默认解锁")
	}
	// 经典模式不应有 map_01 解锁
	if pm.IsMapUnlocked("classic", "map_01") {
		t.Fatal("经典模式不应解锁 map_01")
	}
}

func TestClassicModeUnlockChain(t *testing.T) {
	s := newMemStorage()
	pm := persistence.NewProgressManager(s)

	// 通关 map_c01 应解锁 map_c02
	pm.RecordGameResult("classic", "map_c01", 40, true)
	if !pm.IsMapUnlocked("classic", "map_c02") {
		t.Fatal("通关 map_c01 应解锁 classic 的 map_c02")
	}
	// 不应影响战役模式
	if pm.IsMapUnlocked("casual", "map_c02") {
		t.Fatal("classic 通关不应解锁 casual 的 map_c02")
	}
}

// newMemStorage 创建测试用临时文件存储（每次独立目录）。
func newMemStorage() persistence.Storage {
	dir, _ := os.MkdirTemp("", "defense2-test-*")
	s, _ := persistence.NewFileStorage(dir)
	return s
}
