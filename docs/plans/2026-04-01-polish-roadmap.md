# Defense2 打磨路线图

## 现状快照

| 维度 | 现状 | 短板 |
|------|------|------|
| 塔系统 | 1 种基础塔 + 33 能力 + tier 随机 | 只有 1 种塔，10 套精灵闲置 |
| 敌人 | 75 精灵，~15 种有独立行为 | 60 种仅外观差异，行为同质 |
| 战灵 | 5 种，单帧静态 | 与 7 帧敌人/5 帧塔对比粗糙 |
| 地图 | 8 张(easy×1/normal×2/hard×3/extreme×2) | 纯布局差异，无环境主题 |
| 道具 | 6 种消耗品 | 无图标，纯色圆替代 |
| 音效 | 126 WAV，27 常量绑定 | 99 个 WAV 未接入代码 |
| 音乐 | 0 首 | 完全缺失 |
| 后处理 | 9 shader | bloom 因引擎 bug 禁用 |
| 教程 | 5 条纯文本 | 无高亮/箭头/持久化 |
| 持久化 | 高分 modeID_mapID | 无成就/解锁/设置存储 |

---

## P0：核心补全

### P0-1 背景音乐

**问题：** 0 首 BGM，静默中打塔防缺乏沉浸感。

**方案：**

| 曲目 | 场景 | 风格 | 时长 |
|------|------|------|------|
| menu.ogg | Title/Select | 轻松电子，低节奏 | 60-90s 循环 |
| battle.ogg | Stage 战斗 | 紧张节拍，层次递进 | 90-120s 循环 |
| boss.ogg | Boss 波 | 激烈，鼓点加密 | 60-90s 循环 |

代码改动：
- `audio/manager.go` 新增 BGM 播放（OGG Vorbis 流式，Ebitengine `audio/vorbis` 包）
- 新增方法：`PlayBGM(name)` / `StopBGM(fadeDur)` / `SetBGMVolume(v)`
- BGM 音量默认 0.3，SFX 音量默认 0.8，独立控制
- 切换时机：
  - `NewStageScene()` → `PlayBGM("battle")`
  - `Spawner` 检测到 Boss 波 → `PlayBGM("boss")`
  - Boss 波结束 → `PlayBGM("battle")`
  - `NewSelectScene()` → `PlayBGM("menu")`
  - 胜利/失败 → `StopBGM(0.5)` + 播放 victory/defeat SFX

资源获取：Suno AI 生成或 Pixabay/freesound 免版权库。

**工作量：** ~150 行代码 + 3 个 OGG 文件

---

### P0-2 多塔种

**问题：** `towers.json` 仅 1 种 "basic"(哨兵)。10 套精灵(sentinel/shotgun/prism/cyclone/railgun/ricochet/mortar/hydra/nova/fortress)已有但未配置。

**方案：** 配置 5 种基础塔，各有不同的属性倾向和默认攻击方式。

| 塔 | 精灵 | 攻击方式 | 特点 | 费用 | tier 倾向 |
|----|------|---------|------|------|-----------|
| 哨兵 sentinel | sentinel | projectile | 均衡 | 50 | B/B/B |
| 霰弹 shotgun | shotgun | scatter | 近程高伤 | 60 | A/B/D |
| 棱光 prism | prism | wideBeam | 远程穿透 | 70 | C/C/S |
| 旋刃 cyclone | cyclone | spin_aoe | 范围持续 | 80 | B/A/C |
| 穿甲 railgun | railgun | pierce | 远程单体 | 100 | S/D/A |

tier 倾向说明（damage/speed/range）：
- 哨兵 B/B/B → budget 6 (2+2+2)，各项中等
- 霰弹 A/B/D → budget 6 (3+2+0)，高伤中速短程
- 棱光 C/C/S → budget 6 (1+1+4)，低伤低速超远程
- 旋刃 B/A/C → budget 6 (2+3+1)，中伤高速短程
- 穿甲 S/D/A → budget 6 (4+0+3)，超高伤极低速远程

改动点：
- `config/towers/towers.json` 新增 4 个塔定义（仿照 basic 格式）
- `RollTowerStats()` 的 `TierBudget=6` 不变，每种塔的 tier 分布通过 `tierBias` 字段倾向特定档位
- 建造菜单已支持多塔（`buildBuildMenuData` 遍历 `towerDefs`）

**工作量：** ~80 行 JSON + `randomize.go` 加 tierBias 支持 ~50 行

---

## P1：体验打磨

### P1-1 道具图标

**问题：** `assets/items/` 为空，ItemPanel 用 `draw.FilledCircle` 替代。

**方案：**
- 6 张 64x64 PNG，设计风格与现有 `assets/icons/` 一致（线条图标，单色填充）
- 命名：`item-base-damage.png` / `item-pot-damage.png` / `item-base-speed.png` / `item-pot-speed.png` / `item-base-range.png` / `item-pot-range.png`
- `generate-assets.mjs` 扩展 `items` 类型，JSON 描述 → SVG → PNG
- `item_panel.go` 改用 `render.GlobalIcons().Get("item-xxx")` + `draw.Sprite`
- 拖拽浮标也用精灵替代纯色圆

**工作量：** ~40 行代码 + 6 张图

### P1-2 战灵多帧动画

**问题：** 5 战灵各 1 张 PNG。敌人 7 帧(walk×4+hit×2+static)、塔 5 帧(idle×2+attack×3)。

**方案：**
- 每战灵 4 帧：idle-0/idle-1/attack-0/attack-1
- 文件命名遵循 `anim/loader.go` 的自动检测：`{key}-idle-0.png` / `{key}-attack-0.png`
- `generate-assets.mjs` 已支持 wardens 类型，补充帧定义
- `WardenRenderer` 已有 `Animator` 字段，接入即可
- idle 在非攻击时 0.5s 切换帧，attack 在开火时播放

**工作量：** ~30 行代码 + 20 张图(5×4)

### P1-3 音效绑定补全

**问题：** 126 个 WAV 仅 27 个绑定。以下已有 WAV 但代码未调用：

| 类别 | 未绑定 WAV | 应绑定位置 |
|------|-----------|-----------|
| CC/状态 | slow-apply, stun-impact, freeze-hit, freeze-chain, root-apply, knockback | `combat/crowd_control.go` ApplySlow/Stun/Freeze/Root |
| 灼烧/中毒 | burn-ignite, burn-tick, poison-tick, poison-pulse, bleed-tick | `combat/apply_hit.go` 或 buff.onApply |
| 护盾 | shield-break, hit-shield | `combat/damage_pipeline.go` 护盾破碎 |
| Boss | boss-phase, boss-invincible, boss-summon-minions | boss 行为触发点 |
| 敌人特殊 | teleport-blink, mirror-copy, split-pop, stealth-reveal, berserk-activate | `enemy/` 行为实现后绑定 |
| 暴击 | crit-hit | `combat/apply_hit.go` IsCrit 分支 |
| 链弹 | bounce-hit, electric-chain | bounce 能力命中 |
| 雷击 | thunder-strike, thunder-charge | `combat/thunder_strike.go` |
| 道具 | (缺) | 需新增 item-use.wav 或复用 upgrade |
| 反射 | reflect-hit | reflector 敌人行为 |
| 回血 | regen-tick, medic-heal, death-heal | healer/regen 敌人行为 |

优先级：先绑 CC/状态(6 个) + 护盾(2 个) + 暴击(1 个) = 9 个高频音效。其余随敌人行为实现逐步补充。

**工作量：** 第一批 ~50 行，全部 ~150 行

### P1-4 教程升级

**问题：** 5 条 `tutorial.CurrentMessage()` 纯文本叠在画面顶部，无引导、无持久化。

**方案：**

新增 `internal/core/tutorial/guide.go`（替换当前 tutorial.go）：

```
步骤流程：
1. "欢迎！这是你的基地，保护它不被入侵"     → 高亮路径终点
2. "点击【造塔】建造第一座防御塔"             → 高亮 ActionBar 造塔按钮
3. "选择一种塔，点击空位放置"                  → 高亮建造位格子
4. "点击【开波】放出敌人"                      → 高亮 TopBar 开波按钮
5. (等待第一波通过)
6. "干得好！点击塔查看详情"                    → 高亮已建的塔
7. "点击【升级】提升塔的战力"                  → 高亮升级按钮
8. "打开【道具】，拖拽道具到塔上强化属性"      → 高亮 ActionBar 道具按钮
9. "当塔解锁能力时，选择一个来特化它"          → (等能力选择触发)
10. "教程完成！祝你好运"                       → 自动关闭
```

视觉引导：
- `hud/tutorial_overlay.go`：全屏半透明遮罩(alpha=150) + 目标区域矩形挖洞(clear)
- 箭头：draw.ThickLine 画 V 形箭头指向目标区域
- 文本框：底部居中，半透明背景 + 白色文字 + "点击继续"提示
- 步骤间可点击跳过，也可按 ESC 跳过全部

持久化：`os.UserConfigDir()/defense2/tutorial_done` 文件存在则不再触发

**工作量：** ~350 行代码

### P1-5 敌人行为差异化

**问题：** 75 种精灵仅 ~15 种有行为代码。`config/settings.json` 的 `specialHints` 描述了 60+ 种设计意图但未实现。

**方案：** 优先实现 10 种与现有系统兼容的行为（不需要新机制）。

| 行为 | 实现方式 | 对应原型 | 可见性 |
|------|---------|---------|--------|
| healer | 每 3s 对半径 80px 内敌人回 5%maxHP | en-medic, en-priest | 绿色十字浮字 |
| stealth | 入场 alpha=0.15 持续 3s，受击解除 | su-shadow, su-phantom | 半透明渲染 |
| teleporter | HP<50% 传送到路径 70% 位置(一次) | su-blink, st-blink | 蓝色闪烁粒子 |
| splitter | 死亡时在原位生成 2 个 40%HP 小怪 | or-swarm, ch-swarm | 分裂动画 |
| buffer | 光环 +20% 移速给半径 100px 内敌人 | en-banner, or-drum | 黄色环脉冲 |
| regenerator | 每秒回 1%maxHP | na-troll, or-troll | 绿色+N浮字 |
| armored | 前半段路径减伤 40%，后半段正常 | ch-iron, en-knight | 盾牌图标 |
| berserker | HP<30% 时移速×1.5 | or-berserker | 红色光环 |
| reflector | 受击反弹 15%伤害给射手塔 | st-mirror | 白色闪烁 |
| shielded+ | 初始护盾=30%maxHP，破后 2s 内减速 50% | en-guard, ch-wall | 蓝色护盾条 |

实现位置：`enemy/behavior.go` 或拆分为 `enemy/behaviors/` 子目录。`Spawner` 在 `Spawn()` 时根据 archetype 的 `behavior` 字段注入回调。

**工作量：** ~500 行代码 + 配置

---

## P2：锦上添花

### P2-1 地图环境主题

**问题：** 8 张地图纯布局差异，背景色统一 `MapGradientTop=#193549 / Bot=#1b4332`。

**方案：**

| 地图 | 主题 | 渐变上/下 | 环境粒子 | 光照色温 |
|------|------|----------|---------|---------|
| map_01 Winding Canyon | 峡谷 desert | #2a1a0a / #1a1208 | 沙尘飘动 | 暖黄 |
| map_02 Crossroads | 森林 forest | #0a1a0f / #051208 | 落叶飘落 | 自然绿 |
| map_03 Iron Bastion | 钢铁 tech | #0a0f1a / #080a15 | 电弧闪烁 | 冷蓝 |
| map_04 Spiral Fortress | 古堡 stone | #1a150f / #0f0a08 | 火把余烬 | 暖橙 |
| map_05 Dual Battlefront | 冰原 ice | #0a1520 / #051018 | 雪花飘落 | 冷白 |
| map_06 Labyrinth Corridor | 地下 dark | #0a0a10 / #050508 | 暗影浮动 | 紫暗 |
| map_07 Extreme Narrows | 火山 lava | #200a0a / #150505 | 火星上升 | 炽红 |
| map_08 Arena | 虚空 void | #0a0520 / #050315 | 星光闪烁 | 紫蓝 |

改动：
- 地图 JSON 加 `"theme": "desert"` 字段
- `theme/colors.go` 新增 `MapThemes map[string]MapThemeColors`
- `render/draw_map.go` 按 theme 切换渐变色
- `particle/` 新增 4 种环境粒子预设（沙尘/雪花/落叶/星光，复用 Ambient 框架）
- 光照色温通过 `lighting.kage` 的光色参数调整

**工作量：** ~200 行代码 + 8 套色板 JSON

### P2-2 成就系统

15 个成就分 3 档：

| 档次 | 成就 | 触发条件 |
|------|------|---------|
| 铜 | 初次胜利 | 通关任意地图 |
| 铜 | 塔防新手 | 建造 10 座塔（累计） |
| 铜 | 首个 Boss | 击杀首个 Boss |
| 银 | 完美主义 | 任意关卡 3 星 |
| 银 | 连杀达人 | 单局 20 连杀 |
| 银 | 道具大师 | 单局使用 10 个道具 |
| 银 | 全能战士 | 5 种塔各建造至少 1 座 |
| 银 | 富甲一方 | 单局持有 1000 金币 |
| 金 | 全图三星 | 所有地图 3 星通关 |
| 金 | 百杀 | 单局击杀 100 敌人 |
| 金 | 零泄漏 | 通关 Hard 难度不泄漏 |
| 金 | 速通 | 10 分钟内通关任意关卡 |
| 钻 | 大师 | Extreme 难度全图通关 |
| 钻 | 完美大师 | Extreme 难度全图 3 星 |
| 钻 | 不灭传说 | Endless 模式坚持 50 波 |

代码结构：
- `internal/core/achievement/achievement.go`：定义 + Tracker
- 触发挂 EventBus（已有 7 种事件类型）
- 持久化：`os.UserConfigDir()/defense2/achievements.json`
- UI：解锁时 Toast + 结算画面新增成就栏

**工作量：** ~350 行代码

### P2-3 Bloom 修复

等 Ebitengine 升级修复 `DrawRectShader` 在 v2.9.9 的 runtime crash。代码已就绪（4 shader + 4 BloomPreset），只需 `BloomEnabled = true`。

### P2-4 设置界面

**方案：**

```
┌─────────────────────────┐
│       设    置          │
├─────────────────────────┤
│ 音乐音量  [====----] 40%│
│ 音效音量  [=======─] 80%│
│ 画    质  [高] 中  低   │
│ 语    言  中文 [English]│
├─────────────────────────┤
│        [ 返回 ]         │
└─────────────────────────┘
```

- 新增 `internal/scene/settings.go`
- 滑块组件 `hud/slider.go`（复用 draw.RoundRect + 拖拽）
- 从暂停菜单增加"设置"按钮入口
- 持久化：`os.UserConfigDir()/defense2/settings.json`
- 启动时读取并应用

**工作量：** ~400 行代码

### P2-5 解锁进度

- 初始开放：map_01 + sentinel 塔 + prince 战灵
- 通关 map_01 → 解锁 map_02 + shotgun 塔
- 通关 map_02 → 解锁 map_03/04 + core 战灵 + prism 塔
- 通关 map_04 → 解锁 map_05/06 + chain 战灵 + cyclone 塔
- 通关 map_06 → 解锁 map_07/08 + skystrike/envoy 战灵 + railgun 塔
- `ProgressManager` 扩展 `Unlocks map[string]bool`，JSON 持久化
- Select 场景锁定未解锁项（灰显 + 锁图标 + 解锁条件文字）

**工作量：** ~250 行代码

### P2-6 局后统计

Result 场景扩展：
- 新增统计数据收集（已有 telemetry 基础）
- DPS 时间曲线（简易折线图，draw.Line 连线）
- 金币收支：建塔/升级/击杀/道具
- 最高连杀、最强单塔 DPS、最多击杀塔
- 本地排行榜（按 map+difficulty，前 10 名）

**工作量：** ~500 行代码

---

## P3：长期演进

| 项目 | 说明 | 前置依赖 |
|------|------|---------|
| 随机事件 | 波间弹出增益/减益选择（类 Roguelike） | P0-2 多塔种 |
| 每日挑战 | 固定种子 + 特殊规则 | P2-2 成就 |
| 关卡编辑器 | 拖拽地图编辑 + 导出 JSON | P2-1 地图主题 |
| 多人竞速 | WebSocket 同屏，相同波次谁先死 | 网络层 |
| Micro-LLM | 已有引擎框架，可做 AI 解说/动态难度 | 模型训练 |

---

## 执行顺序

```
Sprint 1 ─ 核心完整
  P0-2 多塔种         ~130 行   纯配置，1 天
  P0-1 背景音乐       ~150 行   需音乐资源，1-2 天

Sprint 2 ─ 听觉完善
  P1-3 音效绑定(第一批) ~50 行   9 个高频音效，半天
  P1-1 道具图标        ~40 行   需图片资源，半天

Sprint 3 ─ 视觉一致
  P1-2 战灵动画        ~30 行   需图片资源，半天
  P2-1 地图主题        ~200 行  纯代码+色板，1 天

Sprint 4 ─ 可玩性深度
  P1-5 敌人行为(前 5 种) ~250 行  纯代码，1-2 天
  P1-3 音效绑定(第二批)  ~100 行  跟随行为实现，半天

Sprint 5 ─ 新手体验
  P1-4 教程升级        ~350 行  1-2 天
  P2-4 设置界面        ~400 行  1-2 天

Sprint 6 ─ 留存系统
  P2-2 成就系统        ~350 行  1-2 天
  P2-5 解锁进度        ~250 行  1 天
  P2-6 局后统计        ~500 行  2 天
```
