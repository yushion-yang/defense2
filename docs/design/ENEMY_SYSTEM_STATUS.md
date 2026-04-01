# 怪物系统现状梳理

> 截至 2026-04-01 的完整系统状态快照。

---

## 一、架构总览

怪物 = **原型** + **标志** + **Buff 模板** + **能力列表**。没有"固定类型"，所有多样性来自组合。

```
波次规划器（planner.js）
  ├─ 选原型（SPAWN_TABLE 权重抽取）
  ├─ 注入 Buff（按波次段从 BUFF_POOLS 抽取）
  ├─ 标记 Elite/Boss（按规则）
  └─ 选 Boss 模板（BOSS_POOL）
         ↓
createEnemyFromSpec({ archetype, flags, buffs, bossTemplate })
         ↓
initializeGameEnemy()  ← 难度缩放 + 能力挂载 + 生命周期注册
         ↓
运行时实体
```

### 四层构成

| 层 | 数据来源 | 决定什么 | 数量 |
|----|---------|---------|:----:|
| 原型 | `config/enemies/enemies-core.json` | 基础数值比例、视觉标识、移动方式 | 12+1 |
| 标志 | 波次规划器运行时赋予 | Elite(×3HP) / Boss(×30HP) | 2 |
| Buff 模板 | `config/enemies/enemy-buff-templates.json` | 附加能力（再生/狂暴/反弹/闪现…） | 17 |
| Boss 模板 | `config/enemies/boss-templates.json` | Boss 专属行为（阶段/传送/召唤…） | 7 |

---

## 二、12 个基础原型

| 原型 | 中文 | HP 倍 | 速度倍 | 奖励倍 | 半径 | 移动 | 核心特征 |
|------|------|:-----:|:------:|:------:|:----:|------|---------|
| normal | 步兵 | 1.18 | 1.04 | 0.82 | 18 | 地面 | 基准参照物 |
| runner | 疾行 | 0.74 | 1.84 | 0.72 | 16 | 地面 | 高速低血 |
| tank | 重甲 | 2.85 | 0.78 | 1.35 | 24 | 地面 | 厚血慢速 |
| armored | 护甲兵 | 1.40 | 0.85 | 1.10 | 20 | 地面 | 中等偏硬 |
| shielded | 护盾兵 | 1.28 | 1.02 | 0.94 | 18 | 地面 | 自带护盾(0.82×HP) |
| swarm | 虫群 | 0.50 | 2.08 | 0.60 | 12 | 地面 | 小体型+极高速，需 AoE 清理 |
| stealth | 隐身兵 | 0.92 | 1.32 | 1.10 | 16 | 地面 | 出生 3s 隐身 |
| splitter | 分裂体 | 1.60 | 0.86 | 0.70 | 20 | 地面 | 死后分裂 2 个子体 |
| teleporter | 传送兵 | 0.80 | 1.00 | 1.20 | 16 | 地面 | 每 5s 跳过路径段 |
| healer | 治疗兵 | 1.06 | 1.14 | 1.02 | 20 | 地面 | 治疗周围友军 |
| buffer | 旗手 | 0.68 | 1.08 | 1.30 | 16 | 地面 | 光环加速周围友军 |
| flying | 飞行斥候 | 0.90 | 1.10 | 0.85 | 16 | 飞行 | 无视地面路径 |

另有 `dummy`（HP×200，不动）仅用于测试。

### 数值缩放公式

```
baseHp    = (46 + wave × 18) × lateHpMultiplier
hp        = round(baseHp × archetype.hpScale)
speed     = (44 + wave × 3.5) × archetype.speedScale × lateSpeedMultiplier
reward    = max(5, round((12 + wave × 1.5) × archetype.rewardScale × lateRewardMultiplier))
shield    = round(baseHp × archetype.shieldScale)     [仅 shielded 有]
```

后期乘数（wave > 5 后逐波复利）：HP +12.5%/波，速度 +5.5%/波，奖励 -递减至 0.45×。

---

## 三、标志系统

| 标志 | HP | 速度 | 奖励 | 特殊 | 视觉 |
|------|:--:|:----:|:----:|------|------|
| elite | ×3 | ×1.1 | ×2 | 金色边框 + "精" | 大一号 |
| boss | ×30 | 不变 | ×5 | 控制免疫(2s CD)、%HP上限5%/hit | 加宽血条 + 5段刻度 |

- Elite 和 Boss 互斥
- 任何原型都可以被标记
- 另有运行时精英晋升（HP×4, 速度×0.9, 奖励×2, 半径×1.4, +15%HP护盾）

---

## 四、17 个 Buff 模板

### 状态增强类（5）

| 模板 | 效果 | 默认参数 |
|------|------|---------|
| berserk | HP<50% 时加速 | threshold:0.5, speedMult:1.5 |
| regen | 持续回血（%HP/秒） | hpPerSec:2% |
| armorBuff | 增加护甲（遗留字段） | armor:10 |
| damageReduce | 受伤减免比例 | ratio:20% |
| corruptImmune | 免疫腐化标记 | — |

### 光环类（3）

| 模板 | 效果 | 默认参数 |
|------|------|---------|
| healAura | 治疗周围友军 | radius:60, heal:5/s |
| speedAura | 加速周围友军 | radius:80, bonus:20% |
| timewarp | 区域加速（含自身） | radius:100, bonus:30% |

### 死亡触发类（3）

| 模板 | 效果 | 默认参数 |
|------|------|---------|
| deathSplit | 死后分裂小怪 | count:2, child:swarm |
| deathSlow | 留减速区域 | radius:60, factor:0.5, dur:3s |
| empBurst | 禁用周围塔攻速 | radius:80, dur:2s |

### 主动能力类（6）

| 模板 | 效果 | 默认参数 |
|------|------|---------|
| blink | 定时闪现前进 | interval:8s, range:100 |
| siphonShield | 每受 N 击获得护盾 | every 5 hits, shield:10%HP |
| reflect | 反弹伤害比例 | ratio:15% |
| revive | 死后复活 | hp:50%, maxRevives:1 |
| spawnMinions | 定时召唤小怪 | interval:10s, count:2 |
| stealthOnHit | 受击后隐身 | hits:3, dur:2s |

### Buff 注入规则

| 波次段 | 可用池 | 最大数 |
|--------|--------|:------:|
| 1-5 | （无） | 0 |
| 6-15 | berserk, regen, armorBuff, speedAura | 1 |
| 16-25 | +reflect, healAura, damageReduce, blink | 2 |
| 26+ | +timewarp, revive, spawnMinions, deathSplit | 2 |

---

## 五、7 个 Boss 模板

| 模板 | 行为 | 参数 |
|------|------|------|
| bossPhase | 每 25%HP 阶段转换（加速/狂暴） | thresholds:[0.75,0.5,0.25] |
| bossTeleport | 定时传送到随机路径点 | interval:15s |
| bossSpawnMinions | 定时召唤 5 个小怪 | interval:20s, count:5 |
| bossReflect | 反弹 25% 伤害 | ratio:0.25 |
| bossRotateWeakness | 循环切换弱点属性 | interval:10s |
| bossGoldSteal | 每次命中偷 1 金 | goldPerHit:1 |
| bossAura | 友军 +30%速度 | radius:120 |

### Boss 阶段转换（硬编码）

- 70% HP：2s 无敌 + 召唤 4 个 normal 小怪
- 30% HP：2s 无敌 + 全场怪物 5s 加速 + Boss 狂暴(1.5×速度)

---

## 六、完整能力清单（enemyAbilities.js）

### 伤害减免类（4）

| 能力 | 字段 | 效果 |
|------|------|------|
| damageCap | `enemy.damageCap` | 单次受伤硬上限（固定值，如 60） |
| damageCapPercent | `enemy.damageCapPercent` | 单次受伤 %maxHP 上限（如 8%） |
| elementResist | `enemy._elementResist` | 魔法/元素伤害减免比例（如 30%） |
| projectileResist | `enemy._projectileResist` | 弹射伤害减免比例（如 40%） |

> 沉默（silence）状态可禁用 damageCap 和 damageCapPercent。

### 控制免疫类（3）

| 能力 | 效果 |
|------|------|
| slowImmune | 永久减速免疫 |
| stunImmune | 永久眩晕免疫 |
| controlImmune | 永久全控制免疫 |

### 死亡效果类（9）

| 能力 | 效果 |
|------|------|
| splitOnDeath | 死后分裂子体（count×30%HP，1.4×速度） |
| deathArmorBuff | 死后给周围友军加护甲 |
| deathSpeedBuff | 死后给周围友军加速 |
| deathSplit | 死后分裂(15%HP, 1.5×速度) |
| deathSlowZone | 死后留减速场 |
| deathTransform | 延迟后变形为其他类型 |
| deathCurse | 诅咒击杀者（减伤 25%） |
| deathCurseAll | 诅咒全部塔（减伤 20%） |
| deathCorruptionSpread | 死后向周围传播腐化层数 |
| deathSpawnRush | 死后生成 6 个极速 runner |
| goldStealOnDeath | 死后扣除玩家金币 |

### 光环/被动类（7）

| 能力 | 效果 |
|------|------|
| damageReductionAura | 周围友军受伤 -30% |
| rangeDebuffAura | 削弱周围塔射程 -20% |
| vulnerableAura | 自身增伤 +15%（自损） |
| phalanxArmor | 周围友军加护甲光环 |
| heraldBuff | 周围族群友军叠加 HP 加成 |
| stationaryHealAura | 驻留时周围 %HP 治疗光环 |
| bossAura | Boss 专属速度+护甲光环 |
| silenceAura | 沉默周围塔（降低攻速） |

### 主动能力类（13）

| 能力 | 效果 |
|------|------|
| empBurst | HP 阈值触发 AoE 攻速削弱 |
| singleHeal | 定时单体治疗友军 |
| siphonShield | 受击 N 次获得护盾 |
| blink | 定时闪现前进 |
| bloomHeal | 自愈 + 友军护盾 |
| symbioticLink | 共享 50% 受到的伤害给附近友军 |
| martyrBuff | 死后友军获得 HP+速度加成 |
| blockShield | 定时给周围友军护盾 |
| spawnMinions | 定时召唤小怪 |
| revive | 死后复活(50%HP, 最多1次) |
| bossTeleport | 定时路径跳跃 |
| hitCountStealth | 受击 N 次后隐身 |
| onHitReflect | 被攻击时降低攻击者攻速 |
| rotateWeakness | 循环切换弱点属性 |

### 其他

| 能力 | 效果 |
|------|------|
| flying | 设为飞行单位 |
| loopOnLeak | 到达终点后循环回路径起点 |
| goldSteal | 泄漏时偷金/每秒偷金 |
| towerDebuff | 范围削弱塔攻速 |
| summonMinions | 死后召唤小怪 |

---

## 七、伤害管线（processDamage）

```
输入伤害
  │
  ├─ Step 1: 免疫检查（untargetable > invincible > damageImmune）
  │          → pure 类型无视所有三种
  │
  ├─ Step 2: Boss %HP 伤害上限（isPercentHp 时，上限 5%maxHP/hit）
  │
  ├─ Step 3: 攻击者增伤 buff（damageUp，乘法叠加）
  │
  ├─ Step 4: 目标减伤 buff（damageDown，乘法叠加，地板 0.2 = 最多减 80%）
  │
  ├─ Step 4.5: 目标伤害上限（damageCap 固定值 / damageCapPercent %HP）
  │            → 沉默状态可禁用
  │
  ├─ Step 5: 护盾吸收（多护盾按剩余时间升序消耗 + 遗留单护盾字段）
  │          → pure 类型无视护盾
  │
  ├─ Step 6: HP 扣减
  │
  ├─ Step 7: 阈值触发（50% HP、Boss 阶段等）
  │
  └─ Step 8: 死亡判定
```

### 四种伤害类型

| 类型 | 无视减伤 | 无视护盾 | 无视无敌 |
|------|:-------:|:-------:|:-------:|
| physical | - | - | - |
| magic | - | - | - |
| true | **是** | - | - |
| pure | **是** | **是** | **是** |

### 管线外的额外计算（applyDamageToEnemy）

在进入管线前/后还有：
- 闪避事件（完全回避）
- 元素抗性（减免魔法/火/冰/雷）
- 弹射抗性（减免弹射伤害）
- 全局暴击（1.5×）
- 处决阈值（低血量 2×伤害）
- 瞬杀概率
- 回声打击（50%额外伤害）
- DOT 传播
- 反射器反弹
- 阶段护盾（75%/50%/25% HP 触发护盾）

---

## 八、现有减伤机制汇总

| 机制 | 来源 | 效果 | 绕过方式 |
|------|------|------|---------|
| damageDown buff | buff 系统 | 乘法减伤(地板 0.2) | true/pure 伤害 |
| damageCap | 能力 | 固定值硬上限 | silence 禁用 |
| damageCapPercent | 能力 | %HP 硬上限 | silence 禁用 |
| elementResist | 能力 | 元素伤害减免 | 物理/true 伤害 |
| projectileResist | 能力 | 弹射伤害减免 | 非弹射伤害 |
| 护盾 | 多来源 | 先消耗再扣血 | pure 伤害 |
| 无敌 | buff | 完全格挡 | pure 伤害 |
| 伤害免疫 | buff | 完全格挡 | pure 伤害 |
| 不可选中 | buff | 完全格挡 | pure 伤害 |
| 闪避 | 事件 | 概率完全回避 | — |
| Boss %HP 上限 | 管线 | 5%maxHP/hit | 非 %HP 伤害 |

---

## 九、波次规划

### 出怪表（SPAWN_TABLE）

| 原型 | 最低波次 | 权重 |
|------|:-------:|:----:|
| normal | 1 | 10 |
| runner | 3 | 6 |
| tank | 5 | 4 |
| armored | 5 | 3 |
| shielded | 7 | 3 |
| swarm | 6 | 5 |
| stealth | 6 | 2 |
| flying | 5 | 2 |
| splitter | 7 | 2 |
| healer | 8 | 2 |
| buffer | 5 | 2 |
| teleporter | 6 | 1 |

### 每波数量

```
总数 = base(6) + wave × perWave(2)
```

### 精英/Boss 规则

- 精英：每 4 波出一个
- Boss：胜利目标波后从 BOSS_POOL 随机选

---

## 十、关键文件索引

| 文件 | 作用 |
|------|------|
| `config/enemies/enemies-core.json` | 12 原型定义 |
| `config/enemies/enemy-buff-templates.json` | 17 个 Buff 模板 |
| `config/enemies/boss-templates.json` | 7 个 Boss 模板 |
| `src/core/combat/enemy/factory.js` | 创建 + 初始化 |
| `src/core/combat/enemy/enemyAbilities.js` | 能力挂载（40+ 种） |
| `src/core/combat/enemy/enemyEvents.js` | 波次事件应用 + 精英晋升 |
| `src/core/combat/enemy/buffTemplates.js` | Buff/Boss 模板应用 + 旧类型映射 |
| `src/core/combat/enemy/tickEnemies.js` | 每帧更新（13步） |
| `src/core/combat/enemy/behaviors.js` | 纯行为函数 |
| `src/core/combat/enemy/lifecycle.js` | 生命周期处理器 |
| `src/core/combat/enemy/movement.js` | 路径移动 |
| `src/core/combat/damagePipeline.js` | 统一伤害管线（8步） |
| `src/core/combat/damage/applyDamage.js` | 完整伤害流程（含副作用） |
| `src/core/combat/damage/settleDeaths.js` | 死亡结算 |
| `src/core/combat/wave/planner.js` | 波次组成规划 |
| `src/core/combat/dot.js` | DOT 系统(burn/poison/bleed) |
| `src/core/combat/crowdControl.js` | 控制效果(stun/slow/root) |
| `src/core/combat/buff/buffManager.js` | Buff 增删查改 |
| `src/core/combat/buff/stackRules.js` | Buff 叠加规则 |

---

## 十一、当前问题与设计空白

### 已有但不充分的机制

1. **damageCap/damageCapPercent 仅 2 个怪用**：ironhide(固定60)、colossus(%HP 8%)，其他怪完全没有伤害上限保护
2. **元素抗性(elementResist)是全有全无**：要么抗所有元素，要么完全不抗，缺乏针对性
3. **护甲系统已删除**：原有的 LoL 式护甲公式已被移除，怪物缺乏渐进式物理减伤
4. **damageDown buff 极少自然出现**：几乎只有事件系统偶尔注入，怪物自身不会获得

### 完全缺失的机制

1. **无基于攻击者攻击力的伤害上限**：高攻塔对任何非 Boss 怪都能全额输出
2. **无物理/远程/近战分类抗性**：所有物理伤害一视同仁
3. **无针对特定攻击模式的反制**：如对 AoE/单体/DOT/弹射 的差异化抗性
4. **无怪物阵型/编队效果**：同类怪聚集没有协同加成
5. **无适应性机制**：怪物不会根据玩家的塔阵容调整行为

### 数值断层风险

- **塔伤害范围**：基础 8~30，核心技能爆发 15×（450 点最高）
- **怪物 HP 范围**：wave 1 约 54~155，wave 10 约 170~520，wave 20 约 430~1300
- **高爆发 vs 低血**：核心技能（如 nukeBomb 450 伤害）可以秒杀 wave 15 前的大多数非 tank/非 boss 怪
- **缺乏中间层防御**：怪物只有"完全裸奔"和"Boss 级保护"两个极端
