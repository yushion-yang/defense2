// stage_input.go — StageScene 的输入处理和交互状态机。
// 包含 handleInput、handlePausedInput、handleWardenSelection
// 以及相机控制、手势初始化等辅助方法。
package scene

import (
	"fmt"

	gameAudio "defense2/internal/audio"
	"defense2/internal/core/enemy"
	"defense2/internal/core/event"
	"defense2/internal/core/game"
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
		case modeTowerSel:
			s.selectedTower = nil
			s.imode = modeIdle
		case modeUpgrade:
			s.choicePanel.Close()
			s.imode = modeTowerSel
		case modeSpawnMenu, modeSpawnPlace:
			s.spawnMode = false
			s.spawnType = ""
			s.imode = modeIdle
		default:
			s.prePauseMode = s.imode
			s.imode = modePaused
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		if s.imode == modeBuildMenu {
			s.imode = modeIdle
		} else {
			s.imode = modeBuildMenu
			s.selectedTower = nil
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
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if s.imode == modeTowerSel && s.selectedTower != nil {
			s.trySellTower(s.selectedTower.X, s.selectedTower.Y)
			s.imode = modeIdle
		}
	}

	// F2: 调试覆盖层
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		s.debugOverlay.Toggle()
	}

	// 测试模式专用快捷键
	if s.testMode {
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

	// ── Tap → 游戏操作 ──
	if !g.JustTapped() {
		return
	}
	tapX, tapY := g.TapPos()
	ftx, fty := float32(tapX), float32(tapY)
	wtx, wty := s.screenToWorld(tapX, tapY)

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
	if hud.ToggleButtonHitTest(ftx, fty, true) {
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
		return
	case "build":
		if s.imode == modeBuildMenu {
			s.imode = modeIdle
		} else {
			s.imode = modeBuildMenu
			s.selectedTower = nil
			s.wardenPanelOpen = false
		}
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

	// 按交互模式分发 Tap
	switch s.imode {
	case modeIdle:
		clicked := s.towerAtPixel(wtx, wty)
		if clicked != nil {
			s.selectedTower = clicked
			s.imode = modeTowerSel
		} else if s.wardenPanelOpen {
			s.wardenPanelOpen = false
		}

	case modeBuildMenu:
		idx := hud.BuildMenuHitTest(ftx, fty, s.buildMenuTotalCards(), len(s.towerDefs))
		if idx == -2 || idx == -1 {
			s.imode = modeIdle
		} else if idx >= 0 {
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
			s.enemies.Spawn(wtx, wty, 100, 0, 0, s.spawnType, cfg)
			label := s.spawnType
			if cfg.Label != "" {
				label = cfg.Label
			}
			hud.ShowToast("已放置: " + label)
		}

	case modeTowerSel:
		// "选择能力(N)" 按钮点击 → 打开 ChoicePanel
		if hud.HitTestAbilityBtn(ftx, fty) && s.selectedTower != nil {
			s.openAbilityChoicePanel()
		} else if hud.InfoPanelUpgradeHitTest(ftx, fty, s.selectedTower != nil) {
			s.tryUpgradeTower()
		} else if hud.InfoPanelSellHitTest(ftx, fty, s.selectedTower != nil) {
			s.trySellTower(s.selectedTower.X, s.selectedTower.Y)
			s.imode = modeIdle
		} else {
			clicked := s.towerAtPixel(wtx, wty)
			if clicked != nil && clicked != s.selectedTower {
				s.selectedTower = clicked
			} else {
				s.selectedTower = nil
				s.imode = modeIdle
			}
		}
	}
}

// handlePausedInput 暂停菜单输入。
func (s *StageScene) handlePausedInput() {
	s.gesture.Update()
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyP) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		s.imode = s.prePauseMode
		return
	}
	if s.gesture.JustTapped() {
		tx, ty := s.gesture.TapPos()
		action := hud.PauseMenuHitTest(float32(tx), float32(ty))
		switch action {
		case hud.PauseResume:
			s.audioMgr.PlaySafe(gameAudio.SFXUIClick)
			s.imode = s.prePauseMode
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
	mx, my := g.CursorPos()
	s.wardenOverlay.Update(mx, my, g.JustTapped())
}

// tryUpgradeTower 为选中的塔购买 10 点永久强度。
func (s *StageScene) tryUpgradeTower() {
	t := s.selectedTower
	if t == nil {
		return
	}
	cost := tower.StrengthBuyCost
	if s.gold < cost {
		return
	}
	spent := t.BuyStrength()
	s.gold -= spent
	if t.Strength != nil {
		t.Strength.AddPermanent(10)
	}
	s.bus.Emit(event.EvtTowerUpgraded, event.TowerUpgradedPayload{TowerKey: t.Key, Spent: spent})
	s.showNotify(fmt.Sprintf("强度+10 (-$%d)", spent))
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
		if hud.ToggleButtonHitTest(fx, fy, true) || hud.ToggleButtonHitTest(fx, fy, false) {
			return true
		}
		if y > float64(game.ScreenHeight)-120 {
			return true
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
	mapW := s.gameMap.Width()
	mapH := s.gameMap.Height()
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

// needsCamera 返回地图是否需要相机。
func (s *StageScene) needsCamera() bool {
	mapW := s.gameMap.Width() + s.gameMap.OffsetX*2
	mapH := s.gameMap.Height() + s.gameMap.OffsetY*2
	return mapW > float64(game.ScreenWidth) || mapH > float64(game.ScreenHeight)
}

// openAbilityChoicePanel 打开能力选择覆盖层。
func (s *StageScene) openAbilityChoicePanel() {
	t := s.selectedTower
	if t == nil {
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

	// 构建 ChoiceOption（描述替换模板占位符为实际值）
	var effStr float64
	if t.Strength != nil {
		effStr = t.Strength.Effective()
	}
	opts := make([]hud.ChoiceOption, len(choices))
	for i, c := range choices {
		desc := FormatAbilityDisplay(&c, effStr)
		opts[i] = hud.ChoiceOption{
			Label:       c.Label,
			Description: desc,
			Tier:        "normal",
			Data:        c.Type,
		}
	}

	catName := tower.CategoryName(nextCat)
	s.choicePanel.Show("选择"+catName, opts, func(idx int, opt hud.ChoiceOption) {
		abilType, _ := opt.Data.(string)
		if abilType != "" && t.AddAbility(abilType) {
			tower.ClearPendingChoice(t, nextCat)
			hud.ShowToast("获得能力: " + opt.Label)
		}
	})
	s.imode = modeUpgrade
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
