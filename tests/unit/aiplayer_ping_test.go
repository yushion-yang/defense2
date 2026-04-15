//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/aiplayer"
)

// TestPingSendAndConsume 验证基本的 Send → HasPending → Consume 流程。
func TestPingSendAndConsume(t *testing.T) {
	ps := aiplayer.NewPingState()

	// 初始状态：无 pending，无冷却
	if ps.HasPending() {
		t.Error("new PingState should have no pending")
	}
	if ps.OnCooldown() {
		t.Error("new PingState should not be on cooldown")
	}

	// 发送 ping
	ok := ps.Send(5, 10, 330.0, 630.0)
	if !ok {
		t.Error("first Send should succeed")
	}
	if !ps.HasPending() {
		t.Error("after Send, should have pending")
	}
	if !ps.OnCooldown() {
		t.Error("after Send, should be on cooldown")
	}

	// 消费 ping
	req := ps.Consume()
	if req == nil {
		t.Fatal("Consume should return non-nil request")
	}
	if req.Row != 5 || req.Col != 10 {
		t.Errorf("got row=%d col=%d, want 5,10", req.Row, req.Col)
	}
	if req.X != 330.0 || req.Y != 630.0 {
		t.Errorf("got X=%.1f Y=%.1f, want 330.0,630.0", req.X, req.Y)
	}

	// 消费后无 pending
	if ps.HasPending() {
		t.Error("after Consume, should have no pending")
	}
	// 二次消费返回 nil
	if ps.Consume() != nil {
		t.Error("second Consume should return nil")
	}
}

// TestPingCooldown 验证冷却机制。
func TestPingCooldown(t *testing.T) {
	ps := aiplayer.NewPingState()
	ps.Send(1, 2, 100, 200)
	ps.Consume() // 消费掉，但冷却仍在

	// 冷却中 Send 应失败
	ok := ps.Send(3, 4, 300, 400)
	if ok {
		t.Error("Send during cooldown should return false")
	}

	// Tick 足够时间使冷却过期（10 秒冷却）
	for i := 0; i < 700; i++ {
		ps.Tick(1.0 / 60.0) // ~11.67 秒
	}

	if ps.OnCooldown() {
		t.Error("cooldown should expire after 11.67s of ticking")
	}

	// 冷却结束后可以再次 Send
	ok = ps.Send(7, 8, 500, 600)
	if !ok {
		t.Error("Send after cooldown should succeed")
	}
}

// TestPingConsumeWithoutSend 验证无 Send 时 Consume 返回 nil。
func TestPingConsumeWithoutSend(t *testing.T) {
	ps := aiplayer.NewPingState()
	if ps.Consume() != nil {
		t.Error("Consume without Send should return nil")
	}
}

// TestAIPlayerSendPing 通过 AIPlayer 公开接口验证 ping 集成。
func TestAIPlayerSendPing(t *testing.T) {
	ops := &mockOps{}
	ap := aiplayer.New(aiplayer.Config{
		ZoneProvider: aiplayer.NewZone(24, 13),
		OwnerID:      1,
		StartGold:    200,
		Ops:          ops,
		CellSize:     60,
	})

	// 首次 ping 应成功
	ok := ap.SendPing(5, 15, 930, 330)
	if !ok {
		t.Error("first SendPing should succeed")
	}

	// 冷却中 ping 应失败
	ok = ap.SendPing(6, 16, 990, 390)
	if ok {
		t.Error("SendPing during cooldown should fail")
	}
	if !ap.PingOnCooldown() {
		t.Error("should be on cooldown after SendPing")
	}
}

// TestAIPlayerPingTriggersBuild 验证 ping 同意时触发造塔。
func TestAIPlayerPingTriggersBuild(t *testing.T) {
	ops := &mockOps{}
	// 使用高 Compliance 个性确保同意
	ap := aiplayer.NewWithPersonality(aiplayer.Config{
		ZoneProvider: aiplayer.NewZone(24, 13),
		OwnerID:      1,
		StartGold:    200,
		Ops:          ops,
		CellSize:     60,
	}, aiplayer.Personality{
		Aggression: 0.5,
		Economy:    0.5,
		Risk:       0.5,
		Reaction:   0.5,
		Compliance: 0.95, // 高配合度，确保同意
	})

	// 发送 ping 到 AI 区域的格子
	ap.SendPing(5, 15, 930, 330)

	snap := aiplayer.AISnapshot{
		Wave:     1,
		MaxWaves: 12,
		TowerDefs: []aiplayer.AITowerDef{
			{Key: "basic", Cost: 50, Damage: 10, Range: 100, Index: 0},
		},
		BuildCells: []aiplayer.AICell{
			{Row: 5, Col: 15, X: 930, Y: 330},
		},
		PathPoints: []aiplayer.AIPathPoint{
			{X: 900, Y: 300},
			{X: 950, Y: 350},
		},
		WardenReady: true,
		WaveActive:  true,
	}

	// Tick 足够时间让 ping 响应 + 行动执行
	for i := 0; i < 300; i++ {
		ap.Tick(1.0/60.0, snap)
	}

	if ops.builtCount == 0 {
		t.Error("high-compliance AI should build at pinged location")
	}
}
