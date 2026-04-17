# 文件职责表

> 自动生成，勿手动编辑。运行 `make index` 更新。

共 292 个文件, 74638 行代码, 47 个包。

## achievement (215 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `achievement.go` | 215 | Achievement tracking and persistence. |

## aiplayer (3843 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `action.go` | 64 | 延迟行动队列。 |
| `aiplayer.go` | 1116 | AI 玩家主结构体。 |
| `awareness.go` | 332 | 局势感知系统。 |
| `behavior.go` | 176 | - |
| `bubble.go` | 287 | 思维气泡数据模型。 |
| `coop_zone.go` | 93 | N 分区 Zone 系统。 |
| `cooperation.go` | 245 | 协作意识系统。 |
| `decision.go` | 1113 | AI 决策引擎。 |
| `personality.go` | 38 | AI 个性系统。 |
| `ping.go` | 65 | - |
| `spectator.go` | 138 | - |
| `sprite.go` | 116 | AI 精灵状态机。 |
| `zone.go` | 60 | 区域划分逻辑。 |

## anim (370 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `anim.go` | 104 | 帧动画控制器。 |
| `lib.go` | 99 | 动画帧库（共享）+ 轻量播放状态（per-instance）。 |
| `loader.go` | 167 | 动画帧加载器。 |

## audio (591 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `loader.go` | 111 | 音效批量加载器。 |
| `manager.go` | 416 | 音效管理器。 |
| `resample.go` | 64 | PCM pitch shifting via linear-interpolation resampling. |

## autoplay (7966 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `ability_scenarios.go` | 352 | JSON 驱动的能力级测试场景。 |
| `anomaly.go` | 1302 | 运行时异常检测器。 |
| `assertion.go` | 525 | 场景断言框架。 |
| `controller.go` | 384 | AutoPlay 控制器。 |
| `coverage.go` | 378 | 覆盖矩阵与测试计划生成器。 |
| `recorder.go` | 589 | JSON 数据收集器。 |
| `report.go` | 364 | 汇总报告生成器。 |
| `sim_report.go` | 399 | 仿真测试平衡报告生成器。 |
| `strategy.go` | 178 | AutoPlay 核心类型与策略接口定义。 |
| `strategy_balance.go` | 525 | 数值平衡测试策略与场景。 |
| `strategy_champion.go` | 276 | "最强玩法"策略。 |
| `strategy_competent.go` | 1522 | 仿真测试"合理玩家"策略。 |
| `strategy_focus.go` | 86 | 单塔极限策略。 |
| `strategy_greedy.go` | 152 | 贪心启发策略。 |
| `strategy_random.go` | 90 | 随机模糊策略。 |
| `strategy_scenario.go` | 535 | 脚本化场景策略。 |
| `strategy_simulation.go` | 221 | 仿真测试场景矩阵。 |
| `strategy_visual.go` | 88 | 视觉目录策略。 |

## buff (668 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `buff.go` | 69 | - |
| `dot.go` | 61 | 持续伤害（DoT）计时与伤害计算。 |
| `ids.go` | 43 | Buff ID 唯一标识符常量。 |
| `init.go` | 55 | 全局堆叠规则单例管理。 |
| `list.go` | 373 | BuffList 核心容器，管理活跃 buff 的增删查改和持续时间衰�... |
| `rules.go` | 67 | 堆叠规则定义与 JSON 配置加载。 |

## combat (1604 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `apply_hit.go` | 443 | 统一命中处理（战斗系统的核心枢纽）。 |
| `attack.go` | 130 | 攻击方式分发框架。 |
| `beam.go` | 72 | 光束数据与对象池。 |
| `cc_ids.go` | 14 | CC 回调类型唯一标识符常量。 |
| `crowd_control.go` | 118 | 控制效果（CC）韧性系统。 |
| `damage_pipeline.go` | 235 | 8步伤害管线（战斗系统的最终伤害计算层）。 |
| `damage_type.go` | 24 | 伤害类型系统。 |
| `handler_barrage.go` | 194 | 连射攻击方式（自管理）。 |
| `handler_radial.go` | 78 | 360度环射直线穿透攻击方式。 |
| `handler_scatter.go` | 73 | 锥形散射攻击方式。 |
| `handler_spinaoe.go` | 124 | 旋转范围伤害攻击方式（自管理）。 |
| `handler_widebeam.go` | 99 | 宽光束攻击方式（贯穿路径所有敌人）。 |

## config (2479 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `ability_config.go` | 137 | 能力配置数据结构与加载。 |
| `audit.go` | 56 | 配置审计工具。 |
| `balance_config.go` | 307 | 游戏平衡配置的 Go 映射。 |
| `buff_config.go` | 25 | buff 堆叠规则配置加载。 |
| `classic_preset_config.go` | 127 | 经典模式预设炮塔配置加载。 |
| `classic_waves_config.go` | 210 | 经典模式逐波出怪配置加载。 |
| `economy_spec.go` | 67 | 经济系统规格加载器。 |
| `enemy_config.go` | 291 | 敌人原型配置与能力系统的加载层。 |
| `i18n_resolve.go` | 81 | 配置加载后统一覆盖显示文本。 |
| `loader.go` | 217 | 配置加载核心，是整个配置系统的基础层。 |
| `platform_config.go` | 78 | 平台差异化配置。 |
| `scenario_config.go` | 93 | 测试场景快照的数据结构与加载。 |
| `settings_config.go` | 66 | 全局设置配置加载。 |
| `spawner_config.go` | 199 | 出怪系统完整配置加载（config/systems/spawner.json）。 |
| `tier_presets.go` | 61 | 塔属性档位预设加载。 |
| `tower_config.go` | 116 | 塔配置数据结构与加载。 |
| `validate.go` | 108 | 配置校验。 |
| `vfx_config.go` | 55 | VFX 特效目录配置加载。 |
| `warden_config.go` | 132 | 战灵配置加载。 |
| `wave_composition_config.go` | 53 | 波次出怪组合类型定义和辅助函数。 |

## debug (162 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `perf.go` | 162 | frame time + GC performance tracker. |

## descriptor (5820 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `ability_store.go` | 158 | 自定义能力持久化存储（CRUD）。 |
| `adapter.go` | 145 | EffectResult → HitResult/TickResult 适配层。 |
| `blueprint.go` | 134 | 定制炮塔蓝图数据结构与校验。 |
| `blueprint_store.go` | 146 | 蓝图持久化存储（CRUD）。 |
| `blueprint_to_def.go` | 128 | 蓝图→塔定义转换。 |
| `budget.go` | 94 | 预算计算系统。 |
| `condition.go` | 221 | 条件门接口及 11 种内置条件类型。 |
| `derive_ability_table.go` | 386 | 从描述符自动派生 AbilityDef 元数据表。 |
| `describe.go` | 271 | 从能力描述符生成人类可读的中文描述。 |
| `descriptor.go` | 702 | 能力描述符 JSON 模式与解析器。 |
| `descriptor_ability.go` | 375 | 描述符驱动能力的 tower.Ability/Ticker 适配器。 |
| `edit_state.go` | 731 | EditState ↔ AbilityDescriptor 双向转换。 |
| `effect.go` | 213 | 效果接口及 13 种具体效果实现。 |
| `effect_result.go` | 96 | 统一效果输出结构与上下文。 |
| `init.go` | 69 | 描述符能力的双轨注册初始化。 |
| `interpreter.go` | 218 | 描述符运行时解释器。 |
| `loader.go` | 86 | 能力描述符表加载器。 |
| `marshal.go` | 336 | AbilityDescriptor 的 JSON 序列化/反序列化。 |
| `prebuilt_blueprints.go` | 87 | 预制蓝图加载器。 |
| `primitive_meta.go` | 385 | 基元元数据注册表，供能力编辑器 UI 使用。 |
| `scaler.go` | 168 | 数值缩放器接口及基础实现。 |
| `selector.go` | 385 | 目标选择器接口及 9 种实现。 |
| `trigger.go` | 47 | 触发器类型定义及解析。 |
| `validate_ability.go` | 239 | 自定义能力平衡性校验与费用计算。 |

## draw (1135 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `batch_compact.go` | 32 | 线段批量缓冲区压缩工具。 |
| `circle.go` | 284 | draw 包最大的绘图文件，包含所有基础图元函数。 |
| `dashed.go` | 126 | 虚线绘制，与 line_batch 批量化集成。 |
| `glow_layer.go` | 65 | 辉光离屏合成层。 |
| `gradient.go` | 129 | 渐变图像生成与缓存。 |
| `hover.go` | 166 | 全局长按悬浮追踪器，为触摸设备模拟鼠标悬浮行为。 |
| `line_batch.go` | 129 | 透明批量化 vector 线段，draw 包的核心性能优化之一。 |
| `roundrect.go` | 163 | - |
| `scale.go` | 41 | HiDPI 缩放核心，draw 包的基石。 |

## easing (54 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `easing.go` | 54 | centralized easing/interpolation utilities. |

## economy (41 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `economy.go` | 41 | 经济系统。 |

## enemy (3012 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `ability_type_ids.go` | 52 | 怪物能力类型唯一标识符常量。 |
| `behaviors.go` | 434 | 敌人行为系统：每帧驱动的主动行为与能力计时器。 |
| `enemy.go` | 546 | 敌人实体定义与状态效果处理。 |
| `events.go` | 47 | 波次敌人事件系统。 |
| `ids.go` | 32 | 敌人子系统唯一标识符常量。 |
| `lifecycle.go` | 132 | 敌人生命周期钩子系统（观察者模式）。 |
| `movement.go` | 154 | 敌人沿路径点列表移动。 |
| `pool.go` | 404 | 敌人对象池：生成、击杀、遍历。 |
| `spawn_config.go` | 84 | 敌人生成配置。 |
| `spawner.go` | 1127 | 波次出怪管理器（753 行，本项目最复杂的单文件）。 |

## event (159 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `bus.go` | 159 | 全局事件总线。 |

## game (162 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `constants.go` | 37 | 全局常量。 |
| `quality.go` | 125 | quality level system with adaptive frame-rate downgrade. |

## gamemap (156 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `gamemap.go` | 156 | 运行时地图状态。 |

## gamemode (1101 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `base.go` | 85 | Mode 接口的默认空实现（Null Object 模式）。 |
| `config_ruleset.go` | 47 | 配置驱动的 TowerRuleset 实现。 |
| `difficulty.go` | 54 | 难度配置加载。 |
| `hook.go` | 173 | 模式钩子系统（运行时扩展点）。 |
| `mode.go` | 183 | 游戏模式框架。 |
| `mode_config.go` | 126 | 配置驱动模式系统的 JSON 数据结构与加载。 |
| `session.go` | 113 | 游戏会话运行时容器。 |
| `tower_ruleset.go` | 58 | 模式级塔加载规则。 |
| `universal.go` | 262 | 配置驱动的通用游戏模式。 |

## hud (5041 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `ability_helpers.go` | 60 | 能力显示相关的共享工具。 |
| `action_bar.go` | 166 | Bottom-center pill-shaped action toolbar. |
| `ai_overlay.go` | 106 | AI 玩家精灵和思维气泡渲染。 |
| `build_menu.go` | 560 | Build tower popup panel. |
| `choice_panel.go` | 329 | 通用选择面板（N 选 1）。 |
| `debug_overlay.go` | 106 | 调试覆盖层（F2 切换）。 |
| `debug_panel.go` | 207 | 调试面板（仅测试模式）。 |
| `info_panel.go` | 615 | Bottom-center tower detail panel. |
| `item_panel.go` | 280 | Item popup panel (2x3 grid), positioned above ActionBar. |
| `layout.go` | 9 | HUD layout helpers. |
| `mascot_overlay.go` | 130 | Mascot character overlay for the guide system. |
| `minimap.go` | 84 | 右上角小地图 HUD。 |
| `pause_menu.go` | 117 | 暂停菜单覆盖层。 |
| `primitive_picker.go` | 251 | 原语选择器弹窗组件。 |
| `spawn_menu.go` | 222 | 造怪选择菜单（仅测试模式）。 |
| `speech_bubble.go` | 195 | Reusable speech bubble component for the mascot guide system. |
| `toast.go` | 94 | Temporary on-screen notification toast. |
| `toggle_btn.go` | 73 | 左下/右下角的收起/展开切换按钮。 |
| `top_bar.go` | 221 | Centered pill-shaped top status bar. |
| `tutorial_overlay.go` | 76 | Tutorial guided overlay. |
| `warden_panel.go` | 187 | Bottom-right warden info panel. |
| `warden_select_overlay.go` | 430 | 战灵选择覆盖层（Stage 内使用）。 |
| `wave_announce.go` | 269 | Wave start announcement with slide-in/out animation. |
| `wave_panel.go` | 254 | 左侧抽屉式波次预览面板。 |

## i18n (165 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `i18n.go` | 148 | - |
| `resolve_config.go` | 17 | 配置加载后的显示文本解析。 |

## input (259 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `gesture.go` | 187 | 统一手势识别器（按→拖→放 三态状态机）。 |
| `manager.go` | 72 | 简化版输入管理器（旧接口，部分场景仍在使用）。 |

## item (187 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `item.go` | 187 | 道具系统。 |

## learning (1506 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `features.go` | 544 | 特征提取系统。 |
| `loader.go` | 24 | 权重模型加载器。 |
| `trainer.go` | 387 | 在线学习训练器。 |
| `weights.go` | 551 | 权重模型与神经网络评分。 |

## llm (754 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `connector.go` | 392 | - |
| `knowledge.go` | 75 | - |
| `prompt.go` | 287 | - |

## loader (132 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `tower_loader.go` | 132 | 塔配置→运行时定义转换器。 |

## main (1218 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `main.go` | 500 | AutoPlay 自动对局工具入口（无头模式批量跑关卡）。 |
| `train.go` | 214 | AI 学习系统离线训练管线。 |
| `visual.go` | 78 | 可视化自动对局模式。 |
| `main.go` | 42 | 游戏入口（桌面 + WASM 通用）。 |
| `main.go` | 384 | 写入 OP 测试用自定义能力和蓝图到 ~/.defense2/。 |

## mascot (1151 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `action.go` | 13 | - |
| `conditions.go` | 582 | - |
| `context.go` | 55 | - |
| `dialog.go` | 20 | - |
| `loader.go` | 48 | - |
| `mascot.go` | 433 | - |

## mobile (21 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `mobile.go` | 21 | Android 移动端入口。 |

## particle (445 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `emitter.go` | 235 | preset particle emitter functions. |
| `particle.go` | 210 | GPU-friendly particle pool with batch rendering. |

## persistence (840 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `progress.go` | 606 | 玩家进度管理。 |
| `storage.go` | 154 | 持久化存储。 |
| `storage_js.go` | 80 | WASM 环境下基于浏览器 localStorage 的持久化存储。 |

## physics (164 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `grid.go` | 164 | 空间网格索引，用于高效碰撞检测。 |

## pipeline (916 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `orchestrator.go` | 162 | Pipeline 编排器：每帧 tick 的核心调度中枢。 |
| `sys_endgame.go` | 42 | 波次奖励 + 胜负判定步骤。 |
| `sys_projectile.go` | 36 | 弹射物移动 + 命中检测步骤。 |
| `sys_spawn.go` | 140 | 敌人生命周期相关的 TickSystem 集合。 |
| `sys_tower.go` | 51 | 塔相关步骤：动画、能力、技能、战斗。 |
| `sys_warden.go` | 44 | 战灵行为 + 战灵技能步骤。 |
| `tick_abilities.go` | 137 | 塔能力 tick 管线：每帧重建 buff → 执行能力逻辑 → 收集�... |
| `tick_combat.go` | 304 | 战斗管线：塔→弹射物→敌人的完整战斗流程。 |

## postprocess (847 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `bloom.go` | 23 | bloom preset configurations for different game contexts. |
| `effects.go` | 184 | screen-level effect state manager. |
| `lighting.go` | 47 | simplified point-light system state manager. |
| `pipeline.go` | 465 | GPU 后处理管线。 |
| `shaders.go` | 128 | embeds and compiles Kage shaders for the post-processing pipeline. |

## projectile (342 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `pool.go` | 298 | 弹射物环形缓冲区对象池：发射、追踪、回收。 |
| `projectile.go` | 44 | 弹射物实体定义。 |

## render (3502 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `batch_compact.go` | 40 | 批量渲染缓冲区压缩工具。 |
| `cull.go` | 41 | viewport culling for large-map camera rendering. |
| `damage_colors.go` | 25 | 伤害类型 → 显示颜色映射。 |
| `draw_beam.go` | 191 | 光束渲染模块。 |
| `draw_enemy.go` | 645 | 敌人渲染模块。 |
| `draw_map.go` | 444 | 地图渲染模块。 |
| `draw_projectile.go` | 332 | 弹道体渲染模块。 |
| `draw_tower.go` | 371 | 塔渲染模块。 |
| `draw_tower_buff.go` | 43 | 炮塔金灵 buff 强化特效。 |
| `draw_warden.go` | 248 | warden rendering. |
| `floattext.go` | 139 | 浮动文本系统。 |
| `font.go` | 221 | 字体管理与文本渲染工具。 |
| `icon.go` | 64 | 图标管理器。 |
| `impact_vfx.go` | 138 | 命中冲击特效系统。 |
| `mascot_sprite.go` | 95 | Mascot sprite loader for the guide system. |
| `screenshake.go` | 64 | 屏幕震动效果。 |
| `splash_vfx.go` | 115 | splash ability impact ring VFX. |
| `trail_batch.go` | 286 | 弹道尾迹批量渲染器。 |

## scene (18941 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `ability_edit.go` | 1944 | 自定义能力编辑场景（完整管线编辑器）。 |
| `audio_preview.go` | 762 | Audio preview scene. |
| `autoplay_types.go` | 163 | AutoPlayer 接口和数据类型定义。 |
| `bestiary.go` | 472 | 图鉴场景。 |
| `blueprint_edit.go` | 1603 | 蓝图编辑场景（4步向导）。 |
| `campaign_select.go` | 499 | 战役模式关卡选择场景。 |
| `game.go` | 503 | 顶层游戏管理器，实现 Ebitengine 的 ebiten.Game 接口。 |
| `lang_select.go` | 126 | 首次语言选择场景。 |
| `loading.go` | 290 | 加载场景。 |
| `map_editor.go` | 808 | Map Editor scene (tower slot editor). |
| `result.go` | 703 | 结算场景（游戏结束后的统计与评价画面）。 |
| `scene.go` | 52 | 场景系统的核心接口定义。 |
| `select.go` | 576 | 模式选择场景（卡片式 UI）。 |
| `settings.go` | 390 | 设置场景。 |
| `settings_persist.go` | 80 | 设置持久化（音量/画质）。 |
| `stage.go` | 4462 | 游戏主战斗场景（~3600行，本项目最核心的文件）。 |
| `stage_info_vm.go` | 686 | 塔信息面板的 ViewModel 构建器。 |
| `stage_input.go` | 1091 | StageScene 的输入处理和交互状态机（11 种模式）。 |
| `stage_types.go` | 123 | StageScene 的类型定义和常量。 |
| `stage_warden_vm.go` | 179 | 战灵选择数据构建（从 config 加载并转为 hud.WardenOption）。 |
| `test_select.go` | 512 | 测试模式场景选择器。 |
| `title.go` | 145 | 标题场景（游戏启动首屏）。 |
| `tower_workshop.go` | 822 | 炮塔工坊场景（预制/自定义 能力+蓝图 四 Tab 管理中心）。 |
| `vfx_preview.go` | 1127 | VFX preview scene. |
| `warden_select.go` | 365 | 战灵选择场景。 |
| `wave_preview.go` | 458 | Wave Preview scene. |

## sprite (73 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `cache.go` | 52 | 图像缓存。 |
| `parser.go` | 21 | PNG 图像解析器。 |

## strength (293 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `chain.go` | 152 | 链网络系统（Union-Find 并查集）。 |
| `strength.go` | 141 | 塔的战力(Strength)数据系统。 |

## telemetry (143 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `telemetry.go` | 143 | 全局遥测计数器。 |

## theme (701 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `colors.go` | 344 | - |
| `ds.go` | 105 | - |
| `layout.go` | 252 | - |

## timescale (105 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `timescale.go` | 105 | - |

## tower (1681 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `ability.go` | 170 | 塔能力系统核心接口与注册表。 |
| `ability_ids.go` | 69 | 能力类型唯一标识符常量。 |
| `lifecycle.go` | 85 | 塔生命周期钩子系统（观察者模式）。 |
| `pool.go` | 301 | 塔对象池与空间索引。 |
| `randomize.go` | 153 | 塔随机属性生成与能力解锁顺序随机化。 |
| `targeting.go` | 110 | 塔索敌逻辑。 |
| `tower.go` | 311 | 塔实体定义。 |
| `upgrade.go` | 482 | 塔能力槽解锁与选择系统。 |

## tutorial (151 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `tutorial.go` | 151 | 新手教程系统。 |

## types (1192 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `chain.go` | 201 | 聚能战灵。 |
| `core_mech.go` | 165 | 机甲战灵。 |
| `envoy.go` | 235 | 金灵战灵。 |
| `prince.go` | 306 | 火灵战灵。 |
| `skystrike.go` | 285 | 水灵战灵。 |

## ui (1833 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `attr_row.go` | 55 | 属性值行组件。 |
| `card_grid.go` | 68 | 网格卡片布局组件。 |
| `icon_label.go` | 77 | 图标 + 单行文本组件。 |
| `label.go` | 118 | 安全单行文本组件。 |
| `layout.go` | 487 | 自适应布局组件。 |
| `panel_box.go` | 83 | 面板容器组件。 |
| `paragraph.go` | 72 | 安全多行换行文本组件。 |
| `scale_text.go` | 82 | 三段式缩放文本组件。 |
| `scroll_panel.go` | 104 | 可滚动面板组件。 |
| `text_fit.go` | 159 | 自适应文本工具集。 |
| `tooltip.go` | 79 | 浮动提示面板组件。 |
| `widget.go` | 449 | 可复用 UI 组件。 |

## vfx (1867 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `vfx_enemy.go` | 308 | 敌人视觉效果（解耦版）。 |
| `vfx_mascot.go` | 125 | Mascot-themed visual effects. |
| `vfx_projectile.go` | 162 | 弹道视觉效果（解耦版）。 |
| `vfx_stage.go` | 350 | stage-level visual effects (extracted from stage.go). |
| `vfx_tower.go` | 488 | 炮塔视觉效果（解耦版）。 |
| `vfx_warden.go` | 434 | 战灵视觉效果（解耦版）。 |

## warden (630 行)

| 文件 | 行数 | 职责 |
|------|------|------|
| `state.go` | 453 | 战灵公共状态基座。 |
| `warden.go` | 177 | 战灵系统框架。 |

