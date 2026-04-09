# TODO — 待办事项

> 基于 `docs/design/` 设计文档提取。按优先级排列。
> 截至 2026-04-09 大部分已完成，仅剩少量暂缓项。

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

- [x] **Boss 行为系统**：BossState 挂载在 Enemy.BossData，FullBossState(wave) 按波次递进启用行为
- [x] **bossPhase 阶段转换**：HP 阈值（75%/50%/25%）每阶段加速 10%
- [x] **bossSpawnMinions 召唤**：10 波后启用，20s 间隔召唤 3+wave/10 个 normal 小怪
- [x] **bossReflect 反伤**：20 波后启用，CalcBossReflectDamage 返回 15% 反伤值
- [ ] **bossRotateWeakness 循环弱点**：需伤害类型系统支持，暂缓
- [x] **bossGoldSteal 偷金**：30 波后启用，BossGoldSteal 返回偷金数
- [x] **bossAura Boss 光环**：TickBossAura 120px 范围 +30% 速度

## P3 — 高级 Buff 模板

- [x] **reflect 反伤**：ApplyBuffTemplate 映射 ReflectPercent → Enemy.ReflectPercent（伤害管线需读取此值，由调用方处理）
- [x] **revive 复活**：Pool.Kill 检查 ReviveHPPercent，首次死亡时恢复 HP 并跳过死亡流程
- [ ] **blink 闪现**：与 Teleporter 机制类似，可复用 UpdateTeleport（间隔更短/距离更远），暂缓
- [x] **deathSplit 死亡分裂**：ApplyBuffTemplate 映射 DeathSplitCount → Enemy.SplitCount（P0 已实现）
- [ ] **deathSlow 死亡减速区**：需区域效果系统支持，暂缓
- [ ] **spawnMinions 召唤小兵**：Boss 版已实现（P2），普通敌人版可复用 BossSpawnMinions 机制，暂缓
- [ ] **empBurst 电磁脉冲**：需塔禁用机制，暂缓
- [ ] **timewarp 时间扭曲**：需塔负面光环机制，暂缓

## P4 — 系统改进

- [ ] **BuffList 接入 Enemy**：将敌人状态效果从直接字段迁移到 BuffList 系统（统一堆叠规则/回调机制），暂缓
- [x] **buff 模板接线**：berserk/regen/healAura/speedAura/damageReduce/reflect/revive/deathSplit 已接线（8/13），剩余 5 个需前置系统支持
