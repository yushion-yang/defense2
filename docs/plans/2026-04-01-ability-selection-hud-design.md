# Tower Ability Selection HUD Design

## Context

当前塔升级系统有以下问题：
1. `RollAbilityChoices` 每帧重新调用，选项不稳定（随机 bug）
2. 能力选择以 info panel 内联小按钮形式呈现，不够醒目
3. 缺少已获取能力的完整展示
4. 能力解锁是全局波次驱动但代码按塔单独判断

## Design

### 数据模型

**Tower struct 新增字段**：
```go
// PendingChoices 缓存每个能力位的 3 选 1 选项。
// key = category index (0-5), value = 3 个 AbilityDef。
// 建塔时一次性为所有已解锁位 roll；新波次解锁时追加 roll。
PendingChoices map[int][3]*config.AbilityDef
```

**选项确定时机**：
1. **建塔时** (`Pool.Place`)：根据当前全局 `wavesCleared` 计算已解锁位数，为每个已解锁且未填充的能力位 roll 3 个选项，缓存到 `PendingChoices`
2. **新波次解锁时**：全局波次触发（每 2 波），遍历所有塔，为新解锁的能力位 roll 并缓存
3. **选择后**：从 `PendingChoices` 删除对应 category key

**全局解锁计算**：`UnlockedSlots = min(wavesCleared / WavesPerUnlock, MaxAbilitySlots)`。建塔时传入 `wavesCleared` 用于初始化。

### HUD 交互流程

```
modeTowerSel (选中一个塔)
  Info Panel 显示:
    - 塔属性（伤害/攻速/射程）
    - 已获取能力列表（6 个 slot，已选的显示 icon+name，未选的灰色）
    - 若有待选能力位：显示「选择能力 (N)」按钮（N=待选数量）
    - 强度购买 + 卖塔按钮

  点击「选择能力」→ 进入 modeUpgrade
    - ChoicePanel 覆盖层弹出
    - 标题: 类别名称（如"攻击模式"/"控制效果"）
    - 3 张卡片：能力 icon + label + display 描述
    - 点选一张 → AddAbility → 关闭覆盖层 → 回到 modeTowerSel
    - 如果还有待选能力位，info panel 继续显示「选择能力」按钮
    - ESC/点外部 → 取消回到 modeTowerSel（不选，保持待选状态）
```

### 已获取能力展示

Info Panel 中能力展示区重构为 **6 slot 网格**：
- 每个 slot 对应一个类别（按 UnlockOrder 排列）
- 已选：icon + 能力名（彩色）
- 已解锁未选（有 PendingChoices）：闪烁金色边框 + "待选择"
- 未解锁：灰色锁定图标 + 类别名

### 关键文件变更

| 文件 | 变更 |
|------|------|
| `internal/core/tower/tower.go` | Tower 加 `PendingChoices` 字段 |
| `internal/core/tower/upgrade.go` | 新增 `RollAndCachePendingChoices(t, wavesCleared)`、`ClearPendingChoice(t, cat)` |
| `internal/core/tower/pool.go` | `Place` 后调用 `RollAndCachePendingChoices` |
| `internal/scene/stage.go` | 波次结束时遍历所有塔 roll 新解锁位 |
| `internal/render/hud/info_panel.go` | 重构能力展示区为 6-slot 布局 + "选择能力(N)" 按钮 |
| `internal/render/hud/choice_panel.go` | 适配能力选择数据（现有泛型面板） |
| `internal/scene/stage_input.go` | `modeUpgrade` 交互处理 + handleAbilityChoice 改用缓存 |
| `internal/scene/stage_info_vm.go` | VM 加 SlotVM 数据 + PendingCount |

### Verification

1. `make run` → 建塔 → info panel 显示 6 slot（第一个闪烁"待选择"）
2. 点"选择能力" → ChoicePanel 弹出 3 张攻击方式卡
3. 选一张 → 回到 info panel，slot 0 显示选中的能力
4. 第 3 波结束 → 新 slot 解锁闪烁
5. 第 5 波建新塔 → 自动有 2 个已解锁位（攻击 + 1 随机）
6. `go build ./...` 编译通过
