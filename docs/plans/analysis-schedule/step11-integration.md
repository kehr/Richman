# Step 11: Integration & Final Wiring

**依赖：** Step 4（scheduler）+ Step 5（handlers）+ Step 9（ScheduleTab）+ Step 10（HoldingSection）
**设计依据：** TRD 全文、PRD §状态空间

## 任务目标

将 ScheduleTab 注册到 SettingsPage，确认 eat barrel 导出 TimePicker，执行完整 lint，提交 PRD/TRD/Plan 文档，准备 ff-merge。

## 涉及文件

- 修改：`frontend/src/pages/settings/SettingsPage.tsx`（注册 schedule tab）
- 修改：`frontend/src/ui-kit/eat/index.ts`（如 TimePicker 未导出则添加）
- 修改：`frontend/src/features/schedule/index.ts`（确认 barrel 完整）
- 确认：`backend/cmd/server/main.go`（schedule 路由已注册）

## 执行步骤

- [ ] 检查 `eat/index.ts` 是否有 `TimePicker` 导出：`grep TimePicker frontend/src/ui-kit/eat/index.ts`；若无则在 antd 导出块中添加 `TimePicker`
- [ ] 修改 `SettingsPage.tsx`，在 `items` 数组中新增 schedule tab：
  - key: `"schedule"`
  - label: `t("tabs.schedule")`
  - icon: `<CalendarClock size={14} />` from `lucide-react`（验证 lucide-react 中 CalendarClock 是否存在：`node -e "const {CalendarClock}=require('lucide-react');console.log(!!CalendarClock)"` 若不存在改用 `<Clock size={14} />`）
  - content: `<ScheduleTab />`
  - 在 `TAB_KEYS` 常量中添加 `"schedule"`
- [ ] 执行 `cd frontend && pnpm lint:all` 全量验证
- [ ] 执行 `cd backend && make check` 全量验证
- [ ] 将 PRD/TRD/Plan 文档一并 commit：`git add docs/ && git commit -m "docs: add analysis-schedule PRD, TRD, and implementation plan"`
- [ ] 在 worktree 内执行 `git fetch origin && git rebase origin/main`，解决冲突
- [ ] 切回主仓库，执行 `git merge --ff-only feat/analysis-schedule`
- [ ] 执行 `git push origin main`
- [ ] 清理：`git worktree remove .claude/worktrees/analysis-schedule && git branch -d feat/analysis-schedule`

## 验证标准

- `pnpm lint:all` 全量通过（无 Biome/tsc/depcruiser 错误）
- `make check` 全量通过（无 lint/build 错误）
- 设置页出现「调度策略」第六个 Tab（类型正确、可渲染）
- 持仓详情页分析元信息侧边栏出现频率和窗口覆盖控件
- ff-merge 成功，push 到 origin/main
