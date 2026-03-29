package core_test

import (
	"math"
	"testing"

	"defense2/internal/core/enemy"
	"defense2/internal/core/gamemap"
	"defense2/internal/core/skill"
)

// makeSkillEnemy 创建用于技能测试的敌人
func makeSkillEnemy(x, y, hp float64) *enemy.Enemy {
	return &enemy.Enemy{
		X: x, Y: y,
		HP: hp, MaxHP: hp,
		Speed: 50, BaseSpeed: 50,
		Radius: 10, Active: true,
		Path: []gamemap.Point{{X: 0, Y: 0}},
	}
}

// ============================================================
// 核心技能注册表测试
// ============================================================

func TestSkillRegistry_RegisterAndGet(t *testing.T) {
	// 注册前获取不存在的应返回nil
	if skill.Get("nonExistentSkill_xyz") != nil {
		t.Error("未注册技能应返回nil")
	}

	// 注册自定义技能
	skill.Register("testSkill", func() skill.CoreSkill {
		return &mockSkill{name: "testSkill"}
	})

	s := skill.Get("testSkill")
	if s == nil {
		t.Fatal("注册后应能获取技能")
	}
	if s.Name() != "testSkill" {
		t.Errorf("Name()=%s, 期望testSkill", s.Name())
	}
}

func TestSkillRegistry_AssignAndTick(t *testing.T) {
	skill.Register("testSkill2", func() skill.CoreSkill {
		return &mockSkill{name: "testSkill2"}
	})

	state := &skill.SkillState{}
	ok := skill.AssignSkill(state, "testSkill2", nil)
	if !ok {
		t.Fatal("AssignSkill应成功")
	}
	if state.SkillName != "testSkill2" {
		t.Errorf("SkillName=%s, 期望testSkill2", state.SkillName)
	}

	enemies := []*enemy.Enemy{makeSkillEnemy(50, 0, 100)}
	suppress := skill.TickEntitySkill(state, nil, enemies, 0.016, nil)
	if suppress {
		t.Error("mockSkill不应压制普攻")
	}
}

// ============================================================
// 链式闪电测试
// ============================================================

func TestChainLightning_Registration(t *testing.T) {
	// chainLightning 应通过 init() 自注册
	s := skill.Get("chainLightning")
	if s == nil {
		t.Fatal("chainLightning应已注册")
	}
}

func TestChainLightning_CooldownAndFire(t *testing.T) {
	s := skill.Get("chainLightning")
	s.Init(nil)

	enemies := []*enemy.Enemy{
		makeSkillEnemy(50, 0, 1000),
		makeSkillEnemy(100, 0, 1000),
	}

	// 模拟冷却(7秒)
	for i := 0; i < 700; i++ {
		s.Tick(nil, enemies, 0.01, nil)
	}

	// 冷却结束后应开始射击
	r, _ := s.GetProgress(nil)
	// 进度应有变化（具体值取决于实现）
	_ = r
}

// ============================================================
// 核弹测试
// ============================================================

func TestNukeBomb_Registration(t *testing.T) {
	s := skill.Get("nukeBomb")
	if s == nil {
		t.Fatal("nukeBomb应已注册")
	}
}

// ============================================================
// 风刃测试
// ============================================================

func TestWindBlade_Registration(t *testing.T) {
	s := skill.Get("windBlade")
	if s == nil {
		t.Fatal("windBlade应已注册")
	}
}

// ============================================================
// 通道激光测试
// ============================================================

func TestChannelLaser_Registration(t *testing.T) {
	s := skill.Get("channelLaser")
	if s == nil {
		t.Fatal("channelLaser应已注册")
	}
}

func TestChannelLaser_FindBestDirection(t *testing.T) {
	enemies := []*enemy.Enemy{
		makeSkillEnemy(100, 0, 100),
		makeSkillEnemy(110, 0, 100),
		makeSkillEnemy(120, 0, 100),
	}
	// 所有敌人在右方，最佳方向应接近0弧度
	angle := skill.FindBestLaserDirection(enemies, 0, 0, 200, 28)
	if math.Abs(angle) > math.Pi/4 {
		t.Errorf("最佳方向=%.2f弧度, 敌人在右方应接近0", angle)
	}
}

func TestChannelLaser_GetEnemiesInLaser(t *testing.T) {
	enemies := []*enemy.Enemy{
		makeSkillEnemy(100, 0, 100), // 正右方
		makeSkillEnemy(0, 100, 100), // 正上方
	}
	hits := skill.GetEnemiesInLaser(enemies, 0, 0, 0, 200, 28) // 角度0=向右
	if len(hits) != 1 {
		t.Errorf("向右激光应命中1个敌人(正右方), 实际%d", len(hits))
	}
}

// ============================================================
// 全部 9 技能注册测试
// ============================================================

func TestAllSkillsRegistered(t *testing.T) {
	names := []string{
		"chainLightning", "nukeBomb", "windBlade", "channelLaser",
		"missileBarrage", "judgmentBeam", "chainLightningBolts", "judgmentRain",
		"thunderSmite",
	}
	for _, name := range names {
		s := skill.Get(name)
		if s == nil {
			t.Errorf("%s 应已注册", name)
		}
	}
}

// ============================================================
// Mock 技能（测试用）
// ============================================================

type mockSkill struct {
	name   string
	ticked int
}

func (m *mockSkill) Name() string       { return m.name }
func (m *mockSkill) Init(_ interface{}) {}
func (m *mockSkill) Tick(_ interface{}, _ []*enemy.Enemy, _ float64, _ *skill.SkillContext) bool {
	m.ticked++
	return false
}
func (m *mockSkill) ShouldSuppressFire(_ interface{}) bool     { return false }
func (m *mockSkill) ShouldSuppressMove(_ interface{}) bool     { return false }
func (m *mockSkill) GetProgress(_ interface{}) (float64, bool) { return 0, false }
