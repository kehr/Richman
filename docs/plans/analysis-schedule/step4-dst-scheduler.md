# Step 4: DST Logic + Scheduler Rewrite

**依赖：** Step 3（service 层已实现）
**设计依据：** TRD §DST 感知逻辑、§Scheduler 重设计

## 任务目标

实现 NYSE DST 感知逻辑，并将 `scheduler.go` 从硬编码三条 cron 改为动态读取数据库配置。

## 涉及文件

- 创建：`backend/internal/service/schedule/dst.go`
- 修改：`backend/internal/service/analysis/scheduler.go`（重构，保留 `NewScheduler`/`Start`/`Stop` 签名，重写内部逻辑）

## 执行步骤

- [ ] 阅读现有 `backend/internal/service/analysis/scheduler.go` 全文，记录：现有 cron entry 的注册方式、`runJob` 的函数签名、如何拿到 userID 列表
- [ ] 创建 `dst.go`，实现：
  - `IsEDT(t time.Time) bool`：根据 NYSE 规则判断是否夏令时（3 月第二个周日 02:00 EST 至 11 月第一个周日 02:00 EST）
  - `USWindowTimes(isEDT bool) (preTime, postTime time.Time)`：返回当日美股默认盘前盘后时间（Asia/Shanghai）
  - `NextDSTTransition(now time.Time) time.Time`：返回下次 DST 切换时间（用于定时刷新）
- [ ] 修改 `scheduler.go`，重写 `Start()` 方法：
  - 调用 `ListActiveUserScheduleSettings` 加载所有用户配置
  - 按市场 × 窗口 × 用户维度注册 cron 条目，条目 ID 格式 `{userID}:{market}:{pre|post}`
  - 无配置用户退化到系统默认（保持原有行为）
  - 新增 `ReloadUser(userID int64)` 方法：移除该用户所有条目并重新注册
  - 在 DST 切换时间点注册一次性 cron 回调，自动更新 us_pre_custom=false / us_post_custom=false 的用户时间并重载
  - 保持 `runJob`/`processUserJob` 核心分析逻辑不变，仅改触发时机判断（加 frequency 间隔检查）
- [ ] 执行 `cd backend && make check` 确认编译通过
- [ ] `git add backend/internal/service/ && git commit -m "feat(scheduler): rewrite dynamic cron loading with DST awareness"`

## 验证标准

- `make check` 通过
- `IsEDT` 对已知日期（如 2025-07-01 → true，2025-01-01 → false）返回正确值（走读验证）
- `Start()` 调用后不 panic，日志显示已加载用户配置条目数
