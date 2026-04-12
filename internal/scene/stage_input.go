// stage_input.go — StageScene 的输入处理和交互状态机。
// 包含 handleInput、handlePausedInput、handleWardenSelection
// 以及相机控制、手势初始化等辅助方法。
package scene

import (
	"fmt"
	"math"

	gameAudio "defense2/internal/audio"
	"defense2/internal/core/achievement"
	"defense2/internal/core/enemy"
	"defense2/internal/core/event"
	"defense2/internal/core/game"
	"defense2/internal/core/item"
	"defense2/internal/core/tower"
	"defense2/internal/input"
	"defense2/internal/render/draw"
	"defense2/internal/render/hud"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// handleInput 基于手势识别器 + 交互状态机处理输入。
func (s *StageScene) handleInput() {
	g := s.gesture
	// 状态机控制拖拽权限（modeTowerSel 下禁用，防止点击取消被误判为拖拽）
	switch s.imode {
	case modeIdle, modeBuildPlace, modeSpawnPlace:
		g.DragEnabled = s.needsCamera()
	default:
		g.DragEnabled = false
	}
	g.Update()

	// 道具拖拽模式：跟踪松手释放
	if s.imode == modeItemDrag && s.dragItemActive {
		// 更新拖拽悬停目标
		mx, my := g.CursorPos()
		wtx, wty := s.screenToWorld(mx, my)
		s.dragHoverTower = s.towerAtPixel(wtx, wty)

		// 检测鼠标/触摸释放
		released := inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
		if !released {
			if ids := inpututil.AppendJustReleasedTouchIDs(nil); len(ids) > 0 {
				released = true
			}
		}
		if released {
			target := s.dragHoverTower
			if target != nil && !target.Selling {
				item.ApplyItem(target, s.dragItemKind)
				s.inventory.Use(s.dragItemKind)
				s.gameStats.ItemsUsed++
				hud.ShowToast(item.Defs[s.dragItemKind].Name + " → " + target.Label)
				s.audioMgr.PlaySafe(gameAudio.SFXUIClick)
				s.tutorial.Trigger("item_use")
				// 成就: 道具使用次数
				s.achieveTracker.SessionItemsUsed++
				if s.achieveTracker.SessionItemsUsed >= achievement.ThresholdOf("item_master") {
					if s.achieveTracker.Unlock("item_master") {
						hud.ShowToast("成就解锁: 道具大师")
					}
				}
			} else {
				hud.ShowToast("请拖拽到炮塔上使用")
			}
			s.dragItemActive = false
			s.dragHoverTower = nil
			s.imode = modeItemPanel
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.dragItemActive = false
			s.dragHoverTower = nil
			s.imode = modeItemPanel
		}
		return
	}

	mx, my := g.CursorPos()
	fmx, fmy := float32(mx), float32(my)

	// ── 拖拽 → 平移相机 ──
	if g.IsDragging() && s.needsCamera() {
		dx, dy := g.DragDelta()
		s.camX -= dx
		s.camY -= dy
		s.clampCamera()
	}

	// ── 滚轮 ──
	_, sy := g.ScrollDelta()
	if sy != 0 && s.debugPanelOpen {
		hud.DebugPanelScroll(sy)
	} else if sy != 0 && s.needsCamera() {
		s.camY -= sy * 3
		s.clampCamera()
	}

	// ── Hover 更新 ──
	if s.imode == modeSpawnMenu {
		s.spawnHoverIdx = hud.SpawnMenuHoverTest(fmx, fmy, len(s.spawnEntries()))
	} else {
		s.spawnHoverIdx = -1
	}

	// ── 键盘快捷键 ──
	// modeUpgrade: ChoicePanel 每帧更新（在键盘和 tap 之前）
	if s.imode == modeUpgrade && s.choicePanel != nil && s.choicePanel.IsActive() {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.choicePanel.Close()
			s.imode = modeTowerSel
			return
		}
		mx, my := draw.CursorPos()
		clicked := g.JustTapped()
		s.choicePanel.Update(mx, my, clicked)
		if !s.choicePanel.IsActive() {
			// 选择回调已关闭面板 → 回到 modeTowerSel
			s.imode = modeTowerSel
		}
		return // modeUpgrade 吃掉所有输入
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		switch s.imode {
		case modeBuildMenu, modeBuildPlace:
			s.selectedTower = nil
			s.imode = modeIdle
			s.audioMgr.PlayAt(gameAudio.SFXUIClose, gameAudio.VolUI)
		case modeTowerSel:
			s.selectedTower = nil
			s.imode = modeIdle
			s.audioMgr.PlayAt(gameAudio.SFXUIClose, gameAudio.VolUI)
		case modeUpgrade:
			s.choicePanel.Close()
			s.imode = modeTowerSel
		case modeSpawnMenu, modeSpawnPlace:
			s.spawnMode = false
			s.spawnType = ""
			s.imode = modeIdle
			s.audioMgr.PlayAt(gameAudio.SFXUIClose, gameAudio.VolUI)
		case modeItemPanel:
			s.imode = modeIdle
			s.itemPanelOpen = false
			s.dragItemActive = false
			s.audioMgr.PlayAt(gameAudio.SFXUIClose, gameAudio.VolUI)
		case modeItemDrag:
			s.dragItemActive = false
			s.imode = modeItemPanel
		default:
			s.prePauseMode = s.imode
			s.imode = modePaused
			s.postPipeline.Effects.SetDesaturation(0.7, 3.0, 0.5, 0.5, 0.6)
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		if s.imode == modeBuildMenu || s.imode == modeBuildPlace {
			s.imode = modeIdle
			s.audioMgr.PlayAt(gameAudio.SFXUIClose, gameAudio.VolUI)
		} else {
			// 清除造怪/道具状态
			s.spawnMode = false
			s.spawnType = ""
			s.itemPanelOpen = false
			s.dragItemActive = false
			s.selectedTower = nil
			s.enterBuildMode()
			s.audioMgr.PlayAt(gameAudio.SFXUIOpen, gameAudio.VolUI)
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		if s.imode == modeItemPanel {
			s.imode = modeIdle
			s.itemPanelOpen = false
			s.audioMgr.PlayAt(gameAudio.SFXUIClose, gameAudio.VolUI)
		} else if s.inventory.TotalCount() > 0 {
			// 清除造怪/建造状态
			s.spawnMode = false
			s.spawnType = ""
			s.imode = modeItemPanel
			s.itemPanelOpen = true
			s.selectedTower = nil
			s.audioMgr.PlayAt(gameAudio.SFXUIOpen, gameAudio.VolUI)
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyU) && s.imode == modeTowerSel {
		s.tryUpgradeTower()
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) && (s.imode == modeIdle || s.imode == modeTowerSel) {
		if !s.spawner.WaveActive && !s.spawner.AllDone {
			s.tryStartWave()
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		s.gameSpeed = 1
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		s.gameSpeed = 2
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		s.gameSpeed = 3
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		// modePaused 由 handlePausedInput 处理，此处只处理非暂停→暂停
		s.prePauseMode = s.imode
		s.imode = modePaused
		s.postPipeline.Effects.SetDesaturation(0.7, 3.0, 0.5, 0.5, 0.6)
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if s.imode == modeTowerSel && s.selectedTower != nil {
			s.trySellTower(s.selectedTower.X, s.selectedTower.Y)
			s.imode = modeIdle
			return
		}
	}

	// F2: 调试覆盖层
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		s.debugOverlay.Toggle()
	}

	// 测试模式专用快捷键
	if s.testMode {
		// Delete: 消灭距离鼠标最近的怪物
		if inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
			if s.hoveredEnemy != nil && s.hoveredEnemy.Active && !s.hoveredEnemy.IsDying() {
				s.enemies.Kill(s.hoveredEnemy)
				hud.ShowToast("已消灭: " + s.hoveredEnemy.Archetype)
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyD) {
			s.debugPanelOpen = !s.debugPanelOpen
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyG) {
			s.gold += 500
			hud.ShowToast("+500 金币")
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyK) {
			s.enemies.Each(func(e *enemy.Enemy) {
				e.HP = 0
			})
			hud.ShowToast("清除全场敌人")
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyN) {
			s.enemies.Each(func(e *enemy.Enemy) {
				e.HP = 0
			})
			s.spawner.WaveActive = false
			prevWave := s.spawner.Wave
			s.spawner.StartNextWave()
			if s.spawner.Wave > prevWave {
				s.onWaveTransition(prevWave)
			}
			hud.ShowToast("跳到下一波")
		}
	}

	// ── Hover 更新（每帧） ──
	if s.imode == modeBuildMenu {
		s.buildHoverIdx = hud.BuildMenuHoverTest(fmx, fmy, s.buildMenuTotalCards())
	} else {
		s.buildHoverIdx = -1
	}

	// 测试模式：始终跟踪距离鼠标最近的敌人
	if s.testMode {
		s.hoveredEnemy = s.nearestEnemy(float64(fmx), float64(fmy))
	}

	// 道具面板：检测按下开始拖拽（不等松开）
	if s.imode == modeItemPanel {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
			cards := s.buildItemPanelCards()
			idx := hud.ItemPanelHitTest(fmx, fmy, cards)
			if idx >= 0 {
				s.dragItemKind = item.Kind(idx)
				s.dragItemActive = true
				s.imode = modeItemDrag
				return
			}
		}
	}

	// ── Tap → 游戏操作 ──
	if !g.JustTapped() {
		return
	}
	tapX, tapY := g.TapPos()
	ftx, fty := float32(tapX), float32(tapY)
	wtx, wty := s.screenToWorld(tapX, tapY)

	// 教程点击推进（无事件要求的步骤可任意点击推进）
	if s.tutorial.ClickAdvance() {
		return
	}

	// 调试面板点击（优先级最高）
	if s.testMode && s.debugPanelOpen {
		actions := s.debugActions()
		idx := hud.DebugPanelHitTest(ftx, fty, actions)
		if idx == -2 {
			s.debugPanelOpen = false
			return
		}
		if idx >= 0 && idx < len(actions) && actions[idx].Action != nil {
			actions[idx].Action()
			return
		}
	}

	// 左下角切换按钮（波次面板）
	if hud.WavePanelHandleHitTest(ftx, fty, &s.wavePanelState) {
		s.wavePanelOpen = !s.wavePanelOpen
		return
	}
	// 右下角切换按钮（战灵面板）
	if hud.ToggleButtonHitTest(ftx, fty, false) {
		s.wardenPanelOpen = !s.wardenPanelOpen
		if s.wardenPanelOpen {
			if s.imode == modeBuildMenu || s.imode == modeBuildPlace || s.imode == modeTowerSel {
				s.imode = modeIdle
			}
			s.selectedTower = nil
		}
		return
	}

	// TopBar 按钮
	topBtn := hud.TopBarHitTest(ftx, fty)
	switch topBtn {
	case "start":
		if !s.spawner.WaveActive && !s.spawner.AllDone {
			s.tryStartWave()
		}
		return
	case "speed":
		if s.testMode {
			switch s.gameSpeed {
			case 1:
				s.gameSpeed = 2
			case 2:
				s.gameSpeed = 3
			case 3:
				s.gameSpeed = 10
			default:
				s.gameSpeed = 1
			}
		} else {
			if s.gameSpeed == 1 {
				s.gameSpeed = 2
			} else {
				s.gameSpeed = 1
			}
		}
		return
	case "menu":
		s.prePauseMode = s.imode
		s.imode = modePaused
		s.postPipeline.Effects.SetDesaturation(0.7, 3.0, 0.5, 0.5, 0.6)
		return
	case "spawn":
		if s.imode == modeSpawnMenu || s.imode == modeSpawnPlace {
			s.imode = modeIdle
			s.spawnMode = false
			s.spawnType = ""
		} else {
			s.imode = modeSpawnMenu
			s.spawnMode = true
			s.spawnType = ""
			s.selectedTower = nil
		}
		return
	case "debug":
		s.debugPanelOpen = !s.debugPanelOpen
		return
	case "screenshot":
		// 已在 Update() 早期拦截处理，这里只需消费点击防止穿透到交互模式
		return
	}

	// ActionBar 按钮
	actionBtn := hud.ActionBarHitTest(ftx, fty)
	switch actionBtn {
	case "build":
		if s.imode == modeBuildMenu || s.imode == modeBuildPlace {
			s.imode = modeIdle
			s.audioMgr.PlayAt(gameAudio.SFXUIClose, gameAudio.VolUI)
		} else {
			s.selectedTower = nil
			s.wardenPanelOpen = false
			s.itemPanelOpen = false
			s.tutorial.Trigger("build")
			s.enterBuildMode()
			s.audioMgr.PlayAt(gameAudio.SFXUIOpen, gameAudio.VolUI)
		}
		return
	case "items":
		if s.inventory.TotalCount() == 0 {
			s.itemBtnState.TriggerDisabledShake()
			return
		}
		if s.imode == modeItemPanel {
			s.imode = modeIdle
			s.itemPanelOpen = false
			s.audioMgr.PlayAt(gameAudio.SFXUIClose, gameAudio.VolUI)
		} else {
			s.imode = modeItemPanel
			s.itemPanelOpen = true
			s.selectedTower = nil
			s.wardenPanelOpen = false
			s.audioMgr.PlayAt(gameAudio.SFXUIOpen, gameAudio.VolUI)
		}
		return
	}

	// 按交互模式分发 Tap
	switch s.imode {
	case modeIdle:
		clicked := s.towerAtPixel(wtx, wty)
		if clicked != nil {
			s.selectedTower = clicked
			s.imode = modeTowerSel
			s.audioMgr.PlayAt(gameAudio.SFXUIOpen, gameAudio.VolUI)
			s.tutorial.Trigger("tower_select")
		} else if s.wardenPanelOpen {
			s.wardenPanelOpen = false
		}

	case modeBuildMenu:
		idx := hud.BuildMenuHitTest(ftx, fty, s.buildMenuTotalCards(), len(s.towerDefs))
		if idx == -2 || idx == -1 { // close button or outside panel
			s.imode = modeIdle
			s.audioMgr.PlayAt(gameAudio.SFXUIClose, gameAudio.VolUI)
		} else if idx >= 0 { // buildable card
			s.selectedDef = idx
			s.selectedTower = nil
			s.imode = modeBuildPlace
		}

	case modeBuildPlace:
		placed := s.tryPlaceTower(wtx, wty)
		if placed {
			s.imode = modeIdle
		} else if s.towerAtPixel(wtx, wty) != nil {
			hud.ShowToast("此位置已有塔")
		}

	case modeSpawnMenu:
		entries := s.spawnEntries()
		if idx := hud.SpawnMenuHitTest(ftx, fty, len(entries)); idx >= 0 {
			s.spawnType = entries[idx].Name
			s.imode = modeSpawnPlace
			label := entries[idx].Name
			if entries[idx].Label != "" {
				label = entries[idx].Label
			}
			hud.ShowToast("点击地图放置: " + label)
		} else {
			s.imode = modeIdle
			s.spawnMode = false
			s.spawnType = ""
		}

	case modeSpawnPlace:
		if cfg, ok := s.spawner.Archetypes[s.spawnType]; ok {
			label := s.spawnType
			if cfg.Label != "" {
				label = cfg.Label
			}
			if s.spawnMoving {
				// 造动怪：找最近路径点放置，赋予路径让它行走
				path := s.gameMap.PickPath()
				bestIdx, bestDist := 0, math.MaxFloat64
				for i, pt := range path {
					d := math.Hypot(pt.X-wtx, pt.Y-wty)
					if d < bestDist {
						bestDist = d
						bestIdx = i
					}
				}
				// 确保前方至少有一个路径点可走
				if bestIdx >= len(path)-1 {
					bestIdx = len(path) - 2
				}
				if bestIdx < 0 {
					bestIdx = 0
				}
				pt := path[bestIdx]
				// baseHP/baseSpd 传原始基准值，Spawn 内部会乘 cfg 的 Scale
				baseHP := 1000.0
				baseSpd := 50.0
				e := s.enemies.Spawn(pt.X, pt.Y, baseHP, baseSpd, bestIdx+1, s.spawnType, cfg)
				if e != nil {
					e.Path = path
				}
				hud.ShowToast("动怪: " + label + "  (按Esc退出)")
			} else {
				// 造静怪：标记为木桩怪，不移动
				e := s.enemies.Spawn(wtx, wty, 1000, 0, 0, s.spawnType, cfg)
				if e != nil {
					e.IsDummy = true
				}
				hud.ShowToast("静怪: " + label + "  (按Esc退出)")
			}
		}

	case modeTowerSel:
		// "选择能力(N)" 按钮点击 → 打开 ChoicePanel
		if hud.HitTestAbilityBtn(ftx, fty) && s.selectedTower != nil {
			s.openAbilityChoicePanel()
		} else if hud.HitTestUnlockBtn(ftx, fty) && s.selectedTower != nil {
			s.tryUnlockAbilitySlot()
		} else if hud.InfoPanelUpgradeHitTest(ftx, fty, s.selectedTower != nil) {
			s.tryUpgradeTower()
		} else if hud.InfoPanelBulkUpgradeHitTest(ftx, fty, s.selectedTower != nil) {
			s.tryBulkUpgradeTower()
		} else if hud.InfoPanelSellHitTest(ftx, fty, s.selectedTower != nil) {
			s.trySellTower(s.selectedTower.X, s.selectedTower.Y)
			s.imode = modeIdle
			s.audioMgr.PlayAt(gameAudio.SFXUIClose, gameAudio.VolUI)
		} else {
			clicked := s.towerAtPixel(wtx, wty)
			if clicked != nil && clicked != s.selectedTower {
				s.selectedTower = clicked
			} else {
				s.selectedTower = nil
				s.imode = modeIdle
				s.audioMgr.PlayAt(gameAudio.SFXUIClose, gameAudio.VolUI)
			}
		}

	case modeItemPanel:
		// 点击面板外 → 关闭（卡片点击已在上面的 press-down 检测中处理，
		// ActionBar 点击已在上面 ActionBarHitTest 中 return，不会到达此处）
		if !hud.ItemPanelContains(ftx, fty) {
			s.imode = modeIdle
			s.itemPanelOpen = false
			s.audioMgr.PlayAt(gameAudio.SFXUIClose, gameAudio.VolUI)
		}
	}
}

// handlePausedInput 暂停菜单输入。
func (s *StageScene) handlePausedInput() {
	s.gesture.Update()
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyP) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		s.imode = s.prePauseMode
		s.postPipeline.Effects.SetDesaturation(0.0, 4.0, 0.5, 0.5, 0.6)
		return
	}
	if s.gesture.JustTapped() {
		tx, ty := s.gesture.TapPos()
		action := hud.PauseMenuHitTest(float32(tx), float32(ty))
		switch action {
		case hud.PauseResume:
			s.audioMgr.PlaySafe(gameAudio.SFXUIClick)
			s.imode = s.prePauseMode
			s.postPipeline.Effects.SetDesaturation(0.0, 4.0, 0.5, 0.5, 0.6)
		case hud.PauseSettings:
			s.audioMgr.PlaySafe(gameAudio.SFXUIClick)
			s.switcher.SwitchScene(NewSettingsScene(s.switcher, s))
		case hud.PauseRestart:
			s.audioMgr.PlaySafe(gameAudio.SFXUIClick)
			s.switcher.SwitchScene(NewStageSceneWithOpts(s.switcher, s.initOpts))
		case hud.PauseQuit:
			s.audioMgr.PlaySafe(gameAudio.SFXUIClick)
			s.switcher.SwitchScene(NewSelectScene(s.switcher))
		}
	}
}

// handleWardenSelection 战灵选择覆盖层输入。
func (s *StageScene) handleWardenSelection() {
	if s.wardenOverlay == nil || !s.wardenOverlay.Active {
		s.imode = modeIdle
		return
	}
	g := s.gesture
	g.DragEnabled = false
	g.Update()
	// ESC: 跳过战灵选择，直接进入战斗
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.wardenReady = true
		s.imode = modeIdle
		return
	}
	mx, my := g.CursorPos()
	s.wardenOverlay.Update(mx, my, g.JustTapped())
}

// tryUpgradeTower 为选中的塔购买 10 点永久强度。
func (s *StageScene) tryUpgradeTower() {
	t := s.selectedTower
	if t == nil {
		return
	}
	cost := tower.StrengthBuyCost()
	if s.gold < cost {
		return
	}
	spent := t.BuyStrength() // 内部已调用 AddPermanent + RecalcStats
	s.gold -= spent
	s.gameStats.GoldSpent += spent
	s.bus.Emit(event.EvtTowerUpgraded, event.TowerUpgradedPayload{TowerKey: t.Key, Spent: spent})
	s.showNotify(fmt.Sprintf("强度+10 (-$%d)", spent))
}

// tryBulkUpgradeTower 为选中的塔一次性购买 50 点永久强度（花费 5 倍单次费用）。
func (s *StageScene) tryBulkUpgradeTower() {
	t := s.selectedTower
	if t == nil {
		return
	}
	cost := tower.StrengthBuyCost() * 5
	if s.gold < cost {
		return
	}
	// 5 次购买合并
	for i := 0; i < 5; i++ {
		t.BuyStrength()
	}
	s.gold -= cost
	s.gameStats.GoldSpent += cost
	s.bus.Emit(event.EvtTowerUpgraded, event.TowerUpgradedPayload{TowerKey: t.Key, Spent: cost})
	s.showNotify(fmt.Sprintf("强度+50 (-$%d)", cost))
}

// tryUnlockAbilitySlot 花钱解锁选中塔的下一个能力槽位并 roll 候选选项。
func (s *StageScene) tryUnlockAbilitySlot() {
	t := s.selectedTower
	if t == nil {
		return
	}
	if !tower.CanUnlockMore(t) {
		return
	}
	def := s.findTowerDef(t)
	cost := tower.NextUpgradeCost(t, def)
	if s.gold < cost {
		return
	}
	cat := tower.UnlockNextSlot(t)
	if cat < 0 {
		return
	}
	s.gold -= cost
	s.gameStats.GoldSpent += cost
	t.Cost += cost  // 累计到塔总投资（影响卖出退款）
	t.PaidUnlocks++ // 记录付费解锁次数（影响下次费用）
	catName := tower.CategoryName(cat)
	hud.ShowToast(fmt.Sprintf("解锁: %s (-$%d)", catName, cost))
	s.audioMgr.PlayAt(gameAudio.SFXUIOpen, gameAudio.VolUI)
}

// tryPlaceTower 和 trySellTower 保留在 stage.go 中（涉及经济/粒子等更多依赖）。

// ─── 相机和空间查询 ───

// newStageGesture 创建配置好的手势识别器。
func newStageGesture() *input.Gesture {
	g := input.NewGesture()
	g.ToLogical = func(x, y float64) (float64, float64) {
		return x / draw.Scale, y / draw.Scale
	}
	g.IsOnUI = func(x, y float64) bool {
		fx, fy := float32(x), float32(y)
		if hud.TopBarHitTest(fx, fy) != "" {
			return true
		}
		if hud.ToggleButtonHitTest(fx, fy, false) {
			return true
		}
		if y > float64(game.ScreenHeight)-120 {
			return true
		}
		// ActionBar 区域
		if abx, aby, abw, abh := hud.ActionBarRect(); abw > 0 {
			if fx >= abx && fx <= abx+abw && fy >= aby && fy <= aby+abh {
				return true
			}
		}
		return false
	}
	return g
}

// screenToWorld 将屏幕坐标转换为世界坐标。
func (s *StageScene) screenToWorld(sx, sy float64) (float64, float64) {
	return sx + s.camX, sy + s.camY
}

// clampCamera 将相机偏移夹紧到地图范围内。
func (s *StageScene) clampCamera() {
	mapW := s.gameMap.PixelWidth()
	mapH := s.gameMap.PixelHeight()
	screenW := float64(game.ScreenWidth)
	screenH := float64(game.ScreenHeight)

	maxX := mapW + s.gameMap.OffsetX*2 - screenW
	maxY := mapH + s.gameMap.OffsetY*2 - screenH
	if maxX < 0 {
		maxX = 0
	}
	if maxY < 0 {
		maxY = 0
	}

	if s.camX < 0 {
		s.camX = 0
	}
	if s.camX > maxX {
		s.camX = maxX
	}
	if s.camY < 0 {
		s.camY = 0
	}
	if s.camY > maxY {
		s.camY = maxY
	}
}

// enterBuildMode enters build mode. If only one tower type, skip menu and go directly to place mode.
func (s *StageScene) enterBuildMode() {
	if len(s.towerDefs) == 1 {
		s.selectedDef = 0
		s.imode = modeBuildPlace
	} else {
		s.imode = modeBuildMenu
	}
}

// needsCamera 返回地图是否需要相机。
func (s *StageScene) needsCamera() bool {
	mapW := s.gameMap.PixelWidth() + s.gameMap.OffsetX*2
	mapH := s.gameMap.PixelHeight() + s.gameMap.OffsetY*2
	return mapW > float64(game.ScreenWidth) || mapH > float64(game.ScreenHeight)
}

// openAbilityChoicePanel 打开能力选择覆盖层。
func (s *StageScene) openAbilityChoicePanel() {
	t := s.selectedTower
	if t == nil {
		return
	}

	if s.testMode {
		s.openTestAbilityChoicePanel(t)
		return
	}

	nextCat := tower.NextPendingCategory(t)
	if nextCat < 0 {
		return
	}
	choices, ok := t.PendingChoices[nextCat]
	if !ok || len(choices) == 0 {
		return
	}

	// 构建 ChoiceOption（使用与炮塔 HUD 一致的分段描述）
	var effStr float64
	if t.Strength != nil {
		effStr = t.Strength.Effective()
	}
	opts := make([]hud.ChoiceOption, len(choices))
	for i, c := range choices {
		segs := buildAbilitySegments(&c, effStr)
		opts[i] = hud.ChoiceOption{
			Label:    c.Label,
			Segments: segs,
			Tier:     "normal",
			Icon:     c.Icon,
			Data:     c.Type,
		}
	}

	catName := tower.CategoryName(nextCat)
	s.choicePanel.Show("选择"+catName, opts, func(idx int, opt hud.ChoiceOption) {
		abilType, _ := opt.Data.(string)
		if abilType != "" && t.AddAbility(abilType) {
			tower.ClearPendingChoice(t, nextCat)
			hud.ShowToast("获得能力: " + opt.Label)
			s.bus.Emit(event.EvtTowerUpgraded, event.TowerUpgradedPayload{TowerKey: t.Key})
		}
	})
	s.imode = modeUpgrade
	s.audioMgr.PlayAt(gameAudio.SFXUIOpen, gameAudio.VolUI)
}

// openTestAbilityChoicePanel 测试模式：先选类别，再展示该类别全部能力。
func (s *StageScene) openTestAbilityChoicePanel(t *tower.Tower) {
	// 收集所有空槽类别（按类别索引固定顺序：攻击/CC/命中/光环/DoT/范围）
	var emptyCats []int
	for cat := 0; cat < len(t.AbilitySlots); cat++ {
		if t.AbilitySlots[cat] == "" {
			emptyCats = append(emptyCats, cat)
		}
	}
	if len(emptyCats) == 0 {
		hud.ShowToast("所有能力位已满")
		return
	}

	// 只剩一个空槽：直接展示该类别全部能力
	if len(emptyCats) == 1 {
		s.openTestCategoryAbilities(t, emptyCats[0])
		return
	}

	// 多个空槽：先让玩家选类别
	opts := make([]hud.ChoiceOption, len(emptyCats))
	for i, cat := range emptyCats {
		count := len(tower.AbilitiesForCategory(cat))
		opts[i] = hud.ChoiceOption{
			Label:       tower.CategoryName(cat),
			Description: fmt.Sprintf("共%d个能力可选", count),
			Tier:        "normal",
			Data:        cat,
		}
	}
	s.choicePanel.Show("选择能力类别", opts, func(idx int, opt hud.ChoiceOption) {
		cat, _ := opt.Data.(int)
		s.openTestCategoryAbilities(t, cat)
	})
	s.imode = modeUpgrade
}

// openTestCategoryAbilities 测试模式：展示指定类别的全部能力供选择。
func (s *StageScene) openTestCategoryAbilities(t *tower.Tower, cat int) {
	choices := tower.AllChoicesForCategory(cat)
	if len(choices) == 0 {
		hud.ShowToast("该类别无可用能力")
		return
	}

	var effStr float64
	if t.Strength != nil {
		effStr = t.Strength.Effective()
	}
	opts := make([]hud.ChoiceOption, len(choices))
	for i, c := range choices {
		segs := buildAbilitySegments(&c, effStr)
		opts[i] = hud.ChoiceOption{
			Label:    c.Label,
			Segments: segs,
			Tier:     "normal",
			Icon:     c.Icon,
			Data:     c.Type,
		}
	}

	catName := tower.CategoryName(cat)
	s.choicePanel.Show("[测试] "+catName, opts, func(idx int, opt hud.ChoiceOption) {
		abilType, _ := opt.Data.(string)
		if abilType != "" && t.AddAbility(abilType) {
			tower.ClearPendingChoice(t, cat)
			hud.ShowToast("获得能力: " + opt.Label)
			s.bus.Emit(event.EvtTowerUpgraded, event.TowerUpgradedPayload{TowerKey: t.Key})
		}
	})
	s.imode = modeUpgrade
}

// enemyAtPixel 返回像素位置上最近的敌人（碰撞半径内），无则返回 nil。
// nearestEnemy 返回距离像素位置最近的活跃敌人（无距离限制）。
func (s *StageScene) nearestEnemy(px, py float64) *enemy.Enemy {
	var best *enemy.Enemy
	bestDist := math.MaxFloat64
	s.enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() {
			return
		}
		d := math.Hypot(e.X-px, e.Y-py)
		if d < bestDist {
			bestDist = d
			best = e
		}
	})
	return best
}

func (s *StageScene) enemyAtPixel(px, py float64) *enemy.Enemy {
	var best *enemy.Enemy
	bestDist := 20.0 // 最大拾取半径
	s.enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() {
			return
		}
		dx := e.X - px
		dy := e.Y - py
		d := math.Sqrt(dx*dx + dy*dy)
		r := e.Radius
		if r < 12 {
			r = 12 // 最小拾取半径
		}
		if d <= r && d < bestDist {
			bestDist = d
			best = e
		}
	})
	return best
}

// towerAtPixel 返回像素位置上的塔，无塔返回 nil。
func (s *StageScene) towerAtPixel(px, py float64) *tower.Tower {
	gm := s.gameMap
	cs := float64(gm.CellSize)
	fx := (px - gm.OffsetX) / cs
	fy := (py - gm.OffsetY) / cs
	if fx < 0 || fy < 0 {
		return nil
	}
	t := s.towers.At(int(fy), int(fx))
	if t != nil && t.Selling {
		return nil
	}
	return t
}
