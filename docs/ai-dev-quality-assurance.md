# AI 开发质量保证方法论

> 基于本项目 50+ 个已发现 bug 的根因分析，总结 AI 写代码时 bug 是如何被引入的，以及如何系统性预防。

---

## 一、Bug 是如何被引入的

### 假设前提

如果满足以下条件，理论上不会有 bug：
1. 设计文档**完备**（覆盖所有边界情况）
2. AI **精确理解**设计意图（无翻译损耗）
3. AI **一次性**实现所有系统（无多 session 割裂）
4. 系统间**无隐式依赖**（全部显式声明）

**现实中每一条都不成立。** 以下是本项目真实 bug 的根因分类：

### 1.1 翻译损耗（设计意图 → 代码实现）

> **你说的和 AI 理解的不是同一件事。**

| 真实案例 | 你的意图 | AI 的理解 | 结果 |
|---------|---------|---------|------|
| JSON tag 错误 | `abilities.json` 用 `"type"` 字段标识能力 | struct tag 写成 `json:"name"` | 能力系统整体失效 |
| scatter 弹丸数 | 配置声明 `extraPellets` 随强度缩放 | handler 硬编码 `scatterPellets=3` | 强度不影响弹丸数 |
| 卖塔退款率 | 文档说 70% 回收 | economy.go 写了 50% | 玩家实际只拿到一半 |

**根因**：AI 在生成代码时，对"配置字段名"、"具体数值"这类细节的记忆是模糊的。它会根据"合理猜测"填充，而不是严格查证。

### 1.2 集成裂缝（正确的 A + 正确的 B = 错误的 A+B）

> **每个系统单独看是对的，但拼在一起就出问题。**

| 真实案例 | 系统 A | 系统 B | 裂缝 |
|---------|--------|--------|------|
| ProcessDamage 未接入 | 伤害管线 7 步完整正确 | ApplyHit 直接 `HP-=` | 管线写好了但没人调用 |
| BossRush 永不结束 | BossRush 需要 5 个 Boss | Spawner 每 5 波才出 1 个 Boss | 5 波只出 1 个，永远达不到 5 |
| WaveCleared 不触发 | onWaveTransition 逻辑正确 | 手动开波路径没捕获 prevWave | 事件链断裂 |
| 弹道追踪 | 弹射物有 Target 字段 | 初始化时 VX/VY 计算一次后不更新 | fire-and-forget |

**根因**：AI 在不同 session 中分别实现系统 A 和系统 B。每个 session 的上下文窗口只看到部分代码，不知道另一个系统的约束。

### 1.3 多 Session 漂移

> **第 1 个 session 的决策，第 5 个 session 已经忘了。**

| 真实案例 | Session N 的决策 | Session M 的实现 | 冲突 |
|---------|----------------|-----------------|------|
| 敌人原型全 normal | Session 1: pool.go 硬编码 archetype="normal" | Session 3: 加了 13 种原型配置 | 配置加了但 Spawn 不读 |
| Hero 系统残留 | Session 2: 删除了 hero 系统 | Session 4: 某处仍引用 heroUnit | 编译通过但运行异常 |
| 减速下限两处定义 | Session 1: combat 包定义 MinSpeedRatio | Session 2: enemy 包也定义一份 | 改了一处忘了另一处 |

**根因**：AI 每个 session 从头构建上下文。CLAUDE.md 和 memory 能缓解但不能完全解决——它们是摘要，不是完整代码。

### 1.4 隐式假设

> **你以为 AI 知道，AI 以为你知道，结果谁都不知道。**

| 真实案例 | 你的隐式假设 | AI 的隐式假设 | 结果 |
|---------|-------------|-------------|------|
| 战灵出生位置 | 战灵应该 teleport 到轨道位置 | 初始位置 (600,270) 是安全的 | 开局闪现到屏幕中央 |
| 敌人池复用 | Spawn 时应该重置所有字段 | 只重置了"重要"字段 | HitFlash 残留 |
| Boss HP 缩放 | Boss 应该随波次变强 | 固定 3 倍 HP（"合理默认值"） | 后期 Boss 被秒杀 |

**根因**：设计文档再详细，也不可能写"Boss HP 倍率应该是 `8+wave` 而不是 `3`"这种具体数值。AI 填的默认值未必符合实际平衡需求。

### 1.5 紧急修复引入新 Bug

> **修一个 bug 的过程中引入另一个。**

| 真实案例 | 原始修复 | 引入的问题 |
|---------|---------|-----------|
| 修复 SVG 渲染 → 改用 PNG | PNG 路径约定与 SVG 不同 | faction="base" vs 目录 "core" 不匹配 |
| HitCallback 加 killed 参数 | 修改了函数签名 | 部分调用点没更新，编译通过但行为错误 |

---

## 二、系统性预防方案

### 2.1 设计文档 → 代码的验证层

**原则**：不信任 AI 的"翻译"，用机器验证代替人工 review。

```
设计文档(.md)
    ↓ AI 实现
源代码(.go)
    ↓ 自动验证
契约测试(tests/contracts/)  ← 这层阻止翻译损耗
```

**具体做法**：

| 验证类型 | 工具 | 防什么 |
|---------|------|--------|
| 配置契约 | `tests/contracts/config_rules_test.go` | JSON 字段缺失/范围错误 |
| JSON↔Go 映射 | `docs/review-guides/09-config-consistency.md` | json tag 不匹配 |
| 常量一致性 | grep 脚本 | 多处定义不一致 |
| API lint | `tests/lint/forbidden_api_test.go` | 直接调用 ebiten API |

**建议新增**：
```go
// tests/contracts/ability_mapping_test.go
// 验证 abilities.json 中每个 type 在 config_ability.go 的 switch 中有对应 case
func TestAbilityConfigMapping(t *testing.T) {
    table := config.GlobalAbilityTable()
    for name := range table {
        if _, ok := tower.Registry[name]; !ok {
            t.Errorf("ability %q in config but not registered in code", name)
        }
    }
}
```

### 2.2 集成验证 → 数据流断言

**原则**：不验证单个系统，验证系统间的数据流。

```
系统 A → [数据流断言] → 系统 B
```

**具体做法**：

| 数据流 | 断言 | 工具 |
|--------|------|------|
| 塔命中 → 伤害管线 | `grep 'HP\s*-=' 只出现在 ProcessDamage` | lint 脚本 |
| 击杀 → 事件 → 金币 | 回归测试: 杀敌前后 gold 差值 = KillGold | regression sim |
| 波次 → 清除事件 → 能力解锁 | autoplay: wavesCleared == wave-1 | anomaly detector |
| 配置 → 代码 → 遥测 | autoplay: 34 种能力 → 遥测覆盖率 | coverage report |

### 2.3 多 Session 连续性 → 契约文件

**原则**：跨 session 的约束用代码固化，不靠 AI 记忆。

| 约束类型 | 固化方式 | 示例 |
|---------|---------|------|
| 接口契约 | Go interface | Mode 接口的 11 个方法 |
| 数值约定 | const + 注释 | `MinSpeedRatio = 0.2 // 减速下限，combat 和 enemy 两包共用` |
| 注册完整性 | init() 自注册 + 测试 | 每种模式 init 注册 → 测试验证全注册 |
| 行为规范 | CLAUDE.md | 渲染用 draw.* 不用 vector.* |

**CLAUDE.md 规则优先级**：
```
1. 编译器强制（interface、类型系统）  ← 最可靠
2. 测试强制（contract test、lint test）  ← 次之
3. CLAUDE.md 约定（文字规则）  ← 最弱，容易被忽略
```

### 2.4 隐式假设 → 显式断言

**原则**：凡是"应该如此"的地方，都加运行时检查。

```go
// 不好：隐式假设 Boss HP 应该合理
bossCfg.HpScale *= 3

// 好：显式断言 + 可配置
bossHPMul := 8.0 + float64(s.Wave)
bossCfg.HpScale *= bossHPMul
// autoplay anomaly detector: boss_too_weak 检测存活 <5s
```

**autoplay 异常检测器就是"显式断言"的运行时版本**——把隐式假设变成 26 条检测规则。

### 2.5 修复流程 → 防止二次引入

**原则**：每次修 bug 必须回答两个问题。

```
1. 这个 bug 的回归测试是什么？（写 test 或加 anomaly 规则）
2. 同类 bug 还可能出现在哪里？（全局搜索同模式代码）
```

---

## 三、实操 Checklist

### 3.1 AI 写完代码后，人做什么

```
□ 1. make check-all（编译+lint+test）
□ 2. 浏览 git diff，关注：
     - JSON tag 是否正确（对照配置文件）
     - 新函数是否被调用（不是"写了就有用"）
     - 接口方法是否全实现
□ 3. 跑一轮 autoplay（--scenario 相关场景）
□ 4. 看 report.json 的 anomalies 和 coverage_gaps
```

### 3.2 新系统交付标准

```
□ 配置文件有对应的契约测试
□ 核心函数有 ≥1 个单元测试
□ 与其他系统的交互点有回归测试
□ autoplay coverage 中该系统的维度不为 0
□ review-guides/ 中有对应审核文档
```

### 3.3 Bug 修复标准

```
□ 回归测试覆盖该 bug 的触发条件
□ anomaly.go 有对应检测规则（运行时兜底）
□ 全局搜索同模式代码确认无同类问题
□ memory 记录 bug 根因（防止 AI 下次犯同样的错）
```

---

## 四、质量保障层次总结

```
第 1 层：编译器                     ← 免费，100% 可靠
第 2 层：单元测试 + 契约测试         ← make test，秒级反馈
第 3 层：回归 Sim（headless）        ← 纯逻辑验证，无需 GPU
第 4 层：autoplay 异常检测（26 条）   ← 运行时不变量，覆盖隐式假设
第 5 层：AI 源码审核（review-guides） ← 深层逻辑 + 跨系统一致性
第 6 层：人工游玩                    ← "感觉"层，AI 无法完全替代
```

每修一个 bug，应该往上层固化——让它在更早、更便宜的层被拦截。
