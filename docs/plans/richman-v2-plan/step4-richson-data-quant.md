# Step 4: richson Data Sources + Quant Engine (Layer 1)

> Phase 2 | 并行组 R2 (可与 Step 5 同时执行) | 前置: Step 2

## 任务目标

实现 richson 全部外部数据源 wrapper（FRED, yfinance, AKShare, Polymarket, CFTC COT, WGC, stooq）和进程内 TTL 缓存层，以及量化评分引擎的全部模块：四维指标计算器、维度评分 + 百分位、LLM 定性调整映射、综合置信度、支撑/阻力位、市场体制检测、事件监控、回撤计算、冲突检测、D4 ATR 动态权重。

## 涉及文件

### 创建

**数据源层：**
- `richson/src/richson/datasources/__init__.py`
- `richson/src/richson/datasources/fred.py`
- `richson/src/richson/datasources/yahoo.py`
- `richson/src/richson/datasources/akshare_client.py`
- `richson/src/richson/datasources/polymarket.py`
- `richson/src/richson/datasources/cot.py`
- `richson/src/richson/datasources/wgc.py`
- `richson/src/richson/datasources/stooq.py`
- `richson/src/richson/datasources/cache.py`

**量化引擎：**
- `richson/src/richson/core/__init__.py`
- `richson/src/richson/core/scoring.py` -- 维度评分 + 百分位
- `richson/src/richson/core/adjustment.py` -- LLM 定性->数值映射
- `richson/src/richson/core/confidence.py` -- 综合置信度计算
- `richson/src/richson/core/support_resistance.py` -- 支撑/阻力位
- `richson/src/richson/core/regime.py` -- 市场体制检测
- `richson/src/richson/core/event_monitor.py` -- Polymarket 事件概率变动监控
- `richson/src/richson/core/indicators/__init__.py`
- `richson/src/richson/core/indicators/d1_macro_rates.py`
- `richson/src/richson/core/indicators/d2_dollar_liquidity.py`
- `richson/src/richson/core/indicators/d3_structural_demand.py`
- `richson/src/richson/core/indicators/d4_technical_position.py`

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| 数据源清单 | SS3.2 数据获取 | richson SS3.1, SS10 |
| 进程内 TTL 缓存 | - | richson SS10 |
| 四维黄金模型 | SS3.3 黄金模型 | richson SS7.2 |
| 各维度子指标和权重 | SS3.3 | richson SS3.1 |
| 百分位标准化 | SS3.4 | richson SS7.2 |
| LLM 调整上限 +/-15 | SS3.1 Layer 2 | richson SS7.4 |
| 置信度计算 | SS3.4 | richson SS7.5 |
| 支撑/阻力位 | SS5.2.3 | richson SS7.6 |
| 市场体制 3 分类 | SS4.2.1 | richson SS5.3 |
| 事件概率监控 | SS3.6 | richson SS6.2 |
| 回撤计算 | SS5.2.3 | richson SS7.7 |
| 冲突检测 | SS3.4 | richson SS7.8 |
| D4 ATR 动态权重 | - | richson SS7.3 |
| stooq fallback | - | richson SS3.1 |

## 关键约束

- 所有第三方库 API 名称必须先从 pyproject.toml 安装后验证，禁止凭记忆直接写
- 数据源 wrapper 需处理网络超时和降级（返回 None / partial data）
- cache.py 使用 cachetools TTLCache 或等效方案，key 粒度到 asset_code + indicator
- 评分引擎模块须为纯计算函数（接收 DataFrame 输入，返回 dict 输出），不直接调用数据源
- 维度评分范围 0-100，权重总和 = 1.0
- LLM 调整值上限 +/-15，adjustment.py 需做裁剪

## 验证标准

- [ ] 每个数据源 wrapper 可独立运行（带 API key 时能获取数据，无 key 时优雅降级）
- [ ] scoring.py 单元测试：给定固定输入 DataFrame，输出可复现的分数
- [ ] regime.py 单元测试：VIX/T10Y2Y 边界值测试三种体制输出
- [ ] confidence.py 单元测试：数据覆盖 full/partial/degraded 三种情况
- [ ] 所有模块 import 无错误
- [ ] pyproject.toml 依赖声明包含全部引用的第三方库

## 变更点清单覆盖

C3.2-C3.14 (13), C5.1-C5.8 (8) = **21 项**
