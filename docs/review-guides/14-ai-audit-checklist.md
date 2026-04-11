# AI 审查清单：系统性 Bug 与优化发现

> 供 AI session 逐项执行的审查清单。每项包含：验证方法、grep 命令、预期结果。
> 根据历史 bug 模式（display mismatch / feature not wired / config drift / state leak / ability interaction / warden issues）推导。

Date: 2026-04-11

---

## 审查方法

每个审查项标注了优先级（P0/P1/P2/P3）和验证方式：
- **grep**: 使用 Grep 工具搜索代码关键字
- **trace**: 追踪调用链确认连通性
- **cross-ref**: 交叉比对 JSON 配置与代码
- **read**: 精读函数逻辑确认正确性

---

## A. Config↔Code 一致性

### A-1. [cross-ref] 能力 JSON 与代码注册完备性
- 提取 `config/abilities/abilities.json` 所有 `"type"` 值（31 个）
- 提取 `internal/core/tower/abilities/config_ability.go` 所有 switch case 标签
- 提取 `internal/core/tower/abilities/scaling.go` 所有 Name() 返回值
- 交叉比对：JSON 中每个 type 必须在代码中有处理
- **已知问题**：`stackDamage`/`percentHpDamage` 在 tower.go struct 中引用但无能力实现

### A-2. [cross-ref] 敌人原型 JSON 与出怪器引用完备性
- 提取 `config/enemies/enemies-core.json` 所有原型 key（18 个）
- 提取 `internal/core/enemy/spawner.go` `waveCompositions` 表中的所有字符串
- 交叉比对：出怪器引用的每个原型必须在 JSON 中存在

### A-3. [cross-ref] 敌人能力 JSON 与代码应用完备性
- 提取 `config/enemies/abilities.json` 所有能力 type（15 个）
- 在 `internal/core/enemy/spawn_config.go` 和 `internal/scene/stage.go` 中搜索每个 type
- 每个能力必须有运行时处理代码

### A-4. [cross-ref] 战灵 JSON 参数与 Init() 默认值
- 对每个战灵（prince/core/chain/skystrike/envoy），比对 `wardens.json` 参数键与 `ParamOr()` 调用
- JSON 中的参数应覆盖所有 ParamOr 的 key

### A-5. [cross-ref] 攻击方式注册完备性
- `tower.go` 定义 8 个 AttackStyle 常量
- `combat/attack.go` init() 注册了 5 个 handler
- 验证哪些方式是 SelfManaged，哪些走 registry

---

## B. 死代码 / 未使用功能

### B-1. [P1][grep] 5 个 scaling 能力无功能（TODO 代码）
- `killUpgrade`/`waveScale`/`periodicCast`(case 2)/`neighborBoost`/`elementSwitch`(fire/lightning)
- 在 `scaling.go` 中计算了 bonus 但赋值给 `_`（TODO 注释）
- 若在 `abilities.json` 中可选，玩家选了等于浪费能力槽
- **验证**：grep 这 5 个名字在 `abilities.json` 中是否存在

### B-2. [grep] Tower 死字段
- `LastPercentHpTarget`/`PercentHpRecentTargets`/`StackDamageCount` 在 tower.go 声明
- 无对应能力实现代码
- **验证**：grep `LastPercentHp|StackDamage` 在 `internal/core/`

### B-3. [grep] VFX 函数调用者验证
- 对 `internal/render/vfx/` 每个导出函数，grep 在 vfx 包外的调用者
- 排除 `vfx_preview.go` 后仍需有至少 1 个游戏逻辑调用者

---

## C. 空指针风险

### C-1. [grep] Tower.Strength 空指针
- grep `\.Strength\.` 在 `internal/core/` 所有文件
- 每个调用点需确认 Strength 已初始化或有 nil guard

### C-2. [grep] Tower.Buffs / Enemy.Buffs 空指针
- grep `\.Buffs\.` 在 `internal/core/`
- Pool.Spawn/initTower 中初始化 Buffs，但回收/死亡路径是否有遗漏？

### C-3. [read] 弹射物目标悬空指针
- `projectile.Pool.Update` 检查 `p.Target.Active`
- 但 `Kill()` 不设 `Active=false`（仅设 DyingTimer），导致弹射物仍追踪濒死敌人
- 当敌人 slot 被复用（新敌人 Spawn 在同一槽位），旧弹射物追踪新敌人
- **验证**：grep `p\.Target` 在 `internal/core/projectile/` 和 `pipeline/tick_combat.go`

### C-4. [read] Skystrike BurstTarget 悬空指针
- `skystrike.go` 存储 `BurstTarget *enemy.Enemy`
- 若目标死后 slot 复用，Active 为 true 但是新敌人
- **验证**：read skystrike.go 确认防御措施

---

## D. 能力系统缺口

### D-1. [trace] Abilities 与 AllAbilities() 同步脆弱性
- `t.Abilities = t.AllAbilities()` 在 `upgrade.go:156` 调用
- 任何修改 `AbilitySlots` 而不重新同步的代码路径会导致能力静默失效
- **验证**：grep `AbilitySlots\[` 赋值在 `internal/core/tower/`

### D-2. [trace] enhance 能力实现位置
- `config_ability.go` 对 `enhance` 返回 nil
- 实际处理可能在 `upgrade.go` 或 `branch.go`
- **验证**：grep `enhance` 在 `internal/core/tower/`

### D-3. [read] multiTarget 能力路径
- `config_ability.go` 对 `multiTarget` 返回 nil
- 实际处理在 `tick_combat.go` 通过 `multiTargetCount`
- **验证**：trace `multiTarget` 从能力获取到战斗应用的完整链路

---

## E. 敌人行为完备性

### E-1. [trace] 5 种行为全部被 TickBehaviors 调度
- healer/stealth/splitter/buffer/regenerator
- **验证**：read `behaviors.go` TickBehaviors 的 switch-case

### E-2. [grep] SpeedBuff 每帧重置模式
- SpeedBuff 必须每帧清零再由 buffer 行为设置
- **验证**：grep `SpeedBuff` 在 `internal/core/enemy/`

### E-3. [read] 分裂子体继承问题
- DeathSpawn 子体使用 DefaultSpawnConfig()，丢失原型能力
- **验证**：read `pool.go` deathSpawn 生成逻辑

---

## F. 战灵系统（P0 级问题区域）

### F-1. [P0][trace] 战灵 SetTemp 被 ClearTransient 清除
- **已确认 bug**：执行顺序为 Step 4（战灵 Tick → SetTemp）→ Step 6（ClearTransient 清除所有 Temp）
- 导致链战灵和金灵的强度 buff 每帧被清零
- 这是 bug record 中「金灵没有给炮塔加强度」和「串联没有加属性」的根因
- **验证**：read `stage.go` 确认 step 4 和 step 6 的顺序

### F-2. [P0][read] chainActive 标志引用错误的战灵类型
- `stage.go` 中 `chainActive := s.wardenType == "envoy"`
- 应该是 `s.wardenType == "chain"`
- 导致链战灵的全局连锁网络从不激活，金灵却激活了不相关的连锁系统
- **验证**：read `stage.go` 搜索 `chainActive`

### F-3. [trace] 战灵攻击→伤害→VFX 完整链路
- 每个战灵：Tick → 目标选择 → ApplyDamage → ProcessDamage → VFX
- **验证**：对 5 种战灵逐一 trace

### F-4. [read] 金灵 OnSpecial 五星芒触发
- bug record：「金灵释放的第一次技能，没有给炮塔展示五星芒特效」
- **验证**：trace envoy OnSpecial → stage.go OnWardenSpecial 回调 → VFX

---

## G. 经济 / 进度系统

### G-1. [read] 击杀金币流转
- 从 ProcessDamage kill → stage.go emitKill → gold 增加
- **验证**：trace 完整金币流转链路

### G-2. [grep] Enemy.Reward 字段是否为死数据
- `enemy.Reward` 在 stage.go 设置但 emitKill 使用 `economy.KillGold() * rewardScale`
- **验证**：grep `\.Reward\b` 在 `internal/` 确认是否有实际消费方

### G-3. [read] goldPassive 能力是否正确返回金币
- `config_ability.go` 的 goldPassive OnTick 应返回 `TickResult{GoldEarned: N}`
- **验证**：read goldPassive case

### G-4. [cross-ref] 4 种游戏模式经济公式自洽
- 已有契约测试覆盖 campaign 模式
- **验证**：对 endless/timed/bossRush 模式逐一检查 Calc() 公式

---

## H. 渲染 / 交互一致性

### H-1. [grep] HiDPI 合规
- 禁止直接使用 `vector.*`、`ebiten.CursorPosition`、`ebiten.TouchPosition`
- **验证**：grep 违规调用在 `internal/`（排除 draw 包自身）

### H-2. [grep] 精灵 key 映射完备
- `pool.go spriteKeyForStyle()` 对每个 AttackStyle 返回 sprite key
- **验证**：每个返回值在 `assets/towers/` 中有对应目录

### H-3. [trace] 教程步骤触发
- 8 步教程在各种 UI 重构后是否仍能触发
- **验证**：read `internal/core/tutorial/` 确认触发条件

### H-4. [read] 成就触发完备性
- 15 个成就通过 EventBus 触发
- **验证**：对每个成就条件，grep 对应事件是否在正确时机发射

---

## 优先级汇总

| 优先级 | 数量 | 关键项 |
|--------|------|--------|
| **P0** | 2 | F-1 战灵 SetTemp 被清除, F-2 chainActive 标志错误 |
| **P1** | 3 | B-1 scaling 能力无功能, D-1 Abilities 同步脆弱, F-4 金灵五星芒 |
| **P2** | 8 | A-1~A-5 配置一致性, C-3/C-4 悬空指针, E-3 分裂子体 |
| **P3** | 7 | B-2/B-3 死代码, G-2 死字段, H-1~H-4 渲染/交互 |

---

## 执行建议

1. 先修 F-1 + F-2（P0，战灵系统根因 bug）
2. 再修 B-1（5 个无功能能力需要决定：实现 or 从 JSON 移除）
3. 按 P2 逐项审查，每发现问题记录到 `docs/bug/record.md`
4. P3 项作为持续改进在后续 session 处理
