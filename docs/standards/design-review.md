# 设计完备性审查规范

本规范定义任何非平凡设计在进入编码或写 design doc 之前必须执行的强制检查流程。所有涉及状态机、跨层改动、异步副作用、生命周期钩子、数据库字段变更的任务都必须走完本规范的 5 个 Pass，才能把设计呈现给用户或落地成文档。

## 适用范围

满足以下任意一条即为「非平凡设计」，必须执行完整的 5 个 Pass：

- 引入新的布尔标记、状态字段、生命周期阶段
- 修改任何跨前后端契约（API、DTO、Schema）
- 触及 react-query 缓存、路由守卫、导航跳转、mutation 时序
- 修改含有既有 useEffect 副作用或 ref-based 单次触发逻辑的组件
- 需要修改迁移文件或既有 SQL 语句
- 需要引入新的用户可选择的退出/跳过/dismiss 交互
- 文件清单超过 3 个，或涉及多个独立模块

只满足以下条件可以走简化流程（跳过 Pass 1、Pass 3）：

- 纯样式调整、仅影响单个组件的文案/颜色/间距
- 修复 lint/format/type 错误
- 补一个已有接口的测试

## 执行时机

执行顺序必须是：

```
brainstorming skill 收集需求
    ↓
列出初版设计（§1..§N）
    ↓
【强制】Pass 1 → Pass 5 审查
    ↓
把发现的 gap 呈现给用户并迭代修复
    ↓
写 design doc 到 docs/superpowers/specs/ 或 docs/prds/
    ↓
进入 writing-plans 或 executing-plans 阶段
```

不允许的顺序：

- 在 Pass 完成之前直接写 design doc
- 把 gap 审查留到用户主动要求「检查一下有没有漏」才做
- 把 Pass 留到编码阶段再补

## Pass 1 状态空间枚举

目标：确保设计涵盖所有可能的状态组合，不只是 happy path。

对设计引入的每一个布尔标记、枚举字段、可选生命周期阶段，列出维度并枚举所有组合。对 N 个布尔维度共 2^N 个组合逐个分类：

- **Valid target** —— 设计明确支持用户/系统停留在这个状态
- **Transient** —— 允许在转换过程中短暂存在，必须在有限时间内收敛
- **Forbidden** —— 设计必须通过原子更新、约束或校验阻止这个组合出现

示例：三个维度 `{completed, skipped, has_holdings}` 共 8 个组合。必须明确：

- `{completed=true, skipped=true, *}` → Forbidden，通过 `MarkCompleted` SQL 同时清 `skipped_at` 保证互斥
- `{completed=false, skipped=true, has_holdings=0}` → Valid target，Dashboard 必须能渲染 nudge + empty hero 并存
- `{completed=true, skipped=false, has_holdings=0}` → Valid target，Dashboard 仅渲染 empty hero，无 nudge

Pass 1 的产物是一张状态表，必须附在设计呈现中。没有状态表的设计不允许进入下一个 Pass。

## Pass 2 文件不变量提取

目标：在修改任何既有文件之前，先明确记录它当前的隐含契约，防止新改动无意中违反。

对「要修改的文件」清单里的每一个文件：

1. Read 整个文件，不允许只读概要
2. 写一行「现有契约 X」—— 用一句话描述这个文件当前依赖什么前提才能正确工作
3. 写一行「我的改动对 X 的影响」—— 明确声明保留、修改或替换该契约

常见的需要显式提取的契约类型：

- useEffect 的依赖前提（是否假设组件不会 remount）
- useRef 的单次触发假设（是否假设组件生命周期单次）
- SQL 的幂等性（是否依赖 `COALESCE` 或 `WHERE` 条件实现）
- React Query 的 query key 共享（是否依赖别处会被同时失效）
- 路由守卫的布尔逻辑（是否只检查单一字段）
- Early return 分支（是否依赖某个前提组件不会被渲染）
- 表单 onFinish 的副作用顺序（是否依赖 navigate 在 mutation 之后）

Pass 2 的产物是一张「文件 × 现有契约 × 改动影响」三列表，附在设计呈现中。文件清单没有契约分析列的不允许进入下一个 Pass。

## Pass 3 替代路径遍历

目标：除了主 happy path 之外，每条功能路径都必须列出至少 3 条替代路径并逐条验证设计是否 hold。

对每条主 happy path，必须列出并验证以下替代路径：

- **Back 导航** —— 用户在任意中间步骤点浏览器后退或界面回退按钮，组件 remount 时有没有副作用被重新触发？有没有表单状态丢失？
- **Retry after failure** —— 某个 mutation 失败后用户点重试，累积的 state 是否干净？是否会产生重复写入？
- **Cross-session resumption** —— 用户关闭标签页再重新进入，sessionStorage / localStorage / 后端持久状态是否一致？
- **Concurrent mutation** —— 两个 mutation 几乎同时触发，react-query 缓存、URL、后端是否收敛到一致状态？
- **Permanent dismissal** —— 任何「不再提示」「永久关闭」「永久跳过」选项都必须配一条 regret 回归路径。用户后悔了怎么回来？
- **Race between mutation and navigate** —— navigate 是否在 mutation 的缓存失效完成之前触发？是否可能导致守卫使用旧 status 反弹？

每条替代路径都要写明设计如何处理。没有答案的替代路径就是真 gap，必须回到设计。

## Pass 4 Pre-mortem

目标：逼自己列出「这个设计上线后最可能的 5 个 bug」并在呈现设计前修复。

执行方式：写完初版设计后停下来，想象这个功能已经上线两周，用户反馈了 bug。列出最可能的 5 个 bug，按严重度排序。对每一条：

1. 描述用户会看到什么现象
2. 回溯到设计的哪一条假设出了问题
3. 在设计里补上防御

如果 2 分钟内想不出 5 条，说明对设计的理解深度不够，必须回头补 Pass 1 或 Pass 2。不允许跳过或以「都挺稳的」结束。

常见的 bug 分类可用作扫描清单：

- 缓存陈旧（react-query 失效顺序、mutation onSuccess 里没 invalidate 相关 query）
- 导航回路（守卫 + state + mutation 的时序交错）
- 双触发（useEffect 空依赖 + remount、StrictMode 双调）
- 非原子 SQL（互斥字段分两条 UPDATE 写入）
- 视觉叠加（多个条件性组件同时满足时的布局冲突）
- Dismiss 后无 regret 路径
- Reset/rollback 漏清理关联字段
- 错误分支没反馈到 UI（try/catch 吞异常）

Pre-mortem 的产物是一张「潜在 bug × 根因 × 设计防御」表，附在设计呈现中。

## Pass 5 攻击自己的推荐项

目标：每个多选题的「推荐项」都必须通过自我反驳测试后才能呈现给用户。

对 AskUserQuestion 或类似选项的每一个「推荐项」，必须花 30 秒以上问自己：

- 这个选项下最容易被忽略的副作用是什么？
- 选这个选项的用户 3 个月后最可能的痛点是什么？
- 这个推荐隐含了什么假设，如果这个假设不成立会怎样？

如果自我反驳找出任何能落地的担忧，要么修正推荐项，要么在选项的 description 里把这个风险写清楚让用户知情。

不允许的做法：

- 把默认选项直接标「推荐」而不反驳
- 反驳发现问题但选择「应该问题不大」继续推荐
- 用「用户可以之后再改」作为推荐风险的借口

## 产物规范

执行完 Pass 1-5 后，设计呈现必须包含以下内容：

1. 初版设计（§1..§N）
2. 状态空间表（Pass 1 产物）
3. 文件契约影响表（Pass 2 产物）
4. 替代路径验证清单（Pass 3 产物）
5. Pre-mortem bug 表（Pass 4 产物）
6. 每个推荐项的自反驳说明（Pass 5 产物，内联在 AskUserQuestion 的 description 里或单独列出）
7. 由 Pass 审查发现并已修复的 gap 列表
8. 剩余待用户决策的 gap 列表

只有以上 8 项齐全时才能写 design doc。

## 反模式与自检

遇到以下念头时必须强制停下来执行相应的 Pass：

| 内心独白 | 必须立即执行 |
|---------|-------------|
| 这个文件我读过了不用再看 | Pass 2 |
| Happy path 很清楚剩下的是细节 | Pass 1 + Pass 3 |
| 这个功能挺简单的 | Pass 4 |
| 实现阶段再处理这个细节 | Pass 1..Pass 4 全部 |
| skill 流程说要往前走 | 允许暂停，skill 流程是脚手架不是合约 |
| 用户应该自己会想到 | Pass 5 |
| 我之前类似的设计没问题 | Pass 2 +Pass 3，不同上下文前提不同 |

## 例外情况

以下情况可以跳过 Pass 1-5：

- 纯文案修正
- 单个 CSS 属性调整
- 修 lint/format/type 错误
- 为已有接口补缺失的测试
- 把一个已有常量的值改为另一个值（不改语义）
- 文档修正

这些情况直接改即可。但一旦涉及任何 state / async / schema / cross-layer / lifecycle，立即回归完整 Pass。

## 背景

2026-04-09 Richman onboarding UX 设计过程中，初版设计遗漏了 5 个阻塞级 gap 和 6 个重要技术细节，全部在用户主动要求「检查下有没有错漏」时才浮出水面。每一个遗漏都可以通过上述 5 个 Pass 之一的机械性检查提前发现。本规范把这次事件的教训固化为流程，防止同类遗漏在后续任务中重现。
