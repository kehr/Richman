# Step 19: Deployment Config + Integration

> Phase 5 | 并行组 R10 (单独执行) | 前置: 全部 Steps 1-18 完成

## 任务目标

配置双服务部署：docker-compose 新增 richson 服务定义、两服务 healthcheck 配置、端口隔离（richson 仅 expose 不 publish）、环境变量模板完善。确保全部组件端到端可运行。

## 涉及文件

### 修改

- `docker-compose.yml` -- 新增 richson 服务 + healthcheck 配置
- `richson/.env.example` -- 完善（Step 2 创建的基础上补充生产变量）
- `backend/.env.example` -- 确认 v2 变量完整（Step 13 追加的基础上校验）

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| docker-compose richson 定义 | - | richson SS12.1 |
| richson .env.example | - | richson SS12.2 |
| richman .env.example 追加 | - | richman SS10.3 |
| richson 端口隔离 (expose) | - | richson SS21.1 |
| 两服务 healthcheck | - | richman SS22.2 / richson SS11.3 |

## 关键约束

- richson 端口使用 `expose`（不 publish），仅 docker 内部网络可访问
- richman healthcheck 依赖 richson healthcheck（级联检查）
- docker-compose 确保 richson 先于 richman 启动（depends_on + healthcheck）
- 两服务共享同一 PostgreSQL 实例
- richson 内部地址：`http://richson:8001`（docker 服务名解析）
- 生产环境 INTERNAL_API_KEY 通过 docker secrets 或环境变量注入

## 验证标准

- [ ] `docker-compose up -d` 两服务均启动成功
- [ ] richson 健康检查通过（docker-compose ps 显示 healthy）
- [ ] richman 健康检查通过
- [ ] richman 能访问 richson（GET http://richson:8001/health 从 richman 容器内）
- [ ] richson 端口从宿主机不可直接访问
- [ ] 两服务共享 PostgreSQL 且各自 migration 正常
- [ ] 前端 dev server 能通过 richman API 获取 v2 数据

## 端到端验证清单

以下为全链路冒烟测试，确认前后端全部串联：

- [ ] 访问 /market 展示标的卡片墙 + 体制判断条
- [ ] 点击黄金卡片进入 /market/GLD 详情页
- [ ] 分析 Tab 展示 K 线 + 维度面板
- [ ] 未登录执行 Tab 显示 Demo Plan + RegisterCTA
- [ ] 注册新用户（含邀请码 + disclaimer）
- [ ] 登录后导航栏切换为三项
- [ ] /briefing 展示持仓卡片（需先添加持仓）
- [ ] 触发持仓分析，job 轮询成功
- [ ] 设置页显示邀请码 + 邮件开关
- [ ] 邮件推送开关可切换

## 变更点清单覆盖

F1-F5 = **5 项**
