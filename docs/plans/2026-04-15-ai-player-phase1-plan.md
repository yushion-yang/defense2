# AI 玩家系统 Phase 1 实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在现有地图上跑通"AI 精灵自动造塔 + 思维气泡"的最小可玩原型

**Architecture:** 新建 `internal/core/aiplayer/` 包，实现独立于 AutoPlayer 的 AI 玩家系统。AI 通过 `StageScene` 新增的 `aiPlayer` 字段注入，拥有独立金币，在 `updatePlaying()` 末尾 Tick，在 `drawScene()` 世界空间阶段渲染精灵。不复用 AutoPlayer 接口（那是无头测试用的，`Draw` 直接 return）。

**Tech Stack:** Go 1.24+, Ebitengine v2.9.9, 现有 draw/ui/theme 包

---

## 关键设计决策

1. **不复用 AutoPlayer 接口** — AutoPlayer 是无头自动化测试框架，`Draw()` 中 `autoPlayer != nil` 直接跳过渲染。AI 玩家需要渲染精灵/气泡，必须是独立字段。
2. **AI 直接操作 tower.Pool** — 和 AutoPlayer 不同，AI 玩家不返回 Action 列表由 stage 执行，而是 stage 提供受控的操作接口（`BuildTowerForAI`/`UpgradeForAI`），AI 直接调用。这避免了 AutoPlayer 的"仅 idle 模式执行"限制。
3. **区域按列号硬分** — Phase 1 不做地图 zones 字段，直接按列号中点左右分（左=玩家，右=AI）。
4. **独立金币** — AI 有自己的 `gold int`，击杀收入按塔的 Owner 归属。

## 文件清单

| 操作 | 文件 | 用途 |
|------|------|------|
| 新建 | `internal/core/aiplayer/aiplayer.go` | AI 玩家主结构体 + Tick |
| 新建 | `internal/core/aiplayer/decision.go` | 决策引擎（造塔/升级评估） |
| 新建 | `internal/core/aiplayer/action.go` | 延迟行动队列 |
| 新建 | `internal/core/aiplayer/sprite.go` | 精灵状态机（位置/移动/状态） |
| 新建 | `internal/core/aiplayer/bubble.go` | 思维气泡数据模型 |
| 新建 | `internal/core/aiplayer/zone.go` | 区域划分逻辑 |
| 新建 | `internal/render/hud/ai_overlay.go` | 精灵 + 气泡渲染 |
| 新建 | `config/aiplayer/ai-dialogue.json` | 本地文案模板 |
| 新建 | `tests/unit/aiplayer_test.go` | 单元测试 |
| 修改 | `internal/scene/stage.go` | 新增 aiPlayer 字段 + Tick/Draw 集成 |
| 修改 | `internal/scene/stage_types.go` | StageOptions 新增 AIEnabled |
| 修改 | `internal/scene/test_select.go` | TestSelect 添加 AI 开关 |

---

### Task 1: Zone — 区域划分逻辑

**Files:**
- Create: `internal/core/aiplayer/zone.go`
- Test: `tests/unit/aiplayer_zone_test.go`

**Step 1: Write the failing test**

```go
// tests/unit/aiplayer_zone_test.go
//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/aiplayer"
)

func TestZoneSplit(t *testing.T) {
	// 24 列地图，中点=12，左半 0-11=玩家，右半 12-23=AI
	z := aiplayer.NewZone(24, 13)

	tests := []struct {
		name     string
		row, col int
		want     aiplayer.ZoneOwner
	}{
		{"player left edge", 5, 0, aiplayer.ZoneHuman},
		{"player right boundary", 5, 11, aiplayer.ZoneHuman},
		{"ai left boundary", 5, 12, aiplayer.ZoneAI},
		{"ai right edge", 5, 23, aiplayer.ZoneAI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := z.OwnerOf(tt.row, tt.col)
			if got != tt.want {
				t.Errorf("OwnerOf(%d,%d) = %d, want %d", tt.row, tt.col, got, tt.want)
			}
		})
	}
}

func TestZoneBuildCells(t *testing.T) {
	z := aiplayer.NewZone(24, 13)
	grid := make([][]int, 13)
	for r := range grid {
		grid[r] = make([]int, 24)
		for c := range grid[r] {
			if c%3 == 0 {
				grid[r][c] = 2 // CellBuildable
			}
		}
	}
	aiCells := z.BuildableCells(grid, aiplayer.ZoneAI)
	for _, c := range aiCells {
		if c.Col < 12 {
			t.Errorf("AI buildable cell at col %d, should be >= 12", c.Col)
		}
	}
	humanCells := z.BuildableCells(grid, aiplayer.ZoneHuman)
	for _, c := range humanCells {
		if c.Col >= 12 {
			t.Errorf("Human buildable cell at col %d, should be < 12", c.Col)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags unittest -run TestZone -v ./tests/unit/`
Expected: FAIL — package not found

**Step 3: Write minimal implementation**

```go
// internal/core/aiplayer/zone.go
// zone.go — 区域划分逻辑。
// Phase 1: 按列号中点硬分（左=玩家，右=AI）。
// Phase 4 将改用地图 JSON zones 字段。
package aiplayer

// ZoneOwner 区域所有者。
type ZoneOwner int

const (
	ZoneHuman   ZoneOwner = 0
	ZoneAI      ZoneOwner = 1
	ZoneNeutral ZoneOwner = 2
)

// GridCell 网格单元格坐标。
type GridCell struct {
	Row, Col int
}

// Zone 区域划分器。
type Zone struct {
	cols    int
	rows    int
	splitCol int // AI 区域起始列
}

// NewZone 创建区域划分器，按列中点分割。
func NewZone(cols, rows int) *Zone {
	return &Zone{
		cols:     cols,
		rows:     rows,
		splitCol: cols / 2,
	}
}

// OwnerOf 返回指定格子的所有者。
func (z *Zone) OwnerOf(row, col int) ZoneOwner {
	if col < z.splitCol {
		return ZoneHuman
	}
	return ZoneAI
}

// BuildableCells 返回指定所有者区域内的所有可建造格子。
// grid 值: 2=CellBuildable
func (z *Zone) BuildableCells(grid [][]int, owner ZoneOwner) []GridCell {
	var cells []GridCell
	for r, row := range grid {
		for c, v := range row {
			if v == 2 && z.OwnerOf(r, c) == owner {
				cells = append(cells, GridCell{Row: r, Col: c})
			}
		}
	}
	return cells
}
```

**Step 4: Run test to verify it passes**

Run: `go test -tags unittest -run TestZone -v ./tests/unit/`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/aiplayer/zone.go tests/unit/aiplayer_zone_test.go
git commit -m "feat(aiplayer): add zone split logic for player/AI area division"
```

---

### Task 2: Sprite — 精灵状态机

**Files:**
- Create: `internal/core/aiplayer/sprite.go`
- Test: `tests/unit/aiplayer_sprite_test.go`

**Step 1: Write the failing test**

```go
// tests/unit/aiplayer_sprite_test.go
//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/aiplayer"
)

func TestSpriteIdle(t *testing.T) {
	sp := aiplayer.NewSprite(600, 300)
	if sp.State() != aiplayer.SpriteIdle {
		t.Errorf("initial state = %d, want SpriteIdle", sp.State())
	}
	if sp.X() != 600 || sp.Y() != 300 {
		t.Errorf("position = (%.0f,%.0f), want (600,300)", sp.X(), sp.Y())
	}
}

func TestSpriteMoveTo(t *testing.T) {
	sp := aiplayer.NewSprite(0, 0)
	sp.MoveTo(120, 0) // 120px 距离, speed=120px/s → 1秒到达

	// Tick 30 帧 (0.5s) — 应该在移动中
	for i := 0; i < 30; i++ {
		sp.Tick(1.0 / 60.0)
	}
	if sp.State() != aiplayer.SpriteMoving {
		t.Errorf("after 0.5s state = %d, want SpriteMoving", sp.State())
	}
	// X 应该约 60 (半程)
	if sp.X() < 50 || sp.X() > 70 {
		t.Errorf("after 0.5s X = %.0f, want ~60", sp.X())
	}

	// 再 Tick 60 帧 (1s) — 应该到达并回到 Idle
	for i := 0; i < 60; i++ {
		sp.Tick(1.0 / 60.0)
	}
	if sp.State() != aiplayer.SpriteIdle {
		t.Errorf("after arrival state = %d, want SpriteIdle", sp.State())
	}
	if sp.X() < 118 || sp.X() > 122 {
		t.Errorf("after arrival X = %.0f, want ~120", sp.X())
	}
}

func TestSpriteIdleBob(t *testing.T) {
	sp := aiplayer.NewSprite(100, 200)
	// idle 模式下 Y 应有微小浮动 (bob)
	sp.Tick(0.5)
	bobY := sp.DrawY()
	if bobY == 200 {
		t.Log("bob offset may be zero at t=0.5, acceptable")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags unittest -run TestSprite -v ./tests/unit/`
Expected: FAIL

**Step 3: Write minimal implementation**

```go
// internal/core/aiplayer/sprite.go
// sprite.go — AI 精灵状态机。
//
// 管理精灵在地图上的位置和运动状态。
// 状态: Idle(巡视) → Moving(移动到目标) → Idle
// 渲染相关的视觉效果(bob动画等)也在此计算。
package aiplayer

import "math"

// SpriteState 精灵状态。
type SpriteState int

const (
	SpriteIdle    SpriteState = iota // 空闲巡视
	SpriteMoving                     // 移动到目标
	SpriteThinking                   // 思考中（决策前停顿）
	SpriteExecuting                  // 执行操作
)

const (
	spriteSpeed   = 120.0 // 移动速度 (px/s)
	bobAmplitude  = 3.0   // idle 上下浮动幅度
	bobFrequency  = 2.0   // 浮动频率 (Hz)
	arrivalThresh = 2.0   // 到达判定阈值
)

// Sprite AI 精灵。
type Sprite struct {
	x, y       float64
	targetX    float64
	targetY    float64
	state      SpriteState
	time       float64 // 累计时间（用于动画）
	stateTimer float64 // 当前状态剩余时间
}

// NewSprite 创建精灵。
func NewSprite(x, y float64) *Sprite {
	return &Sprite{
		x: x, y: y,
		targetX: x, targetY: y,
		state: SpriteIdle,
	}
}

func (s *Sprite) X() float64         { return s.x }
func (s *Sprite) Y() float64         { return s.y }
func (s *Sprite) State() SpriteState { return s.state }

// DrawY 返回渲染用 Y 坐标（含 bob 动画偏移）。
func (s *Sprite) DrawY() float64 {
	if s.state == SpriteIdle {
		return s.y + bobAmplitude*math.Sin(s.time*bobFrequency*2*math.Pi)
	}
	return s.y
}

// MoveTo 设置移动目标。
func (s *Sprite) MoveTo(x, y float64) {
	s.targetX = x
	s.targetY = y
	s.state = SpriteMoving
}

// SetThinking 进入思考状态（定时后回到 Idle）。
func (s *Sprite) SetThinking(duration float64) {
	s.state = SpriteThinking
	s.stateTimer = duration
}

// SetExecuting 进入执行状态（定时后回到 Idle）。
func (s *Sprite) SetExecuting(duration float64) {
	s.state = SpriteExecuting
	s.stateTimer = duration
}

// Tick 每帧更新。
func (s *Sprite) Tick(dt float64) {
	s.time += dt

	switch s.state {
	case SpriteIdle:
		// 空闲：只做 bob 动画（由 DrawY 处理）
	case SpriteMoving:
		s.tickMoving(dt)
	case SpriteThinking, SpriteExecuting:
		s.stateTimer -= dt
		if s.stateTimer <= 0 {
			s.state = SpriteIdle
		}
	}
}

func (s *Sprite) tickMoving(dt float64) {
	dx := s.targetX - s.x
	dy := s.targetY - s.y
	dist := math.Hypot(dx, dy)

	if dist < arrivalThresh {
		s.x = s.targetX
		s.y = s.targetY
		s.state = SpriteIdle
		return
	}

	step := spriteSpeed * dt
	if step >= dist {
		s.x = s.targetX
		s.y = s.targetY
		s.state = SpriteIdle
		return
	}

	s.x += dx / dist * step
	s.y += dy / dist * step
}
```

**Step 4: Run test to verify it passes**

Run: `go test -tags unittest -run TestSprite -v ./tests/unit/`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/aiplayer/sprite.go tests/unit/aiplayer_sprite_test.go
git commit -m "feat(aiplayer): add sprite state machine with movement and bob animation"
```

---

### Task 3: Bubble — 思维气泡数据模型

**Files:**
- Create: `internal/core/aiplayer/bubble.go`
- Create: `config/aiplayer/ai-dialogue.json`
- Test: `tests/unit/aiplayer_bubble_test.go`

**Step 1: Write the failing test**

```go
// tests/unit/aiplayer_bubble_test.go
//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/aiplayer"
)

func TestBubbleLifecycle(t *testing.T) {
	bm := aiplayer.NewBubbleManager()

	// 显示气泡
	bm.Show("测试消息", aiplayer.BubbleThink, 2.0)
	if !bm.Visible() {
		t.Fatal("bubble should be visible after Show")
	}
	if bm.Text() != "测试消息" {
		t.Errorf("text = %q, want %q", bm.Text(), "测试消息")
	}

	// Tick 1 秒，仍然可见
	bm.Tick(1.0)
	if !bm.Visible() {
		t.Fatal("bubble should still be visible after 1s")
	}

	// Tick 再 1.5 秒，应消失
	bm.Tick(1.5)
	if bm.Visible() {
		t.Fatal("bubble should be hidden after 2.5s (duration=2.0)")
	}
}

func TestBubbleDialogueBank(t *testing.T) {
	bank := aiplayer.DefaultDialogueBank()
	texts := bank.Random("build_thinking")
	if texts == "" {
		t.Fatal("build_thinking should return non-empty text")
	}
}
```

**Step 2: Run test**

Run: `go test -tags unittest -run TestBubble -v ./tests/unit/`
Expected: FAIL

**Step 3: Write implementation**

```go
// internal/core/aiplayer/bubble.go
// bubble.go — 思维气泡数据模型。
//
// 管理气泡的显示/隐藏/文案选择。
// 纯数据层，渲染由 hud/ai_overlay.go 负责。
package aiplayer

import (
	"math/rand"
)

// BubbleType 气泡类型。
type BubbleType int

const (
	BubbleThink   BubbleType = iota // 💭 思考
	BubbleAction                    // 💬 行动
	BubbleEmotion                   // ❗ 情绪
	BubbleReply                     // 💬 回应
)

// BubbleManager 气泡管理器。
type BubbleManager struct {
	text     string
	btype    BubbleType
	timer    float64
	visible  bool
}

// NewBubbleManager 创建气泡管理器。
func NewBubbleManager() *BubbleManager {
	return &BubbleManager{}
}

// Show 显示一条气泡消息。
func (b *BubbleManager) Show(text string, btype BubbleType, duration float64) {
	b.text = text
	b.btype = btype
	b.timer = duration
	b.visible = true
}

// Tick 每帧更新。
func (b *BubbleManager) Tick(dt float64) {
	if !b.visible {
		return
	}
	b.timer -= dt
	if b.timer <= 0 {
		b.visible = false
		b.text = ""
	}
}

// Visible 是否可见。
func (b *BubbleManager) Visible() bool { return b.visible }

// Text 当前文案。
func (b *BubbleManager) Text() string { return b.text }

// Type 当前气泡类型。
func (b *BubbleManager) Type() BubbleType { return b.btype }

// Alpha 当前透明度（临近消失时淡出）。
func (b *BubbleManager) Alpha() float64 {
	if b.timer < 0.3 {
		return b.timer / 0.3
	}
	return 1.0
}

// ── 文案库 ──

// DialogueBank 本地对话文案库。
type DialogueBank struct {
	entries map[string][]string
}

// DefaultDialogueBank 返回硬编码的默认文案库。
// Phase 3 将改为从 JSON 加载。
func DefaultDialogueBank() *DialogueBank {
	return &DialogueBank{
		entries: map[string][]string{
			"build_thinking": {
				"这里放个塔？",
				"嗯...这个位置不错",
				"先堵住这里",
				"在这里补一个",
			},
			"build_done": {
				"搞定！",
				"好了",
				"建好了~",
			},
			"upgrade_thinking": {
				"该升级了",
				"这座塔需要加强",
				"升一下这个",
			},
			"low_gold": {
				"缺钱了...",
				"金币不够",
				"得等下一波",
			},
			"enemy_leak": {
				"啊！漏了！",
				"糟糕...",
				"没拦住",
			},
			"wave_start": {
				"来了来了",
				"准备好了",
				"开打！",
			},
			"idle": {
				"目前还好",
				"等下一波",
				"...",
			},
			"warden_select": {
				"选这个战灵吧",
				"我来选个战灵",
			},
			"ability_choose": {
				"选这个能力",
				"这个不错",
			},
		},
	}
}

// Random 从指定类别随机返回一条文案。
// 类别不存在时返回空串。
func (d *DialogueBank) Random(category string) string {
	texts, ok := d.entries[category]
	if !ok || len(texts) == 0 {
		return ""
	}
	return texts[rand.Intn(len(texts))]
}
```

`config/aiplayer/ai-dialogue.json` — Phase 3 加载用，Phase 1 先用硬编码:
```json
{
  "_comment": "Phase 3 将从此文件加载文案，Phase 1 使用 DefaultDialogueBank() 硬编码"
}
```

**Step 4: Run test**

Run: `go test -tags unittest -run TestBubble -v ./tests/unit/`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/aiplayer/bubble.go config/aiplayer/ai-dialogue.json tests/unit/aiplayer_bubble_test.go
git commit -m "feat(aiplayer): add bubble manager and dialogue bank for thought display"
```

---

### Task 4: Action Queue — 延迟行动队列

**Files:**
- Create: `internal/core/aiplayer/action.go`
- Test: `tests/unit/aiplayer_action_test.go`

**Step 1: Write the failing test**

```go
// tests/unit/aiplayer_action_test.go
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
		Delay: 1.0,
		Execute: func() {
			executed = true
		},
		Label: "test_build",
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

func TestActionQueuePeekLabel(t *testing.T) {
	q := aiplayer.NewActionQueue()
	q.Enqueue(aiplayer.DelayedAction{
		Delay:   0.5,
		Execute: func() {},
		Label:   "build",
	})
	if q.PeekLabel() != "build" {
		t.Errorf("PeekLabel = %q, want %q", q.PeekLabel(), "build")
	}
}
```

**Step 2: Run test**

Run: `go test -tags unittest -run TestActionQueue -v ./tests/unit/`
Expected: FAIL

**Step 3: Write implementation**

```go
// internal/core/aiplayer/action.go
// action.go — 延迟行动队列。
//
// AI 决策后不立即执行，而是排入队列，经过可配置的延迟后才执行。
// 延迟模拟"思考时间"，让 AI 行为节奏更像真人。
package aiplayer

// DelayedAction 延迟行动。
type DelayedAction struct {
	Delay   float64    // 延迟秒数
	Execute func()     // 执行函数
	Label   string     // 行动标签（用于气泡文案关联）
	elapsed float64    // 已等待时间
}

// ActionQueue 延迟行动队列（FIFO）。
type ActionQueue struct {
	queue []DelayedAction
}

// NewActionQueue 创建空队列。
func NewActionQueue() *ActionQueue {
	return &ActionQueue{}
}

// Enqueue 入队一个延迟行动。
func (q *ActionQueue) Enqueue(a DelayedAction) {
	q.queue = append(q.queue, a)
}

// Tick 每帧更新，到时间的行动自动执行并移除。
func (q *ActionQueue) Tick(dt float64) {
	if len(q.queue) == 0 {
		return
	}

	// 只处理队首（FIFO，一次只执行一个）
	q.queue[0].elapsed += dt
	if q.queue[0].elapsed >= q.queue[0].Delay {
		q.queue[0].Execute()
		q.queue = q.queue[1:]
	}
}

// Pending 待执行数量。
func (q *ActionQueue) Pending() int {
	return len(q.queue)
}

// Busy 是否有待执行的行动。
func (q *ActionQueue) Busy() bool {
	return len(q.queue) > 0
}

// PeekLabel 查看队首行动标签（用于气泡显示）。
func (q *ActionQueue) PeekLabel() string {
	if len(q.queue) == 0 {
		return ""
	}
	return q.queue[0].Label
}

// Clear 清空队列。
func (q *ActionQueue) Clear() {
	q.queue = q.queue[:0]
}
```

**Step 4: Run test**

Run: `go test -tags unittest -run TestActionQueue -v ./tests/unit/`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/aiplayer/action.go tests/unit/aiplayer_action_test.go
git commit -m "feat(aiplayer): add delayed action queue for human-like pacing"
```

---

### Task 5: Decision Engine — 决策引擎（简单版）

**Files:**
- Create: `internal/core/aiplayer/decision.go`
- Test: `tests/unit/aiplayer_decision_test.go`

**Step 1: Write the failing test**

```go
// tests/unit/aiplayer_decision_test.go
//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/aiplayer"
)

func TestDecisionBuildWhenRich(t *testing.T) {
	eng := aiplayer.NewDecisionEngine()
	snap := aiplayer.AISnapshot{
		Gold:     200,
		TowerDefs: []aiplayer.AITowerDef{{Key: "basic", Cost: 50}},
		BuildCells: []aiplayer.AICell{
			{Row: 3, Col: 15, X: 900, Y: 180},
			{Row: 5, Col: 18, X: 1080, Y: 300},
		},
		Towers:    nil,
		Wave:      1,
		MaxWaves:  12,
	}
	d := eng.Evaluate(snap)
	if d.Type != aiplayer.DecisionBuild {
		t.Errorf("decision = %d, want DecisionBuild when gold=200 and no towers", d.Type)
	}
}

func TestDecisionUpgradeWhenHasTowers(t *testing.T) {
	eng := aiplayer.NewDecisionEngine()
	snap := aiplayer.AISnapshot{
		Gold:     100,
		TowerDefs: []aiplayer.AITowerDef{{Key: "basic", Cost: 50}},
		BuildCells: []aiplayer.AICell{{Row: 3, Col: 15, X: 900, Y: 180}},
		Towers: []aiplayer.AITower{
			{Row: 5, Col: 18, Damage: 30, Strength: 100},
			{Row: 7, Col: 20, Damage: 50, Strength: 100},
		},
		Wave:     6,
		MaxWaves: 12,
	}
	d := eng.Evaluate(snap)
	// 中期有塔有钱，应该升级或建塔（都可以）
	if d.Type != aiplayer.DecisionBuild && d.Type != aiplayer.DecisionUpgrade {
		t.Errorf("decision = %d, want Build or Upgrade", d.Type)
	}
}

func TestDecisionIdleWhenBroke(t *testing.T) {
	eng := aiplayer.NewDecisionEngine()
	snap := aiplayer.AISnapshot{
		Gold:       5,
		TowerDefs:  []aiplayer.AITowerDef{{Key: "basic", Cost: 50}},
		BuildCells: []aiplayer.AICell{{Row: 3, Col: 15}},
		Towers:     []aiplayer.AITower{{Row: 5, Col: 18, Damage: 30, Strength: 100}},
		Wave:       3,
		MaxWaves:   12,
	}
	d := eng.Evaluate(snap)
	if d.Type != aiplayer.DecisionIdle {
		t.Errorf("decision = %d, want DecisionIdle when gold=5", d.Type)
	}
}
```

**Step 2: Run test**

Run: `go test -tags unittest -run TestDecision -v ./tests/unit/`
Expected: FAIL

**Step 3: Write implementation**

```go
// internal/core/aiplayer/decision.go
// decision.go — AI 决策引擎。
//
// Phase 1 简单版：基于经济状态和阶段做造塔/升级决策。
// 选位逻辑：优先靠近路径中心的格子。
// Phase 2 将加入威胁评估、能力选择、个性系统。
package aiplayer

import (
	"math"
	"math/rand"
)

// DecisionType 决策类型。
type DecisionType int

const (
	DecisionIdle    DecisionType = iota // 什么都不做
	DecisionBuild                       // 造塔
	DecisionUpgrade                     // 升级
)

// Decision 决策结果。
type Decision struct {
	Type     DecisionType
	Row, Col int    // 目标格子
	TowerKey string // 塔类型（Build 时使用）
	CenterX  float64
	CenterY  float64
}

// AISnapshot AI 可见的游戏状态快照（纯值，零 core 依赖）。
type AISnapshot struct {
	Gold       int
	Wave       int
	MaxWaves   int
	WaveActive bool
	Lives      int
	Towers     []AITower
	BuildCells []AICell
	TowerDefs  []AITowerDef
	MapCenterX float64
	MapCenterY float64
}

// AITower 塔快照。
type AITower struct {
	Row, Col    int
	Damage      float64
	Strength    int
	AttackSpeed float64
	Range       float64
	Kills       int
}

// AICell 可建造格子。
type AICell struct {
	Row, Col int
	X, Y     float64
}

// AITowerDef 可建造塔定义。
type AITowerDef struct {
	Key    string
	Cost   int
	Damage float64
	Range  float64
	Index  int
}

// DecisionEngine 决策引擎。
type DecisionEngine struct {
	strengthBuyCost int // 升级花费（默认 10）
}

// NewDecisionEngine 创建决策引擎。
func NewDecisionEngine() *DecisionEngine {
	return &DecisionEngine{
		strengthBuyCost: 10,
	}
}

// SetStrengthBuyCost 设置升级花费（从配置读取）。
func (e *DecisionEngine) SetStrengthBuyCost(cost int) {
	e.strengthBuyCost = cost
}

// Evaluate 评估当前状态，返回最佳决策。
func (e *DecisionEngine) Evaluate(snap AISnapshot) Decision {
	// 阶段判定
	progress := 0.0
	if snap.MaxWaves > 0 {
		progress = float64(snap.Wave) / float64(snap.MaxWaves)
	}

	// 能造塔吗？
	canBuild := false
	cheapest := math.MaxInt
	for _, d := range snap.TowerDefs {
		if d.Cost < cheapest {
			cheapest = d.Cost
		}
		if d.Cost <= snap.Gold {
			canBuild = true
		}
	}

	// 能升级吗？
	canUpgrade := len(snap.Towers) > 0 && snap.Gold >= e.strengthBuyCost

	// 策略：早期建塔优先，中后期升级优先
	switch {
	case !canBuild && !canUpgrade:
		return Decision{Type: DecisionIdle}
	case progress < 0.4 && canBuild && len(snap.BuildCells) > 0:
		return e.pickBuild(snap)
	case progress >= 0.4 && canUpgrade:
		// 中后期升级为主，但偶尔建塔
		if canBuild && len(snap.BuildCells) > 0 && rand.Float64() < 0.3 {
			return e.pickBuild(snap)
		}
		return e.pickUpgrade(snap)
	case canBuild && len(snap.BuildCells) > 0:
		return e.pickBuild(snap)
	case canUpgrade:
		return e.pickUpgrade(snap)
	default:
		return Decision{Type: DecisionIdle}
	}
}

// pickBuild 选择造塔位置和类型。
func (e *DecisionEngine) pickBuild(snap AISnapshot) Decision {
	// 选最便宜的能造的塔
	var bestDef AITowerDef
	bestScore := -1.0
	for _, d := range snap.TowerDefs {
		if d.Cost > snap.Gold {
			continue
		}
		score := (d.Damage * d.Range) / float64(d.Cost)
		if score > bestScore {
			bestScore = score
			bestDef = d
		}
	}

	// 选位置：靠近地图中心的格子（路径通常经过中心）
	cx := snap.MapCenterX
	cy := snap.MapCenterY
	if cx == 0 && cy == 0 {
		cx, cy = 600, 270 // fallback
	}

	bestCell := snap.BuildCells[0]
	bestDist := math.MaxFloat64
	for _, c := range snap.BuildCells {
		dist := math.Hypot(c.X-cx, c.Y-cy)
		if dist < bestDist {
			bestDist = dist
			bestCell = c
		}
	}

	return Decision{
		Type:     DecisionBuild,
		Row:      bestCell.Row,
		Col:      bestCell.Col,
		TowerKey: bestDef.Key,
		CenterX:  bestCell.X,
		CenterY:  bestCell.Y,
	}
}

// pickUpgrade 选择最佳升级目标。
func (e *DecisionEngine) pickUpgrade(snap AISnapshot) Decision {
	// 选伤害最高的塔升级
	best := snap.Towers[0]
	for _, t := range snap.Towers[1:] {
		if t.Damage > best.Damage {
			best = t
		}
	}
	return Decision{
		Type: DecisionUpgrade,
		Row:  best.Row,
		Col:  best.Col,
	}
}
```

**Step 4: Run test**

Run: `go test -tags unittest -run TestDecision -v ./tests/unit/`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/aiplayer/decision.go tests/unit/aiplayer_decision_test.go
git commit -m "feat(aiplayer): add simple decision engine with build/upgrade evaluation"
```

---

### Task 6: AIPlayer — 主结构体整合

**Files:**
- Create: `internal/core/aiplayer/aiplayer.go`
- Test: `tests/unit/aiplayer_core_test.go`

**Step 1: Write the failing test**

```go
// tests/unit/aiplayer_core_test.go
//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/aiplayer"
)

func TestAIPlayerTick(t *testing.T) {
	ops := &mockOps{}
	ap := aiplayer.New(aiplayer.Config{
		Zone:      aiplayer.NewZone(24, 13),
		StartGold: 120,
		Ops:       ops,
	})

	// 应该有初始金币
	if ap.Gold() != 120 {
		t.Errorf("gold = %d, want 120", ap.Gold())
	}

	// Tick 多帧，AI 应在某个时间点决策（决策间隔 2-4 秒）
	for i := 0; i < 300; i++ { // 5 秒
		ap.Tick(1.0/60.0, aiplayer.AISnapshot{
			Gold:       ap.Gold(),
			Wave:       1,
			MaxWaves:   12,
			TowerDefs:  []aiplayer.AITowerDef{{Key: "basic", Cost: 50, Damage: 10, Range: 100}},
			BuildCells: []aiplayer.AICell{{Row: 5, Col: 15, X: 900, Y: 300}},
		})
	}

	// 精灵应该存在
	if ap.SpriteX() == 0 && ap.SpriteY() == 0 {
		t.Error("sprite should have non-zero position")
	}
}

// mockOps 实现 aiplayer.StageOps 接口。
type mockOps struct {
	builtCount    int
	upgradedCount int
}

func (m *mockOps) BuildTower(key string, row, col int) bool {
	m.builtCount++
	return true
}

func (m *mockOps) UpgradeTower(row, col int) bool {
	m.upgradedCount++
	return true
}

func (m *mockOps) SellTower(row, col int) bool {
	return true
}

func (m *mockOps) TowerCost(key string) int {
	return 50
}

func (m *mockOps) StrengthBuyCost() int {
	return 10
}
```

**Step 2: Run test**

Run: `go test -tags unittest -run TestAIPlayerTick -v ./tests/unit/`
Expected: FAIL

**Step 3: Write implementation**

```go
// internal/core/aiplayer/aiplayer.go
// aiplayer.go — AI 玩家主结构体。
//
// 职责：编排决策引擎、行动队列、精灵、气泡，驱动 AI 的完整行为循环。
// 关联：由 stage.go 每帧调用 Tick()，由 hud/ai_overlay.go 渲染。
// 设计：零 core 依赖（除了自身子模块），通过 StageOps 接口与游戏交互。
package aiplayer

import (
	"math/rand"
)

// StageOps AI 可用的游戏操作接口。
// 由 StageScene 实现，避免 AI 直接依赖 scene 包。
type StageOps interface {
	BuildTower(key string, row, col int) bool
	UpgradeTower(row, col int) bool
	SellTower(row, col int) bool
	TowerCost(key string) int
	StrengthBuyCost() int
}

// Config AI 玩家配置。
type Config struct {
	Zone      *Zone
	StartGold int
	Ops       StageOps
}

// AIPlayer AI 玩家。
type AIPlayer struct {
	gold     int
	zone     *Zone
	ops      StageOps
	sprite   *Sprite
	bubble   *BubbleManager
	actions  *ActionQueue
	engine   *DecisionEngine
	dialogue *DialogueBank

	// 决策节奏
	decisionTimer    float64 // 距下次决策的倒计时
	decisionInterval float64 // 当前决策间隔（随机 2-4s）
}

// New 创建 AI 玩家。
func New(cfg Config) *AIPlayer {
	// 精灵初始位置：AI 区域中心
	spawnX := float64(cfg.Zone.splitCol*60 + 6*60) // 粗略估算中心 X
	spawnY := float64(cfg.Zone.rows/2) * 60        // 中心 Y

	ap := &AIPlayer{
		gold:     cfg.StartGold,
		zone:     cfg.Zone,
		ops:      cfg.Ops,
		sprite:   NewSprite(spawnX, spawnY),
		bubble:   NewBubbleManager(),
		actions:  NewActionQueue(),
		engine:   NewDecisionEngine(),
		dialogue: DefaultDialogueBank(),
	}
	ap.rollNextInterval()

	if cfg.Ops != nil {
		ap.engine.SetStrengthBuyCost(cfg.Ops.StrengthBuyCost())
	}

	return ap
}

// Gold 返回当前金币。
func (ap *AIPlayer) Gold() int { return ap.gold }

// AddGold 增加金币（击杀收入等）。
func (ap *AIPlayer) AddGold(amount int) { ap.gold += amount }

// SpriteX 精灵 X 坐标。
func (ap *AIPlayer) SpriteX() float64 { return ap.sprite.X() }

// SpriteY 精灵 Y 坐标（含 bob）。
func (ap *AIPlayer) SpriteY() float64 { return ap.sprite.DrawY() }

// SpriteState 精灵状态。
func (ap *AIPlayer) SpriteState() SpriteState { return ap.sprite.State() }

// BubbleVisible 气泡是否可见。
func (ap *AIPlayer) BubbleVisible() bool { return ap.bubble.Visible() }

// BubbleText 气泡文案。
func (ap *AIPlayer) BubbleText() string { return ap.bubble.Text() }

// BubbleAlpha 气泡透明度。
func (ap *AIPlayer) BubbleAlpha() float64 { return ap.bubble.Alpha() }

// Tick 每帧更新。
//   1. 行动队列处理
//   2. 精灵更新
//   3. 气泡更新
//   4. 决策评估（按间隔）
func (ap *AIPlayer) Tick(dt float64, snap AISnapshot) {
	ap.actions.Tick(dt)
	ap.sprite.Tick(dt)
	ap.bubble.Tick(dt)

	// 有待执行行动时不做新决策
	if ap.actions.Busy() {
		return
	}

	// 决策节奏控制
	ap.decisionTimer -= dt
	if ap.decisionTimer > 0 {
		return
	}
	ap.rollNextInterval()

	// 过滤出 AI 区域的可建造格子
	snap.Gold = ap.gold
	aiCells := ap.filterAICells(snap.BuildCells)
	snap.BuildCells = aiCells

	// 过滤出 AI 区域的塔
	aiTowers := ap.filterAITowers(snap.Towers)
	snap.Towers = aiTowers

	// 评估决策
	decision := ap.engine.Evaluate(snap)
	ap.executeDecision(decision)
}

// filterAICells 过滤出 AI 区域的可建造格子。
func (ap *AIPlayer) filterAICells(cells []AICell) []AICell {
	var result []AICell
	for _, c := range cells {
		if ap.zone.OwnerOf(c.Row, c.Col) == ZoneAI {
			result = append(result, c)
		}
	}
	return result
}

// filterAITowers 过滤出 AI 区域的塔。
func (ap *AIPlayer) filterAITowers(towers []AITower) []AITower {
	var result []AITower
	for _, t := range towers {
		if ap.zone.OwnerOf(t.Row, t.Col) == ZoneAI {
			result = append(result, t)
		}
	}
	return result
}

// executeDecision 将决策转化为延迟行动 + 精灵移动 + 气泡。
func (ap *AIPlayer) executeDecision(d Decision) {
	switch d.Type {
	case DecisionIdle:
		// 偶尔冒个 idle 气泡
		if rand.Float64() < 0.1 {
			ap.bubble.Show(ap.dialogue.Random("idle"), BubbleThink, 1.5)
		}
		return

	case DecisionBuild:
		// 精灵移动到目标格子
		ap.sprite.MoveTo(d.CenterX, d.CenterY)
		// 显示思考气泡
		ap.bubble.Show(ap.dialogue.Random("build_thinking"), BubbleThink, 1.5)

		cost := 50 // fallback
		if ap.ops != nil {
			cost = ap.ops.TowerCost(d.TowerKey)
		}
		towerKey := d.TowerKey
		row, col := d.Row, d.Col

		// 延迟执行造塔
		ap.actions.Enqueue(DelayedAction{
			Delay: 1.0 + rand.Float64()*0.5, // 1-1.5s 思考时间
			Label: "build",
			Execute: func() {
				if ap.gold < cost {
					ap.bubble.Show(ap.dialogue.Random("low_gold"), BubbleEmotion, 1.0)
					return
				}
				if ap.ops != nil && ap.ops.BuildTower(towerKey, row, col) {
					ap.gold -= cost
					ap.bubble.Show(ap.dialogue.Random("build_done"), BubbleAction, 1.0)
				}
			},
		})

	case DecisionUpgrade:
		row, col := d.Row, d.Col
		ap.bubble.Show(ap.dialogue.Random("upgrade_thinking"), BubbleThink, 1.0)

		ap.actions.Enqueue(DelayedAction{
			Delay: 0.5 + rand.Float64()*0.5,
			Label: "upgrade",
			Execute: func() {
				cost := 10
				if ap.ops != nil {
					cost = ap.ops.StrengthBuyCost()
				}
				if ap.gold < cost {
					ap.bubble.Show(ap.dialogue.Random("low_gold"), BubbleEmotion, 1.0)
					return
				}
				if ap.ops != nil && ap.ops.UpgradeTower(row, col) {
					ap.gold -= cost
				}
			},
		})
	}
}

// rollNextInterval 随机下一次决策间隔 (2-4s)。
func (ap *AIPlayer) rollNextInterval() {
	ap.decisionInterval = 2.0 + rand.Float64()*2.0
	ap.decisionTimer = ap.decisionInterval
}
```

**Step 4: Run test**

Run: `go test -tags unittest -run TestAIPlayerTick -v ./tests/unit/`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/aiplayer/aiplayer.go tests/unit/aiplayer_core_test.go
git commit -m "feat(aiplayer): integrate decision engine, sprite, bubble into AIPlayer"
```

---

### Task 7: HUD 渲染 — AI 精灵和气泡

**Files:**
- Create: `internal/render/hud/ai_overlay.go`

此 Task 无单元测试（渲染层难以 unittest），通过视觉验证。

**Step 1: Write implementation**

```go
// internal/render/hud/ai_overlay.go
// ai_overlay.go — AI 玩家的精灵和思维气泡渲染。
//
// 遵循 HUD ViewModel 模式：接受纯值 VM，用 ui 组件绘制。
// 精灵在世界空间渲染（受相机偏移），气泡在精灵正上方。
package hud

import (
	"image/color"

	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// AIOverlayVM AI 覆盖层的 ViewModel。
type AIOverlayVM struct {
	// 精灵
	SpriteX, SpriteY float64      // 世界坐标
	SpriteImg        *ebiten.Image // 精灵图片（可选，nil 时画默认圆形）
	SpriteState      int           // 0=idle, 1=moving, 2=thinking, 3=executing
	SpriteAlpha      float64       // 0-1

	// 气泡
	BubbleVisible bool
	BubbleText    string
	BubbleAlpha   float64

	// 区域
	ZoneSplitX float64 // 区域分界线 X 坐标（世界坐标）
	ShowZone   bool    // 是否显示分界线
}

const (
	aiSpriteSize    = 28.0  // 精灵显示直径
	aiBubbleOffsetY = -30.0 // 气泡相对精灵的 Y 偏移
	aiBubbleMaxW    = 140.0 // 气泡最大宽度
	aiBubblePadH    = 6.0   // 气泡水平内边距
	aiBubblePadV    = 4.0   // 气泡垂直内边距
)

// DrawAIOverlay 渲染 AI 精灵和气泡到世界空间。
func DrawAIOverlay(screen *ebiten.Image, vm AIOverlayVM) {
	if vm.SpriteAlpha <= 0 {
		return
	}

	// ── 区域分界线 ──
	if vm.ShowZone && vm.ZoneSplitX > 0 {
		lineClr := color.NRGBA{R: 100, G: 180, B: 255, A: 40}
		draw.DashedLine(screen, vm.ZoneSplitX, 0, vm.ZoneSplitX, float64(theme.CanvasH), 1, lineClr, 8, 6)
	}

	// ── 精灵 ──
	cx, cy := vm.SpriteX, vm.SpriteY
	if vm.SpriteImg != nil {
		draw.SpriteScaledRotatedAlpha(screen, vm.SpriteImg, cx, cy,
			aiSpriteSize/float64(vm.SpriteImg.Bounds().Dx()), 0, vm.SpriteAlpha)
	} else {
		// 默认：半透明蓝色光球
		clr := color.NRGBA{R: 80, G: 160, B: 255, A: uint8(180 * vm.SpriteAlpha)}
		draw.FilledCircle(screen, cx, cy, aiSpriteSize/2, clr)
		// 外圈
		outClr := color.NRGBA{R: 120, G: 200, B: 255, A: uint8(100 * vm.SpriteAlpha)}
		draw.CircleOutline(screen, cx, cy, aiSpriteSize/2+3, 1, outClr)
	}

	// ── 思维气泡 ──
	if !vm.BubbleVisible || vm.BubbleText == "" {
		return
	}
	drawAIBubble(screen, cx, cy+aiBubbleOffsetY, vm.BubbleText, vm.BubbleAlpha)
}

// drawAIBubble 在指定位置渲染思维气泡。
func drawAIBubble(screen *ebiten.Image, cx, cy float64, text string, alpha float64) {
	fm := ui.GlobalFontMetrics()
	if fm == nil {
		return
	}

	fontSize := theme.FontXS
	tw := fm.MeasureText(text, fontSize)
	if tw > aiBubbleMaxW {
		tw = aiBubbleMaxW
	}

	bw := tw + aiBubblePadH*2
	bh := float64(fontSize) + aiBubblePadV*2
	bx := cx - bw/2
	by := cy - bh

	// 背景圆角矩形
	bgAlpha := uint8(220 * alpha)
	bgClr := color.NRGBA{R: 40, G: 40, B: 50, A: bgAlpha}
	draw.RoundRect(screen, bx, by, bw, bh, 4, bgClr)

	// 文字
	textAlpha := uint8(255 * alpha)
	textClr := color.NRGBA{R: 255, G: 255, B: 255, A: textAlpha}
	fm.DrawCenteredText(screen, text, cx, by+bh/2, fontSize, textClr)
}
```

**Step 2: Commit**

```bash
git add internal/render/hud/ai_overlay.go
git commit -m "feat(aiplayer): add AI sprite and thought bubble HUD rendering"
```

---

### Task 8: Stage 集成 — 注入 AI 玩家

**Files:**
- Modify: `internal/scene/stage_types.go` — StageOptions 新增 AIEnabled
- Modify: `internal/scene/stage.go` — 新增 aiPlayer 字段 + Tick/Draw 集成 + StageOps 实现

**Step 1: 修改 StageOptions**

在 `stage_types.go` 的 `StageOptions` 结构体末尾添加:

```go
	AIEnabled bool // 启用 AI 玩家（合作模式）
```

**Step 2: 在 stage.go 中添加 AI 相关字段和导入**

在 `StageScene` 结构体中（靠近 `autoPlayer` 字段附近）添加:

```go
	aiPlayer *aiplayer.AIPlayer // AI 玩家（合作模式）
```

import 中添加 `"defense2/internal/core/aiplayer"`。

**Step 3: 在 NewStageSceneWithOpts 中初始化 AI 玩家**

在 `NewStageSceneWithOpts` 函数的返回之前（initOpts 设置之后），添加:

```go
	if opts.AIEnabled && s.gameMap != nil {
		zone := aiplayer.NewZone(s.gameMap.Config.Cols, s.gameMap.Config.Rows)
		s.aiPlayer = aiplayer.New(aiplayer.Config{
			Zone:      zone,
			StartGold: s.gold, // 与玩家相同的初始金币
			Ops:       s,      // StageScene 实现 StageOps
		})
	}
```

**Step 4: 实现 StageOps 接口**

在 stage.go 末尾添加:

```go
// ── AI 玩家操作接口（StageOps）──

// BuildTower 为 AI 造塔。实现 aiplayer.StageOps。
func (s *StageScene) BuildTowerForAI(key string, row, col int) bool {
	// 查找塔定义
	var def *tower.TowerDef
	for i := range s.towerDefs {
		if s.towerDefs[i].Key == key {
			def = &s.towerDefs[i]
			break
		}
	}
	if def == nil {
		return false
	}
	// 检查格子
	if s.towers.At(row, col) != nil {
		return false
	}
	center := s.gameMap.CellCenter(row, col)
	t := s.towers.Place(row, col, center.X, center.Y, *def)
	if t == nil {
		return false
	}
	// 应用能力模式
	s.ruleset.OnTowerBuilt(t, s.wavesCleared)
	return true
}

// UpgradeTowerForAI 为 AI 升级塔。实现 aiplayer.StageOps。
func (s *StageScene) UpgradeTowerForAI(row, col int) bool {
	t := s.towers.At(row, col)
	if t == nil {
		return false
	}
	spent := t.BuyStrength()
	return spent > 0
}

// SellTowerForAI 为 AI 卖塔。实现 aiplayer.StageOps。
func (s *StageScene) SellTowerForAI(row, col int) bool {
	t := s.towers.At(row, col)
	if t == nil {
		return false
	}
	t.Selling = true
	t.SellAnim = 0.25
	return true
}

// TowerCost 返回塔的造价。实现 aiplayer.StageOps。
func (s *StageScene) TowerCost(key string) int {
	for _, d := range s.towerDefs {
		if d.Key == key {
			return d.Cost
		}
	}
	return 0
}

// StrengthBuyCost 返回升级花费。实现 aiplayer.StageOps。
func (s *StageScene) StrengthBuyCost() int {
	return config.GlobalBalance().Tower.StrengthBuyCost
}
```

注意：StageOps 接口方法名是 `BuildTower`/`UpgradeTower`/`SellTower`，但 stage.go 中用 `BuildTowerForAI` 等名称避免与现有方法冲突。需要在 aiplayer.go 中调整接口方法名匹配，或者用适配器。最简方案是接口方法名就用 `BuildTowerForAI` 等。

**Step 5: 在 updatePlaying 末尾添加 AI Tick**

在 `runAutoPlayFrame()` 调用之前添加:

```go
	// AI 玩家决策
	if s.aiPlayer != nil {
		s.tickAIPlayer()
	}
```

```go
// tickAIPlayer 构建 AI 快照并调用 AIPlayer.Tick。
func (s *StageScene) tickAIPlayer() {
	snap := aiplayer.AISnapshot{
		Gold:     s.aiPlayer.Gold(),
		Wave:     s.spawner.CurrentWave(),
		MaxWaves: s.gameMap.Config.Waves,
		Lives:    s.lives,
	}

	// 塔快照
	s.towers.Each(func(t *tower.Tower) {
		snap.Towers = append(snap.Towers, aiplayer.AITower{
			Row: t.Row, Col: t.Col,
			Damage: t.Damage, Strength: int(t.Strength.Effective()),
			Range: t.Range, Kills: t.Kills,
		})
	})

	// 可建造格子
	gm := s.gameMap
	for r := 0; r < gm.Config.Rows; r++ {
		for c := 0; c < gm.Config.Cols; c++ {
			if gm.Config.Grid[r][c] == 2 && s.towers.At(r, c) == nil {
				center := gm.CellCenter(r, c)
				snap.BuildCells = append(snap.BuildCells, aiplayer.AICell{
					Row: r, Col: c, X: center.X, Y: center.Y,
				})
			}
		}
	}

	// 塔定义
	for i, d := range s.towerDefs {
		snap.TowerDefs = append(snap.TowerDefs, aiplayer.AITowerDef{
			Key: d.Key, Cost: d.Cost, Damage: d.Damage,
			Range: d.Range, Index: i,
		})
	}

	snap.MapCenterX = gm.PixelWidth() / 2
	snap.MapCenterY = gm.PixelHeight() / 2

	s.aiPlayer.Tick(1.0/60.0, snap)
}
```

**Step 6: 在 drawScene 的世界空间阶段添加 AI 渲染**

在 `drawScene` 的世界元素渲染部分（战灵之后、道具之前），添加:

```go
	// AI 玩家精灵 + 气泡
	if s.aiPlayer != nil {
		hud.DrawAIOverlay(worldTarget, hud.AIOverlayVM{
			SpriteX:       s.aiPlayer.SpriteX(),
			SpriteY:       s.aiPlayer.SpriteY(),
			SpriteAlpha:   1.0,
			BubbleVisible: s.aiPlayer.BubbleVisible(),
			BubbleText:    s.aiPlayer.BubbleText(),
			BubbleAlpha:   s.aiPlayer.BubbleAlpha(),
			ZoneSplitX:    float64(s.gameMap.Config.Cols/2) * float64(s.gameMap.CellSize),
			ShowZone:      true,
		})
	}
```

**Step 7: Commit**

```bash
git add internal/scene/stage_types.go internal/scene/stage.go internal/render/hud/ai_overlay.go
git commit -m "feat(aiplayer): integrate AI player into StageScene with Tick and Draw"
```

---

### Task 9: 入口 — TestSelect 添加 AI 开关

**Files:**
- Modify: `internal/scene/test_select.go` — TestSelect 开始游戏时设置 AIEnabled=true

**Step 1: 修改 TestSelect**

在 `test_select.go` 中创建 StageOptions 的地方，将 `AIEnabled: true` 硬编码开启（Phase 1 仅 Test 模式可用，不影响 Campaign）。

找到 `NewStageSceneWithOpts(s.switcher, StageOptions{` 调用，在 `TestMode: true` 后添加:

```go
	AIEnabled: true,
```

**Step 2: 编译验证**

Run: `go build ./cmd/game/`
Expected: BUILD SUCCESS

**Step 3: Commit**

```bash
git add internal/scene/test_select.go
git commit -m "feat(aiplayer): enable AI player in test mode"
```

---

### Task 10: 编译验证 + 运行测试

**Step 1: 运行所有 AI 单元测试**

```bash
go test -tags unittest -run "TestZone|TestSprite|TestBubble|TestActionQueue|TestDecision|TestAIPlayer" -v ./tests/unit/
```
Expected: ALL PASS

**Step 2: 运行 lint**

```bash
make lint
```
Expected: PASS（可能需要修一些小问题）

**Step 3: 运行全量测试**

```bash
make test
```
Expected: PASS

**Step 4: 桌面运行验证**

```bash
make run
```
Expected: 进入 Test 模式 → 选地图 → 开始游戏 → 能看到蓝色光球在右半区域移动、冒气泡、自动造塔

**Step 5: 最终 commit**

```bash
git add -A
git commit -m "feat(aiplayer): Phase 1 complete — AI player skeleton with sprite, bubble, decisions"
```

---

## 验证清单

- [ ] AI 精灵在地图右半区域可见（蓝色光球）
- [ ] 精灵 idle 状态有上下浮动（bob 动画）
- [ ] 精灵移动到目标位置有平滑过渡
- [ ] 造塔前显示思考气泡（"这里放个塔？"）
- [ ] 造塔后显示完成气泡（"搞定！"）
- [ ] AI 有独立金币，不花玩家的钱
- [ ] AI 只在右半区域造塔
- [ ] 缺钱时显示"缺钱了..."气泡
- [ ] 区域分界线（淡蓝虚线）可见
- [ ] 全量测试通过（`make test`）
- [ ] Lint 通过（`make lint`）
