# Step 06 LLM Vision 抽象与 Claude 实现

## 任务目标

新增 VisionProvider 接口和它的 Claude 实现，为 step07 的截图识别服务提供能力底座。与现有的纯文本 Provider 接口解耦，但共享同一个 factory 入口。

## 涉及文件

创建：
- `backend/internal/llm/vision.go`（VisionProvider 接口 + VisionRequest / VisionResponse 类型）
- `backend/internal/llm/claude_vision.go`（Claude 实现）
- `backend/internal/llm/claude_vision_test.go`
- `backend/internal/llm/vision_factory.go`（按 env 创建 VisionProvider）

修改：
- `backend/internal/llm/factory.go`（如已有 factory，扩展暴露 NewVisionProvider）
- `backend/internal/config/config.go`（新增 LLM_VISION_PROVIDER / VISION_API_KEY 等环境变量）
- `backend/.env.example`（新增对应 env 模板）

## 设计依据

- TRD §4.2 LLM Vision 抽象定义
- TRD §4.6 降级策略（超时 / 5xx / 解析失败）
- 工程规范 `docs/standards/backend.md` 三层架构、配置集中加载

## 实施要点

- VisionProvider 接口仅一个方法 AnalyzeImage(ctx, req) (resp, err)，不引入 multipart / http 细节
- Claude 实现使用 Anthropic Messages API 的 vision 能力（model = claude-sonnet-4-6）
- ImageData 转 base64 后嵌入 messages 的 image content block
- 超时通过 ctx 控制，默认 30s，可由 config 覆盖
- 错误分类：网络 / 5xx / 解析失败 → 返回结构化 error，调用方据此降级
- 不在本 step 实现具体 prompt（prompt 在 step07 的 screenshot service 内部组装）
- 单元测试 mock HTTP server，覆盖：
  - 成功响应解析
  - 网络超时
  - 5xx 错误
  - 响应 JSON 格式不符
- 配置项不直接 os.Getenv，遵守 backend.md 的"配置集中加载"原则

## 验证标准

1. `go test ./internal/llm/...` 通过
2. mock 测试覆盖 4 种典型场景
3. `.env.example` 含全部新变量及说明
4. `make check` 通过

## 依赖说明

- 无前置 step 强依赖（与 step01-05 解耦）
- 但建议在 Phase 1-2 早期完成，因为 step07 直接依赖

## 预估提交

- commit 1: `feat(llm): add vision provider abstraction`
- commit 2: `feat(llm): add claude vision implementation`
