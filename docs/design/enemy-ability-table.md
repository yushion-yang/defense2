# 怪物能力配置表

## 1. 基础原型 (enemies-core.json)

| 原型 | 中文名 | HP倍率 | 速度倍率 | 奖励倍率 | 半径 | 移动 | 特殊能力 |
|------|--------|--------|---------|---------|------|------|---------|
| normal | 步兵 | 1.18 | 1.04 | 0.82 | 18 | 地面 | 无 |
| runner | 疾行 | 0.74 | 1.84 | 0.72 | 16 | 地面 | 无（纯高速） |
| tank | 重甲 | 2.85 | 0.78 | 1.35 | 24 | 地面 | 无（纯高HP） |
| armored | 护甲兵 | 1.40 | 0.85 | 1.10 | 20 | 地面 | 无（高HP低速） |
| shielded | 护盾兵 | 1.28 | 1.02 | 0.94 | 18 | 地面 | 护盾(82%MaxHP) |
| swarm | 虫群 | 0.50 | 2.08 | 0.60 | 12 | 地面 | 无（最快最脆） |
| stealth | 隐身兵 | 0.92 | 1.32 | 1.10 | 16 | 地面 | 出场隐身 3s |
| splitter | 分裂体 | 1.60 | 0.86 | 0.70 | 20 | 地面 | 死亡分裂 2 子体 |
| teleporter | 传送兵 | 0.80 | 1.00 | 1.20 | 16 | 地面 | 每 5s 传送跳 1 段路径 |
| healer | 治疗兵 | 1.06 | 1.14 | 1.02 | 20 | 地面 | 每 2.5s 治疗半径 105 内友方 24%HP |
| buffer | 旗手 | 0.68 | 1.08 | 1.30 | 16 | 地面 | 光环 R100: +20%速 +10甲 |
| flying | 飞行斥候 | 0.90 | 1.10 | 0.85 | 16 | 飞行 | 无（基础飞行） |
| dummy | 木桩 | 10000 | 0 | 0 | 24 | 地面 | 不移动（测试用） |

### 原型属性公式

```
实际HP    = baseHP × hpScale     (baseHP = 52 + wave × 21)
实际速度  = baseSpeed × speedScale (baseSpeed = 58 + wave × 5)
击杀奖励  = baseReward × rewardScale
```

### 特殊能力参数

| 能力 | 参数 | 说明 |
|------|------|------|
| 隐身 | duration=3s | 出场隐身,不可被选中目标,溅射/AoE可强制显形(HitFlash>0) |
| 分裂 | count=2, hpRatio=30%, speedScale=×1.4, radius=×0.7 | 子体不会再分裂 |
| 传送 | interval=5s, skip=1段 | 冷却从满 interval 开始,跳过 pathOrder 中的段数 |
| 治疗 | scale=24%HP, radius=105px, interval=2.5s | 治疗量=自身HP×24% |
| 旗手光环 | range=100px, speedUp=+20% | 每帧对范围内友方设置 SpeedBuff |
| 护盾 | shieldScale=82%MaxHP | 伤害优先扣护盾 |

---

## 2. Buff 模板 (波次注入)

游戏运行时会随机给普通怪注入 buff，Boss/Elite 不受注入。

| 模板ID | 名称 | 类别 | 效果 | 注入波段 |
|--------|------|------|------|---------|
| berserk | 狂暴 | offense | HP≤50%时速度×1.5 | 6+ |
| regen | 回血 | defense | 每秒回复 2%MaxHP | 6+ |
| healAura | 治疗光环 | utility | 每 2s 治疗 R80 内友方 10HP | 6+ |
| speedAura | 速度光环 | utility | R80 内友方 +20%移速 | 6+ |
| damageReduce | 减伤 | defense | 受伤 -30% | 16+ |
| reflect | 反伤 | defense | 反弹 15%伤害 | 16+ |
| revive | 复活 | death | 首死后 50%HP 复活(一次性) | 26+ |
| deathSplit | 死亡分裂 | death | 死亡分裂 2 个子体 | 26+ |
| deathSlow | 死亡减速 | death | 死亡时 R60 减速50% 3s | (未注入,模板存在) |
| spawnMinions | 召唤 | offense | 召唤 3 个 normal | (未注入,模板存在) |

### 注入规则

| 波段 | 最大 buff 数 | buff 池 |
|------|-------------|---------|
| 1-5 | 0 | 无 |
| 6-15 | 1 | berserk, regen, healAura, speedAura |
| 16-25 | 2 | + damageReduce, reflect |
| 26+ | 2 | + revive, deathSplit |

- 每个普通怪 30% 概率获得 buff，70% 无 buff
- Boss 和 Elite 不受波次 buff 注入

### Flag 标记

| Flag | 效果 |
|------|------|
| elite | HP×3, 速度×1.1, 奖励×2, 标记 Elite=true |
| boss | HP×30, 奖励×5, 标记 Boss=true |

---

## 3. Boss 模板 (boss-templates.json)

### Boss 阶段系统

| 阶段 | HP阈值 | 效果 |
|------|--------|------|
| Phase 1 | HP≤75% | BaseSpeed +10% |
| Phase 2 | HP≤50% | BaseSpeed +10% (累计+21%) |
| Phase 3 | HP≤25% | BaseSpeed +10% (累计+33%) |

### Boss 光环

| 参数 | 值 |
|------|-----|
| 半径 | 120px |
| 速度加成 | +30% |
| 护甲加成 | +10 |

### 波次解锁的 Boss 能力

| 解锁波次 | 能力 | 参数 |
|---------|------|------|
| 10+ | 召唤小兵 | 每 20s 召唤 3+wave/10 个 normal |
| 20+ | 反伤 | 15% |
| 30+ | 偷金 | 每次被击中扣 1 金 |

### 预设 Boss 类型

| Boss | 能力概述 | 应对策略 |
|------|---------|---------|
| boss-iron | 30甲 + 10%伤害上限 | 破甲 + 持续输出 |
| boss-storm | 传送 + 免减速 | 全路径布防 |
| boss-grove | 3%/s回血 + 治疗光环 | DPS必须压过回复 |
| boss-legion | 光环加速加甲 + 召唤 | 先杀小怪再集火 |
| boss-hive | 持续召唤 + 死亡爆兵 | 溅射/AoE全程开 |
| boss-shadow | 受击隐身 + 反射攻速 | 多塔分散攻击 |
| boss-abyss | 全免疫 + 5%伤害上限 | 极高持续DPS |
| boss-chaos | 死亡全塔诅咒 + 扣50金 | 做好经济准备 |
| boss-sky | 飞行 + 召唤飞行斥候 | 对空火力全开 |
| boss-omega | 40甲 + 4%上限 + 弱点轮换 | 观察弱点集火 |

---

## 4. 扩展原型 (specialHints 中定义，部分未实装)

### 钢铁系 (st-)

| 原型 | 名称 | 能力 |
|------|------|------|
| st-plating | 附甲工兵 | 死亡给周围加甲 |
| st-bunker | 掩体车 | 范围友方 -30%受伤 |
| st-overload | 过载体 | 半血释放EMP禁塔 |
| st-welder | 焊接兵 | 修复友方 |
| st-magnetize | 磁化兵 | 抗投射物 -40% |

### 能量系 (en-)

| 原型 | 名称 | 能力 |
|------|------|------|
| en-phase | 相位兵 | 15%闪避 |
| en-disrupt | 干扰者 | 削射程 -20% |
| en-overcharge | 超载体 | 死亡加速友方 |
| en-siphon | 虹吸兵 | 受击吸血转盾 |
| en-flicker | 闪烁虫 | 周期瞬移 |

### 自然系 (na-)

| 原型 | 名称 | 能力 |
|------|------|------|
| na-spore | 孢子母体 | 死亡释放3孢子 |
| na-bloom | 绽放体 | 自愈 + 给友方盾 |
| na-rootwalker | 藤行者 | 全免CC + 死亡留减速区 |
| na-seedling | 种子兵 | 死后变tank |
| na-symbiont | 共生体 | 分摊伤害 |

### 秩序系 (or-)

| 原型 | 名称 | 能力 |
|------|------|------|
| or-phalanx | 方阵兵 | 近距互相加甲 |
| or-martyr | 殉道者 | 死亡永久强化同波怪 |
| or-warden | 守望者 | 给友方格挡盾 |
| or-inquisitor | 审判官 | 标记攻击者降攻速 |
| or-herald | 传令官 | 存活时全队加HP |

### 召唤系 (su-)

| 原型 | 名称 | 能力 |
|------|------|------|
| su-hive | 母巢虫 | 持续生产虫群 |
| su-revenant | 亡魂兵 | 首死复活 |
| su-cocoon | 虫茧 | 死亡爆出5个快怪 |
| su-leech | 寄附虫 | 漏了重走不扣命 |
| su-totem | 图腾兽 | 不动但持续治疗 |

### 混沌系 (ch-)

| 原型 | 名称 | 能力 |
|------|------|------|
| ch-backlash | 反噬体 | 死亡诅咒击杀塔 |
| ch-drain | 汲魂兵 | 12%吸血 |
| ch-goldrot | 蚀金虫 | 漏怪扣30金 |
| ch-corruptor | 腐化先驱 | 死亡传播腐化 |
| ch-abyssal | 深渊行者 | 全免疫但受伤+15% |

### 飞行系 (fly-)

| 原型 | 名称 | 能力 |
|------|------|------|
| fly-scout | 飞行斥候 | 基础飞行 |
| fly-heavy | 飞行重甲 | 高HP飞行 |
| fly-dart | 疾风翼 | 极速飞行 |
| fly-shield | 飞盾卫 | 60%护盾 |
| fly-ghost | 幽灵翼 | 出场隐身3s |
| fly-bomber | 投弹手 | 死亡减速脉冲 |
| fly-medic | 飞行奶妈 | 治疗飞行友方 |
| fly-split | 裂翼虫 | 死亡分裂2个疾风翼 |

---

## 5. 特殊机制说明

### 隐身规则
- 出场后隐身 `StealthDuration` 秒
- 隐身期间: Stealthed=true, 渲染 alpha=15%, targeting.go 跳过
- 解除条件: timer 到期 **或** HitFlash > 0 (被溅射/AoE 命中)

### 分裂规则
- 死亡时生成 `SplitCount` 个子体
- 子体: HP = 父体MaxHP × SplitHPRatio(30%), 速度 = 父体BaseSpeed × 1.4, 半径 = 父体 × 0.7
- 子体继承父体路径和 PathIndex
- 子体 SplitCount=0 (不递归分裂)

### 传送规则
- 每 `TeleportInterval` 秒跳过 `TeleportSkip` 段路径
- 冷却从放置时 = TeleportInterval 开始 (首次需等满间隔)
- 跳过后直接出现在目标路径点

### 治疗规则
- 每 `HealInterval` 秒对半径 `HealRadius` 内友方治疗 `HealPower` HP
- HealPower = 自身HP × healScale (配置 0.24 = 24%)
- 不治疗自己

### 狂暴规则
- HP降到 `BerserkThreshold`(50%) 以下时触发
- 一次性效果: 速度 × BerserkSpeedScale(1.5)
- BerserkTriggered 标记防重复触发

### 复活规则
- 首次死亡: HP恢复到 MaxHP × ReviveHPPercent(50%)
- ReviveUsed=true 后不再复活
- 复活时 HitFlash=0.3 (白闪效果)
- 复活绕过分裂检查 (不触发死亡分裂)

### 反伤规则
- 受到伤害时反弹 ReflectPercent(15%) 给攻击者
- 当前代码中反伤对塔无实际效果 (塔无HP)
