# Step 21 端到端验证与全量 lint

## 任务目标

最后一步收尾。跑通完整的用户动线确保所有 step 集成正确，运行全量 lint / test / build，修复所有遗留问题。

## 涉及文件

可能修改：
- 任何遗漏的文件
- 可能增加端到端测试脚本（可选，MVP 不强制）

## 设计依据

- PRD §1 - §10 全部章节作为验收标准
- TRD §1 - §9 全部技术约束

## 实施要点

### 9 条端到端动线检查

每条动线必须能从浏览器手动跑通，不出现 500 / 黑屏 / 数据不一致：

1. **新用户注册全流程**
   - 访问 / → 重定向 /login
   - 点"用邀请码注册"→ /register
   - 填邀请码 + 邮箱 + 密码注册成功
   - 自动跳 /onboarding/welcome
   - 走完 4 步 onboarding
   - 落到 /dashboard 看到首张决策卡（含变化徽章 = first_analysis）

2. **添加多个持仓后查看 Dashboard**
   - 在 Portfolio 添加 3 个不同类型的持仓
   - 回 Dashboard 点"重新分析"
   - 等待完成后看到 3 张卡，部分卡有变化徽章

3. **点决策卡进入详情页**
   - Dashboard 卡点击 → /decision-cards/:id
   - 5 区块 + 右侧 meta 栏全部正确渲染
   - 执行计划展开能看到所有步骤的 rationale

4. **截图批量导入持仓**
   - Portfolio 点"📷 截图批量导入"
   - 上传一张 mock 测试图（dev 环境用 mock vision provider 或真实 LLM）
   - 双栏对照修改 → 确认导入
   - 列表正确更新

5. **设置总资金后金额全局可见**
   - Settings → 账户 → 设置总资金 100000
   - 立即回 Dashboard 顶部 strip / 决策卡 / Portfolio 列表
   - 所有百分比位置都附带 ¥ 金额
   - 清空总资金后金额全部消失

6. **修改风险偏好后下次分析权重变化**
   - Settings → 账户 → 风险偏好 = 激进
   - Dashboard 重新分析
   - 在某张决策卡详情页观察权重微调 trace（催化剂权重应升高、位置降低）

7. **推送链接回流**
   - 模拟发一封测试邮件（或飞书 / 微信测试发送）
   - 复制邮件中的决策卡链接，登出后访问该链接
   - 跳到 /login?returnTo=...
   - 登录后正确回流到原决策卡详情页

8. **帮助页锚点跳转**
   - 决策卡详情页徽章旁边的 ? 图标 → 跳 /help#badge
   - 三维维度旁的 ? → /help#dimensions
   - 信心度旁的 ? → /help#confidence

9. **Onboarding 守卫边界**
   - 已完成 onboarding 的用户访问 /onboarding/welcome → 跳 /dashboard
   - 未完成的用户访问 /dashboard → 跳 /onboarding/welcome
   - dev 环境 Settings 重置 onboarding 后再访问 /dashboard → 重新进入 wizard

### 全量检查命令

- 后端：`cd backend && make check`（lint + test + build）
- 前端：`cd frontend && pnpm lint:all && pnpm build`
- 数据库：`make migrate-down` 至 005 然后 `make migrate-up` 至 008，验证迁移幂等

### 隐私守卫验证

- 用 grep 检查推送 adapter 的 render 函数参数类型，确认无 totalCapital / amount 字段
- 在 dev 模式（-tags debug）跑一次完整分析，确认 privacy_guard 没有报警
- 检查 LLM 请求日志，确认 prompt 上下文不含金额信息

## 验证标准

1. 9 条动线全部跑通
2. 后端 `make check` 通过，无任何 lint / test / vet / build 错误
3. 前端 `pnpm lint:all && pnpm build` 通过
4. 迁移正反向都通
5. 隐私守卫验证通过
6. 文档同步更新（README、CHANGELOG 如有）

## 依赖说明

- 前置：step01 - step20 全部完成

## 预估提交

- commit 1: `chore: end-to-end verification and lint cleanup`
- 视情况可能 0-N 次小修复 commit
