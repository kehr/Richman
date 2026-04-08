# Step 06 安装 framer-motion 与测试环境 mock

## 任务目标

安装 framer-motion 作为前端依赖，mock `window.matchMedia` 到 vitest setup 让 `useReducedMotion` 在 jsdom 里正常工作，为后续所有使用动画的 step 扫清障碍。

## 涉及文件

修改：
- `frontend/package.json`（新增 framer-motion 依赖）
- `frontend/pnpm-lock.yaml`（pnpm 自动更新）
- `frontend/src/test/setup.ts`（新增 matchMedia mock）

## 设计依据

- PRD §5 依赖变更
- PRD §6.4 测试环境 setup
- PRD 附录 D Pass 4 Pre-mortem bug 5：`useReducedMotion` 在 jsdom 返回 undefined 导致测试挂掉

## 实施要点

- `cd frontend && pnpm add framer-motion` 安装
- 确认 lock file 更新，确认版本写入 `dependencies` 而非 `devDependencies`
- `test/setup.ts` 新增 `Object.defineProperty(window, "matchMedia", ...)`，mock 返回 `{ matches: false, media, onchange, addListener, removeListener, addEventListener, removeEventListener, dispatchEvent }`，所有方法用 `vi.fn()` 填充
- 不要 touch eat barrel（framer-motion 不是 antd，直接从 `"framer-motion"` 导入即可）
- 不要修改 dependency-cruiser 配置（framer-motion 是第三方库）

## 验证标准

1. `cd frontend && pnpm lint:all` 通过
2. `pnpm test -- --run` 通过，无 matchMedia 相关警告
3. `pnpm build` 成功；onboarding chunk 新增 framer-motion 体积
4. 临时写一个测试用例 `import { useReducedMotion } from "framer-motion"; const r = useReducedMotion();` 在 jsdom 环境能正常 render 不报错
5. 主 bundle（非 onboarding）体积无明显增长（framer-motion 只随 onboarding lazy chunk 加载）

## 依赖说明

无硬依赖，可以与 step05 并行。但 step07 起所有前端 step 都依赖本 step 的 framer-motion 可用性。
