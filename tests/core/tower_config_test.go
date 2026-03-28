package core_test

import (
	"testing"

	"defense2/internal/config"
)

func TestLoadTowersCoreJSON(t *testing.T) {
	fd, err := config.LoadTowerFile("config/towers/towers-core.json")
	if err != nil {
		t.Fatalf("加载失败: %v", err)
	}
	if fd.Meta.Faction != "base" {
		t.Fatalf("预期 faction=base, 实际 %s", fd.Meta.Faction)
	}
	if len(fd.Towers) < 3 {
		t.Fatalf("预期至少 3 座塔, 实际 %d", len(fd.Towers))
	}
	// 检查 laser 塔属性
	laser, ok := fd.Towers["laser"]
	if !ok {
		t.Fatal("应包含 laser 塔")
	}
	if laser.BaseRange != 200 {
		t.Fatalf("laser 射程预期 200, 实际 %.0f", laser.BaseRange)
	}
}

func TestLoadAllTowers(t *testing.T) {
	all, err := config.LoadAllTowers()
	if err != nil {
		t.Fatalf("加载全部塔失败: %v", err)
	}
	if len(all) < 8 {
		t.Fatalf("预期至少 8 座塔, 实际 %d", len(all))
	}
	// towers-core 和 towers 应合并
	if _, ok := all["laser"]; !ok {
		t.Fatal("应包含 core 塔 laser")
	}
}
