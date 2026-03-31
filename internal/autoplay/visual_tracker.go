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
	wardenType    string
}

// NewVisualTracker 创建视觉追踪器。
func NewVisualTracker(ss *Screenshotter) *VisualTracker {
	return &VisualTracker{
		screenshotter: ss,
		towerTypes:    make(map[string]bool),
		archetypes:    make(map[string]bool),
		attackStyles:  make(map[string]bool),
		statusFX:      make(map[string]bool),
	}
}

// Check 每帧检查游戏状态，检测新的视觉内容并请求截图。
// 返回本帧需要截图的数量（用于判断是否需要渲染本帧）。
func (v *VisualTracker) Check(state *GameState) int {
	count := 0

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

	// ── 每种攻击方式射击视觉（塔有目标 = 正在射击）──
	for _, t := range state.Towers {
		if t.HasTarget && t.AttackStyle != "" && !v.attackStyles[t.AttackStyle] {
			v.attackStyles[t.AttackStyle] = true
			v.screenshotter.RequestCapture(fmt.Sprintf("attack_%s.png", t.AttackStyle))
			count++
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

	// ── 波次公告（波次刚切换后 5 帧内截图，此时公告动画正在显示）──
	// 由 controller 在波次变化时调用 RequestWaveAnnounce
	// 这里不做，交给 controller 处理

	// ── 战灵外观（激活后截图）──
	if state.WardenReady && !v.wardenActive && (state.WardenX != 0 || state.WardenY != 0) {
		v.wardenActive = true
		v.screenshotter.RequestCapture("warden_active.png")
		count++
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
