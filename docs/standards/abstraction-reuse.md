# 抽象复用原则

本文档定义项目级的抽象与复用设计规范，前后端均强制遵守。核心思想：**基础设施和横切关注点必须经过抽象层封装，业务代码不直接接触原始 API**。

## 总则

任何时候发现以下情况，必须先建立抽象再实现：

- 直接调用基础设施（localStorage、sessionStorage、os.Getenv、http.Client……）
- 在多个地方重复相似的模板代码（try/catch 包裹、参数构造、错误映射……）
- 同一类数据（key 字符串、配置名、枚举值）散落在多个文件里
- 一个关注点（序列化、错误处理、认证头）在调用方各自重复实现

**判断标准：** 如果删掉抽象层后，调用方需要知道"怎么做"而不只是"做什么"，就说明抽象是必要的。


## 前端抽象规则

### 分层模型

```
pages / features          ← 业务代码（只知道"做什么"）
    ↓ 依赖
domain/                   ← 基础设施抽象（封装"怎么做"）
    storage/              ← localStorage 读写
    http/                 ← fetch 封装、认证头、错误处理
    auth/                 ← token 读写
    money/                ← 金额格式化
    ui/                   ← 主题、通用 UI 状态
    ...
ui-kit/eat                ← 组件库 barrel（封装 antd 直接依赖）
```

### 新建 domain 模块的时机

遇到以下情况时，在 `domain/` 新建模块，而不是在调用方就地实现：

| 触发场景 | 抽象方式 |
|---|---|
| 读写 localStorage / sessionStorage | `domain/storage/local-storage.ts` 原语 + `useLocalStorage` hook |
| 发起 HTTP 请求 | `domain/http/client.ts` 的 `request()` |
| 格式化金额/百分比 | `domain/money/format.ts` |
| 读取认证 token | `domain/auth/storage.ts` |
| 持久化 UI 状态（主题、语言） | `domain/ui/` |

### 禁止事项（前端）

```typescript
// 禁止：直接调用 localStorage
localStorage.getItem("some_key");
localStorage.setItem("some_key", value);
localStorage.removeItem("some_key");

// 禁止：key 字符串字面量散落在业务代码中
useLocalStorage("richman_last_task_id", null);

// 禁止：在 feature/page 中重复序列化逻辑
const raw = localStorage.getItem("key");
const parsed = raw ? JSON.parse(raw) : null;
```

```typescript
// 正确：通过抽象层
import { StorageKeys, storageGet } from "@/domain/storage/local-storage";
import { useLocalStorage } from "@/domain/storage/use-local-storage";

const [taskId, setTaskId] = useLocalStorage(StorageKeys.lastAnalysisTaskId, null);
```

### React Hook 封装原则

横切关注点的状态逻辑，如果有多个组件需要，必须提取为自定义 hook，放在 `domain/` 或对应 feature 的 hook 文件中：

```typescript
// 禁止：在 3 个组件里各自实现相同的读写逻辑
// 正确：提取为 useLocalStorage / useThemeMode / useAnalysisTask
```

Hook 命名规则：
- 基础设施 hook → `domain/<模块>/use-<关注点>.ts`
- 业务 hook → `features/<feature>/use-<行为>.ts`


## 后端抽象规则

### 分层模型

```
API handlers      ← HTTP 路由（只做参数绑定和响应序列化）
    ↓
Service           ← 业务编排（调用 repo、外部服务）
    ↓
Repo              ← 数据访问（只做 SQL，不含业务判断）
    ↓
Infrastructure    ← 外部依赖封装（llm/、datasource/、config/）
```

每一层只能向下依赖，**不可越层、不可反向**。

### 新建基础设施抽象的时机

| 触发场景 | 抽象方式 |
|---|---|
| 调用外部 HTTP API（LLM、行情数据源） | `internal/<provider>/` package，暴露 interface |
| 读取配置 | `internal/config/` 统一结构体，通过依赖注入传递 |
| 数据库访问 | `internal/repo/` + sqlc 生成，service 不直接写 SQL |
| 跨域共享逻辑（加密、哈希、时间计算） | `internal/<util>/` package |

### 禁止事项（后端）

```go
// 禁止：在 service / handler 中直接读环境变量
os.Getenv("OPENAI_API_KEY")

// 禁止：在 handler 中直接写 SQL
db.Query("SELECT * FROM holdings WHERE ...")

// 禁止：在 service 中直接构造 http.Client 并发请求
resp, _ := http.Get("https://api.openai.com/...")

// 禁止：在多个 service 中各自实现相同的错误映射
if err != nil {
    return fmt.Errorf("...")  // 相同模式重复 N 次
}
```

### Interface 优先原则

对可能替换的外部依赖（LLM 提供商、数据源、通知渠道），必须定义 interface，具体实现通过依赖注入传入，不可在业务代码中直接实例化：

```go
// 正确：通过 interface 解耦
type Provider interface {
    ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}

// Service 依赖 interface，不依赖具体实现
type Synthesizer struct {
    resolver llm.Resolver  // interface
}
```


## 通用原则

### 集中注册，不散落

常量、配置 key、枚举值等需要在多处引用的标识符，必须在**一个文件**中集中定义，其他地方引用而非各自硬编码：

- 前端：`StorageKeys`（storage key）、`QUERY_KEY`（TanStack Query key）、路由路径常量
- 后端：`model/` 包中的枚举常量、`config/` 中的配置字段名

### 错误处理统一收口

- 前端：`domain/http/client.ts` 的 `request()` 统一处理 HTTP 错误；业务代码不重复判断 `response.ok`
- 后端：`middleware/error_handler.go` 统一序列化错误响应；service 层只返回语义化 error，不拼 HTTP 状态码

### 禁止"就地抽象"

不要在某个 feature/page 内部为了"复用"而提取 helper——这种 helper 会成为孤立的内部工具，不可被其他模块使用。真正需要复用的逻辑，直接放到对应的 `domain/` 或 `internal/` 层：

```
禁止：features/portfolio/utils/formatAmount.ts
正确：domain/money/format.ts
```


## 代码审查检查项

任何 PR 中出现以下模式时，必须在 review 中要求改正：

- [ ] 直接调用 `localStorage` / `sessionStorage`（非 `domain/storage/`）
- [ ] 直接调用 `os.Getenv`（非 `config.Config` 结构体字段）
- [ ] 直接在 handler 层写 SQL 或直接实例化 DB 连接
- [ ] 业务代码中出现硬编码的 key 字符串（storage key、query key、配置名）
- [ ] 相同的错误处理模板代码在多个文件中重复出现
- [ ] 新增外部依赖（API 调用、IO 操作）没有对应的 interface 定义
