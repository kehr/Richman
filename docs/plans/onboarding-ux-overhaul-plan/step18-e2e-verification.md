# Step 18 端到端验收与执行报告总结

## 任务目标

全量运行前后端 lint / test / build，手动执行 6 条主路径的冒烟测试，确认所有功能行为符合 PRD，补齐执行报告最终段落供用户验收。

## 涉及文件

修改：
- `docs/reports/onboarding-ux-overhaul-execution-report.md`（追加最终段落）

## 设计依据

- PRD §7 实施顺序 step 8
- PRD 附录 C Pass 3 的 6 条主路径
- 用户的最终验收标准

## 实施要点

### 全量自动化验证

- 前端：`cd frontend && pnpm lint:all` 必须全绿（Biome + tsc + dependency-cruiser）
- 前端：`pnpm test -- --run` 必须全绿
- 前端：`pnpm build` 必须成功
- 后端：`cd backend && go vet ./...` 必须通过
- 后端：`go build ./...` 必须成功
- 后端：`go test ./...` 必须全绿
- 任何失败必须在本 step 内修复或在执行报告中明确标注为未决项

### 手动冒烟 6 条主路径

1. **新用户完整走完 4 步**：注册 → welcome → 选 category → 录持仓 → 分析完成 → dashboard
2. **新用户 step 2 回退到 step 1 再前进**：验证 categories 保留
3. **新用户 step 3 跳过全流程**：点 header 右上「跳过引导」→ Modal 确认 → dashboard 看到 nudge
4. **从 nudge 重入**：点「开始引导」→ 跳 welcome → 中途点 sidebar 的 dashboard 链接 → 回到 dashboard 不被反弹（guard 因 skipped=true 放行）
5. **从 nudge 永久关闭 → 从 Settings 重入**：dashboard 点 nudge「不再提示」→ 进 Settings → 点「重新走一遍引导」→ Popconfirm → welcome
6. **键盘导航**：在 welcome 按 → 前进 → 在 categories 按 ← 回退 → 按 Esc 触发 skip Modal → 取消 → 在 input 里按 ← 不触发回退

### 执行报告最终段落

在 `docs/reports/onboarding-ux-overhaul-execution-report.md` 追加：
- 全部 18 个 step 的 commit SHA 汇总
- 自动化验证结果（lint / test / build）
- 6 条手动冒烟路径的结果（PASS / FAIL + 备注）
- 已修复问题列表
- 未处理的观察项（优先级低，不阻塞合并）
- 最终 verdict：`READY FOR MERGE` 或 `NEEDS REWORK`

## 验证标准

1. 所有自动化命令全绿
2. 所有 6 条手动路径冒烟通过（或明确记录失败原因）
3. 执行报告最终段落齐全
4. 用户口头或书面确认「可以合并」后方可进入 finishing-a-development-branch 阶段

## 依赖说明

前置：step01-17 全部完成
