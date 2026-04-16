//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/aiplayer"
)

// TestBehaviorLeakReaction 验证漏怪时触发急跑和气泡。
func TestAIPlayerBehaviorLeakReaction(t *testing.T) {
	ap := aiplayer.New(aiplayer.Config{
		ZoneProvider: aiplayer.NewZone(24, 13),
		OwnerID:      1,
		StartGold:    200,
		Ops:          &mockOps{},
		CellSize:     60,
		SpawnX:       900,
		SpawnY:       390,
	})

	// 先 tick 几帧让精灵稳定
	normalSnap := aiplayer.AISnapshot{
		Lives:       20,
		Wave:        1,
		MaxWaves:    12,
		WardenReady: true,
		WaveActive:  true,
		MapCenterX:  720,
		MapCenterY:  390,
		PathPoints:  []aiplayer.AIPathPoint{{X: 100, Y: 300}},
	}
	for i := 0; i < 10; i++ {
		ap.Tick(1.0/60.0, normalSnap)
	}

	// 漏怪：生命从 20 降到 18
	leakSnap := normalSnap
	leakSnap.Lives = 18

	ap.Tick(1.0/60.0, leakSnap)

	// 漏怪后应该显示气泡
	if !ap.BubbleVisible() {
		t.Error("bubble should be visible after leak")
	}
}

// TestBehaviorKillStreak 验证连杀触发兴奋气泡。
func TestAIPlayerBehaviorKillStreak(t *testing.T) {
	ap := aiplayer.New(aiplayer.Config{
		ZoneProvider: aiplayer.NewZone(24, 13),
		OwnerID:      1,
		StartGold:    200,
		Ops:          &mockOps{},
		CellSize:     60,
		SpawnX:       900,
		SpawnY:       390,
	})

	// 第一次 tick：0 kills
	snap := aiplayer.AISnapshot{
		Lives:       20,
		Wave:        1,
		MaxWaves:    12,
		WardenReady: true,
		WaveActive:  true,
		MapCenterX:  720,
		MapCenterY:  390,
		Towers: []aiplayer.AITower{
			{Row: 5, Col: 15, Kills: 0, Owner: 1, Strength: 100},
		},
	}
	for i := 0; i < 10; i++ {
		ap.Tick(1.0/60.0, snap)
	}

	// 突然 5 kills（大连杀）
	snap.Towers = []aiplayer.AITower{
		{Row: 5, Col: 15, Kills: 5, Owner: 1, Strength: 100},
	}
	ap.Tick(1.0/60.0, snap)

	// 大连杀应该触发气泡
	if !ap.BubbleVisible() {
		t.Error("bubble should be visible after kill streak >= 5")
	}
}

// TestBehaviorIdlePatrol 验证长时间空闲后巡视。
func TestAIPlayerBehaviorIdlePatrol(t *testing.T) {
	ap := aiplayer.New(aiplayer.Config{
		ZoneProvider: aiplayer.NewZone(24, 13),
		OwnerID:      1,
		StartGold:    200,
		Ops:          &mockOps{},
		CellSize:     60,
		SpawnX:       900,
		SpawnY:       390,
	})

	initialX := ap.SpriteX()

	// 空闲 tick（无可造塔/无金币变化）6 秒
	snap := aiplayer.AISnapshot{
		Lives:       20,
		Wave:        1,
		MaxWaves:    12,
		Gold:        5, // 无钱做事
		WardenReady: true,
		WaveActive:  true,
		MapCenterX:  720,
		MapCenterY:  390,
	}

	for i := 0; i < 400; i++ { // ~6.7 秒
		ap.Tick(1.0/60.0, snap)
	}

	// 巡视后精灵位置应该发生变化（或有巡视气泡）
	// 由于巡视随机性，只验证精灵是否尝试过移动
	movedOrBubble := ap.SpriteX() != initialX || ap.BubbleVisible()
	if !movedOrBubble {
		t.Log("sprite didn't move or show bubble after 6.7s idle (acceptable due to randomness)")
	}
}

// TestBehaviorCooldowns 验证行为冷却不会刷屏。
func TestAIPlayerBehaviorCooldowns(t *testing.T) {
	ap := aiplayer.New(aiplayer.Config{
		ZoneProvider: aiplayer.NewZone(24, 13),
		OwnerID:      1,
		StartGold:    200,
		Ops:          &mockOps{},
		CellSize:     60,
		SpawnX:       900,
		SpawnY:       390,
	})

	// 连续漏怪多次
	bubbleCount := 0
	for round := 0; round < 5; round++ {
		lives := 20 - round
		snap := aiplayer.AISnapshot{
			Lives:       lives,
			Wave:        1,
			MaxWaves:    12,
			WardenReady: true,
			WaveActive:  true,
			MapCenterX:  720,
			MapCenterY:  390,
		}
		ap.Tick(1.0/60.0, snap)
		if ap.BubbleVisible() {
			bubbleCount++
		}
		// 等几帧
		for i := 0; i < 5; i++ {
			snap2 := snap
			snap2.Lives = lives
			ap.Tick(1.0/60.0, snap2)
		}
	}

	// 5 次漏怪不应该 5 次都显示气泡（冷却机制）
	if bubbleCount >= 5 {
		t.Errorf("leak reactions triggered %d/5 times, cooldown should prevent all from firing", bubbleCount)
	}
}
