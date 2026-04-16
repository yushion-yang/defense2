// aiplayer_test.go — AI 玩家集成测试。
//
// 测试道具背包、executeDecision 新分支、快照构建等。
package aiplayer

import "testing"

// mockOps 实现 StageOps 接口，记录调用用于断言。
type mockOps struct {
	buildCalled   bool
	upgradeCalled bool
	sellCalled    bool
	itemUsed      bool
	unlockCalled  bool
	waveCalled    bool
	wardenCalled  bool
	abilityCalled bool
	lastSellRow   int
	lastSellCol   int
	lastItemKind  int
	lastItemRow   int
	lastItemCol   int
	unlockRow     int
	unlockCol     int
	sellRefund    int
	unlockCost    int

	// 行为控制
	buildResult   bool
	sellResult    bool
	itemResult    bool
	unlockResult  bool
}

func (m *mockOps) BuildTowerForAI(key string, row, col, ownerID int) bool {
	m.buildCalled = true
	return m.buildResult
}
func (m *mockOps) UpgradeTowerForAI(row, col int) bool {
	m.upgradeCalled = true
	return true
}
func (m *mockOps) SellTowerForAI(row, col int) bool {
	m.sellCalled = true
	m.lastSellRow = row
	m.lastSellCol = col
	return m.sellResult
}
func (m *mockOps) TowerCost(key string) int        { return 50 }
func (m *mockOps) StrengthBuyCost() int             { return 10 }
func (m *mockOps) StartWave() bool                  { m.waveCalled = true; return true }
func (m *mockOps) SelectWarden(key string) bool     { m.wardenCalled = true; return true }
func (m *mockOps) ChooseAbility(row, col int, slotIndex int, abilityName string) bool {
	m.abilityCalled = true
	return true
}
func (m *mockOps) UseItemForAI(itemKind int, towerRow, towerCol int) bool {
	m.itemUsed = true
	m.lastItemKind = itemKind
	m.lastItemRow = towerRow
	m.lastItemCol = towerCol
	return m.itemResult
}
func (m *mockOps) UnlockAbilitySlotForAI(row, col int) (cost int, ok bool) {
	m.unlockCalled = true
	m.unlockRow = row
	m.unlockCol = col
	if m.unlockResult {
		return m.unlockCost, true
	}
	return 0, false
}
func (m *mockOps) SellRefundAmount(row, col int) int {
	return m.sellRefund
}

// mockZone 实现 ZoneProvider，所有格子都属于 ownerID=1。
type mockZone struct{}

func (z mockZone) OwnerOf(row, col int) int { return 1 }

// ── 道具背包测试 ──

func TestAIPlayerInventory(t *testing.T) {
	ap := New(Config{
		ZoneProvider: mockZone{},
		OwnerID:      1,
		StartGold:    200,
		Ops:          &mockOps{},
	})

	if ap.ItemCount(0) != 0 {
		t.Fatal("inventory should start empty")
	}

	ap.AddItem(0)
	ap.AddItem(0)
	ap.AddItem(2)

	if ap.ItemCount(0) != 2 {
		t.Fatalf("expected 2 baseDamage items, got %d", ap.ItemCount(0))
	}
	if ap.ItemCount(2) != 1 {
		t.Fatalf("expected 1 baseSpeed item, got %d", ap.ItemCount(2))
	}
}

func TestBuildItemSnapshot(t *testing.T) {
	ap := New(Config{
		ZoneProvider: mockZone{},
		OwnerID:      1,
		StartGold:    200,
		Ops:          &mockOps{},
	})

	ap.AddItem(0) // baseDamage
	ap.AddItem(3) // potentialSpeed

	items := ap.buildItemSnapshot()
	if len(items) != 2 {
		t.Fatalf("expected 2 items in snapshot, got %d", len(items))
	}

	// 验证类别名称正确映射
	found := map[string]bool{}
	for _, it := range items {
		found[it.Category] = true
	}
	if !found["baseDamage"] {
		t.Fatal("missing baseDamage in snapshot")
	}
	if !found["potentialSpeed"] {
		t.Fatal("missing potentialSpeed in snapshot")
	}
}

// ── executeDecision 卖塔测试 ──

func TestExecuteDecision_Sell(t *testing.T) {
	ops := &mockOps{sellResult: true, sellRefund: 35}
	ap := New(Config{
		ZoneProvider: mockZone{},
		OwnerID:      1,
		StartGold:    100,
		Ops:          ops,
	})

	ap.executeDecision(Decision{
		Type: DecisionSell,
		Row:  3, Col: 5,
	})

	// 行动应被入队
	if !ap.actions.Busy() {
		t.Fatal("sell action should be enqueued")
	}

	// 执行队列（模拟时间流逝）
	for i := 0; i < 200; i++ {
		ap.actions.Tick(0.016)
	}

	if !ops.sellCalled {
		t.Fatal("SellTowerForAI should have been called")
	}
	if ops.lastSellRow != 3 || ops.lastSellCol != 5 {
		t.Fatalf("sell target mismatch: (%d,%d)", ops.lastSellRow, ops.lastSellCol)
	}
	if ap.lastAction != "sold tower" {
		t.Fatalf("lastAction should be 'sold tower', got '%s'", ap.lastAction)
	}
}

// ── executeDecision 道具使用测试 ──

func TestExecuteDecision_UseItem(t *testing.T) {
	ops := &mockOps{itemResult: true}
	ap := New(Config{
		ZoneProvider: mockZone{},
		OwnerID:      1,
		StartGold:    100,
		Ops:          ops,
	})

	// 给 AI 一个道具
	ap.AddItem(2) // baseSpeed

	ap.executeDecision(Decision{
		Type:     DecisionUseItem,
		Row:      3,
		Col:      5,
		ItemKind: 2,
	})

	// 执行队列
	for i := 0; i < 200; i++ {
		ap.actions.Tick(0.016)
	}

	if !ops.itemUsed {
		t.Fatal("UseItemForAI should have been called")
	}
	if ops.lastItemKind != 2 {
		t.Fatalf("expected itemKind=2, got %d", ops.lastItemKind)
	}
	// 道具应从背包扣除
	if ap.ItemCount(2) != 0 {
		t.Fatalf("item should be consumed, got count %d", ap.ItemCount(2))
	}
}

func TestExecuteDecision_UseItem_EmptyInventory(t *testing.T) {
	ops := &mockOps{itemResult: true}
	ap := New(Config{
		ZoneProvider: mockZone{},
		OwnerID:      1,
		StartGold:    100,
		Ops:          ops,
	})

	// 不给道具，直接执行使用决策
	ap.executeDecision(Decision{
		Type:     DecisionUseItem,
		Row:      3,
		Col:      5,
		ItemKind: 0,
	})

	for i := 0; i < 200; i++ {
		ap.actions.Tick(0.016)
	}

	// 背包为空时不应调用 ops
	if ops.itemUsed {
		t.Fatal("UseItemForAI should NOT be called when inventory is empty")
	}
}

// ── executeDecision 解锁能力槽测试 ──

func TestExecuteDecision_UnlockSlot(t *testing.T) {
	ops := &mockOps{unlockResult: true, unlockCost: 30}
	ap := New(Config{
		ZoneProvider: mockZone{},
		OwnerID:      1,
		StartGold:    200,
		Ops:          ops,
	})

	ap.executeDecision(Decision{
		Type: DecisionUnlockSlot,
		Row:  3, Col: 5,
	})

	for i := 0; i < 200; i++ {
		ap.actions.Tick(0.016)
	}

	if !ops.unlockCalled {
		t.Fatal("UnlockAbilitySlotForAI should have been called")
	}
	// 金币应被扣除
	if ap.gold != 170 {
		t.Fatalf("expected gold=170 after unlock cost 30, got %d", ap.gold)
	}
}

// ── 新增对话类别测试 ──

func TestDialogueBank_NewCategories(t *testing.T) {
	bank := DefaultDialogueBank()
	newCategories := []string{
		"sell_thinking", "sell_done", "item_use",
		"unlock_slot", "wave_early", "boss_prepare",
	}
	for _, cat := range newCategories {
		text := bank.Random(cat)
		if text == "" {
			t.Fatalf("dialogue category '%s' should have entries", cat)
		}
	}
}
