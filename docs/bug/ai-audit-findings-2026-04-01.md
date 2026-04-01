# AI 源码审核发现 (2026-04-01)

> 通过读源码发现的视觉/交互 bug，无需运行游戏。

---

## P0 渲染审核发现（13 个）

### P1 级（2 个）

1. **塔动画状态共享腐化** `draw_tower.go:329`
   - 同类型塔共享 1 个 Animator，多塔同时攻击/空闲时互相重置帧
   - 影响：同类型多塔的攻击动画闪烁
   - 修复：按 InstanceKey 而非 sprKey 缓存 Animator

2. **Dying 敌人除零崩溃** `draw_enemy.go:46`
   - `progress = 1 - DyingTimer/DyingDuration`，DyingDuration=0 时 NaN/Inf
   - 影响：精灵无限放大或消失
   - 修复：加 `if e.DyingDuration <= 0 { return }` 守卫

### P2 级（6 个）

3. **卖塔动画负缩放** `draw_tower.go:57`
   - SellAnim 浮点下溢导致 progress>1 → 负 scale → 精灵翻转
   - 修复：clamp progress 到 [0,1]

4. **ChoicePanel 意外关闭** `choice_panel.go:143`
   - 点击卡片外空白区域直接关闭面板，能力选择可能丢失
   - 修复：加 Dismissible 字段控制

5. **HP bar 死亡后残留橙色拖尾** `draw_enemy.go:215`
   - DisplayHP 滞后于实际 HP，dying 敌人显示幽灵血条
   - 修复：dying 时跳过 HP bar 渲染

6. **战灵拖尾在 (0,0) 闪点** `draw_warden.go:110`
   - TrailHistory 初始全 0，首帧移动前可能在左上角闪 1-2 个点
   - 修复：用哨兵值初始化 TrailHistory

7. **InfoPanel 遮挡 ActionBar** `info_panel.go:287`
   - 能力多时面板高度超出，覆盖底部"建塔/道具"按钮
   - 修复：panelY 底部留出 ActionBarH 高度

8. **HitFlash 衰减不平滑** `draw_enemy.go:181`
   - 闪白前 60% 时间满亮度，后 40% 突然衰减（阶梯状而非线性）
   - 修复：改用 `HitFlash / initialDuration * 90` 线性衰减

### P3 级（5 个）

9. Fallback 塔炮管不旋转（无 sprite 时）
10. 波次倒计时多显示 1 秒（`int()+1` 应改 `Ceil()`）
11. 灼烧叠加圆偏下，几乎看不到
12. Dying 敌人 Z 序与活敌相同（应先渲后活）
13. 眩晕和定身共用紫色状态点，无法区分

---

## P2 交互审核发现（11 个）

### P0 级（1 个）

1. **战灵选择无 ESC 退出** `stage.go:638`
   - modeWardenSelect 只响应鼠标点击，无键盘退出路径
   - 影响：手柄/键盘玩家可能软锁
   - 修复：handleWardenSelection 加 ESC 处理

### P1 级（3 个）

2. **P 键无 return — 同帧多处理** `stage_input.go:184`
   - 暂停后同帧还执行后续 hover/tap 逻辑
   - 修复：P 键处理后加 return

3. **S 键无 return — S+P 同帧状态冲突** `stage_input.go:170`
   - tryStartWave 可能触发 showWardenSelect，随后 P 覆盖为 modePaused
   - 修复：S 键后加 return

4. **B/I 键跨模式不清理状态** `stage_input.go:146,155`
   - 从 spawnMode 切到 buildMenu 时 spawnMode 标志残留
   - 修复：切模式时清理前一模式的状态标志

### P2 级（5 个）

5. **modeUpgrade ESC 死代码** `stage_input.go:107,126`
   - ESC case 存在但因早期 return 永远不执行
   - 修复：在 modeUpgrade 块内先检查 ESC

6. **showWardenSelect 强制中断任何模式** `stage.go:990`
   - 玩家正在放塔时被战灵选择覆盖打断
   - 修复：只在 modeIdle 时触发

7. **modeSpawnPlace 放怪后不退出** `stage_input.go:423`
   - 放完怪后留在 spawnPlace 模式，需手动 ESC
   - 修复：加提示 toast

8. **物品拖拽释放到空地无反馈** `stage_input.go:49`
   - 拖到非塔区域释放，无任何提示
   - 修复：加 toast "请拖拽到塔上使用"

9. **autoplay 无视当前模式** `stage.go:2456`
   - autoplay actions 在任何面板打开时都执行
   - 修复：加 imode==modeIdle 守卫

### P3 级（2 个）

10. prePauseMode 边缘情况（实际工作正常）
11. MEMORY.md 引用已删除的 modeEvent（文档过期）

---

## 汇总

| 来源 | P0 | P1 | P2 | P3 | 总计 |
|------|----|----|----|----|------|
| 渲染审核 | 0 | 2 | 6 | 5 | 13 |
| 交互审核 | 1 | 3 | 5 | 2 | 11 |
| **合计** | **1** | **5** | **11** | **7** | **24** |
