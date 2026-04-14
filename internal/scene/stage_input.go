// stage_input.go — StageScene 的输入处理和交互状态机（11 种模式）。
//
// 本文件是 StageScene 中最复杂的输入处理模块，核心是一个 11 状态的交互状态机。
// 每帧由 StageScene.Update() 调用 handleInput()，根据当前 interactMode 分发输入事件。
//
// 输入处理优先级（从高到低）：
//  1. 道具拖拽释放检测（modeItemDrag 时独占输入）
//  2. 手势识别器更新（拖拽、滚轮、Hover）
//  3. 键盘快捷键（ESC/B/I/U/S/1/2/3/P/Del/F2 等）
//  4. UI 组件点击（TopBar/ActionBar/WavePanel/WardenPanel/DebugPanel）
//  5. 按 interactMode 分发的 Tap 处理（switch s.imode）
//
// 关键设计决策：
//   - 手势识别器（input.Gesture）统一处理鼠标+触摸输入，提供 JustTapped/IsDragging 等抽象
//   - 拖拽权限由状态机控制：modeTowerSel 下禁用拖拽，防止"点击取消选中"被误判为拖拽
//   - 道具拖拽用 IsMouseButtonJustPressed 启动（按下即触发），用 JustReleased 释放
//   - UI 层 hitTest 返回 string 标识符（"build"/"items"/"start" 等），避免坐标硬编码
//
// 其他入口函数：
//   - handlePausedInput:      暂停菜单输入（ESC/P/Space 恢复 + 4 个菜单按钮）
//   - handleWardenSelection:  战灵选择覆盖层输入（ESC 跳过 + 卡片点击选择）
package scene

import (
	"math"

	gameAudio "defense2/internal/audio"
	"defense2/internal/core/achievement"
	"defense2/internal/core/enemy"
	"defense2/internal/core/event"
	"defense2/internal/core/game"
	"defense2/internal/core/item"
	"defense2/internal/core/tower"
	"defense2/internal/i18n"
	"defense2/internal/input"
	"defense2/internal/render/draw"
	"defense2/internal/render/hud"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// handleInput 主输入处理函数——每帧由 StageScene.Update() 在 statePlaying 时调用。
//
// 执行流程概览（约 550 行）：
//  1. 配置手势识别器的拖拽权限（仅在 idle/buildPlace/spawnPlace 且地图超出屏幕时允许拖拽）
//  2. [modeItemDrag] 道具拖拽独占路径：跟踪悬停目标 → 检测释放 → 应用道具
//  3. 相机拖拽和滚轮处理
//  4. 键盘快捷键处理（ESC 根据模式有不同行为，B=建塔，I=道具 等）
//  5. Hover 状态更新（建塔菜单卡片高亮、造怪菜单高亮）
//  6. [modeItemPanel] 道具面板按下检测（JustPressed 启动拖拽，不等松手）
//  7. Tap 分发：按 interactMode 走不同的 case 分支处理点击
func (s *StageScene) handleInput() {
	g := s.gesture
	// 状态机控制拖拽权限：
	// - idle/buildPlace/spawnPlace 允许拖拽（用于大地图平移）
	// - 其他模式（特别是 modeTowerSel）禁用拖拽，防止"点击空地取消选中"被误判为拖拽起始
	switch s.imode {
	case modeIdle, modeBuildPlace, modeSpawnPlace:
		g.DragEnabled = s.needsCamera()
	default:
		g.DragEnabled = false
	}
	g.Update()

	// ── [modeItemDrag] 道具拖拽独占路径 ──
	// 道具拖拽是特殊的输入模式：从 modeItemPanel 的 JustPressed 进入，
	// 持续跟踪指针位置高亮目标塔，松手时应用道具效果。
	// 此路径 return 退出，不会 fallthrough 到后续的 Tap 分发逻辑。
	if s.imode == modeItemDrag && s.dragItemActive {
		// 更新拖拽悬停目标（高亮可应用道具的塔）
		mx, my := g.CursorPos()
		wtx, wty := s.screenToWorld(mx, my)
		s.dragHoverTower = s.towerAtPixel(wtx, wty)

		// 检测鼠标/触摸释放（需同时检查两种输入源）
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
						hud.ShowToast(i18n.TF("game.achieve.unlocked", i18n.T("game.achieve.item_master")))
					}
				}
			} else {
				hud.ShowToast(i18n.T("game.item.drag_to_tower"))
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

	// 以下是正常（非道具拖拽）的输入处理路径
	mx, my := g.CursorPos()              // 逻辑坐标（已经过 ToLogical 转换）
	fmx, fmy := float32(mx), float32(my) // float32 版本，用于 hud hitTest 函数

	// ── 相机拖拽 → 平移（大地图场景需要，小地图跳过） ──
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
	} else if sy != 0 && s.imode == modeTowerSel {
		hud.InfoPanelScroll(sy)
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
	// modeUpgrade 特殊处理: ChoicePanel（3选1覆盖层）吃掉所有输入，
	// 只响应 ESC（关闭）和鼠标点击（选择选项），其他键盘输入被屏蔽。
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

	// ESC 键行为随交互模式变化（从"关闭面板"到"进入暂停"）：
	// - 有面板打开 → 关闭面板，回到上一级模式
	// - 空闲状态 → 记住当前模式，进入暂停菜单
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
			// 空闲或其他模式：进入暂停
			// prePauseMode 记录暂停前的模式，恢复时还原（如暂停前在 modeTowerSel）
			s.prePauseMode = s.imode
			s.imode = modePaused
			// 后处理去饱和效果：0.7 强度 + 3s 渐入，营造暂停的视觉反馈
			s.postPipeline.Effects.SetDesaturation(0.7, 3.0, 0.5, 0.5, 0.6)
		}
		return
	}
	// B 键: 建塔模式切换（toggle）
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
	// I 键: 道具面板切换（toggle），无道具时不打开
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
	// U 键: 快捷升级（+10 强度），仅在 modeTowerSel 时可用
	if inpututil.IsKeyJustPressed(ebiten.KeyU) && s.imode == modeTowerSel {
		s.tryUpgradeTower()
		return
	}
	// S 键: 开始下一波（仅在空闲/选塔模式且当前没有活跃波次时）
	if inpututil.IsKeyJustPressed(ebiten.KeyS) && (s.imode == modeIdle || s.imode == modeTowerSel) {
		if !s.spawner.WaveActive && !s.spawner.AllDone {
			s.tryStartWave()
		}
		return
	}
	// 1/2/3 键: 游戏速度切换
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		s.gameSpeed = 1
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		s.gameSpeed = 2
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		s.gameSpeed = 3
	}
	// P 键: 暂停/恢复（恢复由 handlePausedInput 处理，此处只处理进入暂停）
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		s.prePauseMode = s.imode
		s.imode = modePaused
		s.postPipeline.Effects.SetDesaturation(0.7, 3.0, 0.5, 0.5, 0.6)
		return
	}
	// Delete/Backspace: 快捷卖塔（仅在选中塔时）
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

	// ── 测试模式专用快捷键 ──
	// 这些快捷键仅在 TestMode 下可用，用于快速调试：
	// Delete=消灭悬停敌人, D=调试面板, G=+500金, K=清屏, N=跳波
	if s.testMode {
		if inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
			if s.hoveredEnemy != nil && s.hoveredEnemy.Active && !s.hoveredEnemy.IsDying() {
				s.enemies.Kill(s.hoveredEnemy)
				hud.ShowToast(i18n.TF("game.debug.killed", s.hoveredEnemy.Archetype))
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyD) {
			s.debugPanelOpen = !s.debugPanelOpen
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyG) {
			s.gold += 500
			hud.ShowToast(i18n.T("game.debug.gold_500"))
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyK) {
			s.enemies.Each(func(e *enemy.Enemy) {
				e.HP = 0
			})
			hud.ShowToast(i18n.T("game.debug.clear_enemies"))
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
			hud.ShowToast(i18n.T("game.debug.next_wave"))
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

	// ── [modeItemPanel] 道具卡片按下检测 ──
	// 关键设计: 用 JustPressed（按下即触发）而非 JustTapped（松开触发），
	// 因为拖拽操作需要在按下瞬间就进入 modeItemDrag，后续跟踪指针移动。
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

	// ── Tap 分发 ──
	// 以下所有逻辑只在"刚点击"（JustTapped）时执行。
	// JustTapped 由手势识别器在松手时触发，自动排除了拖拽操作（拖拽不产生 Tap）。
	if !g.JustTapped() {
		return
	}
	tapX, tapY := g.TapPos()                 // Tap 的逻辑坐标（屏幕空间）
	ftx, fty := float32(tapX), float32(tapY) // float32 版本（hud hitTest 用）
	wtx, wty := s.screenToWorld(tapX, tapY)  // 世界坐标（加上相机偏移）

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

	// ── UI 层 hitTest（按优先级从高到低） ──
	// UI 层点击必须在模式分发之前处理，确保 UI 按钮不被场景逻辑"穿透"。

	// 左下角波次面板切换按钮
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

	// TopBar 按钮（开波/变速/暂停/造怪/调试/截图）
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

	// ActionBar 按钮（底部工具栏：建塔/道具两个按钮）
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

	// ── 按交互模式分发 Tap（核心状态机分支） ──
	// 到达此处的 Tap 没有被任何 UI 组件消费，交由状态机处理。
	switch s.imode {
	case modeIdle:
		// 空闲状态：点击塔→选中，点击空地→关闭战灵面板
		clicked := s.towerAtPixel(wtx, wty)
		if clicked != nil {
			s.selectedTower = clicked
			s.imode = modeTowerSel
			hud.ResetInfoPanelScroll()
			s.audioMgr.PlayAt(gameAudio.SFXUIOpen, gameAudio.VolUI)
			s.tutorial.Trigger("tower_select")
		} else if s.wardenPanelOpen {
			s.wardenPanelOpen = false
		}

	case modeBuildMenu:
		// 建塔面板：点击卡片→进入放塔模式，点击关闭/外部→回 idle
		idx := hud.BuildMenuHitTest(ftx, fty, s.buildMenuTotalCards(), len(s.towerDefs))
		if idx == -2 || idx == -1 { // -2=关闭按钮, -1=面板外部
			s.imode = modeIdle
			s.audioMgr.PlayAt(gameAudio.SFXUIClose, gameAudio.VolUI)
		} else if idx >= 0 { // buildable card
			s.selectedDef = idx
			s.selectedTower = nil
			s.imode = modeBuildPlace
		}

	case modeBuildPlace:
		// 放塔模式：点击可建位→建塔并回 idle，点击已有塔位→提示已占用
		placed := s.tryPlaceTower(wtx, wty)
		if placed {
			s.imode = modeIdle
		} else if s.towerAtPixel(wtx, wty) != nil {
			hud.ShowToast(i18n.T("game.build.occupied"))
		}

	case modeSpawnMenu:
		// 造怪菜单（测试模式）：点击敌人类型→进入放置模式
		entries := s.spawnEntries()
		if idx := hud.SpawnMenuHitTest(ftx, fty, len(entries)); idx >= 0 {
			s.spawnType = entries[idx].Name
			s.imode = modeSpawnPlace
			label := entries[idx].Name
			if entries[idx].Label != "" {
				label = entries[idx].Label
			}
			hud.ShowToast(i18n.TF("game.spawn.place_hint", label))
		} else {
			s.imode = modeIdle
			s.spawnMode = false
			s.spawnType = ""
		}

	case modeSpawnPlace:
		// 造怪放置（测试模式）：点击地图位置生成敌人
		// 两种模式：s.spawnMoving=true → 动怪（找最近路径点放置），false → 静怪（IsDummy）
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
				hud.ShowToast(i18n.TF("game.spawn.moving", label))
			} else {
				// 造静怪：标记为木桩怪，不移动
				e := s.enemies.Spawn(wtx, wty, 1000, 0, 0, s.spawnType, cfg)
				if e != nil {
					e.IsDummy = true
				}
				hud.ShowToast(i18n.TF("game.spawn.static", label))
			}
		}

	case modeTowerSel:
		// 塔选中状态：信息面板上的多个按钮 hitTest（按优先级）：
		// 选择能力 → 解锁槽位 → 升级强度 → 批量升级 → 卖出 → 点击其他塔/空地
		// "选择能力(N)" 按钮点击 → 打开 ChoicePanel
		if hud.HitTestAbilityBtn(ftx, fty) && s.selectedTower != nil {
			s.openAbilityChoicePanel()
		} else if hud.HitTestUnlockBtn(ftx, fty) && s.selectedTower != nil {
			s.tryUnlockAbilitySlot()
		} else if hud.InfoPanelUpgradeHitTest(ftx, fty, s.selectedTower != nil) {
			s.tryUpgradeTower()
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

// handlePausedInput 暂停菜单专用输入处理。
// 暂停状态下只响应：ESC/P/Space=恢复游戏，以及 4 个菜单按钮（继续/设置/重开/退出）。
// 暂停期间 gameSpeed 自动为 0（tick pipeline 不推进），但输入仍需处理。
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

// handleWardenSelection 战灵选择覆盖层输入处理。
// 战灵选择发生在第一波开始前，全屏显示可选战灵卡片。
// 选择完成后 wardenReady=true，进入正常游戏流程。ESC 可跳过选择直接开始。
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

// ─── 塔操作函数 ─────────────────────────────────────────────────────

// tryUpgradeTower 为选中的塔购买 10 点永久强度。
// 强度购买是线性的：每次固定花费 StrengthBuyCost()，获得 +10 永久强度。
// 购买后内部自动 RecalcStats（重算属性），触发 EvtTowerUpgraded 事件（供成就/统计监听）。
func (s *StageScene) tryUpgradeTower() {
	t := s.selectedTower
	if t == nil {
		return
	}
	// 强度购买上限检查（从塔配置读取）
	def := s.findTowerDef(t)
	if def.MaxStrengthBuys >= 0 && t.StrengthPurchases >= def.MaxStrengthBuys {
		hud.ShowToast(i18n.T("game.tower.max_upgrade"))
		return
	}
	cost := tower.StrengthBuyCost()
	if s.gold < cost {
		hud.ShowToast(i18n.T("game.tower.gold_short"))
		return
	}
	spent := t.BuyStrength() // 内部已调用 AddPermanent + RecalcStats
	s.gold -= spent
	s.gameStats.GoldSpent += spent
	s.bus.Emit(event.EvtTowerUpgraded, event.TowerUpgradedPayload{TowerKey: t.Key, Spent: spent})
	s.showNotify(i18n.TF("game.tower.str_up_10", spent))
}

// tryUnlockAbilitySlot 花钱解锁选中塔的下一个能力槽位。
// 战役模式下能力槽位需要付费逐个解锁（测试模式则免费全开）。
// 解锁后自动 roll 3 个候选能力供玩家选择（触发 ChoicePanel）。
// 费用递增：通过 tower.NextUpgradeCost() 根据已付费解锁次数计算。
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
		hud.ShowToast(i18n.T("game.tower.gold_short"))
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
	hud.ShowToast(i18n.TF("game.tower.unlock_slot", catName, cost))
	s.audioMgr.PlayAt(gameAudio.SFXUIOpen, gameAudio.VolUI)
}

// tryPlaceTower 和 trySellTower 保留在 stage.go 中（涉及经济/粒子/成就等更多依赖）。

// ─── 相机和空间查询 ─────────────────────────────────────────────────

// newStageGesture 创建配置好的手势识别器。
//
// 手势识别器统一处理鼠标和触摸输入，提供以下抽象：
//   - JustTapped(): 短点击（松手触发，自动排除拖拽）
//   - IsDragging(): 拖拽中（移动距离超过阈值）
//   - DragDelta():  帧间拖拽偏移量（用于平移相机）
//
// 两个回调用于坐标转换和 UI 穿透防护：
//   - ToLogical:  将物理像素坐标转为逻辑坐标（除以 draw.Scale）
//   - IsOnUI:     判断坐标是否在 UI 区域上，是则禁止拖拽（防止拖拽按钮时移动相机）
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
		// ActionBar 区域（精确矩形检测）
		if abx, aby, abw, abh := hud.ActionBarRect(); abw > 0 {
			if fx >= abx && fx <= abx+abw && fy >= aby && fy <= aby+abh {
				return true
			}
		}
		// ActionBar 上方留 6px 余量，避免紧贴边缘误触发拖拽
		if y > float64(game.ScreenHeight)-60 {
			return true
		}
		return false
	}
	return g
}

// screenToWorld 将屏幕坐标（逻辑像素）转换为世界坐标。
// 世界坐标 = 屏幕坐标 + 相机偏移。用于点击选塔、放塔等需要地图空间定位的操作。
func (s *StageScene) screenToWorld(sx, sy float64) (float64, float64) {
	return sx + s.camX, sy + s.camY
}

// clampCamera 将相机偏移夹紧到地图边界内，防止拖拽超出地图范围。
// 对于小于屏幕的地图（maxX/maxY < 0），偏移固定为 0（不可平移）。
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

// enterBuildMode 进入建塔模式。
// 优化: 只有一种塔型时跳过菜单直接进入放塔模式（减少一次点击）。
func (s *StageScene) enterBuildMode() {
	if len(s.towerDefs) == 1 {
		s.selectedDef = 0
		s.imode = modeBuildPlace
	} else {
		s.imode = modeBuildMenu
	}
}

// needsCamera 返回地图是否需要相机平移（地图尺寸超出屏幕时返回 true）。
func (s *StageScene) needsCamera() bool {
	mapW := s.gameMap.PixelWidth() + s.gameMap.OffsetX*2
	mapH := s.gameMap.PixelHeight() + s.gameMap.OffsetY*2
	return mapW > float64(game.ScreenWidth) || mapH > float64(game.ScreenHeight)
}

// ─── 能力选择系统 ─────────────────────────────────────────────────────

// openAbilityChoicePanel 打开能力选择覆盖层（3 选 1）。
//
// 战役模式流程：
//  1. 找到下一个待选类别（NextPendingCategory）
//  2. 从 PendingChoices 取出该类别的 3 个候选能力
//  3. 用 buildAbilitySegments 生成分段着色描述
//  4. 创建 ChoicePanel 并注册回调（选择后 AddAbility + 清除 Pending）
//  5. 切换到 modeUpgrade（ChoicePanel 吃掉所有输入直到关闭）
func (s *StageScene) openAbilityChoicePanel() {
	t := s.selectedTower
	if t == nil {
		return
	}

	def := s.findTowerDef(t)
	if def.AbilityAcquireMode == "allUnlocked" || def.AbilityAcquireMode == "byWave" {
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
	s.choicePanel.Show(i18n.TF("game.ability.choose_cat", catName), opts, func(idx int, opt hud.ChoiceOption) {
		abilType, _ := opt.Data.(string)
		if abilType != "" && t.AddAbility(abilType) {
			tower.ApplyEnhanceIfPresent(t) // enhance 属性加成统一入口
			tower.ClearPendingChoice(t, nextCat)
			hud.ShowToast(i18n.TF("game.ability.gained", opt.Label))
			s.bus.Emit(event.EvtTowerUpgraded, event.TowerUpgradedPayload{TowerKey: t.Key})
			s.abilitiesPicked = append(s.abilitiesPicked, abilType)
		}
	})
	s.imode = modeUpgrade
	s.audioMgr.PlayAt(gameAudio.SFXUIOpen, gameAudio.VolUI)
}

// openTestAbilityChoicePanel 测试模式的能力选择：先选类别，再展示该类别全部能力。
// 与战役模式不同，测试模式不限制候选数量——展示类别下的所有能力供自由选择。
// 多个空槽时分两步：第一步选类别 → 第二步选该类别下的具体能力。
func (s *StageScene) openTestAbilityChoicePanel(t *tower.Tower) {
	// 收集所有空槽类别（按类别索引固定顺序：攻击/CC/命中/光环/DoT/范围）
	var emptyCats []int
	for cat := 0; cat < len(t.AbilitySlots); cat++ {
		if t.AbilitySlots[cat] == "" {
			emptyCats = append(emptyCats, cat)
		}
	}
	if len(emptyCats) == 0 {
		hud.ShowToast(i18n.T("game.ability.all_full"))
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
			Description: i18n.TF("game.ability.cat_count", count),
			Tier:        "normal",
			Data:        cat,
		}
	}
	s.choicePanel.Show(i18n.T("game.ability.choose_category"), opts, func(idx int, opt hud.ChoiceOption) {
		cat, _ := opt.Data.(int)
		s.openTestCategoryAbilities(t, cat)
	})
	s.imode = modeUpgrade
}

// openTestCategoryAbilities 测试模式：展示指定类别的全部能力供选择。
func (s *StageScene) openTestCategoryAbilities(t *tower.Tower, cat int) {
	choices := tower.AllChoicesForCategory(cat)
	if len(choices) == 0 {
		hud.ShowToast(i18n.T("game.ability.none_available"))
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
	s.choicePanel.Show(i18n.TF("game.ability.test_cat", catName), opts, func(idx int, opt hud.ChoiceOption) {
		abilType, _ := opt.Data.(string)
		if abilType != "" && t.AddAbility(abilType) {
			tower.ApplyEnhanceIfPresent(t) // enhance 属性加成统一入口
			tower.ClearPendingChoice(t, cat)
			hud.ShowToast(i18n.TF("game.ability.gained", opt.Label))
			s.bus.Emit(event.EvtTowerUpgraded, event.TowerUpgradedPayload{TowerKey: t.Key})
		}
	})
	s.imode = modeUpgrade
}

// ─── 空间查询辅助函数 ─────────────────────────────────────────────────

// nearestEnemy 返回距离像素位置最近的活跃敌人（无距离限制，用于测试模式的 hover 高亮）。
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

// enemyAtPixel 返回像素位置上的敌人（碰撞半径内），无匹配返回 nil。
// 拾取半径: max(敌人实际半径, 12px)，确保小型敌人也能被点中。
func (s *StageScene) enemyAtPixel(px, py float64) *enemy.Enemy {
	var best *enemy.Enemy
	bestDist := 20.0 // 最大拾取距离阈值
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
// 将像素坐标转为网格坐标（减去地图偏移，除以格子尺寸），然后查询塔数组。
// 正在卖出的塔（Selling=true）视为不存在，防止对售出动画中的塔操作。
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
