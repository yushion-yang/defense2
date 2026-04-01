# 12 交互状态机审核指导

## 审核目标

审核 interactMode 状态机的模式转换完整性、ESC 退出路径、键盘输入处理、防卡死。
通过读代码发现操作 bug——面板无法关闭、模式转换丢状态、同帧多输入冲突。

## 必读文件

| 文件 | 审核内容 |
|------|---------|
| `internal/scene/stage_types.go` | interactMode 枚举定义（11+ 种模式） |
| `internal/scene/stage_input.go` | handleInput() 主分发 + 各模式输入处理 |
| `internal/scene/stage.go` | Update() 中的模式分发 + 所有 `s.imode =` 赋值 |
| `internal/render/hud/choice_panel.go` | 能力选择面板的交互逻辑 |
| `internal/render/hud/warden_select_overlay.go` | 战灵选择覆盖层交互 |

## 检查项

### A. 模式转换完整性

对每种模式逐一检查：

| # | 模式 | 检查 | 预期 |
|---|------|------|------|
| A1 | modeIdle | ESC 行为 | 进入暂停（不退出游戏） |
| A2 | modeBuildMenu | 退出路径 | ESC/B 键/点击外部/选择塔类型 → 都应能退出 |
| A3 | modeBuildPlace | 退出路径 | ESC/放塔成功/点已有塔(toast) |
| A4 | modeTowerSel | 点外部关闭 | 点击空白区域应回到 modeIdle |
| A5 | modeSpawnMenu | ESC 退出 | ESC → modeIdle |
| A6 | modeSpawnPlace | 放怪后行为 | 留在 spawnPlace（可连续放）还是退出？ |
| A7 | modePaused | 恢复行为 | ESC/P/Space → 恢复 prePauseMode |
| A8 | modeWardenSelect | **ESC 退出** | **必须有键盘退出路径（P0 软锁风险）** |
| A9 | modeUpgrade | ESC 退出 | ESC 应关闭 ChoicePanel → modeTowerSel |
| A10 | modeItemPanel | ESC/I 键退出 | 两种方式都应回 modeIdle |
| A11 | modeItemDrag | 释放行为 | 释放到空地 → 回 modeItemPanel + 反馈 toast |

### B. ESC 键可达性

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| B1 | 每种模式都能 ESC | 搜索所有 `KeyEscape` 处理 | 所有模式有 ESC 退路（直接或通过 default case） |
| B2 | early return 不阻断 ESC | 读 handleInput 的 return 点 | modeUpgrade/modeItemDrag 的 early return 前应检查 ESC |
| B3 | ESC 在 Update() 分发中 | modeWardenSelect/modePaused 在 Update 中处理 | 不走 handleInput → 自己处理 ESC |

### C. 键盘输入冲突

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| C1 | S 键后有 return | 读 tryStartWave 调用后 | 防止 S+P 同帧冲突 |
| C2 | P 键后有 return | 读 modePaused 设置后 | 防止暂停后同帧执行后续逻辑 |
| C3 | B/I 键跨模式清理 | 读模式切换时的状态清理 | 从 spawnMode 切到 buildMenu 时清理 spawnMode 标志 |
| C4 | 速度键(1/2/3)模式守卫 | 读速度键处理 | 是否应在所有模式下生效？ |

### D. 面板交互

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| D1 | ChoicePanel 关闭方式 | 读 choice_panel.go Update | 点卡片选择/点外部关闭/ESC 关闭 — 三种都应工作 |
| D2 | ChoicePanel 强制性 | 是否可被意外关闭 | 能力选择是否应该不可关闭？ |
| D3 | WardenOverlay 关闭 | 读 warden_select_overlay.go | 确认/跳过按钮 + ESC 键 |
| D4 | InfoPanel 点外部 | 读 modeTowerSel 的 tap 处理 | 点空白 → deselect → modeIdle |

### E. 状态保存/恢复

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| E1 | prePauseMode 保存 | 搜索 `prePauseMode =` | 暂停前保存当前模式 |
| E2 | prePauseMode 恢复 | 搜索恢复逻辑 | 恢复后回到暂停前的模式（含放塔/选塔状态） |
| E3 | selectedTower 清理 | 模式切换时 | 进入 modeWardenSelect 时应清理 selectedTower |
| E4 | 强制模式切换 | showWardenSelect 是否检查当前模式 | 应只在 modeIdle 时触发，避免打断操作 |

### F. 拖拽与手势

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| F1 | tap vs drag 区分 | 搜索 gesture.JustTapped | 不应用 raw IsMouseButtonJustPressed（会把拖拽误判为点击） |
| F2 | 物品拖拽反馈 | 读 modeItemDrag 释放处理 | 释放到空地应有 toast 提示 |
| F3 | 相机拖拽冲突 | 读大地图拖拽逻辑 | 拖拽相机不应触发建塔/选塔 |

## 已知问题模式（历史 bug）

- **HUD 点外部无法关闭**: modeTowerSel 的 tap 处理可能被其他条件拦截
- **暂停恢复丢状态**: prePauseMode 未正确保存。已修复
- **拖拽误触**: 从 IsMouseButtonJustPressed 迁移到 gesture.JustTapped。已修复
- **modeUpgrade ESC 死代码**: ESC case 存在但因 early return 不可达。待修复
- **modeWardenSelect 无 ESC**: P0 软锁风险。待修复

## 跨系统关联

- handleInput ← Update() 在 statePlaying 时调用
- autoplay 的 executeAutoPlayAction 绕过 handleInput → 不受模式限制
- EventBus 事件（WaveCleared/EnemyKilled）在任何模式下触发 → 可能改变 UI 状态
- showWardenSelect 在 updatePlaying 头部强制检查 → 可打断任何模式
