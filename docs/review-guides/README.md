# AI 源码审核指导文档

> 每份文档对应一个游戏系统，AI 在新 session 中按文档逐项审核并输出问题报告。

## 使用方法

```
请阅读 docs/review-guides/XX-系统名.md，按照文档中的检查项逐一审核源码，
输出发现的问题（按 P0-P3 严重度分级）。
```

## 文档清单

| 编号 | 文件 | 审核范围 | 预计耗时 |
|------|------|---------|---------|
| 01 | `01-tower-system.md` | 塔定义、建造、卖塔、战力系统 | 5 min |
| 02 | `02-ability-system.md` | 塔能力的实现完整性（含 goldOnKill/deathMark） | 15 min |
| 03 | `03-enemy-system.md` | 敌人原型、行为、Boss、buff 模板 | 10 min |
| 04 | `04-combat-system.md` | 伤害管线、攻击方式、碰撞、CC | 10 min |
| 05 | `05-warden-system.md` | 5 种战灵行为、成长、伤害 | 5 min |
| 06 | `06-wave-economy.md` | 波次公式、经济系统、难度缩放 | 5 min |
| 07 | `07-gamemode-session.md` | 6 种游戏模式胜负判定 | 5 min |
| 08 | `08-event-lifecycle.md` | 事件总线、数据流、生命周期 | 5 min |
| 09 | `09-config-consistency.md` | JSON↔Go 映射、常量一致性 | 5 min |
| 10 | `10-buff-strength.md` | Buff 叠加、战力系统、连锁网络 | 5 min |
| 11 | `11-rendering-visual.md` | 渲染代码、精灵/动画/HP bar/HUD 布局 | 10 min |
| 12 | `12-interaction-statemachine.md` | 交互状态机、ESC 路径、输入冲突 | 10 min |
| 13 | `13-ability-interaction.md` | 炮塔/怪物能力交互、沉默一致性、视觉完整性 | 15 min |

## 新增审核文档

当新增游戏系统时，按以下模板创建新文档：

```markdown
# XX-系统名 审核指导

## 审核目标
一句话描述这个系统做什么。

## 必读文件
列出 AI 必须读取的文件路径。

## 检查项
编号的检查项表格，每项包含：检查内容、验证方法、预期结果。

## 跨系统关联
列出与其他系统的交互点。
```
