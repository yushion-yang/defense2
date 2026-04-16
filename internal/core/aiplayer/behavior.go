// behavior.go -- AI 拟人行为系统。
//
// 追踪游戏事件并触发情绪反应、巡视动作、发呆等拟人行为。
// 设计：纯数据层，不依赖 core 包。由 AIPlayer.Tick 调用。
// 关联：sprite.go 驱动精灵运动，bubble.go 显示气泡文案。
package aiplayer

import "math/rand"

// ── 常量 ──

const (
	killStreakWindow  = 2.0  // 连杀计数窗口（秒）
	killStreakSmall   = 3    // 小连杀阈值
	killStreakBig     = 5    // 大连杀阈值
	idlePatrolTime   = 5.0  // 空闲巡视触发时间（秒）
	daydreamChance   = 0.03 // 每决策周期发呆概率 3%
	economyWorryGold = 20   // 经济焦虑金币阈值
	economyPaceRate  = 0.10 // 每 tick 周期经济焦虑触发概率 10%
	leakSprintSpeed  = 200.0 // 漏怪急跑速度 (px/s)

	// 冷却防刷屏
	leakCooldown       = 5.0 // 漏怪反应冷却
	killStreakCooldown  = 4.0 // 连杀兴奋冷却
	economyCooldown     = 8.0 // 经济焦虑冷却
	patrolCooldown      = 3.0 // 巡视气泡冷却
	daydreamDuration    = 2.0 // 发呆持续时间
)

// BehaviorState 行为状态追踪。
type BehaviorState struct {
	// ── 事件追踪 ──
	prevLives       int
	prevKills       int
	killStreak      int
	killStreakTimer float64
	idleTime        float64 // 无操作累计时间

	// ── 巡视 ──
	patrolTimer   float64
	patrolTargetX float64
	patrolTargetY float64

	// ── 冷却 ──
	leakCD       float64
	killStreakCD float64
	economyCD    float64
	patrolCD     float64

	// ── 标记 ──
	lastDecisionMade bool // 是否刚做了决策（用于重置 idle 时间）
}

// NewBehaviorState 创建行为状态。
func NewBehaviorState(lives int) *BehaviorState {
	return &BehaviorState{prevLives: lives}
}

// NotifyDecision 通知行为系统做了一个决策（重置 idle 时间）。
func (b *BehaviorState) NotifyDecision() {
	b.lastDecisionMade = true
}

// Tick 每帧更新拟人行为。
//
// 处理顺序:
//  1. 冷却递减
//  2. 漏怪急跑
//  3. 连杀兴奋
//  4. 经济焦虑
//  5. 空闲巡视 / 发呆
func (b *BehaviorState) Tick(dt float64, snap AISnapshot, gold int,
	sprite *Sprite, bubble *BubbleManager, dialogue *DialogueBank,
	actionsBusy bool) {

	// ── 1. 冷却递减 ──
	b.leakCD -= dt
	b.killStreakCD -= dt
	b.economyCD -= dt
	b.patrolCD -= dt

	// ── 2. 漏怪急跑 ──
	if snap.Lives < b.prevLives && b.leakCD <= 0 {
		b.leakCD = leakCooldown
		// 精灵急跑向基地区域（地图右侧终点附近）
		baseX := snap.MapCenterX + 200
		baseY := snap.MapCenterY
		if len(snap.PathPoints) > 0 {
			// 取路径终点（最后一个路径点）作为基地方向
			last := snap.PathPoints[len(snap.PathPoints)-1]
			baseX = last.X
			baseY = last.Y
		}
		sprite.MoveTo(baseX, baseY)
		bubble.Show(dialogue.Random("leak_reaction"), BubbleEmotion, 2.0)
		b.idleTime = 0
	}
	b.prevLives = snap.Lives

	// ── 3. 连杀兴奋 ──
	totalKills := b.countTotalKills(snap)
	if totalKills > b.prevKills {
		newKills := totalKills - b.prevKills
		b.killStreak += newKills
		b.killStreakTimer = killStreakWindow
	}
	b.prevKills = totalKills

	b.killStreakTimer -= dt
	if b.killStreakTimer <= 0 {
		b.killStreak = 0
	}

	if b.killStreak >= killStreakBig && b.killStreakCD <= 0 {
		b.killStreakCD = killStreakCooldown
		bubble.Show(dialogue.Random("kill_streak_big"), BubbleEmotion, 2.0)
		b.killStreak = 0
	} else if b.killStreak >= killStreakSmall && b.killStreakCD <= 0 {
		b.killStreakCD = killStreakCooldown
		bubble.Show(dialogue.Random("kill_streak"), BubbleEmotion, 1.5)
		b.killStreak = 0
	}

	// ── 4. 经济焦虑 ──
	if gold < economyWorryGold && !actionsBusy && b.economyCD <= 0 {
		if rand.Float64() < economyPaceRate {
			b.economyCD = economyCooldown
			bubble.Show(dialogue.Random("economy_worry"), BubbleEmotion, 2.0)
			// 来回踱步：在当前位置附近随机偏移
			offsetX := (rand.Float64() - 0.5) * 80
			offsetY := (rand.Float64() - 0.5) * 40
			sprite.MoveTo(sprite.X()+offsetX, sprite.Y()+offsetY)
		}
	}

	// ── 5. idle 时间追踪 ──
	if b.lastDecisionMade {
		b.idleTime = 0
		b.lastDecisionMade = false
	} else if !actionsBusy {
		b.idleTime += dt
	}

	// ── 6. 发呆（每决策周期 3% 概率）──
	// 仅在 idle 且无气泡时触发
	if b.idleTime > 1.0 && !bubble.Visible() && !actionsBusy {
		if rand.Float64() < daydreamChance*dt {
			sprite.SetThinking(daydreamDuration)
			bubble.Show(dialogue.Random("daydream"), BubbleThink, daydreamDuration)
			b.idleTime = 0
			return
		}
	}

	// ── 7. 空闲巡视 ──
	if b.idleTime > idlePatrolTime && sprite.State() == SpriteIdle && b.patrolCD <= 0 {
		b.patrolCD = patrolCooldown
		// 在 AI 区域内随机选一个点巡视
		b.patrolTargetX = snap.MapCenterX + (rand.Float64()-0.3)*300
		b.patrolTargetY = snap.MapCenterY + (rand.Float64()-0.5)*200
		sprite.MoveTo(b.patrolTargetX, b.patrolTargetY)
		if !bubble.Visible() {
			bubble.Show(dialogue.Random("idle_patrol"), BubbleThink, 1.5)
		}
		b.idleTime = 0
	}
}

// countTotalKills 统计快照中所有 AI 塔的总击杀数。
func (b *BehaviorState) countTotalKills(snap AISnapshot) int {
	total := 0
	for _, t := range snap.Towers {
		total += t.Kills
	}
	return total
}
