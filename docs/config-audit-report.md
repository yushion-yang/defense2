# 配置审核报告

> 生成日期: 2026-04-01
> 列出游戏所有可调参数的当前值，供确认是否符合设计意图。
>
> - **JSON** = 配置文件中，无需重编译
> - **Go** = 硬编码在源码中，需重编译

---

## 目录

1. [屏幕与引擎常量](#1-屏幕与引擎常量)
2. [对象池大小](#2-对象池大小)
3. [画质预设](#3-画质预设)
4. [难度模式](#4-难度模式)
5. [经济系统](#5-经济系统)
6. [塔定义](#6-塔定义)
7. [能力定义（34种）](#7-能力定义34种)
8. [塔升级系统](#8-塔升级系统)
9. [战力系统](#9-战力系统)
10. [连锁网络](#10-连锁网络)
11. [敌人原型](#11-敌人原型)
12. [Boss 模板](#12-boss-模板)
13. [Buff 模板](#13-buff-模板)
14. [Buff 叠加规则](#14-buff-叠加规则)
15. [波次系统](#15-波次系统)
16. [伤害管线](#16-伤害管线)
17. [控制效果](#17-控制效果)
18. [伤害类型](#18-伤害类型)
19. [战灵定义](#19-战灵定义)
20. [地图列表](#20-地图列表)
21. [波次难度缩放](#21-波次难度缩放)
22. [出怪间隔倍率](#22-出怪间隔倍率)
23. [战斗设置](#23-战斗设置)
24. [攻击方式映射](#24-攻击方式映射)
25. [⚠️ 发现的不一致](#25-发现的不一致)

---

## 1. 屏幕与引擎常量

来源: `internal/core/game/constants.go` (Go)

| 常量 | 值 | 说明 |
|------|-----|------|
| ScreenWidth | 1200 | 逻辑屏幕宽度(px) |
| ScreenHeight | 540 | 逻辑屏幕高度(px) |
| TargetTPS | 60 | 目标帧率 |

## 2. 对象池大小

来源: `internal/core/game/constants.go` (Go)

| 常量 | 值 | 说明 |
|------|-----|------|
| MaxTowers | 64 | 最大塔数 |
| MaxEnemies | 256 | 最大敌人数 |
| MaxProjectiles | 1024 | 最大弹射物数 |

## 3. 画质预设

来源: `internal/core/game/quality.go` (Go)

| 等级 | 后处理 | 最大粒子 | 光源 | 拖尾长度 | TPS |
|------|--------|---------|------|---------|-----|
| 高 | 开 | 2048 | 4 | 6 | 60 |
| 中 | 开 | 1024 | 2 | 3 | 60 |
| 低 | 关 | 512 | 0 | 1 | 30 |

自适应阈值: 慢帧>14ms×30帧→降级, 快帧<10ms×120帧→升级

## 4. 难度模式

来源: `config/settings.json` (JSON)

| 难度 | 标签 | HP倍率 | 速度倍率 | 奖励倍率 | 初始金币 |
|------|------|--------|---------|---------|---------|
| easy | 简单 | 0.7 | 0.85 | 1.3 | 180 |
| normal | 普通 | 1.0 | 1.0 | 1.0 | 120 |
| hard | 困难 | 1.4 | 1.15 | 0.8 | 100 |
| extreme | 极限 | 2.0 | 1.3 | 0.6 | 80 |

## 5. 经济系统

### JSON 设置 (`config/settings.json` > `economy`)

| 参数 | 值 | 说明 |
|------|-----|------|
| buildCost | 50 | 基础建塔费用 |
| baseUpgradeCost | 60 | 基础升级费用 |
| sellRefundRate | 0.7 | 卖塔退款率 |
| startingGold | 120 | 初始金币 |
| startingLives | 20 | 初始生命 |

### Go 硬编码 (`internal/core/economy/economy.go`)

| 参数 | 值 | 说明 |
|------|-----|------|
| KillReward | 15 | 击杀奖金 |
| WaveBonus | 30 | 波次基础奖金 |
| WaveBonusScale | 5 | 每波递增奖金 |
| InterestRate | 0.05 | 利息率(5%) |
| InterestCap | 50 | 利息上限 |
| SellRefundRatio | **0.5** | ⚠️ 卖塔退款率 |

> ⚠️ **JSON=0.7 vs Go=0.5**，需确认运行时用哪个

### 强度购买 (`internal/core/tower/tower.go`)

| 常量 | 值 | 说明 |
|------|-----|------|
| StrengthBuyCost | 10 | 每次花10金买10强度 |

## 6. 塔定义

来源: `config/towers/towers.json` (JSON)

| Key | 标签 | 造价 | 基础伤害 | 潜力伤害 | 基础攻速 | 潜力攻速 | 基础射程 | 潜力射程 | 攻击方式 | 弹速 |
|-----|------|------|---------|---------|---------|---------|---------|---------|---------|------|
| basic | 哨兵 | 50 | 8 | 14 | 0.4 | 0.6 | 150 | 30 | projectile | 400 |
| shotgun | 霰弹 | 60 | 10 | 18 | 0.35 | 0.55 | 90 | 15 | scatter | 350 |
| prism | 棱光 | 70 | 5 | 9 | 0.3 | 0.5 | 190 | 40 | wideBeam | 0 |
| cyclone | 旋刃 | 80 | 7 | 12 | 0.5 | 0.7 | 110 | 20 | spin_aoe | 0 |
| railgun | 穿甲 | 100 | 15 | 25 | 0.15 | 0.2 | 175 | 35 | pierce | 600 |

属性公式: `属性 = 基础 + 潜力 × (强度 / 100)`

升级费用序列: `[80, 120, 180, 260, 400, 600]`（所有塔共用）

## 7. 能力定义（34种）

来源: `config/abilities/abilities.json` (JSON)

### 攻击模式 (attack) — 9种

| 类型 | 标签 | 缩放维度 | 基础 | 潜力 | 参数 | 参数维度 | 描述 |
|------|------|---------|------|------|------|---------|------|
| enhance | 强化 | statBoost | 0.2 | 0.15 | 0 | — | 一次性全面提升基础属性 |
| scatter | 散射 | extraPellets | 0 | 0.5 | 60 | 扇形角度 | 发射3+{s}颗弹丸 |
| wideBeam | 贯穿光束 | beamWidth | 6 | 4 | 3 | 射程倍率 | 宽{s}光束穿透所有敌人 |
| spinAoe | 旋风 | innerBonus | 0.5 | 0.3 | 0.5 | 内圈比例 | 对周围敌人造成伤害 |
| pierce | 穿刺 | maxPierce | 2 | 1 | 0.85 | 伤害衰减 | 穿透最多{s}个敌人 |
| bounce | 弹射 | maxBounces | 2 | 1 | 0.8 | 伤害衰减 | 弹射最多{s}个敌人 |
| splash | 溅射 | ratio | 0.2 | 0.2 | 50 | 范围px | 命中时对周围{s%}溅射伤害 |
| multiTarget | 多目标 | targets | 1 | 1 | 0 | — | 同时攻击{s}个目标 |
| radial | 环射 | extraShots | 0 | 0.5 | 1.2 | 射程倍率 | 向四周发射3+{s}颗穿刺弹 |

### 控制效果 (cc) — 4种

| 类型 | 标签 | 缩放维度 | 基础 | 潜力 | 参数 | 参数维度 |
|------|------|---------|------|------|------|---------|
| slowPower | 凝滞 | factor | 0.15 | 0.25 | 1.0 | 持续时间 |
| slowDuration | 冰封 | duration | 1.5 | 1.5 | 0.15 | 减速强度 |
| stunChance | 震慑 | chance | 0.08 | 0.12 | 0.3 | 持续时间 |
| stunDuration | 麻痹 | duration | 0.3 | 0.4 | 0.06 | 概率 |

### 伤害加成 (damage) — 6种

| 类型 | 标签 | 缩放维度 | 基础 | 潜力 | 参数 | 参数维度 |
|------|------|---------|------|------|------|---------|
| crit | 暴击 | chance | 0.1 | 0.15 | 1.8 | 暴击倍率 |
| deathMark | 死亡爆破 | explosionDamage | 10 | 15 | 50 | 爆炸半径 |
| distanceDamage | 距离伤害 | maxBonus | 0.2 | 0.3 | 0 | — |
| executionBonus | 斩杀 | damageBonus | 0.2 | 0.3 | 0.5 | HP阈值 |
| flatDamage | 固伤 | damage | 2 | 3 | 0 | — |
| momentum | 蓄势 | bonusRatio | 0.15 | 0.25 | 0 | — |

### 增益光环 (buff) — 6种

| 类型 | 标签 | 缩放维度 | 基础 | 潜力 | 参数 | 参数维度 |
|------|------|---------|------|------|------|---------|
| damageUpAura | 伤害光环 | bonus | 0.05 | 0.1 | 150 | 范围px |
| attackSpeedAura | 攻速光环 | bonus | 0.04 | 0.06 | 150 | 范围px |
| rangeAura | 射程光环 | bonus | 8 | 12 | 150 | 范围px |
| critAura | 暴击光环 | bonus | 0.04 | 0.06 | 150 | 范围px |
| soloBoost | 独行加成 | bonus | 0.1 | 0.2 | 120 | 检测范围 |
| goldPassive | 被动产金 | amount | 1 | 1 | 3 | 间隔秒 |

### 持续伤害 (dot) — 4种

| 类型 | 标签 | 缩放维度 | 基础 | 潜力 | 参数 | 参数维度 |
|------|------|---------|------|------|------|---------|
| burn | 灼烧 | ratio | 0.15 | 0.15 | 2.0 | 持续秒 |
| bleedDot | 流血 | hpPercent | 0.01 | 0 | 3.0 | 持续秒 |
| poison | 中毒 | dps | 3 | 5 | 4.0 | 持续秒 |
| weaken | 虚弱 | amplify | 0.1 | 0.15 | 3.0 | 持续秒 |

### 区域效果 (zone) — 4种

| 类型 | 标签 | 缩放维度 | 基础 | 潜力 | 参数 |
|------|------|---------|------|------|------|
| poisonZone | 毒区 | dps | 1 | 2 | 0 |
| silenceZone | 沉默区 | slowFactor | 0.08 | 0.12 | 0 |
| curseZone | 诅咒区 | hpPercentPerSec | 0.005 | 0.01 | 0 |
| weakenZone | 脆弱区 | amplify | 0.05 | 0.1 | 0 |

## 8. 塔升级系统

来源: `internal/core/tower/upgrade.go` (Go)

| 常量 | 值 | 说明 |
|------|-----|------|
| MaxAbilitySlots | 6 | 最大能力槽（每类1个） |
| WavesPerUnlock | 2 | 每2波解锁1个槽 |
| ChoicesPerUnlock | 3 | 每次解锁3选1 |

解锁公式: `已解锁槽数 = 1 + wavesCleared / 2`（上限6）

## 9. 战力系统

来源: `internal/core/strength/strength.go` (Go)

| 参数 | 值 | 说明 |
|------|-----|------|
| Base | 100 | 初始强度 |

公式: `有效值 = max(0, (Base + 永久 + 临时总和) × 敌人乘法debuff - 敌人减法debuff)`
缩放比: `有效值 / 100.0`

## 10. 连锁网络

来源: `internal/core/strength/chain.go` (Go)

| 常量 | 值 | 说明 |
|------|-----|------|
| ChainDistance | 150px | 连锁距离 |
| ChainStrengthPerTower | 10 | 每塔加成强度 |

连锁加成: 组内塔数 × 10 (2+塔才生效)

## 11. 敌人原型

来源: `config/enemies/enemies-core.json` (JSON)

| 原型 | 标签 | HP倍率 | 速度倍率 | 奖励倍率 | 碰撞半径 | 特殊字段 |
|------|------|--------|---------|---------|---------|---------|
| normal | 步兵 | 1.18 | 1.04 | 0.82 | 18 | — |
| runner | 疾行 | 0.74 | 1.84 | 0.72 | 16 | — |
| tank | 重甲 | 2.85 | 0.78 | 1.35 | 24 | — |
| armored | 护甲兵 | 1.4 | 0.85 | 1.1 | 20 | — |
| shielded | 护盾兵 | 1.28 | 1.02 | 0.94 | 18 | ⚠️ shieldScale=0 |
| swarm | 虫群 | 0.5 | 2.08 | 0.6 | 12 | — |
| stealth | 隐身兵 | 0.92 | 1.32 | 1.1 | 16 | stealthDuration=3 |
| splitter | 分裂体 | 1.6 | 0.86 | 0.7 | 20 | splitCount=2 |
| teleporter | 传送兵 | 0.8 | 1.0 | 1.2 | 16 | teleportInterval=5, skip=1 |
| healer | 治疗兵 | 1.06 | 1.14 | 1.02 | 20 | healScale=0.24, radius=105, interval=2.5 |
| buffer | 旗手 | 0.68 | 1.08 | 1.3 | 16 | auraRange=100, speedUp=0.2, armor=10 |
| flying | 飞行斥候 | 0.9 | 1.1 | 0.85 | 16 | movementType="flying" |
| dummy | 木桩 | 10000 | 0 | 0 | 24 | 测试用 |

> ⚠️ **shielded 的 shieldScale=0**，护盾永远不生效

基础属性公式 (Go): `baseHP = (10 + wave×5) × 难度HP倍率`, `baseSpeed = (50 + wave×3) × 难度速度倍率`

## 12. Boss 模板

来源: `config/enemies/boss-templates.json` (JSON)

| 模板 | 参数 |
|------|------|
| bossPhase | 阈值=[0.75, 0.5, 0.25], 效果=加速/加甲/狂暴 |
| bossTeleport | 间隔=15s, 随机位置 |
| bossSpawnMinions | 间隔=20s, 数量=5, 子怪=normal |
| bossReflect | 反伤比例=0.25 |
| bossRotateWeakness | 切换间隔=10s, 元素=[火/冰/毒/物理] |
| bossGoldSteal | 每次命中偷1金 |
| bossAura | 范围=120, 加速=0.3, 加甲=10 |

Boss 生成参数 (Go): HP倍率=`8+wave`倍, 体型=1.5倍, 每5波1个Boss

## 13. Buff 模板

来源: `internal/core/enemy/buff_templates.go` (Go)

| 模板 | 类别 | 关键参数 |
|------|------|---------|
| berserk | 攻击 | 50%HP时触发, 速度×1.5 |
| regen | 防御 | 每秒回2%最大HP |
| healAura | 辅助 | 治疗量=10, 范围=80, 间隔=2s |
| speedAura | 辅助 | 加速+20% |
| damageReduce | 防御 | 减伤30% |
| empBurst | 攻击 | (无参数) |
| blink | 辅助 | (无参数) |
| deathSplit | 死亡 | 分裂数=2 |
| deathSlow | 死亡 | 减速50%, 范围=60, 持续3s |
| reflect | 防御 | 反伤15% |
| timewarp | 辅助 | (无参数) |
| revive | 死亡 | 复活后50%HP |
| spawnMinions | 攻击 | 召唤3个normal |

## 14. Buff 叠加规则

来源: `internal/core/buff/stack_rules.go` (Go)

| Buff类型 | 叠加模式 | 上限 | 下限 | 优先级 |
|---------|---------|------|------|--------|
| slow | 取最强 | 0.8 | — | — |
| stun | 覆盖 | — | — | — |
| knockup | 覆盖 | — | — | — |
| root | 覆盖 | — | — | — |
| silence | 覆盖 | — | — | — |
| disarm | 覆盖 | — | — | — |
| speedUp | 累加 | 1.4 | — | — |
| damageUp | 乘法 | — | — | — |
| damageDown | 乘法 | — | 0.2 | — |
| fireRateUp | 累加 | 0.5 | — | — |
| invincible | 覆盖 | — | — | 99 |
| damageImmune | 覆盖 | — | — | 90 |
| controlImmune | 覆盖 | — | — | 80 |
| slowImmune | 覆盖 | — | — | 70 |
| stunImmune | 覆盖 | — | — | 70 |
| rootImmune | 覆盖 | — | — | 70 |
| untargetable | 覆盖 | — | — | 100 |
| shield | 独立 | — | — | — |
| dot | 按来源独立 | — | — | — |
| tenacity | 乘法 | — | — | — |

## 15. 波次系统

### 出怪器默认值 (Go)

| 参数 | 值 | 说明 |
|------|-----|------|
| EnemiesPerWave | 5 | 每波基础数量 |
| SpawnInterval | 0.6s | 同波内出怪间隔 |
| WaveInterval | 10s | 波间间隔 |
| FirstWaveInterval | 20s | 首波等待 |
| Boss 间隔 | 每5波 | `wave%5==0` |

每波数量公式: `5 + wave`（⚠️ JSON 写的是 `6 + 2×wave`）

### 波次组合权重 (Go)

| 阶段 | 波次 | 原型(权重) |
|------|------|-----------|
| 1 | 1-3 | normal(100) |
| 2 | 4-6 | normal(70), runner(20), swarm(10) |
| 3 | 7-9 | normal(50), runner(20), tank(15), armored(10), shielded(5) |
| 4 | 10-14 | normal(40), runner(15), tank(15), armored(10), flying(10), healer(5), stealth(5) |
| 5 | 15+ | normal(30), runner(10), tank(15), armored(10), flying(10), healer(5), stealth(5), splitter(5), buffer(5), teleporter(5) |

### 波次 Buff 注入 (Go)

| 波次范围 | 最大Buff数 | Buff池 |
|---------|-----------|--------|
| 1-5 | 0 | (无) |
| 6-15 | 1 | berserk, regen, healAura, speedAura |
| 16-25 | 2 | + reflect, damageReduce |
| 26+ | 2 | + revive, deathSplit |

注入概率: 30%每个非Boss非Elite敌人

## 16. 伤害管线

来源: `internal/core/combat/damage_pipeline.go` (Go)

| 常量 | 值 | 说明 |
|------|-----|------|
| MaxDamageAmplify | 0.5 | 虚弱增伤上限(+50%) |
| Boss %HP 上限 | 0.05 | 单次最多扣5%最大HP |
| 最低伤害 | 1 | 原始伤害>0时保底1点 |

7步管线:
1. 免疫检查 (pure穿透所有)
2. Boss %HP 上限
3. 攻击者增伤 (true/pure跳过)
4. 目标减伤 (true/pure跳过) → 虚弱增伤(上限50%) → 伤害上限(沉默时失效)
5. HP扣减
6. 阈值触发
7. 死亡检查

## 17. 控制效果

来源: `internal/core/combat/crowd_control.go` (Go)

| 常量 | 值 | 说明 |
|------|-----|------|
| MinSpeedRatio | 0.2 | 最低速度=基础×20% |

| CC类型 | 叠加方式 | 韧性影响 | 免疫检查 |
|--------|---------|---------|---------|
| 眩晕 | 刷新(取长) | 持续×(1-韧性) | controlImmune, stunImmune |
| 减速 | 取最强 | 最低 MinSpeedRatio | controlImmune, slowImmune |
| 定身 | 刷新(取长) | 持续×(1-韧性) | controlImmune, rootImmune |

## 18. 伤害类型

来源: `internal/core/combat/damage_type.go` (Go)

| 类型 | 无视减免 | 无视无敌 | 颜色 |
|------|---------|---------|------|
| physical | 否 | 否 | 红 |
| magic | 否 | 否 | 紫 |
| true | 是 | 否 | 金 |
| pure | 是 | 是 | 玫红 |

## 19. 战灵定义

来源: `config/wardens/wardens.json` (JSON)

| Key | 名称 | 类别 | 伤害 | 攻击间隔 | 射程 | 移速 | 击杀成长 | 波次成长 |
|-----|------|------|------|---------|------|------|---------|---------|
| prince | 火灵 | 近战 | 12 | 1.2 | 140 | 350 | 1 | 5 |
| core | 机甲 | 近战 | 20 | 1.2 | 160 | 360 | 1 | 10 |
| chain | 聚能 | 间接 | 12 | 1.2 | 150 | 300 | 0 | 8 |
| skystrike | 水灵 | 间接 | 10 | 1.5 | 140 | 320 | 0 | 10 |
| envoy | 金灵 | 近战 | 12 | 1.2 | 140 | 320 | 0 | 5 |

## 20. 地图列表

来源: `config/level-list.json` (JSON)

| ID | 名称 | 描述 | 波数 | 难度 |
|----|------|------|------|------|
| map_01 | 蜿蜒峡谷 | S形路径，新手友好 | 12 | easy |
| map_02 | 交叉路口 | 双入口交汇 | 15 | normal |
| map_03 | 铁壁防线 | 短路径紧凑布局 | 18 | hard |
| map_04 | 螺旋要塞 | 外到内螺旋 | 15 | normal |
| map_05 | 双线战场 | 两条平行路径 | 18 | hard |
| map_06 | 迷宫回廊 | 多折路径 | 20 | hard |
| map_07 | 极限窄道 | 极少塔位 | 22 | extreme |
| map_08 | 竞技场 | 四面出怪 | 25 | extreme |

## 21. 波次难度缩放

来源: `config/settings.json` > `waves.difficulty` (JSON)

| 参数 | 值 | 说明 |
|------|-----|------|
| hpBase | 52 | 起始HP基数 |
| hpPerWave | 21 | 每波HP增量 |
| speedBase | 58 | 起始速度基数 |
| speedPerWave | 5 | 每波速度增量 |
| rewardBase | 9 | 起始奖励基数 |
| rewardPerWave | 1 | 每波奖励增量 |
| lateWaveStart | 5 | 后期开始波次 |
| lateHpBonusPerWave | 0.125 | 后期每波额外HP缩放 |
| lateSpeedBonusPerWave | 0.055 | 后期每波额外速度缩放 |

> ⚠️ **JSON 公式(hpBase=52, hpPerWave=21) vs Go 公式(10+wave×5)**，两套系统不一致

### 出怪计数 (JSON)

| 参数 | 值 |
|------|-----|
| baseTotalFormula.base | 6 |
| baseTotalFormula.perWave | 2 |
| eliteInterval | 4 |
| spawnBaseInterval | 0.92s |
| spawnMinInterval | 0.18s |
| spawnDecayPerWave | 0.03 |

> ⚠️ **JSON `6+2×wave` vs Go `5+wave`**

## 22. 出怪间隔倍率

来源: `config/settings.json` > `waves.spawnMultipliers` (JSON)

| 原型 | 倍率 | 说明 |
|------|------|------|
| swarm | 0.6 | 最快 |
| runner | 0.74 | 快 |
| normal | 1.0 | 基准 |
| tank | 1.1 | 慢 |
| boss | 1.55 | 最慢 |

（其余原型省略，完整见 settings.json）

## 23. 战斗设置

来源: `config/settings.json` > `combat` (JSON)

| 参数 | 值 | 说明 |
|------|-----|------|
| bossDamageCapRatio | 0.05 | Boss %HP伤害上限(5%) |
| fireRateFloor | 0.18 | 最低射速 |
| slowCap | 0.30 | 最大减速效果(30%) |
| minCCDuration | 0.1 | 最小CC持续时间 |
| armorDivisor | 100 | 护甲除数 |
| maxDamageAmplification | 3 | 最大增伤倍率 |
| dotTickInterval.burn | 0.5s | 灼烧tick |
| dotTickInterval.bleed | 0.5s | 流血tick |
| dotTickInterval.poison | 1.0s | ⚠️ 中毒tick |

> ⚠️ **JSON poison=1.0s vs Go 统一 DotTickInterval=0.5s**
> ⚠️ **JSON maxDamageAmplification=3 vs Go MaxDamageAmplify=0.5**

### 护甲系统

| 参数 | 值 |
|------|-----|
| maxArmor | 50 |
| minArmor | -100 |
| positiveReductionPerPoint | 0.01 (每点减1%) |
| negativeAmplifyPerPoint | 0.01 (每点增1%) |

## 24. 攻击方式映射

来源: `internal/core/tower/tower.go` (Go)

| 能力 | 精灵Key | 标签 |
|------|---------|------|
| enhance | fortress | 堡垒 |
| scatter | shotgun | 霰弹 |
| wideBeam | prism | 棱光 |
| spinAoe | cyclone | 旋刃 |
| bounce | ricochet | 链弹 |
| splash | mortar | 轰炸 |
| multiTarget | hydra | 多管 |
| radial | nova | 星爆 |
| (默认) | sentinel | 哨兵 |

废弃映射: laser→projectile, charge→projectile, aura_dot→spin_aoe

---

## 25. 不一致处理结果（2026-04-01 审核后修复）

| # | 问题 | 处理结果 |
|---|------|---------|
| 1 | **卖塔退款率** JSON=0.7 vs Go=0.5 | ✅ Go 改为 0.7，删除利息系统 |
| 2 | **虚弱增伤上限** JSON=3 vs Go=0.5 | ✅ 以 Go=0.5 为准（JSON 值未被加载） |
| 3 | **中毒tick间隔** JSON=1.0s vs Go=0.5s | ✅ 以 Go=0.5s 为准（统一） |
| 4 | **每波敌人数** JSON=6+2×wave vs Go=5+wave | ✅ 以 Go=5+wave 为准 |
| 5 | **护盾兵** shieldScale=0 | ✅ 护盾机制已删除 |
| 6 | **减速三层值** | ✅ 实际由 Go MinSpeedRatio=0.2 控制（最终下限） |
| 7 | **波次HP公式** JSON vs Go | ✅ Go 改为 JSON 公式: HP=52+21×wave, Speed=58+5×wave |

### 同步删除的系统
- 利息系统（InterestRate/InterestCap）
- 护甲系统
- 护盾机制
- 定身(root) CC 类型
- 魔法/真实/纯粹伤害类型（保留 physical 唯一类型）
- 无参数 buff 模板（empBurst/blink/timewarp）
