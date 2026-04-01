# HUD / FSM 交互审计 (2026-04-01)

## 架构概述

两层状态机控制游戏交互：

- **stageState** (3 值): `statePlaying` / `stateVictory` / `stateDefeat` — 外层生命周期
- **interactMode** (9 值): 控制 `statePlaying` 内的玩家交互

```
modeIdle ──┬── modeBuildMenu ── modeBuildPlace
           ├── modeTowerSel ── modeUpgrade
           ├── modeSpawnMenu ── modeSpawnPlace
           ├── modePaused
           └── modeWardenSelect
```

`Update()` 先分派 stageState，再按 interactMode 路由。`Draw()` 按 interactMode 决定 HUD 可见性。

---

## 已修复的问题

### [FIXED] 手动开波 EvtWaveCleared 不触发

- **根因**: `tryStartWave()` 在 `handleInput()` 中调用 `StartNextWave()` 递增 Wave，`updatePlaying()` 捕获 `prevWave` 时已是新值
- **影响**: 能力解锁、波次奖励、战灵成长全部失效
- **修复**: 提取 `onWaveTransition(prevWave)` 统一处理，所有 `StartNextWave()` 调用点包裹 prevWave 检测

### [FIXED] DragEnabled=false 时移动吞掉点击

- **根因**: `modeTowerSel` 从 DragEnabled 列表移除后，鼠标轻微移动(>5px) 设 `movedBeyond=true` → 松开时既不是 tap 也不是 drag
- **影响**: 展开塔 HUD 后点击空白区域无法取消
- **修复**: gesture.go 松开判定中，`!dragging && !onUI` 时仍判定为 tap

---

## 待修复问题

### #2 TopBar "菜单"按钮不保存 prePauseMode [Medium]

**文件**: `stage_input.go:244`

```go
case "menu":
    s.imode = modePaused  // 未保存 prePauseMode!
```

ESC 和 P 键正确保存了 `s.prePauseMode = s.imode`，但 TopBar "菜单"按钮没有。

**影响**: 点菜单按钮暂停后恢复，之前的交互状态（放塔/选塔/造怪）丢失，回到 modeIdle。

**修复**: 加 `s.prePauseMode = s.imode`。

### #5 胜利/失败时不清除覆盖层状态 [Medium]

**文件**: `stage.go` stateVictory/stateDefeat 设置处

设置 `stateVictory/stateDefeat` 时未重置 `imode`。后续 Draw 仍按旧 imode 渲染：

- `modeUpgrade` 中胜利 → ChoicePanel 全屏遮罩覆盖胜利画面
- `modePaused` 中胜利 → 暂停菜单和胜利画面同时渲染

**修复**: 设置 stateVictory/stateDefeat 时：
```go
s.imode = modeIdle
s.selectedTower = nil
if s.choicePanel != nil { s.choicePanel.Close() }
```

### #3 键盘快捷键未按模式过滤 [Low-Medium]

**文件**: `stage_input.go:108-134`

| 快捷键 | 问题 |
|--------|------|
| S (开波) | 在 modeBuildMenu/modeBuildPlace/modeSpawnMenu 中可触发 |
| 1/2/3 (变速) | 在所有模式中可触发（低风险） |

**修复**: S 键加 `s.imode == modeIdle \|\| s.imode == modeTowerSel` 守卫。

### #6 Delete/Backspace 卖塔不重置 imode [Low]

**文件**: `stage_input.go:131-133`

```go
if inpututil.IsKeyJustPressed(ebiten.KeyDelete) || ... {
    s.trySellTower(...)  // selectedTower 被设为 nil
    // 但 imode 仍是 modeTowerSel!
}
```

**影响**: 留下一帧 `modeTowerSel + selectedTower=nil` 不一致状态。下一次点击自动修正。

**修复**: 加 `s.imode = modeIdle`。

### #7 战灵选择用原始 tap 而非 Gesture [Low-Medium]

**文件**: `stage_input.go:383`

`handleWardenSelection` 使用 `isTapJustPressed()`（原始 inpututil 检查）而非 `g.JustTapped()`。

**影响**: 拖拽手势可能被误判为点击，意外选中战灵。

**修复**: 改用 gesture 系统。

### #12 战灵面板切换留下陈旧 modeTowerSel [Low]

**文件**: `stage_input.go:205-213`

点击战灵面板切换时清除 `selectedTower` 但未重置 `imode`（若当前是 `modeTowerSel`）。

**影响**: 同 #6，`modeTowerSel + selectedTower=nil` 不一致。

**修复**: 加 `if s.imode == modeTowerSel { s.imode = modeIdle }`。

---

## 低优先级 / 设计决策

### #1 panel_fsm.go 死代码

MEMORY.md 记录"已删除"但文件仍存在。建议确认后删除。

### #4 ChoicePanel Close 模式脆弱

点击面板外 → `Close()` 不触发回调 → 正常工作，但 `Close()` 后 `Options/OnSelect` 为 nil。如果回调中引用面板状态会 panic。当前代码安全但模式脆弱。

### #8 战灵选择无 ESC 退出

`modeWardenSelect` 下 ESC 无效（handleInput 不被调用）。可能是设计意图（强制选择），但与 modeUpgrade 的 ESC 行为不一致。

### #9 P 键 modePaused 分支死代码

`handleInput` 中 P 键的 `s.imode == modePaused` 检查永远不可达（modePaused 走 handlePausedInput）。无害但增加理解成本。

### #10 modeBuildPlace 时建塔菜单不可见

`Visible: s.imode == modeBuildMenu` 在 modeBuildPlace 时为 false → 放塔时看不到选了哪种塔。可能是有意设计。

### #11 调试面板优先级高于模式 UI

debug panel 点击检测在 mode switch 之前执行。debug 按钮若与 build menu 重叠会吃掉点击。仅影响测试模式。

---

## 状态转换矩阵

```
From \ To        | Idle | BuildMenu | BuildPlace | TowerSel | Upgrade | SpawnMenu | SpawnPlace | Paused | WardenSel
─────────────────|──────|───────────|────────────|──────────|─────────|───────────|────────────|────────|──────────
modeIdle         |  -   | B/造塔btn |            | tap塔    |         | 调试btn   |            | ESC/P  | 首波倒计时
modeBuildMenu    | ESC  |    -      | tap格子    |          |         |           |            | ESC/P  |
modeBuildPlace   | ESC  |           |     -      |          |         |           |            | ESC/P  |
modeTowerSel     | ESC  |           |            |    -     | 选能力  |           |            | ESC/P  |
modeUpgrade      |      |           |            | 选完/ESC |    -    |           |            |        |
modeSpawnMenu    | ESC  |           |            |          |         |     -     | tap类型    | ESC/P  |
modeSpawnPlace   | ESC  |           |            |          |         |           |     -      | ESC/P  |
modePaused       |      |           |            |          |         |           |            |   -    |
modeWardenSelect |      |           |            |          |         |           |            |        |    -
```

注：Paused 恢复到 prePauseMode（任意模式）。WardenSelect 完成后回到 modeIdle。

---

## 建议修复优先级

1. **#2** TopBar 菜单保存 prePauseMode — 一行修复，用户可感知
2. **#5** 胜败时清除覆盖层 — 几行修复，影响游戏结束体验
3. **#6/#12** 卖塔/战灵面板切换重置 imode — 一行修复，消除不一致状态
4. **#3** S 键开波加模式守卫 — 防止意外操作
5. **#7** 战灵选择改用 Gesture — 保持输入系统一致性
