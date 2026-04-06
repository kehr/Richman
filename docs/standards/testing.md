# 测试规范

## 前端测试

### 技术栈

- Vitest（测试框架）
- React Testing Library（组件测试）
- MSW（API mock）

### 目录结构

测试文件与源文件同目录，使用 `.test.ts` / `.test.tsx` 后缀：

```
features/portfolio/
  api.ts
  api.test.ts
  usePortfolio.ts
  usePortfolio.test.ts
```

### 测试描述

使用中文描述：

```typescript
describe("useHoldings", () => {
	it("should fetch and return holdings list", () => {
		// ...
	});

	it("should handle empty holdings", () => {
		// ...
	});
});
```

### 测试类型

**Hook 测试（TanStack Query）：**

```typescript
import { renderHook, waitFor } from "@testing-library/react";
import { useHoldings } from "./usePortfolio.js";

describe("useHoldings", () => {
	it("should return holdings data", async () => {
		const { result } = renderHook(() => useHoldings(), { wrapper: QueryWrapper });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toHaveLength(3);
	});
});
```

**API 函数测试：**

```typescript
import { fetchHoldings } from "./api.js";

describe("fetchHoldings", () => {
	it("should call correct endpoint", async () => {
		// Use MSW to mock API
		const result = await fetchHoldings();
		expect(result.data).toBeDefined();
	});
});
```

### Mock 策略

| 对象 | 方式 |
|------|------|
| API 请求 | MSW (Mock Service Worker) |
| 路由 | MemoryRouter wrapper |
| i18n | 简化 provider wrapper |
| 认证状态 | Context mock |


## 后端测试

### 技术栈

- Go 标准 testing 包
- testify（断言和 mock）
- testcontainers-go（集成测试用 PostgreSQL 容器）

### 目录结构

测试文件与源文件同目录，使用 `_test.go` 后缀：

```
internal/
  service/
    analysis/
      service.go
      service_test.go
  repo/
    holding.go
    holding_test.go
  analysis/
    trend/
      calculator.go
      calculator_test.go
```

### 测试类型

**单元测试（纯函数）：**

```go
func TestCalculateTrendScore(t *testing.T) {
	tests := []struct {
		name     string
		prices   []float64
		expected TrendResult
	}{
		{
			name:   "upward trend with strong momentum",
			prices: []float64{10, 11, 12, 13, 14},
			expected: TrendResult{
				Direction: DirectionUpward,
				Strength:  0.85,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTrendScore(tt.prices)
			assert.Equal(t, tt.expected.Direction, result.Direction)
			assert.InDelta(t, tt.expected.Strength, result.Strength, 0.1)
		})
	}
}
```

**Service 测试（含依赖）：**

```go
func TestAnalysisService_AnalyzeHolding(t *testing.T) {
	// Setup test dependencies
	repo := setupTestRepo(t)
	service := NewAnalysisService(repo, testConfig)

	// Insert test data
	holdingID := insertTestHolding(t, repo)

	// Execute
	card, err := service.AnalyzeHolding(context.Background(), holdingID)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, card.TrendSummary)
	assert.InRange(t, card.Confidence, 0.0, 100.0)
}
```

**Repo 测试（测试数据库）：**

```go
func TestHoldingRepo_ListByUser(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Insert test data
	userID := int64(10001)
	insertTestHoldings(t, db, userID, 3)

	// Query
	holdings, err := ListHoldingsByUser(context.Background(), db, userID)

	// Assert
	require.NoError(t, err)
	assert.Len(t, holdings, 3)
	// Verify soft delete filter
	for _, h := range holdings {
		assert.Equal(t, int16(0), h.IsDeleted)
	}
}
```

### Mock 策略

| 对象 | 方式 |
|------|------|
| 数据库 | testcontainers-go（真实 PostgreSQL）或 sqlc mock |
| LLM API | 接口 mock（实现 LLMProvider 接口的 mock 结构体） |
| 外部数据源 | httptest.Server mock |
| 配置 | 测试专用 config 构造函数 |

### 原则

- 测试之间独立，不依赖执行顺序
- 清晰的 arrange-act-assert 结构
- setup 创建测试数据，teardown 清理
- 不测试框架行为，聚焦业务逻辑
- 量化计算（趋势/位置/催化剂）必须有充分的边界测试
- LLM 综合层测试使用固定的 mock 输入，验证输出结构正确


## 测试命令

### 前端

```bash
pnpm test              # 运行所有测试
pnpm test:watch        # Watch 模式
pnpm test:coverage     # 覆盖率报告
```

### 后端

```bash
go test ./...                    # 运行所有测试
go test ./internal/analysis/...  # 运行特定包测试
go test -race ./...              # 竞态检测
go test -cover ./...             # 覆盖率
```


## 测试优先级

MVP 阶段优先保证以下测试覆盖：

1. **三维分析引擎** -- 趋势/位置/催化剂的量化计算逻辑（核心业务）
2. **权重和信心度计算** -- 确保矩阵输出正确
3. **持仓成本计算** -- 快速模式和明细模式的成本加权逻辑
4. **API 端点** -- 关键接口的请求/响应格式
5. **推送适配器** -- 各渠道消息格式正确性
