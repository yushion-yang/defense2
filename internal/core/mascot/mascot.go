package mascot

import "math/rand"

// MascotVM is the view-model snapshot for rendering.
type MascotVM struct {
	Visible          bool
	HasDialog        bool
	Text             string
	Expression       string
	CanClick         bool // true = click to advance
	AbilityReady     bool
	AbilityHintShown bool    // true when ability-hint bubble is visible (click = use ability)
	AbilityHintText  string  // 独立于普通对话的技能提示文本（可与 Text 同时存在）
	CooldownPct      float64 // 0.0 = ready, 1.0 = full cooldown
}

// Guide is the mascot state machine that drives dialog playback.
type Guide struct {
	dialogs  []Dialog
	locale   string          // 当前语言代码，如 "zh"/"en"
	scene    string          // current scene name
	shownIDs map[string]bool // Once dialog IDs that have been shown

	// active dialog state
	active  *Dialog
	lineIdx int
	timer   float64

	// Condition evaluation
	condFuncs    []ConditionFunc
	condState    *ConditionState
	evalInterval float64 // seconds between evaluations (default 5.0)
	evalTimer    float64
	lastCtx      *GameContext

	// Ability system
	abilityCooldown    float64 // remaining seconds (0 = ready)
	abilityCooldownMax float64 // default 30.0
	pendingAction      *MascotAction
	abilityHinted      bool // true after hint dialog was shown this cooldown cycle
	abilityHintActive  bool // true while the hint dialog is being displayed

	// 独立技能提示气泡（不占用 active 对话槽，可与普通对话共存）
	abilityHintText  string  // 当前显示的技能提示文本（空=不显示）
	abilityHintTimer float64 // 提示周期计时器
	abilityHintLines []Line  // 缓存的技能提示文案（从 dialogs 预筛选）
}

// NewGuide creates a Guide preloaded with dialogs.
// shownIDs may be nil; if non-nil it restores previously-seen Once dialog IDs.
func NewGuide(dialogs []Dialog, shownIDs map[string]bool) *Guide {
	shown := shownIDs
	if shown == nil {
		shown = make(map[string]bool)
	}
	guide := &Guide{
		dialogs:            dialogs,
		shownIDs:           shown,
		evalInterval:       5.0,
		abilityCooldownMax: 30.0,
	}
	guide.cacheAbilityHintLines()
	return guide
}

// SetLocale 设置当前语言，影响对话文本的选择。
func (g *Guide) SetLocale(locale string) {
	g.locale = locale
}

// cacheAbilityHintLines 从已加载的 dialogs 中筛选 mascot_ability_hint 触发器的文案，
// 缓存到 abilityHintLines 供独立提示气泡使用。
func (g *Guide) cacheAbilityHintLines() {
	for i := range g.dialogs {
		d := &g.dialogs[i]
		if d.Trigger == "mascot_ability_hint" && len(d.Lines) > 0 {
			g.abilityHintLines = append(g.abilityHintLines, d.Lines[0])
		}
	}
}

// resolveText 根据当前 locale 从多语言映射中选择文本。
// 优先使用当前 locale，缺失或为空时 fallback 到 "zh"。
func (g *Guide) resolveText(texts map[string]string) string {
	if g.locale != "" {
		if t, ok := texts[g.locale]; ok && t != "" {
			return t
		}
	}
	if t, ok := texts["zh"]; ok {
		return t
	}
	// 兜底：返回任意一个值
	for _, t := range texts {
		return t
	}
	return ""
}

// SetScene switches the current scene and auto-triggers any "scene_enter" dialog.
func (g *Guide) SetScene(name string) {
	g.scene = name
	g.tryTrigger("scene_enter")
}

// Trigger fires a named event and starts the best matching dialog (if idle).
func (g *Guide) Trigger(event string) {
	g.tryTrigger(event)
}

// 技能提示气泡周期常量（独立于普通对话，不占用 active 槽）
const (
	abilityHintShowDur float64 = 8.0 // 提示显示时长
	abilityHintHideDur float64 = 5.0 // 提示隐藏间隔
)

// Tick advances auto-advance timers and decrements ability cooldown. dt is in seconds.
func (g *Guide) Tick(dt float64) {
	// Decrement ability cooldown.
	if g.abilityCooldown > 0 {
		g.abilityCooldown -= dt
		if g.abilityCooldown < 0 {
			g.abilityCooldown = 0
		}
	}

	// ── 独立技能提示气泡管理 ──
	// 技能就绪时循环显示提示（显示 8s → 隐藏 5s → 换一条再显示），
	// 不占用 active 对话槽，可与普通对话同时存在。
	g.tickAbilityHint(dt)

	// Decrement condition eval timer.
	if g.condFuncs != nil {
		g.evalTimer -= dt
	}

	if g.active == nil {
		return
	}
	line := &g.active.Lines[g.lineIdx]
	if line.AutoAdvance <= 0 {
		return
	}
	g.timer += dt
	if g.timer >= line.AutoAdvance {
		g.advance()
	}
}

// tickAbilityHint 管理独立技能提示气泡的显隐循环。
// 冷却结束后立即显示第一条提示，之后按 显示8s → 隐藏5s 循环。
func (g *Guide) tickAbilityHint(dt float64) {
	if !g.AbilityReady() || len(g.abilityHintLines) == 0 {
		// 技能未就绪或无文案 → 清空提示，重置 hinted 标记
		g.abilityHintText = ""
		g.abilityHintActive = false
		g.abilityHinted = false
		return
	}

	// 冷却刚结束 → 立即显示第一条提示（不等 hideDur）
	if !g.abilityHinted {
		g.abilityHinted = true
		pick := g.abilityHintLines[rand.Intn(len(g.abilityHintLines))]
		g.abilityHintText = g.resolveText(pick.Text)
		g.abilityHintActive = true
		g.abilityHintTimer = 0
		return
	}

	g.abilityHintTimer += dt

	if g.abilityHintText != "" {
		// 正在显示 → 到时间则隐藏
		if g.abilityHintTimer >= abilityHintShowDur {
			g.abilityHintText = ""
			g.abilityHintActive = false
			g.abilityHintTimer = 0
		}
	} else {
		// 正在隐藏 → 到时间则随机选一条显示
		if g.abilityHintTimer >= abilityHintHideDur {
			pick := g.abilityHintLines[rand.Intn(len(g.abilityHintLines))]
			g.abilityHintText = g.resolveText(pick.Text)
			g.abilityHintActive = true
			g.abilityHintTimer = 0
		}
	}
}

// ClickAdvance manually advances to the next line (or finishes the dialog).
// Works on any line, including auto-advance lines.
func (g *Guide) ClickAdvance() {
	if g.active == nil {
		return
	}
	g.advance()
}

// VM returns a snapshot of the current mascot state for rendering.
func (g *Guide) VM() MascotVM {
	vm := MascotVM{
		Visible:          true,
		AbilityReady:     g.AbilityReady(),
		AbilityHintShown: g.abilityHintActive,
		AbilityHintText:  g.abilityHintText,
	}
	if g.abilityCooldownMax > 0 && g.abilityCooldown > 0 {
		vm.CooldownPct = g.abilityCooldown / g.abilityCooldownMax
		if vm.CooldownPct > 1.0 {
			vm.CooldownPct = 1.0
		}
	}
	if g.active == nil {
		return vm
	}
	line := &g.active.Lines[g.lineIdx]
	vm.HasDialog = true
	vm.Text = g.resolveText(line.Text)
	vm.Expression = line.Expression
	vm.CanClick = true
	return vm
}

// ShownIDs returns the set of Once dialog IDs that have been shown.
// The caller should persist this map and pass it back via NewGuide on restart.
func (g *Guide) ShownIDs() map[string]bool {
	out := make(map[string]bool, len(g.shownIDs))
	for k, v := range g.shownIDs {
		out[k] = v
	}
	return out
}

// InitConditions sets up condition evaluation with the given condition functions.
// If funcs is nil, no automatic condition evaluation occurs.
func (g *Guide) InitConditions(funcs []ConditionFunc) {
	g.condFuncs = funcs
	g.condState = NewConditionState()
	g.evalTimer = g.evalInterval
}

// UpdateContext stores the latest game context and evaluates conditions when the
// evaluation timer has elapsed.
func (g *Guide) UpdateContext(ctx GameContext) {
	ctxCopy := ctx
	g.lastCtx = &ctxCopy

	if g.condFuncs == nil || g.condState == nil {
		return
	}

	if g.evalTimer > 0 {
		return
	}

	// Reset eval timer.
	g.evalTimer = g.evalInterval

	// Evaluate conditions: first non-empty result wins.
	for _, fn := range g.condFuncs {
		trigger := fn(&ctx, g.condState)
		if trigger != "" {
			g.condState.markFired(trigger, ctx.SessionSecs)
			g.tryTrigger(trigger)
			break
		}
	}

	// Update tracking state.
	g.condState.PrevWave = ctx.Wave
	g.condState.PrevInStage = ctx.InStage
	g.condState.PrevInteractMode = ctx.InteractMode
	g.condState.PrevQuality = ctx.QualityLevel

	// Meta tracking: gold/lives/kills for anomaly detection.
	if ctx.InStage {
		g.condState.PrevGold = ctx.Gold
		g.condState.PrevGoldInit = true
		g.condState.PrevLives = ctx.Lives
		g.condState.PrevLivesInit = true
		g.condState.PrevKills = ctx.Kills
	}

	// Meta tracking: pause accumulation (evalInterval is the tick period).
	if ctx.InStage && ctx.Paused {
		g.condState.PauseAccum += g.evalInterval
	} else {
		g.condState.PauseAccum = 0
	}

	// Meta tracking: defeat counting.
	if ctx.Defeat && !g.condState.PrevDefeat {
		g.condState.StageDefeats++
	}
	g.condState.PrevVictory = ctx.Victory
	g.condState.PrevDefeat = ctx.Defeat
}

// HasActiveDialog returns true if a dialog is currently being displayed.
func (g *Guide) HasActiveDialog() bool {
	return g.active != nil
}

// AbilityReady returns true if the mascot ability can be used.
// 经典模式下禁用技能援助（纯靠玩家自己的塔防策略）。
func (g *Guide) AbilityReady() bool {
	return g.abilityCooldown <= 0 && g.lastCtx != nil && g.lastCtx.InStage && !g.lastCtx.IsClassicMode
}

// IsAbilityHintActive returns true if the ability-hint dialog is currently showing.
// When true, clicking the mascot should trigger the ability instead of advancing dialog.
func (g *Guide) IsAbilityHintActive() bool {
	return g.abilityHintActive
}

// RequestHelp requests the mascot to perform its battle assistance ability.
// Returns the action if ability is ready, or nil if on cooldown / not in stage.
func (g *Guide) RequestHelp() *MascotAction {
	if !g.AbilityReady() {
		g.ForceTrigger("mascot_not_ready")
		return nil
	}

	action := &MascotAction{Type: ActionKillWeakEnemy}
	g.pendingAction = action
	g.abilityCooldown = g.abilityCooldownMax
	g.abilityHinted = false
	g.abilityHintActive = false
	g.abilityHintText = ""
	g.abilityHintTimer = 0
	g.ForceTrigger("mascot_help")
	return action
}

// ConsumeAction returns and clears the pending mascot action.
func (g *Guide) ConsumeAction() *MascotAction {
	a := g.pendingAction
	g.pendingAction = nil
	return a
}

// NotifyActionComplete informs the mascot that an action was completed.
func (g *Guide) NotifyActionComplete(t ActionType) {
	_ = t // reserved for future action-type-specific responses
	g.ForceTrigger("mascot_kill_success")
}

// ForceTrigger fires a named event, interrupting any active dialog.
// Used for high-priority events like panic recovery.
// Randomly picks among all matching dialogs (not just highest priority).
func (g *Guide) ForceTrigger(event string) {
	// Finish current dialog if any.
	if g.active != nil {
		g.finishDialog()
	}
	var candidates []*Dialog
	for i := range g.dialogs {
		d := &g.dialogs[i]
		if d.Trigger != event {
			continue
		}
		if d.Scene != "*" && d.Scene != g.scene {
			continue
		}
		candidates = append(candidates, d)
	}
	if len(candidates) > 0 {
		pick := candidates[rand.Intn(len(candidates))]
		g.startDialog(pick)
	}
}

// tryTrigger finds the best matching dialog for the given event and starts it.
// Does nothing if a dialog is already active.
// When multiple dialogs share the highest priority, one is chosen at random.
func (g *Guide) tryTrigger(event string) {
	if g.active != nil {
		return
	}
	var candidates []*Dialog
	bestPri := -1
	for i := range g.dialogs {
		d := &g.dialogs[i]
		if d.Trigger != event {
			continue
		}
		if d.Scene != "*" && d.Scene != g.scene {
			continue
		}
		if d.Once && g.shownIDs[d.ID] {
			continue
		}
		if d.Priority > bestPri {
			bestPri = d.Priority
			candidates = candidates[:0]
			candidates = append(candidates, d)
		} else if d.Priority == bestPri {
			candidates = append(candidates, d)
		}
	}
	if len(candidates) > 0 {
		pick := candidates[rand.Intn(len(candidates))]
		g.startDialog(pick)
	}
}

// startDialog begins playback of a dialog.
func (g *Guide) startDialog(d *Dialog) {
	g.active = d
	g.lineIdx = 0
	g.timer = 0
}

// advance moves to the next line or finishes the dialog.
func (g *Guide) advance() {
	g.lineIdx++
	g.timer = 0
	if g.lineIdx >= len(g.active.Lines) {
		g.finishDialog()
	}
}

// finishDialog ends the current dialog and marks Once dialogs as shown.
func (g *Guide) finishDialog() {
	if g.active.Once {
		g.shownIDs[g.active.ID] = true
	}
	g.active = nil
	g.lineIdx = 0
	g.timer = 0
}
