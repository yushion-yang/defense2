# 游戏优化审查报告

> 2026-04-13 自动扫描生成，覆盖代码质量/性能/配置一致性/UX 四个维度。

---

## 一、性能优化（25 项）

### 高影响

| # | 文件 | 问题 | 建议 |
|---|------|------|------|
| P1 | `core/telemetry/telemetry.go:72` | Record() 在伤害管线热路径每帧 1600+ 次 map 查找，正常游戏不需要遥测 | 增加 `Enabled bool` 开关，默认 false，仅 autoplay 开启 |
| P2 | `core/enemy/behaviors.go:81` | TickBehaviors() 用 `pool.Each`(256 槽) 而非 `EachActive`(~20) | 替换为 `pool.EachActive` |
| P3 | `core/combat/apply_hit.go:309,348` | splash/bounce 内部 `pool.Each` 全扫 256 槽，弹幕+溅射组合下放大 | 替换为 `pool.EachActive` |

### 中影响

| # | 文件 | 问题 | 建议 |
|---|------|------|------|
| P4 | `core/tower/tower.go:222` | AllAbilities() 每次调用创建 map+slice，64 塔/帧 | 缓存结果到 Tower 字段，能力变化时重算 |
| P5 | `core/buff/list.go:262` | Active() 每帧每塔创建完整 buff 快照给 `draw_tower_buff.go` | 增加 `FindByIDPrefix()` 方法，避免拷贝 |
| P6 | `core/tower/targeting.go:80` | FindExtraTargets() 堆分配 candidate+result slice | 用包级预分配缓冲区 `[32]candidate` 栈化 |
| P7 | `core/combat/apply_hit.go:139` | 即时伤害路径合成 Projectile 堆分配 | 改栈分配（`var synthLocal projectile.Projectile`） |
| P8 | `render/draw_tower.go:263` | getTowerFrame() 每帧字符串拼接做 map key | 缓存 animKey 到 Tower 字段 |
| P9 | `core/enemy/behaviors.go:237,297` | tickHealer/tickBuffer 内嵌 `pool.Each` | 替换为 `pool.EachActive` |
| P10 | `core/tower/targeting.go:61` | FindNearestEnemy() 用 `pool.Each` | 替换为 `pool.EachActive` |
| P11 | `core/tower/targeting.go:86` | FindExtraTargets() 用 `pool.Each` | 替换为 `pool.EachActive` |
| P12 | `core/pipeline/tick_abilities.go:93` | Phase 1.5 重置敌人标记用 `enemies.Each` | 替换为 `enemies.EachActive` |
| P13 | `core/projectile/pool.go:233` | 弹射物 Each 全扫 1024 槽无 activeIdx | 增加 activeIdx 或 count-based 提前退出 |
| P14 | `render/draw_beam.go:46` | 光束每条 High 画质 25 次独立 draw call 未批量化 | 改为 DrawTriangles 批量化 |

### 低影响

| # | 文件 | 问题 | 建议 |
|---|------|------|------|
| P15 | `core/buff/list.go:238` | ClearByCategory 每次 make(map) | 改用 bitmask |
| P16 | `core/enemy/lifecycle.go:102` | EmitEvent make([]any) | 预分配复用 |
| P17 | `render/draw_tower.go:340` | 每塔每帧调 GlobalAbilityTable() | 入口取一次传入 |
| P18 | `core/tower/tower.go:207` | RecalcStats 每次调 GlobalBalance() | 同上 |
| P19 | `core/enemy/lifecycle.go:126` | safeApply defer/recover 50ns 开销 | 非热路径，可忽略 |
| P20 | `core/combat/apply_hit.go:39` | blockableStyles map 查找可改 switch | 改 switch |
| P21 | `core/strength/strength.go:44` | Effective() 遍历 3 个 map | 缓存值+dirty flag |

**建议优先级**：P1→P2/P3 批量替换 Each→EachActive（一次性改动，收益最大）→P4~P8。

---

## 二、代码质量（28 项）

### TODO/FIXME 遗留

| # | 文件 | 问题 | 建议 |
|---|------|------|------|
| Q1 | `core/enemy/pool.go:262` | `TODO: 复活能力将通过能力系统实现` | 技能系统已移除，删除 TODO |
| Q2 | `core/tower/upgrade.go:338` | `TODO: integrate element boosts via BuffList` | 转 Jira 工单或删除 |

### 未使用代码

| # | 文件 | 问题 | 建议 |
|---|------|------|------|
| Q3 | `render/hud/pause_menu.go:108` | `panelX` 计算后 `_ = panelX` 丢弃 | 删除无用计算 |
| Q4 | `scene/vfx_preview.go:111` | `sw, sh` 计算后丢弃 | 删除无用代码 |
| Q5 | `scene/stage.go:1891` | `hp, hr, ok := e.GetHealAuraParams()` 中 `hp` 被丢弃 | 改为 `_, hr, ok :=` |
| Q6 | `core/combat/handler_barrage.go:173` | 获取 burstDelay 后丢弃 | 清理不需要的返回值 |
| Q7 | `core/tower/upgrade.go:102` | HasPendingUpgrade 的 wavesCleared 参数未使用 | 移除参数 |
| Q8 | `core/persistence/progress.go:386` | SaveGameResult 零调用的别名方法 | 删除 |
| Q9 | `core/game/constants.go:15` | DesignWidth/DesignHeight 零引用，与 ScreenWidth/ScreenHeight 重复 | 删除 |

### 硬编码魔数

| # | 文件 | 问题 | 建议 |
|---|------|------|------|
| Q10 | `core/combat/damage_pipeline.go:157` | 虚弱增伤上限 `0.5` 硬编码 | 移入 `balance.json combat.weakenCap` |
| Q11 | `core/combat/apply_hit.go:325` | HitFlash 阈值 `0.06`/`0.08`/`0.1` 三处重复 | 提取为命名常量 |
| Q12 | `core/combat/handler_widebeam.go:30` | 光束 duration/width/color 硬编码 | 移入配置 |
| Q13 | `core/enemy/pool.go:38` | spawn 动画时长 `0.5`/`0.3` 硬编码 | 移入 `balance.json` |
| Q14 | `core/enemy/events.go:11` | maxSpeedScale=2.4 硬编码 | 移入 `balance.json` |
| Q15 | `core/game/quality.go:71` | 自适应画质 4 个阈值硬编码 | 移入 settings/balance |

### 错误处理缺失

| # | 文件 | 问题 | 建议 |
|---|------|------|------|
| Q16 | `scene/bestiary.go:82` | `persistence.DefaultStorage()` 忽略 error | 至少 log 记录 |
| Q17 | `scene/bestiary.go:87` | `config.LoadEnemyArchetypes()` 忽略 error | 至少 log 记录 |
| Q18 | `scene/lang_select.go:81` | `persistence.DefaultStorage()` 忽略 error | 统一提供 `DefaultProgressManager()` |
| Q19 | `render/draw_warden.go:54` 等 5 处 | `cache.GetOrParse()` 忽略精灵解析 error | log 记录解析失败的资源路径 |
| Q20 | `core/persistence/progress.go:140` | 存档读取错误被忽略，进度可能静默丢失 | log + 备份恢复策略 |

### 重复代码

| # | 文件 | 问题 | 建议 |
|---|------|------|------|
| Q21 | `handler_radial/scatter/barrage` | Strength 获取三处相同模式 | 提取 `Tower.EffectiveStrength()` |
| Q22 | `apply_hit.go` + `stage.go` ×2 | HitFlash 设置三处完全相同 | 提取 `Enemy.TriggerHitFlash()` |
| Q23 | `bestiary.go` + `lang_select.go` | 存储+进度初始化重复 | 提供 `DefaultProgressManager()` |
| Q24 | 4 个 handler | 读取 AbilityTable 的嵌套模式重复 | 提取 `getAbilityDef()` 辅助函数 |

### 命名/风格不一致

| # | 文件 | 问题 | 建议 |
|---|------|------|------|
| Q25 | `scene/wave_preview.go:214` | 调用两次 `CursorPos()` 分别取 x 和 y | 一次调用取两值 |
| Q26 | 各 handler | fallback 默认值有的用局部变量有的用包级 const | 统一为包级 const |
| Q27 | `handler_radial` vs `handler_scatter` | map 访问方式不一致 (comma-ok vs nil check) | 统一一种方式 |

---

## 三、配置一致性（26 项）

### 死配置（JSON 定义但代码未读取）

| # | 配置字段 | 位置 | 说明 |
|---|---------|------|------|
| C1 | `combat.scatterBasePellets` / `scatterSpreadAngle` | balance.json | 实际从 abilities.json 读取 |
| C2 | `combat.radialBaseShots` / `radialRangeMult` | balance.json | 同上 |
| C3 | `combat.wideBeamRangeMult` | balance.json | 同上 |
| C4 | `combat.wardenProjectileSpeed` / `wardenFireballSpeed` | balance.json | 实际从 wardens.json Params 读取 |
| C5 | `combat.bossPercentHpCap` | balance.json | 已被 spawner.json 取代 |
| C6 | `economy` 区段 | balance.json | 已迁移到 economy.json |
| C7 | `cc.json` / `damage-pipeline.json` / `attribute-pipeline.json` | config/systems/ | 纯文档，代码不加载 |

### 硬编码与配置重复

| # | 文件 | 问题 | 建议 |
|---|------|------|------|
| C8 | `handler_scatter.go:39` | pellets=3, spread=60 与 balance.json 重复 | 移除 balance.json 中的重复定义 |
| C9 | `handler_radial.go:36` | totalShots=4, rangeMult=1.2 同上 | 同上 |
| C10 | `handler_widebeam.go:55` | rangeMult=3.0 同上 | 同上 |
| C11 | `handler_barrage.go:38` | 3 个常量完全无配置化 | 在 abilities.json 中添加字段 |
| C12 | `warden/state.go:316` | 弹速 350 硬编码 | 从配置读取 |

### 值不一致（可能影响运行时）

| # | 文件 | 问题 | 影响 |
|---|------|------|------|
| **C13** | `prince.go:103` vs `balance.json` | **火球速度 500 vs 350** | 代码用 500，balance.json 的 350 从未生效 |
| **C14** | `gamemode/base.go:61` vs `economy.json` | **PerfectBonus.Base 8 vs 2** | fallback 值与配置不一致，加载失败时经济偏差大 |
| C15 | `balance_config.go` vs `economy_spec.go` | KillReward float64 vs int | 语义不统一 |

### 缺少默认值处理

| # | 文件 | 问题 | 建议 |
|---|------|------|------|
| **C16** | `spawner_config.go` defaultSpawnerConfig() | Boss 的 PercentHpCap/DyingDuration 零值 | 添加显式默认值 |
| **C17** | `tower_config.go` GlobalTowerTable() | 加载失败返回 nil 无兜底 | 返回空 map |
| C18 | `balance_config.go` defaultBalance() | 缺 WardenMechProjectileSpeed 默认值 | 添加默认值 350 |
| C19 | `settings_config.go` LoadDifficultyModes() | 无 fallback | 添加 defaultDifficultyModes() |
| C20 | `economy_spec.go` | sync.Once 模式与其他加载器不统一 | 统一为显式初始化模式 |

### 过时注释

| # | 文件 | 问题 |
|---|------|------|
| C21 | `balance_config.go:3` | 列出"经济(economy)"但已迁移 |
| C22 | `validate.go:21` | 引用 `baseFireRate` 但字段已改名 `baseAttackSpeed` |
| C23 | `ability.go:40` / `dot.go:17` | 引用 `dotTickIntervals`(复数) 但实际是 `dotTickInterval`(单数) |
| C24 | `economy.json:8` | 注释说"从 balance.json 读取"但实际已独立 |
| C25 | `prince.go:90` | 注释硬编码数值可能与 wardens.json 不同步 |
| C26 | `tower_config.go:46` | TowerFileMeta 空结构体丢弃所有 _meta 字段 |

---

## 四、UX 与游戏体验（25 项）

### P0 功能缺陷

| # | 文件 | 问题 |
|---|------|------|
| **U1** | `scene/select.go` | **难度按钮 hitTestDiffButtons() 已定义但 Update() 未调用**，点击无响应 |

### P1 严重体验

| # | 文件 | 问题 | 建议 |
|---|------|------|------|
| U2 | `scene/title.go` hitTestNavBackBtn | 返回按钮 70x28px，高度不达标(需44+) | 扩展命中区域 |
| U3 | `scene/settings.go` | 滑块旋钮 14px，触摸无法操作 | 旋钮至少 24px，命中区域 44px |
| U4 | `hud/toggle_btn.go` | 切换按钮 36x36px | 扩展到 44x44px |
| U5 | `hud/warden_select_overlay.go` | 确认/跳过按钮 200x36px | 高度调至 44px |
| U6 | `scene/select.go` + `campaign_select.go` | 难度按钮 80x28px | 高度增至 40px+ |
| U7 | `scene/select.go` | 设置/图鉴按钮 70x28px | 高度扩展 |
| U8 | `scene/result.go` | 结果画面 3 秒动画无法跳过 | 点击跳到 phaseDone |
| U9 | `hud/pause_menu.go` | "重新开始"/"退出"无二次确认 | 增加确认步骤 |
| U10 | `hud/info_panel.go` + `stage_input.go` | 售卖塔无确认，误售不可逆 | 双击确认或撤销机制 |

### P2 中等体验

| # | 文件 | 问题 | 建议 |
|---|------|------|------|
| U11 | 多处 | 最小字号 9-11px，小屏难读 | 最小 12px |
| U12 | 各场景 SwitchScene | 无场景过渡动画，硬切 | 添加 200ms 淡入淡出 |
| U13 | `scene/select.go` | 点击"敬请期待"模式无音效 | 添加 UI 音效 |
| U14 | `scene/settings.go` | 按钮无 hover 高亮 | 悬浮时添加高亮 |
| U15 | `scene/select.go` | hoverDiff 被追踪但 Draw 未使用 | 在 Draw 中应用 hover 效果 |
| U16 | 各场景 ESC | ESC 返回无音效 | 添加 playUIClick |
| U17 | `hud/item_panel.go` | 点击已耗尽道具无反馈 | Toast 或"不可用"音效 |
| U18 | `scene/campaign_select.go` | 版本号硬编码 "v0.1.0" | 改用 game.Version |

### P3 体验打磨

| # | 文件 | 问题 | 建议 |
|---|------|------|------|
| U19 | `hud/top_bar.go` | 速度按钮宽度可能过窄 | 设最小宽度 44px |
| U20 | `hud/build_menu.go` | 关闭按钮是纯文本无按钮感 | 添加背景框或 X 图标 |
| U21 | `hud/toast.go` | Toast 无队列，新覆盖旧 | 实现简单队列(2-3条) |
| U22 | `scene/select.go` | 模式卡选中无动画 | 添加缩放弹跳 150ms |
| U23 | `hud/info_panel.go` | InfoPanel 打开/关闭无动画 | 滑入/滑出 150ms |
| U24 | `hud/build_menu.go` | BuildMenu 打开/关闭无动画 | 同上 |
| U25 | `scene/title.go` | Title 全屏点击无输入冷却 | 新场景 1-2 帧冷却期 |

---

## 五、统计总览

| 维度 | 总数 | 高优先 | 中优先 | 低优先 |
|------|------|--------|--------|--------|
| 性能 | 21 | 3 | 11 | 7 |
| 代码质量 | 27 | — | — | — |
| 配置一致性 | 26 | 4(C13/C14/C16/C17) | 6 | 16 |
| UX 体验 | 25 | 1(P0)+9(P1) | 8 | 7 |
| **合计** | **99** | **17** | **25** | **30** |

## 六、建议修复顺序

### 第一批（高收益低风险，1-2 天）

1. **pool.Each → EachActive 全局替换** (P2/P3/P9-P12) — 搜索替换级，遍历量降 10 倍
2. **遥测开关** (P1) — 改一行 if，消除热路径 1600+/帧 map 查找
3. **U1 难度按钮** — 功能缺陷，直接修复
4. **C13/C14 值不一致** — 确认设计意图后统一

### 第二批（中收益，2-3 天）

5. 触摸尺寸批量修复 (U2-U7) — 统一将按钮高度提升至 44px
6. 死配置清理 (C1-C6) — 移除 balance.json 7 个死字段
7. 默认值补全 (C16-C19) — 防止配置加载失败时异常
8. 魔数配置化 (Q10-Q15) — 提取常量或移入 JSON

### 第三批（打磨，按需）

9. 场景过渡动画 (U12)
10. 光束渲染批量化 (P14)
11. 重复代码提取 (Q21-Q24)
12. UI 反馈补全 (U13-U17)
13. 弹射物池 activeIdx (P13)
