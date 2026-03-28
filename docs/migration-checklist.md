# JS → Go 迁移清单

> 生成日期: 2026-03-28 | 分支: claude/visual-overhaul
>
> **使用方式**: 在每行最后的「决定」列填入:
> - ✅ 迁移
> - ❌ 不要
> - ⏳ 以后再说

## 优先级说明

- **高**: 影响核心游戏循环
- **中**: 丰富游戏深度
- **低**: 锦上添花 / 基础设施

---

| # | 功能 | 分类 | JS 文件 | 说明 | 优先级 | 决定 |
|---|------|------|---------|------|--------|----|
| 1 | 伤害管线 (8步) | 战斗核心 | `damagePipeline.js` | 免疫→Boss %HP上限→攻击加成→目标减免→伤害上限→护盾吸收→HP扣减→阈值触发→死亡 | 高 | 需要 |
| 2 | 伤害类型 | 战斗核心 | `damageTypes.js` | 物理/魔法/真实/纯粹 4种类型 + 穿透规则 | 高 | 需要   |
| 3 | 预留伤害追踪 | 战斗核心 | `damage.js` | reservedDamageMap 防止过杀 | 中 |    |
| 4 | 敌人能力 (40+) | 敌人 | `enemyAbilities.js` | 伤害上限/死亡分裂/偷金/召唤/闪现/复活/弱点轮换/反伤/隐身等 | 高 |    |
| 5 | 敌人行为 | 敌人 | `behaviors.js` | 狂暴(低HP加速)、回血、再生 | 中 |  需要  |
| 6 | 敌人生命周期 | 敌人 | `lifecycle.js` | onDeath/onSpawn/onDamaged 实例级钩子 | 中 |  需要  |
| 7 | 敌人事件处理 | 敌人 | `enemyEvents.js` | 敌人事件分发 | 低 |  需要  |
| 8 | 飞行路径 | 敌人 | `flyingPath.js` | 飞行敌人走直线忽略地形 | 中 | 需要   |
| 9 | 敌人Buff模板 | 敌人 | `buffTemplates.js` | 预定义敌人buff配置 | 低 |  需要  |
| 10 | 控制减免(韧性) | 战斗 | `crowdControl.js` | 韧性减免、减速上限0.7、控制免疫 | 中 |  需要  |
| 11 | 塔战斗模式 | 塔 | `modes.js` | 平衡/速射/狙击 3种模式切换 | 中 |    |
| 12 | 分支特化 | 塔 | `branch.js` | 射程/伤害/溅射/穿透激光/深度冻结 5条分支 | 中 | 需要   |
| 13 | 塔链接系统 | 塔 | `linkSystem.js` | 塔→塔 buff传递 + 类型去重 | 低 |    |
| 14 | 塔协同 | 塔 | `towerSynergy.js` | 近距配对加成(冰霜光环/碎裂) | 低 |    |
| 15 | 核心技能注册表 | 塔 | `coreSkillRegistry.js` | 实体无关可插拔技能框架(init/tick/draw/progress) | 中 | 需要   |
| 16 | 核心技能: 链式闪电 | 塔技能 | `coreSkills/chainLightning.js` | 链式闪电技能 | 中 | 需要   |
| 17 | 核心技能: 核弹 | 塔技能 | `coreSkills/nukeBomb.js` | 核弹技能 | 中 | 需要   |
| 18 | 核心技能: 风刃 | 塔技能 | `coreSkills/windBlade.js` | 风刃技能 | 中 | 需要   |
| 19 | 核心技能: 通道激光 | 塔技能 | `coreSkills/channelLaser.js` | 36扇区扫描 + OBB碰撞检测 | 中 |  需要  |
| 20 | 被动技能系统 | 塔 | `passiveSkill.js` | 导弹弹幕/审判光束/链式闪电/审判之雨 | 中 | 需要   |
| 21 | 缩放能力 | 塔 | `abilities/scaling.js` | 按等级成长维度(multiply/add_capped/inverse) | 低 | 需要   |
| 22 | 塔生命周期钩子 | 塔 | `towerLifecycle.js` | onPlace/onSell/onUpgrade | 低 |需要    |
| 23 | 冷却→通道状态机 | 塔 | `cooldownChannel.js` | 通用冷却-蓄力状态机 | 低 |  需要  |
| 24 | 维度元数据 | 塔 | `dimensionMeta.js` | 每能力缩放规则 | 低 |  需要  |
| 25 | 关键词查询 | 塔 | `keywordQuery.js` | 统一能力查找(兼容新旧格式) | 低 |  需要  |
| 26 | 攻击VFX系统 | 塔 | `attackVFX.js` | 攻击视觉效果管理 | 中 | 需要   |
| 27 | 波次敌人事件 (25+) | 事件 | `builtinEvents.js` | 造价提升/路径反转/飞行增援/全护盾/CC抗性等 | 高 |    |
| 28 | 事件翻译/分层 | 事件 | `eventTranslate.js` | 分层事件难度缩放 | 低 |    |
| 29 | 事件注册表(可扩展) | 事件 | `eventRegistry.js` | 可扩展handler注册(Go目前硬编码map) | 低 |    |
| 30 | 任务系统 | 内容 | `missionSystem.js` | 检查点3选1 + 10个条件检查器 + 6种奖励 | 中 |    |
| 31 | 成就系统 | 内容 | `achievements.js` | 极速击杀/完美通关/极简/大伤害/无尽10波 | 低 |    |
| 32 | Buff堆叠规则 | Buff | `stackRules.js` | 覆盖/最强/叠加 3种规则 | 中 | 需要   |
| 33 | Buff回调 | Buff | `buffManager.js` | onApply/onExpire/onTick | 中 |  需要  |
| 34 | 战力: 敌人减益层 | 战力 | `strengthState.js` | permanent/temp/enemyDebuff 三层结构 | 低 | 需要   |
| 35 | 战力: 射程/效果/眩晕乘数 | 战力 | `strengthSystem.js` | Go目前只有伤害/速度乘数 | 低 |  需要  |
| 36 | 战力: 链网络(Union-Find) | 战力 | `chainNetwork.js` | 150px阈值 +10/塔(Go用格邻接 +5) | 低 |  需要  |
| 37 | 通用选择面板 | UI | `choicePanel.js` | N选1通用UI(任务/事件复用) | 中 |  需要  |
| 38 | 面板FSM(5+互斥) | UI | `panelFSM.js` | Go用更简单的interactMode | 低 |  需要  |
| 39 | 综合会话统计 | 系统 | `GameSession.js` | peakDps/建塔数/总金币/泄漏数等 | 低 |  需要  |
| 40 | 事件总线 | 架构 | `eventBus.js` | 发布/订阅解耦 | 低 |  需要  |
| 41 | 状态选择器 | 架构 | `selectors.js` | 状态查询抽象 | 低 |  需要  |
| 42 | 技能测试模式 | 调试 | `SkillTestMode.js` | 99999金/999命沙盒 | 低 |  需要  |
| 43 | 自动对局/基准测试 | 调试 | `autoPlay.js` | 自动游玩 + 性能基准 | 低 |  需要  |
| 44 | 调试控制台API | 调试 | `consoleAPI.js` | window.TD_DEBUG | 低 |    |
| 45 | 调试覆盖层 | 调试 | `debugOverlay.js` | 运行时信息叠加显示 | 低 | 需要   |
| 46 | 运行时探针 | 调试 | `probes.js` | 性能/状态探针 | 低 |  需要  |
| 47 | 阵容编辑器 | 调试 | `lineupEditor.js` | 塔阵容编排工具 | 低 |  需要  |
| 48 | 设计约束 | 测试 | `design-contracts/` | 8个规则文件 | 低 | 需要   |
| 49 | 验证Schema | 测试 | `schemas/` | 3个配置校验文件 | 低 | 需要   |
| 50 | 雷击VFX | 视觉 | `thunderStrike.js` | 闪电视觉 + 周期性范围伤害 | 低 | 需要   |
| 51 | 击中/点击波纹特效 | 视觉 | — | 视觉反馈 | 低 | 需要   |
| 52 | 碰撞检测库 | 工具 | `collision.js` | 独立碰撞工具(Go用内联距离检查) | 低 |  需要  |
| 53 | 高分按模式记录 | 存档 | `progressManager.js` | Go目前只按地图记录 | 低 | 需要   |
| 54 | 职业统计: 总时长 | 存档 | `progressManager.js` | Go缺少totalTime | 低 |    |
| 55 | 模式偏好持久化 | 存档 | `progressManager.js` | 记住上次选择的模式/难度 | 低 |    |

---

## 已完成移植 (汇总)

- **场景**: 标题 / 选关 / 战灵选择 / 测试选择 / 战斗 / 结算
- **塔**: 8种攻击风格、36个能力(5类)、4级升级、对象池、目标选择
- **敌人**: 13种原型、路径移动、护盾、状态效果(减速/流血/灼烧/眩晕/定身)、对象池、波次生成器(5阶段递进+Boss)
- **弹射物**: 追踪、拖尾、穿透、散射、蓄力、弹跳、对象池
- **战灵**: 5种类型(prince/core_mech/chain/skystrike/envoy)、战力系统
- **战斗**: 8个攻击处理器、光束对象池
- **经济**: 击杀奖励、波次奖金、利息、卖塔退款、难度缩放
- **事件**: 7个友方事件处理器、加权随机选取、分层选择
- **游戏模式**: 6种(战役/无尽/限时/Boss冲刺/挑战/测试)
- **存档**: 文件存储 + 内存回退、高分、地图解锁、教程标记
- **渲染**: 地图/塔/敌人/弹射物/光束/战灵绘制器、HUD(11组件)、主题、绘图原语、精灵缓存、帧动画、飘字、屏幕震动、字体、图标
- **音频**: 110个WAV文件、30+触发点、音效节流
- **输入**: 鼠标 + 触摸、手势检测
- **教程**: 5步引导教程

## 明确不移植

- **英雄系统**: 已从Go版移除 (MEMORY.md 有记录)
- **heroWarden 类型**: 随英雄系统一并移除
- **HTML/CSS 叠加层**: 原生渲染不适用
