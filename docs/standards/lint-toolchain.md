# Lint 工具链纪律

本规范定义 Richman 项目的 lint 工具链版本管理、配置同步、验证闭环的强制要求。所有对代码质量工具的依赖升级、配置迁移、跳过规则的豁免都必须走完本规范的检查点。

## 背景

2026-04-09 在修复 `analysis/synthesis.Synthesizer` nil-panic 时发现：

1. 本机没有安装 `golangci-lint`
2. 项目的 `backend/.golangci.yml` 仍是 v1 格式，而 Homebrew 安装的最新版本是 v2.11.4
3. v1 配置在 v2 下无法加载（`unsupported version of the configuration`），导致 `make lint` 自始至终无法运行
4. 全项目累积了 41 条存量 lint 违规，这些违规没有被任何开发者察觉，因为 lint 工具链在当前本机环境下根本不工作
5. 更早之前的提交能够 merge 进主干这件事，说明「严格 lint 原则」没有在 CI 或本地环境里被有效执行

这是一次典型的「工具链破产 + 纪律静默失效」的系统性失灵。为防止复发，本规范把教训固化为可机械执行的流程。

## 核心原则

1. **lint 工具版本 == 配置文件 schema 版本**：任何工具版本升级必须同步验证配置文件能被加载；任何配置文件 schema 迁移必须同步更新最低要求的工具版本
2. **本地环境和 CI 环境必须跑同一份 lint**：禁止「本地跑不起来就跳过，CI 会跑」的心态；lint 如果跑不起来，优先修工具链不是跳过
3. **零违规入库**：每次文件修改必须跑 lint 并修复全部问题；lint 输出非零就不算闭环
4. **豁免必须显式且有理由**：`//nolint` 指令必须写清原因，不允许空 `//nolint`；配置里的 `disable` / `exclude` 必须有注释说明

## 适用范围

本规范约束 Richman 项目的以下 lint 工具：

| 项 | 工具 | 配置文件 |
|------|------|----------|
| 后端 Go | `golangci-lint` | `backend/.golangci.yml` |
| 前端 TS/TSX | `Biome` | `frontend/biome.json` |
| 前端依赖图 | `dependency-cruiser` | `frontend/.dependency-cruiser.cjs` |

## 工具链版本声明

每种 lint 工具必须在**可执行的位置**声明其最低兼容版本，单一来源避免配置漂移：

### Go / golangci-lint

- 在 `backend/.golangci.yml` 顶部必须声明 `version: "2"`（对应 golangci-lint v2.x）
- 在 `backend/tools/tools.go`（如不存在则创建）用 `_ "github.com/golangci/golangci-lint/..."` 固定版本
- 或者在 `backend/Makefile` 的 `lint` 目标前增加一个 `lint-install` 前置目标，用 `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4` 明确安装对应版本
- CI 脚本必须读取同一个版本号，禁止 CI 脚本写死 `latest`

### 前端工具

- `biome` 版本固定在 `frontend/package.json` 的 `devDependencies`，配合 `pnpm-lock.yaml` 锁死
- `dependency-cruiser` 同上

## 配置文件迁移流程

当工具主版本升级（v1 -> v2 这类 breaking change）时必须按以下顺序执行，不允许跳步：

1. **准备阶段**：在独立分支（`chore/<tool>-v<N>-migration`）创建迁移工作区
2. **先装新版本**：本地和 CI 同步升级到目标版本
3. **配置迁移**：按照工具官方迁移指南更新配置文件 schema
4. **全量盘点**：跑一次 `lint ./...`，记录所有新/老违规数量作为 baseline
5. **修复或豁免**：逐条处理 baseline 里的每一条违规，不允许「先放着以后再修」
6. **验证闭环**：`lint` 必须输出 `0 issues`，然后跑 `test` 和 `build` 确认没有破坏
7. **CI 同步**：更新 CI 脚本里的工具版本号，确保 CI 也在新版本下跑 lint
8. **文档更新**：本规范的「工具链版本声明」段落必须反映新版本号

## 本地开发的强制检查点

每次修改代码后必须跑的命令（不是每次改完一批后，是**每次改完一个文件**后）：

### 后端
```bash
cd backend && make check
```

`make check` 内部已串联 `lint + test + build`，其中 lint 失败会直接中断整条链。不允许用 `go test` / `go vet` 单独代替。

### 前端
```bash
cd frontend && pnpm lint:all && pnpm test
```

### 禁止的做法

- 手动跑 `go test` 但跳过 `make check` 的 lint 阶段
- 因为「lint 工具跑不起来」就跳过 lint，必须先修工具链
- 用 `--no-verify` 跳过 pre-commit hook
- 在 CI 里给 lint 设置 `continue-on-error: true`
- 对 lint 报告的警告做 `grep -v` 隐藏

## 豁免规则

### 单行豁免

允许用 `//nolint:<linter-name> // <理由>` 行内豁免，但必须满足：

1. 明确指定 linter 名，禁止空 `//nolint`
2. 必须在注释后附理由（为什么这里不修）
3. 理由不能是「暂时的」「以后会修」；要么是永久原因（比如 API 兼容）要么是有明确跟踪的 TODO 指向 issue

### 全局豁免

禁止在 `.golangci.yml` 里用 `disable` 关闭某个 linter，除非：

1. 该 linter 本身有 bug 或误报率极高
2. 理由写在配置文件的注释里
3. 同时在 `docs/standards/lint-toolchain.md` 里记录豁免决策和时间

### 阈值放宽

禁止通过调高 lint 阈值（比如 `gocritic.hugeParam.sizeThreshold`）来让问题消失，除非新阈值有基于项目实际数据的合理性论证。

## CI 层强制拦截

无论本地开发者是否执行 lint，CI 必须作为最后防线：

1. `main` 分支 push 前的 PR workflow 必须跑 `cd backend && make check` 和 `cd frontend && pnpm lint:all`
2. CI workflow 文件里 lint 步骤禁止 `continue-on-error: true` 或 `||true`
3. CI 里 golangci-lint 版本号必须和本规范声明的保持一致，否则工具漂移会再次发生

## 复盘触发点

遇到以下情况必须回头读这份规范并检查是否需要更新：

- 某个 lint 规则开始误报，被多个 PR 临时豁免
- 升级工具版本后配置文件不兼容
- CI 和本地 lint 结果不一致
- 有新的 lint 工具被引入项目
- 发现存量违规超过 10 条

## 例外情况

以下场景可跳过本规范的强制检查点：

- 纯文档修改（`.md` 文件，不涉及代码）
- 纯配置修改（`.env.example` / `docker-compose.yml` 等）
- `docs/` 目录下的所有改动

这些场景下允许只跑 markdown lint 或 yaml lint（如果有），不强制跑 Go/TS 的完整 check。
