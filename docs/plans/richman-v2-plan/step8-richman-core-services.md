# Step 8: richman v2 Core Services

> Phase 3 | 并行组 R5 (可与 Step 9, 10 同时执行) | 前置: Step 7

## 任务目标

实现 richman v2 四个核心 service：MarketService（Market Overview + 标的详情聚合 + percentileLabel）、BriefingService（投研简报聚合）、FeedbackService（用户反馈 CRUD）、持仓级分析完整流程（v2_holding.go），以及 ComputeConcentration 集中度计算和现有 Service 扩展。

## 涉及文件

### 创建

- `backend/internal/service/market/service.go` -- MarketService
- `backend/internal/service/briefing/service.go` -- BriefingService
- `backend/internal/service/feedback/service.go` -- FeedbackService
- `backend/internal/service/analysis/v2_holding.go` -- 持仓级分析完整流程

### 修改

- `backend/internal/service/user_settings/` (或 user/) -- UserService 新增 UpdateRiskPreference, UpdateEmailPush
- `backend/internal/service/notification/` -- NotificationService 新增 SendBroadcast

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| MarketService.GetOverview | SS4 Market Overview | richman SS5.2 |
| MarketService.GetAssetDetail | SS5 标的详情 | richman SS5.2 |
| percentileLabel 计算 | SS3.4 历史分位 | richman SS5.2 |
| BriefingService.GetBriefing | SS6 投研简报 | richman SS5.3 |
| FeedbackService.Create | SS6.3 反馈机制 | richman SS5.4 |
| 持仓级分析流程 | SS8 执行计划 | richman SS5.5 |
| 幂等防护 (sync.Map TryLock) | - | richman SS5.5 |
| ComputeConcentration | SS8.2 集中度 | richman SS16 |
| UpdateRiskPreference | SS7.6 风险偏好 | richman SS5.6 |
| UpdateEmailPush | SS10.2 退订 | richman SS7.4.1 |
| SendBroadcast | SS10 推送 | richman SS5.6 |

## 关键约束

- MarketService percentileLabel 使用进程内 TTL 缓存（1 小时），冷启动 <30 天不显示
- BriefingService 步骤 1-4 的 DB 查询用 errgroup 并行执行
- 持仓级分析 7 步流程中步骤 1-5 用 errgroup 并行
- 持仓级分析幂等防护使用 `sync.Map` key = `userID:holdingID`
- FeedbackService 校验 rating 仅接受 "helpful" / "not_helpful"
- ComputeConcentration 三级阈值：red(>30%) / orange(>20%) / blue(>10%)
- 不修改 v1 分析 service，新逻辑在 v2_holding.go 中

## 验证标准

- [ ] `cd backend && make check` 通过
- [ ] MarketService 可实例化并调用 GetOverview（需 DB mock 或集成测试）
- [ ] BriefingService 可聚合持仓 + 分析 + 决策卡片
- [ ] FeedbackService 拒绝非法 rating 值
- [ ] 持仓级分析幂等锁正常工作（并发调用第二个返回 409）
- [ ] ComputeConcentration 三级阈值返回正确
- [ ] UserService.UpdateRiskPreference 校验三个合法值

## 变更点清单覆盖

D3.1-D3.5 (5), D3.10-D3.11 (2), D3.18 (1), D4.1-D4.3 (3) = **11 项**

注：D3.12-D3.17 在 Step 9, D3.6-D3.9 在 Step 10
