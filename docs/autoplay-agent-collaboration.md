# AutoPlay 自动化闭环测试系统

日期: 2026-03-31
状态: 设计
前置依赖: `internal/autoplay/`(已实现), `cmd/autoplay/main.go`(已实现)

## 目标

构建**全自动**的游戏优化闭环: 自动对局 -> AI 分析 -> 自动修复 -> 回归验证, 循环迭代直到无关键问题或达到轮次上限。由 Claude Code cron 驱动, 无人值守运行。

## 流程总览

```
脚本1 (环境检查/增量更新)
  |
  v
脚本2 (自动对局 -> M1 JSON + M2 截图)
  |
  v
脚本3 (AI 分析 M1/M2 -> 新问题写 M4)
  |
  v
M4 为空? --是--> 本轮结束, 生成总结报告
  |
  否
  v
脚本4 (修复 M4 问题 -> make build -> 归档 M3)
  |
  v
build 不过? --> 继续修直到通过
  |
  v
回到脚本2 (回归验证, 进入下一轮循环)
```

## 目录结构

```
docs/autotest/
  last_commit.txt        # 文件A: 上次更新自动对局系统时的 git commit hash
  config.json            # 可选: 闭环运行参数配置
  state.json             # 编排器状态(当前轮次/阶段)
  analysis.md            # 脚本3 产出的完整分析报告(含不可自动修复的观察)
  M1/                    # 自动对局产出 -- JSON 数据(波次统计/DPS/金币曲线等)
  M2/                    # 自动对局产出 -- 截图(关键帧/异常画面等)
  M3/                    # 已处理归档 -- 修复完的 issue 文档(含 fix_result)
  M4/                    # 待修复队列 -- AI 分析发现的可自动修复问题
  archive/               # 已分析的 M1/M2 数据归档
    M1/                  # 已分析的 JSON 数据
    M2/                  # 已分析的截图
```

## 脚本详细说明

### 脚本1 -- 环境检查 / 增量更新

**依赖**: `docs/autotest/last_commit.txt`

**任务**:
1. 读取 `last_commit.txt` 中的 git commit hash
2. `git diff <old_hash>..HEAD` 检查自动对局相关代码是否有改动
3. 如有改动 -> 更新自动对局系统逻辑(autoplay 策略/测试参数等)
4. 更新 `last_commit.txt` 为当前 commit hash
5. 无改动 -> 跳过, 直接进入脚本2

### 脚本2 -- 自动对局 (Runner)

**依赖**: 无(直接运行游戏)
**实现**: `general-purpose` subagent, worktree 隔离

**任务**:
1. `go build cmd/autoplay/main.go` 编译自动对局二进制
2. 按策略执行对局:
   - 第1轮: `--sweep` 全组合扫描(~30 场) -- 建立基线
   - 后续轮: `--targeted` 只跑上轮失败场景(5-15 场) -- 验证修复
   - 最终轮: `--sweep` 全量回归(~30 场) -- 确认无回归
3. 产出写入对应目录:
   - JSON 数据 -> `docs/autotest/M1/` (波次统计/DPS/金币/泄漏/FPS/异常等)
   - 截图 -> `docs/autotest/M2/` (关键帧/异常画面)
4. 对局结束后退出

**超时**: 单轮 10 分钟(turbo 10x 速度, ~30 场约 5 分钟完成)

### 脚本3 -- AI 分析 (Analyst)

**依赖**: `M1/` 和 `M2/` 下的 JSON + 截图
**实现**: `general-purpose` subagent (多模态, 可读图)

**任务**:
1. 从 `M1/`、`M2/` 获取本轮产出
2. 读取 `M3/` (已处理归档) 了解历史修复情况, 避免重复报告
3. 结合项目当前代码状态进行多维分析
4. 发现新问题 -> 写入 `M4/`
5. 未发现新问题 -> `M4/` 为空, 本轮闭环结束

**分析维度**:

| 维度 | 内容 | 判定标准 |
|------|------|----------|
| **平衡性** | 各塔胜率/金币效率/DPS 曲线 | 胜率 <30% 偏弱, >90%(困难) 偏强 |
| **敌人** | 泄漏率/存活时长/特殊机制触发 | 太短=太简单, 太长=拖沓 |
| **异常分类** | anomalies[] 按类型分组/去重/分级 | CRITICAL > HIGH > MEDIUM > LOW |
| **覆盖缺口** | 未测试的能力/敌人/管线步骤 | 建议补充测试场景 |
| **视觉检查** | 截图多模态分析 | UI 重叠/特效残留/布局错位 |

**M4 issue 格式** (JSON):

```json
{
  "id": "BAL-001",
  "category": "balance | anomaly | coverage | visual",
  "severity": "CRITICAL | HIGH | MEDIUM | LOW",
  "title": "Laser tower overpowered on map_03",
  "detail": "描述...",
  "evidence": {
    "sessions": ["相关会话ID"],
    "screenshots": ["截图文件名"],
    "metrics": {}
  },
  "suggested_fix": {
    "type": "config | code",
    "file": "目标文件路径",
    "description": "修复建议"
  }
}
```

### 脚本4 -- 自动修复 (Fixer)

**依赖**: `M4/` 下的待修复文档
**实现**: `general-purpose` subagent
**分支**: `claude/autoplay-fix` (首次运行时创建)

**任务**:
1. 从 `M4/` 取出问题(按严重度降序)
2. 逐个修复:
   - config 类: 修改 JSON 配置 + 校验语法
   - code 类: 修改代码 + `make lint`
3. `make build` + `make test` 验证
   - 通过 -> git commit, 归档至 `M3/`
   - 不通过 -> 继续修 build 错误直到通过; 实在修不掉 -> 回滚本文件改动, 跳过此 issue
4. 完成后回到脚本2 进行回归验证

**安全规则**:

| 规则 | 说明 |
|------|------|
| 禁止破坏性 git 操作 | 不用 `reset --hard` / `checkout --` / `clean` |
| 测试门禁 | `make test` 必须通过才能 commit |
| 范围限制 | 只修改 issue 引用的文件 |
| 文件上限 | 每轮最多修改 5 个文件 |
| 行数上限 | 每文件最多改 50 行 |
| 失败回滚 | 只回滚 Fixer 自己刚改的文件 |
| 分支隔离 | 所有改动在 `claude/autoplay-fix`, 不碰当前分支 |

## 编排器 (Orchestrator)

**实现**: Claude Code `CronCreate`, 7 分钟间隔

**状态机**:

```
IDLE  <-- 初始 / 手动触发
  | /autoplay-start
  v
RUN  -- 启动 Runner (脚本2)
  | Runner 完成
  v
ANALYZE  -- 启动 Analyst (脚本3)
  | Analyst 完成
  v
CHECK  -- M4 为空 || 轮次 >= 上限?
  |            \
  | 否          是
  v              v
FIX  -- 启动   DONE -- 生成
  | Fixer       summary.md
  | (脚本4)
  v
RUN  (下一轮)
```

**收敛判定**:
- `M4/` 为空 (无新问题)
- 或达到最大轮次 (默认 5 轮)

**防死循环**:
- 同一问题连续 N 轮修不掉 -> 标记跳过, 继续下一个
- 看门狗: 15 分钟无进展 -> 强制推进到下一阶段

**状态持久化**: `docs/autotest/state.json`

```json
{
  "current_round": 2,
  "max_rounds": 5,
  "phase": "ANALYZE",
  "started_at": "2026-03-31T14:00:00Z",
  "rounds": [
    {
      "round": 1,
      "runner_status": "done",
      "analyst_status": "done",
      "fixer_status": "done",
      "issues_found": 5,
      "issues_fixed": 3,
      "critical_remaining": 1
    }
  ]
}
```

## 典型时间线

| 时间 | 阶段 | 耗时 |
|------|------|------|
| T+0 | 第1轮: Runner (30 场 @10x) | ~5 min |
| T+7 | 第1轮: Analyst (读 30 JSON + 截图) | ~3 min |
| T+14 | 第1轮: Fixer (修 3-5 个问题) | ~4 min |
| T+21 | 第2轮: Runner (定向 10 场) | ~3 min |
| T+28 | 第2轮: Analyst | ~2 min |
| T+35 | 第2轮: Fixer | ~3 min |
| ... | 第3-5轮或收敛 | ... |
| T+~60 | 最终总结 | |

总计: **约 60 分钟** 完成 5 轮自主优化。

## 手动操作

```
/autoplay-start    # 创建分支 + 初始化 state.json + 创建 cron + 启动第一轮
/autoplay-status   # 查看当前轮次/阶段/问题统计
/autoplay-stop     # 取消 cron + 生成已完成轮次的总结报告
```

## 最终总结报告

闭环结束时生成 `docs/autotest/summary.md`:

```markdown
# AutoPlay 优化总结

## 运行概况
- 轮次: 3 (第3轮收敛)
- 总对局: 70 场 (30 + 25 + 15)
- 总耗时: 42 分钟
- 分支: claude/autoplay-fix (4 commits)

## 问题发现与修复
| ID | 类别 | 严重度 | 状态 | 修复方案 |
|----|------|--------|------|----------|
| BAL-001 | 平衡 | HIGH | 已修复 | Laser 伤害 -15% |
| BUG-001 | 异常 | CRITICAL | 已修复 | 拐角吸附容差 |

## 覆盖率
- 塔: 8/8 (100%)
- 敌人原型: 13/13 (100%)
- 能力: 25/27 (93%)

## 平衡调整
| 目标 | 字段 | 修改前 | 修改后 | 原因 |
|------|------|--------|--------|------|
| laser.levels[1] | damage | 45 | 38 | map_03 extreme 过强 |

## 建议 (未自动修复)
1. 考虑增加 aura_dot 攻击方式的塔
2. silenceZone 能力从未触发 -- 检查是否有塔配置了它
```

## 配置

`docs/autotest/config.json` (在 /autoplay-start 时创建):

```json
{
  "max_rounds": 5,
  "cron_interval_min": 7,
  "runner": {
    "round1_strategy": "sweep",
    "subsequent_strategy": "targeted",
    "final_strategy": "sweep",
    "session_timeout_min": 10,
    "turbo_speed": 10
  },
  "analyst": {
    "balance_win_rate_low": 0.3,
    "balance_win_rate_high": 0.9,
    "anomaly_dedup_threshold": 2
  },
  "fixer": {
    "max_files_per_round": 5,
    "max_lines_per_file": 50,
    "test_command": "make test",
    "lint_command": "make lint",
    "branch": "claude/autoplay-fix"
  },
  "convergence": {
    "max_critical": 0,
    "max_total_issues": 2
  }
}
```

## 不做的事 (YAGNI)

- 不做 Web UI 监控面板
- 不做 Slack/webhook 通知
- 不做并行多分支修复
- 不做 ML 策略优化
- 不自动创建 PR (人工审查分支后手动合并)
- 不跨仓库改动 (仅限本项目)
