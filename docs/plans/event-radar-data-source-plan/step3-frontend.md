# Step 3: frontend 类型扩展 + 组件迁移 + asset-detail 集成 + i18n

## 目标

1. 扩展 EventDto 三个字段，与 backend / richson 对齐
2. 把 page-local `EventRadarSection` 迁移到 `features/event-radar/` 作为唯一来源，并增强为整行可点击 anchor
3. 资产详情页 event-calendar.tsx 复用同一个组件
4. i18n 双语 keys 同步

## 涉及文件

新建：
- `frontend/src/features/event-radar/event-radar-section.tsx`（从 page-local 迁移并增强）

修改：
- `frontend/src/features/event-radar/types.ts`（EventDto 加 sourceUrl / sourceName / releaseId）
- `frontend/src/features/event-radar/index.ts`（barrel 导出 EventRadarSection）
- `frontend/src/pages/market-overview/market-overview-page.tsx`（import 路径改为 `@/features/event-radar`）
- `frontend/src/pages/asset-detail/event-calendar.tsx`（重写为 `<EventRadarSection/>` 包装）
- `frontend/src/i18n/locales/zh/market.json`（新增 `openSourceTooltip` / `sourceLabel`）
- `frontend/src/i18n/locales/en/market.json`（同步英文）

删除：
- `frontend/src/pages/market-overview/components/event-radar-section.tsx`（迁移后移除）

## 设计依据

- PRD §4.1（三端 DTO 字段表）
- PRD §5.1（事件雷达条目展示与点击行为）
- PRD §5.2（sourceUrl 缺失时 fallback）
- PRD §5.3（资产详情页集成）
- PRD §5.4（i18n 增量表）
- PRD §6（数据流图）
- PRD §8.6（已修复 gap：组件统一来源、React key 稳定性、URL 安全防御）
- TRD §4.1（types.ts diff）
- TRD §4.2（EventRow 改造完整伪代码 + key 策略）
- TRD §4.3（barrel 导出）
- TRD §4.4（删除 page-local 组件）
- TRD §4.5（asset-detail event-calendar 重写）
- TRD §4.6（i18n keys）
- frontend.md（Pages+Features 架构，所有 ant-design 通过 ui-kit/eat barrel）

## 验证标准

每个文件改完立即跑：
- `cd frontend && pnpm lint:all`（Biome + tsc + dependency-cruiser）通过
- `cd frontend && pnpm build` 通过
- 手动 grep 确认 `pages/market-overview/components/event-radar-section.tsx` 全局已无引用，删除安全
- 本 step 内不做联调（依赖 Step 1 + Step 2 完成）

视觉验证（Step 4 联调时一并执行）：
- 行情概览页事件雷达整行可点击，hover 背景变浅，cursor=pointer
- 点击 FRED 事件新标签打开 `https://fred.stlouisfed.org/release?rid=N`
- 点击 Polymarket 事件新标签打开 `https://polymarket.com/event/<slug>`
- 资产详情页 risk-tab 下"近期事件"区域展示与行情概览页一致的事件列表
- 中英文切换，事件雷达提示文案 "Source: FRED" / "来源：FRED" 正确显示

## 依赖

- 无前置依赖（types.ts 扩展不依赖后端响应即可编译）
- 与 Step 1 / Step 2 完全独立，可并行执行
- 联调验证延后到 Step 4

## Commit 拆分

按 commit-hygiene 拆 3 个 commit：
1. `feat(frontend): extend event radar dto with source metadata` （types.ts）
2. `feat(frontend): unify event radar section under features module` （新建 features/.../section.tsx + barrel + 改 market-overview-page.tsx import + 删 page-local section）
3. `feat(frontend): reuse event radar in asset detail risk tab` （asset-detail/event-calendar.tsx + i18n 双语 keys）
