# Step 07 截图识别服务与 API

## 任务目标

新增 screenshot service：接受图像字节、调用 VisionProvider、解析响应、按置信度阈值分级、返回结构化结果。同时新增 API 端点 POST /api/v1/portfolio/import-screenshot。

## 涉及文件

创建：
- `backend/internal/service/screenshot/service.go`
- `backend/internal/service/screenshot/service_test.go`
- `backend/internal/service/screenshot/prompts.go`（system + user prompt 文本）
- `backend/internal/service/screenshot/parser.go`（解析 LLM JSON 响应）
- `backend/internal/service/screenshot/parser_test.go`
- `backend/internal/api/v1/screenshot.go`
- `backend/internal/api/v1/screenshot_test.go`

修改：
- `backend/internal/api/v1/router.go`（注册新路由）
- `backend/cmd/server/main.go`（依赖注入 VisionProvider 和 screenshot service）

## 设计依据

- TRD §4 完整截图 OCR 章节
- TRD §4.4 置信度阈值常量
- TRD §4.5 API 端点定义（multipart / 5MB 上限 / 限流）
- TRD §4.6 降级策略
- TRD §4.7 Prompt 结构原则
- PRD §4.3 截图批量导入 Modal

## 实施要点

- service.Recognize 不持久化任何数据：图像 byte 处理完即丢
- 图像大小校验在 service 入口（> 5MB 直接返回错误，不调 LLM）
- 限流：每用户 10 次/小时，使用 redis 或内存令牌桶（MVP 内存即可，TRD 未指定具体方案）
- prompts.go 定义 system 和 user prompt 模板，遵循 TRD §4.7 原则
- parser 解析 LLM 返回的 JSON 字符串到 RecognizeResponse：
  - JSON 不合法 → overallStatus = "failed"
  - 成功但所有字段 confidence < 0.6 → overallStatus = "low_quality"
  - 否则 = "ok"
- API handler 处理 multipart upload，转字节流给 service
- handler 不直接落库，只返回识别结果；前端用户确认后走现有 POST /api/v1/holdings 批量创建
- 单元测试覆盖：
  - 图像过大
  - 限流触发
  - LLM 成功 → 解析 ok
  - LLM 成功但响应 JSON 不合法
  - LLM 超时（mock vision provider 报错）
  - 全部字段低置信度

## 验证标准

1. `go test ./internal/service/screenshot/... ./internal/api/v1/screenshot_test.go` 通过
2. `curl -F file=@test.png http://localhost:8080/api/v1/portfolio/import-screenshot -H "Authorization: Bearer ..."` 在 mock vision 模式下能拿到结构化响应
3. `make check` 通过
4. 限流逻辑可通过单测验证

## 依赖说明

- 前置：step06（VisionProvider 抽象必须先完成）
- 可与 step08 并行

## 预估提交

- commit 1: `feat(screenshot): add recognition service with confidence thresholds`
- commit 2: `feat(api): add import-screenshot endpoint with multipart upload`
