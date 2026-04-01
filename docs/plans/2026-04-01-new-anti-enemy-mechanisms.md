# 现有能力精细化组合与玩法搭配方案

Date: 2026-04-01

## 一、清理：移除 stealth 和 shielded 原型

护盾已取消，隐身也取消。需要清理：

| 位置 | 内容 | 操作 |
|------|------|------|
| `config/enemies/enemies-core.json` | `"stealth"` 和 `"shielded"` 条目 | 删除 |
| `internal/config/enemy_config.go` | `StealthDuration` 字段 | 删除 |
| `internal/core/enemy/spawner.go` | 权重表中 `{"stealth", 5}` | 删除 |
| `internal/core/enemy/enemy.go` | `IsUntargetable` 字段 | 保留(管线免疫检查通用) |
| `internal/core/combat/damage_pipeline.go` | `IsUntargetable` 检查 | 保留(通用免疫) |

清理后剩余 **10 种敌人原型**: normal, runner, tank, armored, swarm, splitter, teleporter, healer, buffer, flying (+dummy 测试用)

---

## 二、现有系统盘点

### 10 种敌人原型

| 原型 | 核心特征 | 威胁维度 |
|------|---------|---------|
| **normal** | 均衡，以量取胜 | 数量压力 |
| **runner** | 0.74HP / 1.84速度 | 速度突防 |
| **tank** | 2.85HP / 0.78速度 | 血量硬扛 |
| **armored** | 1.4HP / 0.85速度 | 中等血量+数量 |
| **swarm** | 0.5HP / 2.08速度 / r=12 | 极小体积+蜂拥 |
| **splitter** | 1.6HP / 死亡分裂2子体 | 死亡后增殖 |
| **teleporter** | 每5s跳1段路径 | 跳过防线 |
| **healer** | 每2.5s治疗105px内24%HP | 团体续航 |
| **buffer** | +20%移速 / +10护甲光环 | 增强友军 |
| **flying** | 走飞行路径 | 路线差异 |

### 8 种攻击方式

| 方式 | 特点 | 天然优势 |
|------|------|---------|
| projectile | 追踪弹，单体 | 精准集火 |
| laser | 即时光束 | 无飞行时间 |
| wideBeam | 穿透光束 | 直线贯穿多目标 |
| scatter | 锥形3弹丸 | 扇面覆盖 |
| charge | 蓄力重弹 | 单发高伤 |
| spin_aoe | 旋转范围伤害 | 近程全覆盖 |
| aura_dot | 持续毒圈 | 无死角磨血 |
| radial | 360度穿刺弹 | 全方位覆盖 |

### 33 种能力 (6 大类)

**attack (8)**: enhance, scatter, wideBeam, spinAoe, pierce, bounce, splash, multiTarget, radial

**cc (4)**: slowPower, slowDuration, stunChance, stunDuration

**damage (6)**: crit, deathMark, distanceDamage, executionBonus, flatDamage, momentum

**buff (6)**: damageUpAura, attackSpeedAura, rangeAura, critAura, soloBoost, goldPassive

**dot (4)**: burn, bleedDot, poison, weaken

**zone (4)**: poisonZone, silenceZone, curseZone, weakenZone

---

## 三、能力协同分析

### 3.1 强力协同组合 (1+1 > 2)

#### 爆发型协同

| 组合 | 效果链 | 适合打谁 |
|------|--------|---------|
| **crit + splash** | 暴击后溅射用的是含 crit bonus 的总伤，AoE 暴击 | swarm, normal 密集群 |
| **crit + bounce** | 每次弹射独立判暴击，弹射链上可多次暴击 | 散布中等密度群 |
| **crit + executionBonus** | 低血暴击双重加成，收割极快 | tank 打到半血后 |
| **deathMark + splash** | 溅射杀一片 → 每个死亡都爆炸 → 连锁清场 | swarm 密集群 |
| **executionBonus + burn** | 灼烧磨到半血 → 斩杀加伤收割 | tank, armored |

#### 控制型协同

| 组合 | 效果链 | 适合打谁 |
|------|--------|---------|
| **slowPower + burn** | 减速延长停留 → 灼烧多跳 → 额外总伤 | runner, normal |
| **stunChance + bleedDot** | 眩晕期间流血白跑 → 控制+消耗 | tank, armored |
| **slowDuration + poison** | 长时间减速 + 长时间中毒 → 持续削血 | 任何非Boss |
| **multiTarget + slowPower** | 多目标全减速 → 整波降速 | swarm, runner 波 |
| **bounce + stunChance** | 弹射链上每跳都判眩晕 → 群体眩晕链 | 中密度群 |

#### 经济/辅助协同

| 组合 | 效果链 | 场景 |
|------|--------|------|
| **goldPassive + damageUpAura** | 产金+辅助周围塔伤害 → 纯辅助定位 | 后排安全位 |
| **soloBoost + distanceDamage** | 独立部署+远距 → 双重加伤叠加 | 路径拐角孤位 |
| **critAura + crit(邻塔)** | 暴击光环给邻塔加暴击率 → 全阵暴击 | 密集塔群 |
| **attackSpeedAura + multiTarget(邻塔)** | 攻速光环加速多目标塔 → 火力倍增 | 核心输出区 |

### 3.2 冗余组合 (避免)

| 组合 | 冗余原因 |
|------|---------|
| splash + spinAoe | spin 已全范围，splash 再炸意义小 |
| bounce + spinAoe | spin 已覆盖，弹射目标大概率已被打过 |
| multiTarget + spinAoe | spin 自管理绕过 multiTarget 逻辑 |
| multiTarget + charge | charge 自管理绕过 multiTarget |
| distanceDamage + spinAoe | spin 射程短(~80px)，距离 bonus 极小 |
| slowPower + slowDuration | 同类CC，后者覆盖前者 |
| stunChance + stunDuration | 同类CC，效果浪费一个槽位 |

---

## 四、塔角色定位与能力搭配方案

基于 upgrade.go 的能力系统（6 类各 1 槽，每 2 波解锁 1 槽，3 选 1），每塔最终可拥有 6 个能力（每类 1 个）。

以下按**塔角色**设计推荐搭配模板：

### 4.1 狙击手型 (远程高单体)

**攻击方式**: projectile / laser / charge

| 槽位 | 推荐能力 | 替代 | 理由 |
|------|---------|------|------|
| attack | pierce / enhance | - | 穿刺增加穿透量，enhance 纯数值提升 |
| cc | stunChance | stunDuration | 远程眩晕锁定高价值目标 |
| damage | crit / executionBonus | distanceDamage | 暴击爆发 或 斩杀收割 |
| buff | soloBoost | critAura | 通常部署在边角，独行加伤 |
| dot | burn | poison | 灼烧按伤害比例，高伤塔灼烧更痛 |
| zone | weakenZone | curseZone | 增伤区配合自身高伤 |

**核心打法**: 高伤单点输出 → 暴击/斩杀收割 → 灼烧补充持续伤害

**克制**: tank(crit爆发+burn持续), armored(executionBonus补刀), teleporter(远距覆盖+stun锁定)

---

### 4.2 群控型 (多目标控制)

**攻击方式**: projectile + multiTarget / scatter

| 槽位 | 推荐能力 | 替代 | 理由 |
|------|---------|------|------|
| attack | scatter / multiTarget | bounce | 天然多目标覆盖 |
| cc | slowPower | slowDuration | 多目标 × 减速 = 整波降速 |
| damage | flatDamage | momentum | 多目标低伤塔靠固伤补数值 |
| buff | attackSpeedAura | damageUpAura | 自身需要高攻速维持控制频率 |
| dot | poison | bleedDot | 固定DPS中毒对低伤塔更划算 |
| zone | silenceZone | poisonZone | 沉默禁用敌人 damageCap |

**核心打法**: 广覆盖减速 → 敌群堆积 → 毒/沉默持续消耗 → 为后排输出塔创造窗口

**克制**: runner(减速拉回), swarm(多目标+毒), normal(群体控制)

---

### 4.3 AOE清场型 (范围爆发)

**攻击方式**: spinAoe / wideBeam / radial

| 槽位 | 推荐能力 | 替代 | 理由 |
|------|---------|------|------|
| attack | spinAoe / wideBeam / radial | splash | 天然范围攻击 |
| cc | stunDuration | slowPower | 范围攻击 × 长眩晕 = 群控 |
| damage | deathMark | crit | 击杀爆炸链式清场 |
| buff | damageUpAura | critAura | 辅助周围塔，自身不缺覆盖 |
| dot | burn | bleedDot | 范围灼烧对密集群最有效 |
| zone | curseZone | poisonZone | %HP 扣血对高HP群也有效 |

**核心打法**: 范围扫射 → 灼烧+诅咒持续掉血 → deathMark 击杀爆炸连锁 → 清场

**克制**: swarm(天然克星), normal密集波, splitter(子体也被范围覆盖)

---

### 4.4 辅助型 (光环增益)

**攻击方式**: projectile / aura_dot (低攻击优先级)

| 槽位 | 推荐能力 | 替代 | 理由 |
|------|---------|------|------|
| attack | enhance | - | 提升自身基础属性，没有冗余 |
| cc | slowPower | - | 即使辅助塔也要有控制贡献 |
| damage | momentum | flatDamage | 攻击附带加伤，聊胜于无 |
| buff | damageUpAura / attackSpeedAura | critAura / rangeAura | 核心定位：光环辅助周围塔 |
| dot | weaken | poison | 给敌人上增伤debuff(实现后) |
| zone | weakenZone | silenceZone | 范围增伤配合周围输出塔 |

**核心打法**: 部署在塔群中心 → 光环加持周围塔 → weakenZone 给敌群上增伤 → 团队 DPS 最大化

**适合场景**: 密集塔群布阵，2-3 输出塔围绕 1 辅助塔

---

### 4.5 经济型 (打钱优先)

**攻击方式**: projectile (低成本)

| 槽位 | 推荐能力 | 替代 | 理由 |
|------|---------|------|------|
| attack | enhance | bounce | 提升基础或弹射赚击杀 |
| cc | slowPower | - | 减速让敌人多吃几下 |
| damage | executionBonus | crit | 确保补刀拿击杀奖励 |
| buff | goldPassive | soloBoost | 核心：被动产金 |
| dot | burn | poison | 灼烧磨血保证击杀 |
| zone | poisonZone | - | 范围磨血抢人头 |

**核心打法**: 前期靠 goldPassive 加速经济 → 中期靠击杀积累金币 → 后期转型或持续产金

---

## 五、波次进度与能力解锁节奏

每 2 波解锁 1 槽，玩家需要规划解锁顺序：

### 推荐解锁优先级

| 解锁波次 | 第1槽(波2) | 第2槽(波4) | 第3槽(波6) | 第4槽(波8) | 第5槽(波10) | 第6槽(波12) |
|---------|-----------|-----------|-----------|-----------|------------|------------|
| **狙击手** | attack(pierce) | damage(crit) | dot(burn) | cc(stun) | buff(soloBoost) | zone(weakenZone) |
| **群控** | attack(scatter) | cc(slowPower) | damage(flatDmg) | dot(poison) | buff(atkSpdAura) | zone(silenceZone) |
| **AOE** | attack(spinAoe) | damage(deathMark) | dot(burn) | cc(stunDur) | buff(dmgUpAura) | zone(curseZone) |
| **辅助** | buff(dmgUpAura) | cc(slowPower) | attack(enhance) | zone(weakenZone) | dot(weaken) | damage(momentum) |
| **经济** | buff(goldPassive) | attack(enhance) | damage(execBonus) | dot(burn) | cc(slowPower) | zone(poisonZone) |

**设计原则**:
- 第1-2槽定义塔的核心身份（输出/控制/辅助/经济）
- 第3-4槽补充互补维度（输出塔加控制，控制塔加输出）
- 第5-6槽为锦上添花（光环/区域效果）

---

## 六、敌人原型 vs 塔组合的克制矩阵

| 敌人 \ 塔型 | 狙击手 | 群控 | AOE清场 | 辅助+输出 |
|------------|--------|------|--------|----------|
| **normal** | ★★★ | ★★★★ | ★★★★★ | ★★★★ |
| **runner** | ★★ | ★★★★★ | ★★★ | ★★★★ |
| **tank** | ★★★★★ | ★★ | ★★ | ★★★★ |
| **armored** | ★★★★ | ★★★ | ★★★ | ★★★★ |
| **swarm** | ★ | ★★★ | ★★★★★ | ★★★ |
| **splitter** | ★★ | ★★★ | ★★★★★ | ★★★ |
| **teleporter** | ★★★★ | ★★★ | ★★ | ★★★ |
| **healer** | ★★★ | ★★★ | ★★★ | ★★(需weaken) |
| **buffer** | ★★★★ | ★★★ | ★★★ | ★★★ |
| **flying** | ★★★★ | ★★★ | ★★ | ★★★ |

**读法**: 5星=强克制，1星=几乎无效

**关键发现**:
- **没有万能塔**，每种都有明显弱点 → 迫使混合布阵
- **tank 只怕狙击手**（高单体伤害+斩杀） → tank 波是检验阵容的关键
- **swarm 只怕 AOE** → swarm 波逼玩家必须有范围输出
- **healer 缺少反制** → weaken 能力实现后才能有效应对（当前靠集火优先击杀 healer）

---

## 七、阵容搭配模板

### 7.1 均衡阵容 (推荐新手)

```
[群控] [AOE] [狙击手]
       [辅助]
```

- 群控在前排减速
- AOE 清群（swarm/normal/splitter）
- 狙击手点杀高价值目标（tank/healer/buffer）
- 辅助居中给三塔上光环

**应对所有敌人类型评分**: 平均 ★★★★

### 7.2 控制链阵容 (高难度)

```
[群控A(slow)] [群控B(stun+bounce)]
         [AOE(deathMark)]
```

- 两个群控交替减速+眩晕，敌群寸步难行
- AOE 配 deathMark 在密集区爆炸连锁
- **弱点**: 缺少单体输出，tank 波困难

### 7.3 狙击矩阵 (反 Boss/Tank)

```
[狙击A(crit)] [辅助(dmgAura+critAura)]
[狙击B(exec)]
```

- 辅助塔给两个狙击手上伤害+暴击光环
- A 暴击爆发，B 斩杀收割
- **弱点**: 缺少 AOE，swarm 波灾难

### 7.4 经济速成 (endless/长局)

```
[经济] [群控]
       [AOE]
```

- 经济塔前期 goldPassive 快速积累
- 群控+AOE 扛住前期压力
- 中后期用积累的金币铺更多塔
- **关键**: 经济塔需要 executionBonus 确保抢到击杀

---

## 八、weaken 能力实现建议

当前 `weaken` 和 `weakenZone` 在 `config_ability.go` 中返回 nil，是唯一需要补全的能力。

### 实现方案

Enemy 新增 2 个字段:
```go
DamageAmplify      float64 // 受伤增加倍率
DamageAmplifyTimer float64 // 持续时间
```

**weaken (OnHit)**: 命中时设置 `e.DamageAmplify = sv, e.DamageAmplifyTimer = pm`

**weakenZone (OnTick)**: 每帧对范围内敌人设置 `DamageAmplify = sv`（离开区域后自然归零）

**管线接入**: 步骤 4 (目标减伤) 之后追加:
```go
if e.DamageAmplify > 0 {
    damage *= (1 + e.DamageAmplify)
}
```

上限 0.5（最多 +50% 受伤）。这样 healer 波终于有了战术解法：上 weaken → 集火 → 即使 heal 也扛不住增伤。

---

## 九、总结

不需要新增能力类型。现有 33 种能力 × 8 种攻击方式 × 6 槽位系统，已经提供了极大的组合空间。核心工作是：

1. **删除 stealth/shielded 原型**，精简到 10 种清晰的敌人定位
2. **实现 weaken/weakenZone**，补上唯一缺失的能力实现
3. **在 UI 层引导玩法搭配**，升级选择时展示能力间的协同/冗余提示
4. **通过波次设计制造压力**，迫使玩家混合布阵而非堆叠单一塔型
