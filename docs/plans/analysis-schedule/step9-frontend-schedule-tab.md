# Step 9: Frontend ScheduleTab Components

**依赖：** Step 7（i18n）+ Step 8（hooks）
**可与 Step 10 并行**
**设计依据：** TRD §前端组件树、PRD §入口 A

## 任务目标

实现设置页「调度策略」Tab 的所有组件。

## 涉及文件

- 创建：`frontend/src/pages/settings/tabs/ScheduleTab.tsx`
- 创建：`frontend/src/pages/settings/tabs/schedule/GlobalFrequencySelector.tsx`
- 创建：`frontend/src/pages/settings/tabs/schedule/MarketWindowCard.tsx`
- 创建：`frontend/src/pages/settings/tabs/schedule/WindowToggleRow.tsx`
- 创建：`frontend/src/pages/settings/tabs/schedule/HKPlaceholderCard.tsx`

## 执行步骤

- [ ] 查看 `frontend/src/pages/settings/tabs/AITab.tsx` 作为同层 Tab 组件参考，理解布局结构和 i18n 使用方式
- [ ] 创建 `WindowToggleRow.tsx`：单行「开关 + 标签 + 时间按钮」，props: `{ enabled, time, isCustom, label, hint, onToggle, onTimeChange, onReset, disabled?, timeRange }`；时间按钮点击触发 TimePicker（使用 `TimePicker` from `@/ui-kit/eat`，验证 eat 中是否有 TimePicker，无则从 antd 导出），5 分钟步进；`isCustom=true` 时显示蓝色已修改状态，有重置按钮
- [ ] 创建 `MarketWindowCard.tsx`：展示单市场（A股/美股）的频率覆盖 + 两个 `WindowToggleRow`；props: `{ market, settings, onUpdate }`；频率下拉含「跟随全局」选项
- [ ] 创建 `GlobalFrequencySelector.tsx`：6 个频率选项的 Grid 选择器（每个窗口/每日/每两日/每三日/每周/自定义）；自定义时显示 InputNumber（1-30）；props: `{ value, customDays, onChange }`
- [ ] 创建 `HKPlaceholderCard.tsx`：灰色虚线卡片，显示「港股」标题 + 「规划中」Badge + 时间说明；不可交互
- [ ] 创建 `ScheduleTab.tsx`：组合上述组件，调用 `useScheduleSettings` + `useUpdateScheduleSettings`；保存时调用 mutation；显示 Ant Design `message.success`
- [ ] 执行 `pnpm lint:all` 验证通过
- [ ] `git add frontend/src/pages/settings/tabs/schedule/ frontend/src/pages/settings/tabs/ScheduleTab.tsx && git commit -m "feat(settings): add schedule tab components"`

## 验证标准

- `pnpm lint:all` 通过（含 tsc）
- `ScheduleTab` 能渲染（类型正确，无运行时断言错误）
- `GlobalFrequencySelector` 自定义选项展开后显示 InputNumber
- `WindowToggleRow` 关闭态整行降低透明度且时间不可点击
