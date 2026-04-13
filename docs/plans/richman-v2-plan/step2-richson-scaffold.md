# Step 2: richson Project Scaffold + DB Layer

> Phase 1 | 并行组 R1 (可与 Step 1, 3 同时执行) | 无前置依赖

## 任务目标

从零创建 richson Python 服务的项目骨架：FastAPI 应用入口、pydantic-settings 配置、Alembic 迁移框架、SQLAlchemy 模型定义、Pydantic request/response schemas、4 张 rs_* 表的 Alembic 迁移脚本和 seed 数据。

## 涉及文件

### 创建（richson/ 目录全部为新建）

- `richson/pyproject.toml`
- `richson/Dockerfile`
- `richson/.env.example`
- `richson/alembic.ini`
- `richson/alembic/env.py`
- `richson/alembic/versions/001_init_schema.py` -- rs_* 四张表 + seed
- `richson/src/richson/__init__.py`
- `richson/src/richson/main.py` -- FastAPI app 入口（仅注册 router + middleware + lifespan）
- `richson/src/richson/config.py` -- pydantic-settings
- `richson/src/richson/db/__init__.py`
- `richson/src/richson/db/models.py` -- SQLAlchemy 2.0 models
- `richson/src/richson/db/repository.py` -- CRUD 操作
- `richson/src/richson/schemas/__init__.py`
- `richson/src/richson/schemas/jobs.py`
- `richson/src/richson/schemas/analysis.py`
- `richson/src/richson/schemas/market.py`
- `richson/src/richson/schemas/events.py`
- `richson/src/richson/schemas/common.py`

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| 项目结构 + 技术栈 | SS3.1 架构设计 | richson SS3.1/SS3.2 |
| 进程模型 | SS3.1 (richson 职责) | richson SS3.3 |
| rs_asset_analyses 表 | SS3.4 分析结果 | richson SS6.2 |
| rs_asset_analysis_dimensions 表 | SS3.4 维度明细 | richson SS6.2 |
| rs_analysis_jobs 表 | SS3.1 异步任务 | richson SS6.2 |
| rs_event_alerts 表 | SS3.6 事件雷达 | richson SS6.2 |
| rs_dimension_definitions 表 | SS3.3 权重配置 | richson SS6.2 |
| 四维权重 seed 数据 | SS3.3 黄金模型 | richson SS6.2 |
| Pydantic schemas | SS5 API 契约 | richson SS5.1-SS5.4 |
| config 环境变量 | - | richson SS12.2 |

## 关键约束

- Python >= 3.12, FastAPI >= 0.115, SQLAlchemy 2.0 + asyncpg
- rs_* 表序列 RESTART WITH 100000（与 richman 序列空间隔离）
- rs_event_alerts + rs_asset_analysis_dimensions 需注意序列 RESTART WITH（已知问题 SS21.4）
- rs_analysis_jobs 需 partial unique index 防止同资产重复 job
- seed 数据: gold_v1.0 四维权重（0.30/0.25/0.25/0.20）
- main.py 仅做 app 构建 + router 注册，不实现业务端点（端点在 Step 6 实现）

## 验证标准

- [ ] `cd richson && uv sync` 依赖安装成功
- [ ] `alembic upgrade head` 创建全部 rs_* 表
- [ ] `alembic downgrade base` 回滚干净
- [ ] rs_dimension_definitions 表有 4 条 gold_v1.0 seed 数据
- [ ] `python -c "from richson.main import app; print(app.title)"` 正常输出
- [ ] `python -c "from richson.db.models import AssetAnalysis"` 无导入错误
- [ ] Pydantic schemas 可正常实例化（基本烟雾测试）

## 变更点清单覆盖

C1.1-C1.6 (6), B1-B6 (6), C6.1-C6.3 (3) = **15 项**
