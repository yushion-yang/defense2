# AutoPlay 覆盖率分析与优化方案

> 基于 sweep 68 局全量运行结果的深度分析。

---

## 一、当前 sweep 逻辑

`--sweep` 调用 `GenerateTestPlan()` 生成 68 个用例，分 7 类：

| # | 类别 | 数量 | 策略 | 目的 |
|---|------|------|------|------|
| 1 | Pairwise | 32 | random/greedy 轮换 | 8 地图 × 4 难度全覆盖 |
| 2 | Focus Tower | 8 | focus (单塔) | 每种塔单独极限测试 |
| 3 | Enemy Specific | 12 | greedy + EnemyFilter | 每种原型单独测试 |
| 4 | Game Mode | 6 | greedy | 6 种模式各 1 局 |
| 5 | Scenario | 4 | 脚本化 | FSM 边界（建塔流/生命周期/零金/快速操作） |
| 6 | Visual Catalog | 2 | visual_catalog | 每种塔建一座截图 |
| 7 | Edge Cases | 4 | 混合 | 极限场景 |

---

## 二、问题诊断

### 2.1 严重覆盖缺口

68 局跑完后，**以下维度覆盖率为 0%**：

| 维度 | 应覆盖 | 实际覆盖 | 根因 |
|------|--------|----------|------|
| 塔类型 | 8 | 0（只有 basic） | greedy 策略只选最便宜的 basic；focus 策略建了其他塔但 `tower_usage` 统计可能是聚合时取最后一局快照 |
| 攻击方式 | 7 | 1（projectile） | 同上，basic 只有 projectile |
| 能力 | 26 | 0 | 能力需要升级解锁 + 战斗触发 + 遥测记录 |
| 伤害管线 | 8 步 | 0 | **`ProcessDamage()` 未被任何战斗代码调用**（已写好但未接入） |
| Buff 类型 | 19 | 0 | 需要敌人模板触发 + 塔能力触发 |
| CC 类型 | 3 | 0 | 需要 freeze/stun 塔的能力触发 |
| 交互模式 | 8 | 1（idle） | autoplay 直接调 API 建塔，不经过 HUD 交互层 |

### 2.2 冗余用例

| 冗余类型 | 数量 | 说明 |
|---------|------|------|
| Pairwise 32 局 | ~28 局冗余 | 32 局全用 basic 塔，策略行为几乎相同，只验证了"不同地图能通关" |
| 难度 × 地图全组合 | 32 局 | 4 难度只影响 HP/速度/奖金倍率，不需要对每张地图都测 4 种难度 |
| random + greedy 轮换 | 16 局 random | random 策略建塔概率极低(2%/帧)，多数局几乎不建塔 |

### 2.3 缺失的测试场景

| 缺失场景 | 影响 |
|---------|------|
| 每种塔 + 每种攻击方式的战斗验证 | 攻击效果/弹道/伤害完全未测 |
| 能力触发验证（bounce/burn/stun 等） | 26 种能力全未触发 |
| 战灵类型单独验证（5 种） | 只测了 prince，其他 4 种战灵从未单独验证 |
| 升级流程验证（强度递增 + 能力解锁） | scenario 只升了 1 次级 |
| 技能释放验证 | 技能系统完全未测 |
| 暂停/恢复 | 从未进入暂停模式 |

---

## 三、优化方案：内容导向测试

### 设计原则

1. **每项游戏内容有且只有 1 个专项用例**
2. **关联内容只需 1 个验证用例**（如 bounce 能力 OK → 其他弹射物能力同理只验证 1 个关联场景）
3. **去掉纯组合爆炸**（不需要 8 地图 × 4 难度 = 32 局）

### 3.1 新的测试清单

#### A. 塔 × 攻击方式（8 局）— 每种塔 1 局

每局用 focus 策略只建一种塔，跑足够波次让塔升级 + 触发能力。

| 用例 ID | 塔 | 攻击方式 | 关联验证 |
|---------|-----|---------|---------|
| `tower_basic` | basic | projectile | 弹射物追踪、伤害浮字 |
| `tower_laser` | laser | laser | 光束渲染、持续伤害 |
| `tower_freeze` | freeze | wideBeam | 减速效果、CC 触发 |
| `tower_electric` | electric | scatter | 散弹数量、多目标命中 |
| `tower_hunter` | hunter | charge | 蓄能动画、高伤单发 |
| `tower_en-04` | en-04 | spin_aoe | 范围伤害、旋转特效 |
| `tower_en-05` | en-05 | aura_dot | DOT 持续伤害 |
| `tower_en-08` | en-08 | pierce | 直线穿透 |

**验证点**: 塔 sprite、攻击动画、弹射物/光束渲染、伤害浮字、能力触发遥测

#### B. 能力验证（按类型分组，~6 局）

将 26 种能力按机制分组，每组 1 个代表性验证：

| 用例 ID | 代表能力 | 同组能力（关联验证） | 验证方式 |
|---------|---------|-------------------|---------|
| `abil_damage_mod` | splash | flatDamage, stackDamage, percentHpDamage, percentHpMinor, deathMark, executionBonus, distanceDamage | 遥测 splash 计数 > 0 |
| `abil_projectile` | bounce | chargeShot, multiTarget | 遥测 bounce 计数 > 0 + 次目标受伤 |
| `abil_cc` | stun | onHitSlow, burn, bleedDot | CC 遥测 > 0 |
| `abil_crit` | crit | critAura | 遥测 crit 计数 > 0 |
| `abil_aura` | damageUpAura | attackSpeedAura, rangeAura, soloBoost | 相邻塔属性提升 |
| `abil_zone` | curseZone | poisonZone, silenceZone | 区域 DOT 遥测 > 0 |
| `abil_economy` | goldPassive | goldOnKill | 金币增长高于基线 |

**实现方式**: scenario 策略 — 建指定塔 → 升级到解锁能力 → 等待战斗触发 → 检查遥测

#### C. 敌人原型（精简为 5 局）

12 种原型按行为分组，每组 1 个代表 + 1 个关联验证：

| 用例 ID | 代表原型 | 同组（关联） | 验证点 |
|---------|---------|------------|--------|
| `enemy_basic_group` | normal | runner, swarm | 路径移动、速度差异 |
| `enemy_tank_group` | tank | armored, shielded | HP 缩放、护甲/护盾 |
| `enemy_special` | splitter | teleporter | 分裂/传送特殊行为 |
| `enemy_support` | healer | buffer | 治疗/增益光环 |
| `enemy_evasion` | stealth | flying | 隐身/飞行路径 |

#### D. 战灵（5 局）— 每种 1 局

| 用例 ID | 战灵 | 验证点 |
|---------|------|--------|
| `warden_prince` | prince | 近战跳跃、击杀计数 |
| `warden_core` | core | AoE 攻击 |
| `warden_chain` | chain | 连锁攻击、ChainLinks 渲染 |
| `warden_skystrike` | skystrike | 远程打击 |
| `warden_envoy` | envoy | 辅助增益 |

#### E. 游戏模式（6 局）

保持现有，每种模式 1 局。但修复 bossRush 后应能通过。

| 用例 ID | 模式 | 验证点 |
|---------|------|--------|
| `mode_campaign` | campaign | 通关 + 败北判定 |
| `mode_endless` | endless | 持续到崩盘 |
| `mode_timed` | timed | 时间耗尽 |
| `mode_bossRush` | bossRush | 每波 Boss + 胜利判定 |
| `mode_challenge` | challenge | 挑战条件 |
| `mode_test` | test | 调试工具 |

#### F. 地图验证（8 局）— 每张地图 1 局

不再 × 4 难度。每张地图 normal 难度 greedy 跑一局，验证路径/布局/多路径。

| 用例 ID | 地图 | 特征 |
|---------|------|------|
| `map_01` ~ `map_08` | 各 1 局 | 单路径/双入口/双线/四面/大地图 等 |

#### G. 难度缩放（4 局）— 固定地图 map_01，只变难度

| 用例 ID | 难度 | 验证点 |
|---------|------|--------|
| `diff_easy` | easy | HP×0.7, 奖金×1.3 |
| `diff_normal` | normal | 基准 |
| `diff_hard` | hard | HP×1.4, 奖金×0.8 |
| `diff_extreme` | extreme | HP×2.0, 奖金×0.6 |

#### H. 交互场景 + 边界（保持 ~8 局）

现有 4 scenario + 4 edge case，保留。

### 3.2 总用例数对比

| 方案 | 用例数 | 覆盖 |
|------|--------|------|
| 旧 sweep | 68 | 塔 0/8, 能力 0/26, 攻击 1/7 |
| **新方案** | **~50** | 塔 8/8, 能力 26/26(分组), 攻击 7/7, 战灵 5/5, 模式 6/6, 地图 8/8, 难度 4/4 |

减少 ~18 个冗余用例，同时填补所有覆盖缺口。

---

## 四、阻塞问题（需先修复才能覆盖）

| 问题 | 影响 | 优先级 |
|------|------|--------|
| `ProcessDamage()` 未被战斗主流程调用 | pipeline/damageType/buff/CC 遥测全为 0 | **P0** |
| greedy 策略只选 basic | 非 focus 用例只测到 1 种塔 | P1（用新方案解决） |
| 能力解锁需要 wavesCleared | autoplay 波次快，能力可能来不及解锁 | P1 |
| 战灵选择在 modeWardenSelect 中 | autoplay 直接调 `activateWarden` 跳过了 UI 流 | P2（交互层无法 autoplay 测） |

---

## 五、落地步骤

1. **先修 P0**: 接入 `ProcessDamage` 到战斗主流程（所有塔命中路径 → 统一走管线）
2. **重写 `GenerateTestPlan()`**: 按本文档 §3.1 的清单实现
3. **补能力验证 scenario**: 指定塔 + 强制升级 + 等待触发
4. **跑新 sweep**: 验证覆盖率从 0% 提升到 80%+
