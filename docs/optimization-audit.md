# 游戏优化审查报告

> 2026-04-13 自动扫描生成，2026-04-14 修复并更新状态。

---

## 修复进度

| 批次 | 已完成 | 剩余 | commit |
|------|--------|------|--------|
| 第一批 | P1-P3, P9-P12, U1, C13, C14 | — | `85b0573` |
| 第二批 | U2-U7, C1-C6, C16-C19, Q10-Q11 | — | `a8d905a` |
| 第三批 | Q1-Q9, Q21-Q27, Q16-Q20, C21-C25 | — | `9a4f64b` |
| **合计** | **63 项已修** | **36 项待办** | |

---

## 剩余待办

### 性能（11 项）

#### 中影响

| # | 文件 | 问题 | 建议 | 难度 |
|---|------|------|------|------|
| P4 | `core/tower/tower.go:222` | AllAbilities() 每次调用创建 map+slice，64 塔/帧 | 缓存结果到 Tower 字段，能力变化时重算 | 低 |
| P5 | `core/buff/list.go:262` | Active() 每帧每塔创建完整 buff 快照给 `draw_tower_buff.go` | 增加 `FindByIDPrefix()` 方法，避免拷贝 | 低 |
| P6 | `core/tower/targeting.go:80` | FindExtraTargets() 堆分配 candidate+result slice | 用包级预分配缓冲区 `[32]candidate` 栈化 | 低 |
| P7 | `core/combat/apply_hit.go:139` | 即时伤害路径合成 Projectile 堆分配 | 改栈分配，需 `go build -gcflags="-m"` 验证逃逸 | 中 |
| P8 | `render/draw_tower.go:263` | getTowerFrame() 每帧字符串拼接做 map key | 缓存 animKey 到 Tower 字段 | 低 |
| P13 | `core/projectile/pool.go:233` | 弹射物 Each 全扫 1024 槽无 activeIdx | 增加 activeIdx 机制（参考 enemy pool 实现） | 高 |
| P14 | `render/draw_beam.go:46` | 光束每条 High 画质 25 次独立 draw call 未批量化 | 改为 DrawTriangles 批量化（参考 trail_batch.go） | 高 |

#### 低影响

| # | 文件 | 问题 | 建议 |
|---|------|------|------|
| P15 | `core/buff/list.go:238` | ClearByCategory 每次 make(map) | 改用 bitmask |
| P16 | `core/enemy/lifecycle.go:102` | EmitEvent make([]any) | 预分配复用 |
| P20 | `core/combat/apply_hit.go:39` | blockableStyles map 查找可改 switch | 改 switch |
| P21 | `core/strength/strength.go:44` | Effective() 遍历 3 个 map | 缓存值+dirty flag |

> P17/P18/P19 已评估为可忽略（GlobalAbilityTable/GlobalBalance 是简单全局变量返回；defer/recover 非热路径），不再列入待办。

---

### 代码质量（2 项）

| # | 文件 | 问题 | 建议 |
|---|------|------|------|
| Q6 | `core/combat/handler_barrage.go:173` | 获取 burstDelay 后 `_ = burstDelay` 丢弃 | 清理不需要的返回值 |
| Q12 | `core/combat/handler_widebeam.go:30` | 光束 duration/width/color 硬编码 | 移入 vfx.json 或 balance.json |

> Q13-Q15（spawn 动画时长/maxSpeedScale/画质阈值）已确认是命名常量，保持现状。

---

### 配置一致性（5 项）

| # | 文件 | 问题 | 建议 |
|---|------|------|------|
| C7 | `config/systems/cc.json` 等 3 个文件 | 纯文档混在可加载配置中 | 文件名加 `_doc_` 前缀或移入 `docs/` |
| C11 | `handler_barrage.go:38` | barrage 3 个常量完全无配置化 | 在 abilities.json 中添加字段 |
| C12 | `warden/state.go:316` | 弹速 350 硬编码 | 移入 wardens.json Params |
| C15 | `balance_config.go` vs `economy_spec.go` | KillReward float64 vs int 类型不统一 | 统一为 int（EconomyBalance 已删，仅 economy.go fallback 需确认） |
| C20 | `economy_spec.go` | sync.Once 模式与其他加载器不统一 | 统一为显式初始化模式或添加注释说明理由 |
| C26 | `tower_config.go:46` | TowerFileMeta 空结构体丢弃 _meta 字段 | 改为 `_` 前缀跳过解析 |

---

### UX 与游戏体验（18 项）

#### P1 严重体验（3 项）

| # | 文件 | 问题 | 建议 | 难度 |
|---|------|------|------|------|
| U8 | `scene/result.go` | 结果画面 3 秒动画无法跳过 | 点击跳到 phaseDone | 低 |
| U9 | `hud/pause_menu.go` | "重新开始"/"退出"无二次确认 | 增加确认步骤（文字变化或小面板） | 中 |
| U10 | `hud/info_panel.go` + `stage_input.go` | 售卖塔无确认，误售不可逆 | 双击确认或短时撤销 | 中 |

#### P2 中等体验（8 项）

| # | 文件 | 问题 | 建议 | 难度 |
|---|------|------|------|------|
| U11 | 多处 | 最小字号 9-11px，小屏难读 | 全局最小 12px | 中（影响面广） |
| U12 | 各场景 SwitchScene | 无场景过渡动画，硬切 | Game.SwitchScene 中添加 TransitionOverlay 200ms | 高 |
| U13 | `scene/select.go` | 点击"敬请期待"模式无音效 | 添加 playUIClick | 低 |
| U14 | `scene/settings.go` | 按钮无 hover 高亮 | 悬浮时绘制高亮背景 | 低 |
| U15 | `scene/select.go` | hoverDiff 被追踪但 Draw 未使用 | 在 Draw 中根据 hoverDiff 高亮 | 低 |
| U16 | 各场景 ESC | ESC 返回无音效 | 添加 playUIClick | 低 |
| U17 | `hud/item_panel.go` | 点击已耗尽道具无反馈 | Toast"道具已用完"或不可用音效 | 低 |
| U18 | `scene/campaign_select.go` | 版本号硬编码 "v0.1.0" | 改用 game.Version | 低 |

#### P3 体验打磨（7 项）

| # | 文件 | 问题 | 建议 | 难度 |
|---|------|------|------|------|
| U19 | `hud/top_bar.go` | 速度按钮宽度可能过窄 | 设最小宽度 44px | 低 |
| U20 | `hud/build_menu.go` | 关闭按钮是纯文本无按钮感 | 添加背景框或 X 图标 | 低 |
| U21 | `hud/toast.go` | Toast 无队列，新覆盖旧 | 实现简单队列(2-3 条) | 中 |
| U22 | `scene/select.go` | 模式卡选中无动画 | 缩放弹跳 150ms | 低 |
| U23 | `hud/info_panel.go` | InfoPanel 打开/关闭无动画 | 底部滑入/滑出 150ms | 中 |
| U24 | `hud/build_menu.go` | BuildMenu 打开/关闭无动画 | 同上 | 中 |
| U25 | `scene/title.go` | Title 全屏点击无输入冷却 | 新场景 1-2 帧冷却期 | 低 |

---

## 剩余统计

| 维度 | 剩余 | 高难度 | 中难度 | 低难度 |
|------|------|--------|--------|--------|
| 性能 | 11 | 2(P13/P14) | 1(P7) | 8 |
| 代码质量 | 2 | 0 | 1(Q12) | 1 |
| 配置一致性 | 6 | 0 | 0 | 6 |
| UX 体验 | 18 | 1(U12) | 5 | 12 |
| **合计** | **37** | **3** | **7** | **27** |

### 建议下一步

**快速收割（低难度，半天可完成 10+）：**
- U8 结果跳过、U13/U14/U15/U16 反馈补全、U18 版本号、U19 按钮宽度、U25 防抖
- C7 文档文件重命名、C15 类型统一、C26 空结构体
- Q6 清理返回值、P20 map→switch

**需要设计的（中高难度，按需规划）：**
- P13 弹射物池 activeIdx（参考 enemy pool，工程量中等）
- P14 光束渲染批量化（参考 trail_batch.go，工程量大）
- U12 场景过渡动画（需新增 TransitionOverlay 组件）
- U9/U10 破坏性操作确认（需 UI 设计）

---

## 已完成项速查

<details>
<summary>展开查看 63 项已完成修复</summary>

### 第一批 (commit 85b0573)
- [x] P1 遥测 Enabled 开关
- [x] P2/P3/P9-P12 pool.Each → EachActive (8 处)
- [x] U1 难度按钮功能修复
- [x] C13 火球速度死配置标记
- [x] C14 PerfectBonus.Base 8→2

### 第二批 (commit a8d905a)
- [x] U2 返回按钮 28→44px
- [x] U3 滑块旋钮 14→24px
- [x] U4 切换按钮 36→44px
- [x] U5 战灵按钮 36→44px
- [x] U6 难度按钮 28→40px
- [x] U7 设置按钮 28→40px
- [x] C1-C3 combat 5 个死字段标记
- [x] C4 warden 2 个死字段已移除
- [x] C5 bossPercentHpCap 死字段标记
- [x] C6 EconomyBalance 结构体移除
- [x] C16 Boss 默认值补全
- [x] C17 GlobalTowerTable nil 兜底
- [x] C18 WardenMechProjectileSpeed 默认值
- [x] C19 defaultDifficultyModes fallback
- [x] Q10 weakenAmplifyMax 常量
- [x] Q11 TriggerHitFlash 统一方法

### 第三批 (commit 9a4f64b)
- [x] Q1 删除过时 TODO
- [x] Q2 TODO 改为中文说明
- [x] Q3 删除 panelX 废弃变量
- [x] Q4 删除 vfx sw/sh 废弃赋值
- [x] Q5 GetHealAuraParams 返回值修正
- [x] Q7 HasPendingUpgrade 移除废参数
- [x] Q8 删除 SaveGameResult 死方法
- [x] Q9 删除 DesignWidth/Height 冗余常量
- [x] Q16-Q18 persistence 错误处理 + DefaultProgressManager
- [x] Q19 精灵解析 5 处错误日志
- [x] Q20 存档读取错误日志
- [x] Q21 Tower.EffectiveStrength 提取
- [x] Q22 Enemy.TriggerHitFlash 提取
- [x] Q23 DefaultProgressManager 提取
- [x] Q24 getAbilityDef 辅助函数
- [x] Q25 wave_preview CursorPos 合并
- [x] Q26 handler fallback 统一包级常量
- [x] Q27 map 访问方式统一(getAbilityDef)
- [x] C21-C25 过时注释修复 (5 处)

</details>
