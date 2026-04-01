# TODO — 待办事项

> 基于 `docs/design/` 设计文档提取。按优先级排列。

---

## P0 — 核心遗漏（直接影响玩法多样性）

- [x] **Healer 配置接线**：SpawnConfig 传递 healScale/healRadius/healInterval → Pool.Spawn 初始化 → 敌人行为 tick 接入 updatePlaying
- [x] **敌人行为系统接入**：UpdateBerserk/UpdateHealing/UpdateRegeneration 从死代码变为每帧调用
- ~~**Stealth 隐身系统**~~：已删除
- ~~**Shield 护盾系统**~~：已删除
- [x] **Splitter 死亡分裂**：Pool.Kill 时检查 SplitCount → SpawnSplitChildren 生成子体（30% HP, 1.4x 速度, 不再分裂）

## P1 — 差异化增强

- [x] **Teleporter 传送**：UpdateTeleport 定时跳过路径段，SpawnConfig 传递 interval + skip
- [x] **Buffer 旗手光环**：UpdateBufferAura 范围加速友军，SpawnConfig 传递 auraRange + auraSpeedUp
- [x] **波次 Buff 自动注入**：Spawner.applyWaveBuffs 按波次段（6/16/26+）30% 概率注入 1-2 个 buff
- [x] **speedAura 模板接线**：ApplyBuffTemplate 映射 SpeedAuraFactor → Enemy.AuraSpeedUp
- [x] **damageReduce 模板接线**：ApplyBuffTemplate 映射 DamageReduce → Enemy.DamageReduceRatio → 伤害管线步骤 4.1

## P2 — Boss 行为系统

- [ ] **Boss 模板加载器**：解析 `config/enemies/boss-templates.json`，运行时按 Boss 指定应用模板
- [ ] **bossPhase 阶段转换**：HP 阈值（75%/50%/25%）切换行为/加速/召唤
- [ ] **bossSpawnMinions 召唤**：定时生成小怪波（interval + count）
- [ ] **bossReflect 反伤**：反弹 25% 受到的伤害给攻击者（需塔可受伤或虚拟反馈）
- [ ] **bossRotateWeakness 循环弱点**：每 10s 切换弱点属性（physical/magic），非弱点伤害减半
- [ ] **bossGoldSteal 偷金**：每次被命中偷取玩家 1 金币
- [ ] **bossAura Boss 光环**：友军 +30% 速度 +10 护甲

## P3 — 高级 Buff 模板

- [ ] **reflect 反伤管线**：敌人受伤时反弹 15% 伤害（需确定反伤目标机制）
- [ ] **revive 复活**：死亡后原地复活（HP 50%），仅触发一次
- [ ] **blink 闪现**：定时向前闪现一段距离（跳过路径段）
- [ ] **deathSplit 死亡分裂**：通用死亡分裂逻辑（与 Splitter 原型共用）
- [ ] **deathSlow 死亡减速区**：死亡时留下减速区域（factor 0.5, radius 60, duration 3s）
- [ ] **spawnMinions 召唤小兵**：定时召唤 normal 类型小怪
- [ ] **empBurst 电磁脉冲**：范围内炮塔短暂禁用（需设计塔禁用机制）
- [ ] **timewarp 时间扭曲**：范围内炮塔攻速降低（需设计负面光环机制）

## P4 — 系统改进

- [ ] **BuffList 接入 Enemy**：将敌人状态效果从直接字段迁移到 BuffList 系统（统一堆叠规则/回调机制），与塔使用相同的 buff 基础设施
- [ ] **buff 模板全部接线**：确保所有 13 个 buff 模板在 ApplyBuffTemplate 中正确映射到 Enemy 字段或 BuffList
