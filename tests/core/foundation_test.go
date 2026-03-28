package core_test

import (
	"math"
	"sync"
	"testing"

	"defense2/internal/core/event"
	"defense2/internal/core/physics"
	"defense2/internal/core/query"
	"defense2/internal/core/session"
)

// ============================================================
// 事件总线测试
// ============================================================

func TestBus_OnEmit(t *testing.T) {
	bus := event.NewBus()
	var received []interface{}
	bus.On("test", func(args ...interface{}) {
		received = append(received, args...)
	})
	bus.Emit("test", "hello", 42)

	if len(received) != 2 {
		t.Fatalf("期望收到2个参数，实际收到%d个", len(received))
	}
	if received[0] != "hello" || received[1] != 42 {
		t.Errorf("参数不匹配: %v", received)
	}
}

func TestBus_Unsubscribe(t *testing.T) {
	bus := event.NewBus()
	count := 0
	unsub := bus.On("test", func(args ...interface{}) { count++ })

	bus.Emit("test")
	if count != 1 {
		t.Fatalf("取消前期望1次调用，实际%d", count)
	}

	unsub()
	bus.Emit("test")
	if count != 1 {
		t.Errorf("取消后期望1次调用，实际%d", count)
	}
}

func TestBus_Off(t *testing.T) {
	bus := event.NewBus()
	count := 0
	bus.On("test", func(args ...interface{}) { count++ })
	bus.On("test", func(args ...interface{}) { count++ })

	bus.Emit("test")
	if count != 2 {
		t.Fatalf("Off前期望2次调用，实际%d", count)
	}

	bus.Off("test")
	bus.Emit("test")
	if count != 2 {
		t.Errorf("Off后期望2次调用，实际%d", count)
	}
}

func TestBus_Clear(t *testing.T) {
	bus := event.NewBus()
	count := 0
	bus.On("a", func(args ...interface{}) { count++ })
	bus.On("b", func(args ...interface{}) { count++ })

	bus.Clear()
	bus.Emit("a")
	bus.Emit("b")
	if count != 0 {
		t.Errorf("Clear后期望0次调用，实际%d", count)
	}
}

func TestBus_MultipleListeners(t *testing.T) {
	bus := event.NewBus()
	results := make([]int, 0, 3)
	bus.On("test", func(args ...interface{}) { results = append(results, 1) })
	bus.On("test", func(args ...interface{}) { results = append(results, 2) })
	bus.On("test", func(args ...interface{}) { results = append(results, 3) })

	bus.Emit("test")
	if len(results) != 3 || results[0] != 1 || results[1] != 2 || results[2] != 3 {
		t.Errorf("多处理器调用顺序不正确: %v", results)
	}
}

func TestBus_ConcurrentSafe(t *testing.T) {
	bus := event.NewBus()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unsub := bus.On("test", func(args ...interface{}) {})
			bus.Emit("test")
			unsub()
		}()
	}
	wg.Wait()
}

func TestBus_ListenerCount(t *testing.T) {
	bus := event.NewBus()
	if bus.ListenerCount("test") != 0 {
		t.Errorf("空总线应有0个监听器")
	}
	unsub := bus.On("test", func(args ...interface{}) {})
	if bus.ListenerCount("test") != 1 {
		t.Errorf("添加后应有1个监听器")
	}
	unsub()
	if bus.ListenerCount("test") != 0 {
		t.Errorf("取消后应有0个监听器")
	}
}

// ============================================================
// 碰撞检测测试
// ============================================================

func TestPointDistance(t *testing.T) {
	tests := []struct {
		x1, y1, x2, y2 float64
		want           float64
	}{
		{0, 0, 3, 4, 5},
		{0, 0, 0, 0, 0},
		{1, 1, 4, 5, 5},
	}
	for _, tt := range tests {
		got := physics.PointDistance(tt.x1, tt.y1, tt.x2, tt.y2)
		if math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("PointDistance(%v,%v,%v,%v)=%v, 期望 %v",
				tt.x1, tt.y1, tt.x2, tt.y2, got, tt.want)
		}
	}
}

func TestCirclesOverlap(t *testing.T) {
	tests := []struct {
		a, b physics.Circle
		want bool
	}{
		{physics.Circle{0, 0, 10}, physics.Circle{15, 0, 10}, true}, // 重叠
		{physics.Circle{0, 0, 5}, physics.Circle{20, 0, 5}, false},  // 分离
		{physics.Circle{0, 0, 10}, physics.Circle{20, 0, 10}, true}, // 刚好接触
		{physics.Circle{0, 0, 10}, physics.Circle{5, 0, 10}, true},  // 包含
	}
	for i, tt := range tests {
		got := physics.CirclesOverlap(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("用例%d: CirclesOverlap=%v, 期望 %v", i, got, tt.want)
		}
	}
}

func TestPointInCircle(t *testing.T) {
	c := physics.Circle{10, 10, 5}
	if !physics.PointInCircle(10, 10, c) {
		t.Error("圆心应在圆内")
	}
	if !physics.PointInCircle(15, 10, c) {
		t.Error("边界点应在圆内")
	}
	if physics.PointInCircle(20, 10, c) {
		t.Error("远点不应在圆内")
	}
}

func TestFindCirclesInRange(t *testing.T) {
	candidates := []physics.Circle{
		{10, 0, 5}, // 距离10
		{30, 0, 5}, // 距离30
		{5, 0, 5},  // 距离5
	}
	results := physics.FindCirclesInRange(0, 0, 20, candidates)
	if len(results) != 2 {
		t.Fatalf("期望2个结果(10和5在范围内)，实际%d", len(results))
	}
	// 应该按距离排序
	if results[0].Index != 2 {
		t.Errorf("最近的应该是索引2(距离5)，实际%d", results[0].Index)
	}
}

func TestLineSegmentIntersectsCircle(t *testing.T) {
	c := physics.Circle{10, 0, 5}

	// 线段穿过圆
	hit := physics.LineSegmentIntersectsCircle(0, 0, 20, 0, c)
	if !hit.Hit {
		t.Error("线段穿过圆应命中")
	}

	// 线段不经过圆
	miss := physics.LineSegmentIntersectsCircle(0, 10, 20, 10, c)
	if miss.Hit {
		t.Error("线段不经过圆不应命中")
	}
}

func TestFindCirclesAlongLine(t *testing.T) {
	candidates := []physics.Circle{
		{5, 0, 2},  // 在线段上
		{10, 0, 2}, // 在线段上
		{5, 20, 2}, // 远离线段
	}
	results := physics.FindCirclesAlongLine(0, 0, 20, 0, candidates, nil)
	if len(results) != 2 {
		t.Errorf("期望命中2个，实际%d", len(results))
	}

	// 排除索引0
	exclude := map[int]bool{0: true}
	results2 := physics.FindCirclesAlongLine(0, 0, 20, 0, candidates, exclude)
	if len(results2) != 1 {
		t.Errorf("排除后期望命中1个，实际%d", len(results2))
	}
}

// ============================================================
// 会话统计测试
// ============================================================

func TestStats_RecordAndTick(t *testing.T) {
	s := session.New()

	s.RecordKill()
	s.RecordKill()
	s.RecordLeak()
	s.RecordDamage(100)
	s.RecordDamage(50)
	s.RecordTowerBuilt()
	s.RecordWaveCleared()
	s.RecordGoldEarned(200)
	s.RecordGoldSpent(80)

	if s.Kills != 2 {
		t.Errorf("击杀数=%d, 期望2", s.Kills)
	}
	if s.Leaked != 1 {
		t.Errorf("泄漏数=%d, 期望1", s.Leaked)
	}
	if s.Damage != 150 {
		t.Errorf("伤害=%.0f, 期望150", s.Damage)
	}
	if s.TowersBuilt != 1 {
		t.Errorf("建塔数=%d, 期望1", s.TowersBuilt)
	}
	if s.WavesCleared != 1 {
		t.Errorf("波次数=%d, 期望1", s.WavesCleared)
	}
	if s.GoldEarned != 200 {
		t.Errorf("金币收入=%.0f, 期望200", s.GoldEarned)
	}
	if s.GoldSpent != 80 {
		t.Errorf("金币支出=%.0f, 期望80", s.GoldSpent)
	}
}

func TestStats_DPSTracking(t *testing.T) {
	s := session.New()

	// 模拟1秒内造成100伤害
	s.RecordDamage(100)
	s.Tick(1.0)

	if s.PeakDPS != 100 {
		t.Errorf("峰值DPS=%.0f, 期望100", s.PeakDPS)
	}

	// 模拟下一秒造成200伤害
	s.RecordDamage(200)
	s.Tick(1.0)

	if s.PeakDPS != 200 {
		t.Errorf("峰值DPS=%.0f, 期望200", s.PeakDPS)
	}

	avg := s.AverageDPS()
	if math.Abs(avg-150) > 1e-9 {
		t.Errorf("平均DPS=%.1f, 期望150.0", avg)
	}

	samples := s.DPSSamples()
	if len(samples) != 2 {
		t.Errorf("DPS快照数=%d, 期望2", len(samples))
	}
}

func TestStats_ElapsedTime(t *testing.T) {
	s := session.New()
	s.Tick(0.5)
	s.Tick(0.3)
	if math.Abs(s.ElapsedTime-0.8) > 1e-9 {
		t.Errorf("经过时间=%.1f, 期望0.8", s.ElapsedTime)
	}
}

// ============================================================
// 状态选择器测试
// ============================================================

func TestGetTowerAtSlot(t *testing.T) {
	towers := []query.TowerInfo{
		{SlotIndex: 0, Damage: 10},
		{SlotIndex: 5, Damage: 20},
		{SlotIndex: 12, Damage: 30},
	}

	found := query.GetTowerAtSlot(towers, 5)
	if found == nil || found.Damage != 20 {
		t.Errorf("应找到SlotIndex=5的塔")
	}

	notFound := query.GetTowerAtSlot(towers, 99)
	if notFound != nil {
		t.Errorf("不存在的SlotIndex应返回nil")
	}
}

func TestGetWavePressureScore(t *testing.T) {
	comp := query.WaveComposition{
		Normal: 5,
		Boss:   1,
	}
	score := query.GetWavePressureScore(10, comp)
	// 10*0.8 + 5*1.0 + 1*8.0 = 8 + 5 + 8 = 21
	expected := 21.0
	if math.Abs(score-expected) > 1e-9 {
		t.Errorf("压力分=%.1f, 期望%.1f", score, expected)
	}
}

func TestGetDefenseReadiness(t *testing.T) {
	tests := []struct {
		name     string
		towers   []query.TowerInfo
		pressure float64
		level    query.ReadinessLevel
	}{
		{
			name: "强势",
			towers: []query.TowerInfo{
				{Damage: 50, AttackSpeed: 2, SplashRadius: 10},
				{Damage: 40, AttackSpeed: 2, SplashRadius: 10},
			},
			pressure: 5,
			level:    query.ReadinessStrong,
		},
		{
			name:     "空塔弱势",
			towers:   nil,
			pressure: 10,
			level:    query.ReadinessWeak,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := query.GetDefenseReadiness(tt.towers, tt.pressure)
			if result.Level != tt.level {
				t.Errorf("准备度=%s, 期望%s (分数=%.1f)", result.Level, tt.level, result.Score)
			}
		})
	}
}
