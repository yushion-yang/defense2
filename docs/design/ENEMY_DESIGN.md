# 怪物设计文档

> 怪物 = 原型 + 标志 + buff 模板。不存在"固定类型"的怪物，所有多样性来自组合。

## 一、架构

```
波次规划器
  ↓ 选择原型 + 随机注入 buff
createEnemyFromSpec({ archetype, flags, buffs, bossTemplate })
  ↓ 查原型基础数值 → 叠加 flags(elite/boss) → 注入 buff 模板
enemy 实例（运行时）
```

### 三层构成

| 层 | 来源 | 决定什么 |
|----|------|---------|
| **原型** | `config/enemies-core.json` | 基础行为模式（HP/速度/移动方式） |
| **标志** | 波次规划器赋予 | 强化层级（elite ×3HP / boss ×30HP） |
| **buff 模板** | `config/enemy-buff-templates.json` | 附加能力（再生/狂暴/反弹/闪现…） |

## 二、12 个基础原型

| 原型 | 中文 | HP 倍 | 速度倍 | 奖励倍 | 移动 | 玩家对策 |
|------|------|:-----:|:------:|:------:|------|---------|
| normal | 步兵 | 1.18 | 1.04 | 0.82 | 地面 | 基准参照 |
| runner | 疾行 | 0.74 | 1.84 | 0.72 | 地面 | 减速/控制 |
| tank | 重甲 | 2.85 | 0.78 | 1.35 | 地面 | 持续输出 |
| armored | 护甲兵 | 1.40 | 0.85 | 1.10 | 地面 | 破甲/真伤 |
| shielded | 护盾兵 | 1.28 | 1.02 | 0.94 | 地面 | 先破盾 |
| swarm | 虫群 | 0.50 | 2.08 | 0.60 | 地面 | AoE 清群 |
| stealth | 隐身兵 | 0.92 | 1.32 | 1.10 | 地面 | 揭隐 |
| splitter | 分裂体 | 1.60 | 0.86 | 0.70 | 地面 | 二次清理 |
| teleporter | 传送兵 | 0.80 | 1.00 | 1.20 | 地面 | 预判位置 |
| healer | 治疗兵 | 1.06 | 1.14 | 1.02 | 地面 | 优先击杀 |
| buffer | 旗手 | 0.68 | 1.08 | 1.30 | 地面 | 优先击杀 |
| flying | 飞行斥候 | 0.90 | 1.10 | 0.85 | 飞行 | 对空塔 |

额外保留 `dummy`（HP×200 木桩）仅用于测试。

### 设计约束
- 每个原型定义**唯一的行为模式**，不允许两个原型行为趋同
- 原型只管基础数值和移动方式，不带任何主动/被动能力
- 所有特殊能力通过 buff 模板注入

## 三、标志系统

| 标志 | 效果 | 视觉 |
|------|------|------|
| `elite` | HP×3, 速度×1.1, 奖励×2 | 金色边框 + "精"字 |
| `boss` | HP×30, 奖励×5, 控制免疫(2s CD), %HP 伤害上限 5% | 加宽血条 + 5 段刻度 + Boss 字样 |

- elite 和 boss 互斥
- 任何原型都可以被标记为 elite 或 boss
- 标志在 `createEnemyFromSpec` 时由 `applyFlags()` 应用

## 四、17 个 Buff 模板

来源：`config/enemy-buff-templates.json`

### 状态增强类
| 模板 | 效果 | 参数 |
|------|------|------|
| berserk | HP<50% 时加速 | threshold:0.5, speedMult:1.5 |
| regen | 持续回血 | hpPerSec:2% |
| armorBuff | 增加护甲 | armor:10 |
| damageReduce | 减少受伤 | ratio:20% |
| corruptImmune | 免疫腐化 | - |

### 光环类
| 模板 | 效果 | 参数 |
|------|------|------|
| healAura | 治疗周围友军 | radius:60, heal:5/s |
| speedAura | 加速周围友军 | radius:80, bonus:20% |
| timewarp | 周围敌人加速 | radius:100, bonus:30% |

### 死亡触发类
| 模板 | 效果 | 参数 |
|------|------|------|
| deathSplit | 分裂小怪 | count:2, child:swarm |
| deathSlow | 留减速区 | radius:60, factor:0.5, dur:3s |
| empBurst | 禁用周围塔 | radius:80, dur:2s |

### 主动能力类
| 模板 | 效果 | 参数 |
|------|------|------|
| blink | 定时闪现 | interval:8s, range:100 |
| siphonShield | 受击充盾 | every 5 hits, shield:10%HP |
| reflect | 反弹伤害 | ratio:15% |
| revive | 死后复活 | hp:50%, maxRevives:1 |
| spawnMinions | 定时召唤 | interval:10s, count:2 |
| stealthOnHit | 受击隐身 | hits:3, dur:2s |

### 使用方式
```js
// 波次规划器自动注入
{ archetype: 'tank', count: 3, buffs: ['armorBuff', 'regen'] }
// buff 随波次递增：前5波无buff，6-15波1个，16-25波2个，26+波2个（更强池）
```

## 五、7 个 Boss 模板

来源：`config/boss-templates.json`

| 模板 | 效果 | 适合原型 |
|------|------|---------|
| bossPhase | 每 25% HP 阶段转换（加速/加甲/狂暴） | tank, armored |
| bossTeleport | 每 15s 闪现到随机路径点 | normal, runner |
| bossSpawnMinions | 每 20s 召唤 5 个小怪 | normal, armored |
| bossReflect | 反弹 25% 伤害 | stealth, tank |
| bossRotateWeakness | 每 10s 切换弱点属性 | tank |
| bossGoldSteal | 每次命中偷 1 金 | runner, normal |
| bossAura | 附近友军 +30% 速度 +10 护甲 | any |

### Boss 生成方式
```js
// 波次规划器在 boss 波（每 5/10 波）从 BOSS_POOL 随机选择
{ archetype: 'tank', count: 1, flags: ['boss'], bossTemplate: 'bossPhase' }
```

## 六、波次 Buff 注入规则

| 波次范围 | 可用 buff 池 | 最大 buff 数 |
|---------|-------------|:----------:|
| 1-5 | （无） | 0 |
| 6-15 | berserk, regen, armorBuff, speedAura | 1 |
| 16-25 | +reflect, healAura, damageReduce, blink | 2 |
| 26+ | +timewarp, revive, spawnMinions, deathSplit | 2 |

越晚的波次，buff 越强、组合越复杂。

## 七、关键文件

| 文件 | 作用 |
|------|------|
| `config/enemies-core.json` | 12 原型定义（数值 + 标签） |
| `config/enemy-buff-templates.json` | 17 个 buff 模板参数 |
| `config/boss-templates.json` | 7 个 Boss 行为模板 |
| `src/core/combat/enemy/factory.js` | createEnemyFromSpec — 组装原型+标志+buff |
| `src/core/combat/enemy/buffTemplates.js` | buff/boss 模板应用 + 旧类型映射 |
| `src/core/combat/wave/planner.js` | SPAWN_TABLE + BUFF_POOLS + BOSS_POOL |
| `src/core/combat/enemy/tickEnemies.js` | 运行时 buff 效果执行 |
| `src/renderer/canvas/drawEnemies.js` | 基于 flag/buff 的视觉渲染 |

## 八、新增怪物方式

**新增原型**（极少需要）：
1. 在 `enemies-core.json` 添加条目
2. 在 `planner.js` SPAWN_TABLE 添加 minWave/weight
3. 创建 `assets/enemies/{key}.svg`

**新增 buff 模板**（常见）：
1. 在 `enemy-buff-templates.json` 添加条目
2. 在 `buffTemplates.js` applyBuffTemplate 添加 case
3. 在 `planner.js` BUFF_POOLS 对应 tier 添加

**新增 Boss 模板**：
1. 在 `boss-templates.json` 添加条目
2. 在 `buffTemplates.js` applyBossTemplate 添加初始化
3. 在 `planner.js` BOSS_POOL 添加

**不需要**：创建新敌人 SVG（原型共享模型）、注册 ability handler（buff 模板自带）。
