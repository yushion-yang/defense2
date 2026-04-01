// visual_tracker.go — 视觉内容截图追踪器。
// 检测游戏内容的首次视觉出现，触发对应截图以构建完整视觉目录。
// 每种视觉内容只截一次，避免重复。
package autoplay

import "fmt"

// VisualTracker 追踪哪些视觉内容已被截图。
type VisualTracker struct {
	screenshotter *Screenshotter

	towerTypes   map[string]bool // 每种塔外观
	archetypes   map[string]bool // 每种敌人外观
	attackStyles map[string]bool // 每种攻击方式射击视觉
	statusFX     map[string]bool // 每种状态效果视觉

	bossAppear    bool
	dyingEnemy    bool
	towerBuilding bool
	waveAnnounce  bool
	wardenActive  bool
	wardenCombat  bool // 战灵战斗中截图
	wardenType    string

	// 延迟截图：首次建塔/攻击后等几帧再截（让特效有时间出现）
	pendingDelayed map[string]int // filename -> 剩余延迟帧数
}

// NewVisualTracker 创建视觉追踪器。
func NewVisualTracker(ss *Screenshotter) *VisualTracker {
	return &VisualTracker{
		screenshotter:  ss,
		towerTypes:     make(map[string]bool),
		archetypes:     make(map[string]bool),
		attackStyles:   make(map[string]bool),
		statusFX:       make(map[string]bool),
		pendingDelayed: make(map[string]int),
	}
}

// Check 每帧检查游戏状态，检测新的视觉内容并请求截图。
// 返回本帧需要截图的数量（用于判断是否需要渲染本帧）。
func (v *VisualTracker) Check(state *GameState) int {
	count := 0

	// ── 处理延迟截图（等特效出现后再截）──
	for fname, frames := range v.pendingDelayed {
		if frames <= 0 {
			v.screenshotter.RequestCapture(fname)
			delete(v.pendingDelayed, fname)
			count++
		} else {
			v.pendingDelayed[fname] = frames - 1
		}
	}

	// ── 每种塔类型首次出现 ──
	for _, t := range state.Towers {
		if !v.towerTypes[t.Key] {
			v.towerTypes[t.Key] = true
			v.screenshotter.RequestCapture(fmt.Sprintf("tower_%s.png", t.Key))
			count++
		}
	}

	// ── 每种敌人原型首次出现 ──
	for _, e := range state.Enemies {
		if !e.Active || e.Dying {
			continue
		}
		if !v.archetypes[e.Archetype] {
			v.archetypes[e.Archetype] = true
			v.screenshotter.RequestCapture(fmt.Sprintf("enemy_%s.png", e.Archetype))
			count++
		}
		// Boss 首次出现
		if e.Boss && !v.bossAppear {
			v.bossAppear = true
			v.screenshotter.RequestCapture("boss_appear.png")
			count++
		}
	}

	// ── 每种攻击方式射击视觉（延迟 10 帧截图，让弹射物/特效飞出来）──
	for _, t := range state.Towers {
		if t.HasTarget && t.AttackStyle != "" && !v.attackStyles[t.AttackStyle] {
			v.attackStyles[t.AttackStyle] = true
			fname := fmt.Sprintf("attack_%s.png", t.AttackStyle)
			v.pendingDelayed[fname] = 10 // 延迟 10 帧
		}
	}

	// ── 每种状态效果首次出现 ──
	for _, e := range state.Enemies {
		if !e.Active {
			continue
		}
		if e.IsSlowed && !v.statusFX["slow"] {
			v.statusFX["slow"] = true
			v.screenshotter.RequestCapture("status_slow.png")
			count++
		}
		if e.IsStunned && !v.statusFX["stun"] {
			v.statusFX["stun"] = true
			v.screenshotter.RequestCapture("status_stun.png")
			count++
		}
		if e.IsBurning && !v.statusFX["burn"] {
			v.statusFX["burn"] = true
			v.screenshotter.RequestCapture("status_burn.png")
			count++
		}
		if e.IsBleeding && !v.statusFX["bleed"] {
			v.statusFX["bleed"] = true
			v.screenshotter.RequestCapture("status_bleed.png")
			count++
		}
		if e.IsRooted && !v.statusFX["root"] {
			v.statusFX["root"] = true
			v.screenshotter.RequestCapture("status_root.png")
			count++
		}
	}

	// ── 敌人死亡动画 ──
	if !v.dyingEnemy {
		for _, e := range state.Enemies {
			if e.Dying {
				v.dyingEnemy = true
				v.screenshotter.RequestCapture("enemy_dying.png")
				count++
				break
			}
		}
	}

	// ── 战灵外观（激活后截图）──
	if state.WardenReady && !v.wardenActive && (state.WardenX != 0 || state.WardenY != 0) {
		v.wardenActive = true
		v.screenshotter.RequestCapture("warden_active.png")
		count++
	}

	// ── 战灵战斗中截图（战灵附近有敌人时）──
	if state.WardenReady && v.wardenActive && !v.wardenCombat {
		for _, e := range state.Enemies {
			if !e.Active || e.Dying {
				continue
			}
			dx := e.X - state.WardenX
			dy := e.Y - state.WardenY
			if dx*dx+dy*dy < 200*200 { // 战灵 200px 范围内有敌人
				v.wardenCombat = true
				v.pendingDelayed["warden_combat.png"] = 15 // 延迟 15 帧让攻击动画出现
				break
			}
		}
	}

	return count
}

// Summary 返回已捕获的视觉内容统计。
func (v *VisualTracker) Summary() map[string]int {
	return map[string]int{
		"tower_types":    len(v.towerTypes),
		"archetypes":     len(v.archetypes),
		"attack_styles":  len(v.attackStyles),
		"status_effects": len(v.statusFX),
	}
}
