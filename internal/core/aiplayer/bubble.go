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
	BubbleThink   BubbleType = iota // 思考
	BubbleAction                    // 行动
	BubbleEmotion                   // 情绪
	BubbleReply                     // 回应
)

// BubbleManager 气泡管理器。
type BubbleManager struct {
	text    string
	btype   BubbleType
	timer   float64
	visible bool
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
			// ── 基础决策文案 ──
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
			"ping_agree": {
				"嗯？放这里？好主意",
				"有道理，我试试",
				"OK，听你的",
			},
			"ping_refuse": {
				"我觉得另一边更急...",
				"等一下，我先处理完这边",
				"嗯...我想放别的地方",
			},
			"ping_received": {
				"嗯？",
				"收到~",
			},

			// ── Phase 2-3: 拟人行为文案 ──
			"leak_reaction": {
				"啊！漏了！",
				"糟糕...",
				"那边失守了",
			},
			"kill_streak": {
				"打得不错！",
				"连杀！",
				"火力全开",
			},
			"kill_streak_big": {
				"太强了！",
				"无人能挡！",
			},
			"economy_worry": {
				"缺钱了...",
				"金币告急",
				"得省着点",
			},
			"idle_patrol": {
				"巡视一下...",
				"四处看看",
			},
			"daydream": {
				"...",
				"嗯...",
				"",
			},

			// ── Phase 2-3: 观战评论文案 ──
			"player_build": {
				"那个位置不错啊",
				"你也在补塔？",
				"好主意",
			},
			"player_sell": {
				"哦？要调整布局？",
				"卖掉了？",
			},
			"player_leak": {
				"那边漏了！",
				"小心！",
				"注意防守",
			},
			"player_rich": {
				"你还在攒钱？",
				"金币够了吧...",
			},
			"player_afk": {
				"你还在吗？",
				"需要帮忙吗？",
			},

			// ── Phase 2-3: LLM 事件触发文案 ──
			"boss_incoming": {
				"Boss来了！",
				"小心Boss！",
				"准备迎战",
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
