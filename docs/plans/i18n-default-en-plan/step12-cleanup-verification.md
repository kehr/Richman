# Step 12: 清理 + 验证

## 任务目标

删除旧 i18n provider 和 JSON 文件、清理废弃 import、执行全量验证确认 PRD 成功标准全部达成。

## 涉及文件

删除：
- 删除: `frontend/src/domain/i18n/provider.tsx`
- 删除: `frontend/src/domain/i18n/en.json`
- 删除: `frontend/src/domain/i18n/zh.json`
- 删除: `frontend/src/domain/i18n/` 目录（如已为空）

修改（如残留旧 import）：
- 检查: 全局搜索 `from "@/domain/i18n/provider"`，确保零引用

新增（预留脚本入口）：
- 修改: `frontend/package.json`（scripts 加 `"i18n:check"` 占位）

## PRD/TRD 引用

- TRD §13（废弃文件清单）
- TRD §14（事后工具预留脚本入口）
- PRD §10（全部 13 条成功标准）

## 验证标准

逐条对照 PRD §10 的 13 条成功标准：

- [ ] 1. 默认英文界面（navigator.language 为 en-* 或未设置时）
- [ ] 2. navigator.language 为 zh-* 的首访看到中文
- [ ] 3. Settings Radio 切换立即生效
- [ ] 4. Sidebar Globe Dropdown 切换立即生效
- [ ] 5. 两入口选中态永远同步
- [ ] 6. 刷新后语言偏好持久
- [ ] 7. 老 localStorage `richman_locale` 被继承
- [ ] 8. `rg '[\u4e00-\u9fff]' frontend/src --type tsx` 只剩 JSON 资源 + 语言选项 label（"中文"）+ 测试固定数据
- [ ] 9. `pnpm lint:all` 通过
- [ ] 10. `pnpm test` 通过
- [ ] 11. Help 页切语言 + deep link + IntersectionObserver 正常
- [ ] 12. AntD DatePicker 月份 / Pagination / Form 默认消息切 zh 后全中文
- [ ] 13. 数字 / 日期 / 货币格式化随语言变化

补充验证：

- [ ] `from "@/domain/i18n/provider"` 零引用
- [ ] `domain/i18n/` 目录已删除
- [ ] `pnpm build` 成功（production build 无编译错误）
- [ ] package.json scripts 有 `"i18n:check"` 占位

## 依赖

- Step 1-11 全部完成

## 实施注意

- 删除 `domain/i18n/provider.tsx` 前必须确认全局零引用（`rg "domain/i18n/provider" frontend/src`）
- 删除后如果 `domain/i18n/` 目录为空，一并删除目录
- `pnpm build` 是 PRD 未列但必须通过的健康检查（production build 比 dev 更严格）
- 如果验证过程中发现遗漏的中文，在本 step 内修复并补充对应 JSON key，不另开 step
- 最终 commit 前再跑一遍 `rg '[\u4e00-\u9fff]' frontend/src --type tsx` 确认无遗漏
