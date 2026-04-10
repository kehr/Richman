# Step 10: Frontend Holding Schedule Section

**依赖：** Step 7（i18n）+ Step 8（hooks）
**可与 Step 9 并行**
**设计依据：** TRD §前端组件树、PRD §入口 B

## 任务目标

在持仓详情页的「分析元信息」侧边栏中新增持仓级调度覆盖控件，并将「下次分析时间」改为读取后端返回的 `nextAnalysisAt`。

## 涉及文件

- 修改：`frontend/src/pages/decision-cards/components/MetaSidebar.tsx`
- 创建：`frontend/src/pages/decision-cards/components/HoldingScheduleSection.tsx`

## 执行步骤

- [ ] 阅读 `MetaSidebar.tsx` 全文，找到：下次分析时间的渲染位置、当前使用 `computeNextAnalysisTime` 的调用代码、`holdingId`（或等效 prop）的传入方式
- [ ] 创建 `HoldingScheduleSection.tsx`：
  - 调用 `useHoldingSchedule(holdingId)` 获取覆盖设置和 `nextAnalysisAt`
  - 渲染「分析频率」Select：选项为「跟随市场默认」+ 全部频率选项（从 `useScheduleSettings` 读取市场名称辅助显示当前继承值）；值变化时调用 `useUpdateHoldingSchedule`
  - 渲染「分析窗口」Select：选项为 `follow / pre / post / both`，i18n key 用 `schedule.holdingOverride.windowOptions.*`
  - `nextAnalysisAt` 从 hook 返回结果中读取并格式化显示
- [ ] 修改 `MetaSidebar.tsx`：
  - 在「下次自动分析」区域替换 `computeNextAnalysisTime` 调用为 `HoldingScheduleSection` 的 `nextAnalysisAt`
  - 将 `HoldingScheduleSection` 嵌入「分析元信息」块中（参照 TRD 组件树位置）
  - 保留 `computeNextAnalysisTime` 作为 `nextAnalysisAt` 为 null 时的 fallback
- [ ] 执行 `pnpm lint:all` 通过
- [ ] `git add frontend/src/pages/decision-cards/components/ && git commit -m "feat(decision-card): add holding schedule override section in meta sidebar"`

## 验证标准

- `pnpm lint:all` 通过
- `MetaSidebar` 无 TypeScript 错误
- `nextAnalysisAt` 为 null 时 fallback 到客户端计算（不崩溃）
- `HoldingScheduleSection` 频率选项列表与 `GlobalFrequencySelector` 的选项一致
