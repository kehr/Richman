# Step 5: richson ADK Agents + Degradation Templates

> Phase 2 | 并行组 R2 (可与 Step 4 同时执行) | 前置: Step 2

## 任务目标

实现 richson 的 Google ADK Agent 层（Layer 2 research agent + Layer 3 interpretation/execution agents）和 LLM 降级时的中/英文文本模板。包含 agent 工厂函数、InMemoryRunner 驱动、LiteLlm 多 provider 支持和降级策略。

## 涉及文件

### 创建

**Agents：**
- `richson/src/richson/agents/__init__.py`
- `richson/src/richson/agents/research_agent.py` -- Layer 2: 信息检索 + 研判
- `richson/src/richson/agents/interpretation_agent.py` -- Layer 3: 解读文本生成
- `richson/src/richson/agents/execution_agent.py` -- Layer 3: 执行计划生成
- `richson/src/richson/agents/prompts/__init__.py`
- `richson/src/richson/agents/prompts/research.py` -- D1/D2/D3 各维度 prompt
- `richson/src/richson/agents/prompts/interpretation.py`
- `richson/src/richson/agents/prompts/execution.py`

**降级模板：**
- `richson/src/richson/templates/__init__.py`
- `richson/src/richson/templates/interpretation_zh.py`
- `richson/src/richson/templates/interpretation_en.py`

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| Layer 2 LLM 检索策略 | SS3.1 Layer 2 职责 | richson SS8.2 |
| Layer 3 文本生成 | SS3.1 Layer 3 职责 | richson SS8.3 |
| 执行计划生成 | SS8.1 条件分支执行计划 | richson SS8.4 |
| Agent 工厂 (create_agent + LiteLlm) | - | richson SS8.5 |
| InMemoryRunner 驱动 | - | richson SS8.5 |
| 降级策略（L2 跳过 / L3 模板） | SS3.1 降级原则 | richson SS8.6 + SS17 |
| LLM 输出验证（来源可追溯） | SS3.1 验证规则 | richson SS8.2 |
| Google ADK 正确 API | - | richson SS8.5 |

## 关键约束

- Google ADK 正确导入路径：`from google.adk.agents import Agent`，`Agent(name=..., model=..., instruction=..., tools=..., output_schema=...)`
- 使用 `InMemoryRunner` 驱动 agent 执行
- 使用 `LiteLlm` 实现多 provider（Claude API / OpenAI API）支持
- research_agent 对 D1/D2/D3 三个维度分别执行一次 LLM 调用（D4 技术位置纯量化，无 LLM 参与）
- LLM 调整上限 +/-15 分，单信源最高 +/-8 分
- 降级模板基于量化基础分生成结构化描述，不需要 LLM
- prompt 模板使用 Python f-string 或 Jinja2，接收维度数据作为上下文变量
- 所有 ADK API 名称必须先 grep google-adk 源码确认存在

## 验证标准

- [ ] `from google.adk.agents import Agent` 导入正常
- [ ] create_agent 工厂函数能创建 Agent 实例（无需真实 API key）
- [ ] 降级模板在给定固定输入时生成合理的中/英文文本
- [ ] prompt 模板无语法错误，变量占位符正确
- [ ] research_agent / interpretation_agent / execution_agent 定义正确，output_schema 与 Pydantic schema 对齐
- [ ] 所有模块 import 无错误

## 变更点清单覆盖

C4.1-C4.5 (5), C7.1-C7.3 (3) = **8 项**
