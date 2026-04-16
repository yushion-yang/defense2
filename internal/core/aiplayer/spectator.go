// spectator.go -- AI 观战评论系统。
//
// 追踪人类玩家的行为变化，生成评论性弹幕。
// 设计：纯数据层，不依赖 core 包。由 AIPlayer.Tick 调用。
// 关联：bubble.go 消费评论文案，aiplayer.go 注入人类玩家数据。
package aiplayer

import "math/rand"

// ── 常量 ──

const (
	spectatorCheckInterval = 3.0  // 检查间隔（秒）
	spectatorCommentCD     = 8.0  // 评论冷却（秒）
	spectatorRichThreshold = 200  // "很有钱"金币阈值
	spectatorAFKTime       = 15.0 // 无操作判定时间（秒）
)

// SpectatorState 观战评论状态。
type SpectatorState struct {
	prevHumanTowers int
	prevHumanGold   int
	prevLives       int

	// ── 计时器 ──
	checkTimer      float64 // 检查间隔计时
	lastCommentTime float64 // 距上次评论的累计时间
	commentCooldown float64 // 评论冷却剩余

	// ── AFK 检测 ──
	humanIdleTime        float64 // 人类无操作累计时间
	humanTowerUnchanged  float64 // 塔数不变的累计时间
}

// NewSpectatorState 创建观战评论状态。
func NewSpectatorState(lives int) *SpectatorState {
	return &SpectatorState{
		prevLives: lives,
	}
}

// Tick 每帧更新观战评论。
//
// 检查人类玩家行为变化并触发评论气泡。
// 评论间隔至少 8 秒，避免刷屏。
func (ss *SpectatorState) Tick(dt float64, snap AISnapshot,
	bubble *BubbleManager, dialogue *DialogueBank) {

	// 冷却递减
	ss.commentCooldown -= dt
	ss.lastCommentTime += dt

	// AFK 时间追踪
	if snap.HumanTowerCount == ss.prevHumanTowers {
		ss.humanTowerUnchanged += dt
	} else {
		ss.humanTowerUnchanged = 0
	}

	// 按间隔检查（非每帧，减少开销）
	ss.checkTimer -= dt
	if ss.checkTimer > 0 {
		return
	}
	ss.checkTimer = spectatorCheckInterval

	// 冷却中不评论
	if ss.commentCooldown > 0 {
		ss.updatePrevState(snap)
		return
	}

	// 气泡正在显示时不抢占
	if bubble.Visible() {
		ss.updatePrevState(snap)
		return
	}

	// ── 评论触发（按优先级排列）──

	// 1. 玩家漏怪（最紧急）
	if snap.Lives < ss.prevLives {
		ss.showComment(bubble, dialogue.Random("player_leak"))
		ss.updatePrevState(snap)
		return
	}

	// 2. 玩家造塔
	if snap.HumanTowerCount > ss.prevHumanTowers {
		ss.showComment(bubble, dialogue.Random("player_build"))
		ss.updatePrevState(snap)
		return
	}

	// 3. 玩家卖塔
	if snap.HumanTowerCount < ss.prevHumanTowers {
		ss.showComment(bubble, dialogue.Random("player_sell"))
		ss.updatePrevState(snap)
		return
	}

	// 4. 玩家很有钱但不造塔
	if snap.HumanGold > spectatorRichThreshold && snap.HumanTowerCount == ss.prevHumanTowers {
		// 50% 概率触发（不是每次都吐槽）
		if rand.Float64() < 0.5 {
			ss.showComment(bubble, dialogue.Random("player_rich"))
			ss.updatePrevState(snap)
			return
		}
	}

	// 5. 玩家长时间不操作
	if ss.humanTowerUnchanged > spectatorAFKTime {
		ss.showComment(bubble, dialogue.Random("player_afk"))
		ss.humanTowerUnchanged = 0 // 重置，避免反复触发
		ss.updatePrevState(snap)
		return
	}

	ss.updatePrevState(snap)
}

// showComment 显示评论并重置冷却。
func (ss *SpectatorState) showComment(bubble *BubbleManager, text string) {
	if text == "" {
		return
	}
	bubble.Show(text, BubbleAction, 2.5)
	ss.commentCooldown = spectatorCommentCD
	ss.lastCommentTime = 0
}

// updatePrevState 更新上一帧的状态快照。
func (ss *SpectatorState) updatePrevState(snap AISnapshot) {
	ss.prevHumanTowers = snap.HumanTowerCount
	ss.prevHumanGold = snap.HumanGold
	ss.prevLives = snap.Lives
}
