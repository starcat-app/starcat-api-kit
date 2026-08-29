# AGENTS.md — starcat-api-kit

> **唯一协作规范源**：本仓库根目录 `AGENTS.md` 是本项目协作规范的唯一正文维护源。
> 开工前还必须阅读并遵守上级 [`../AGENTS.md`](../AGENTS.md) 的跨仓规则。

## 项目概述

Starcat 各 HTTP API 共享的 Go 库：Bearer 鉴权、JSON envelope、CORS、环境变量解析、GitHub Client 与 Rate Limit、标准 ping handler、GitHub PAT Token Pool、隐私安全的请求指标（SQLite 聚合）。**不可部署**；变更通过版本 tag 发布，业务 API 以 `go get` 消费。

## 技术栈

- Go 1.25.0
- `modernc.org/sqlite`（metrics 持久化）
- 无 HTTP 服务、无 Dockerfile、无 Makefile

## 关键目录

```
auth/           # Bearer API Key 中间件
envelope/       # 统一 JSON 响应 envelope
cors/           # CORS / OPTIONS
env/            # 共享环境变量解析
github/         # GitHub Client + Rate Limit
httputil/       # 标准 /api/v1/ping handler
tokenpool/      # GitHub PAT 池轮换
metrics/        # 路由指标采集、SQLite 聚合、受控查询接口
```

## 开发与测试命令

```bash
go mod verify
go test ./...
go test -race ./...
gofmt -s -l .          # CI 格式检查，有输出则失败
go vet ./...
```

CI（`.github/workflows/ci.yml`）：checkout → setup-go（go.mod）→ verify → gofmt · vet · test -race。

业务 API 引用已发布版本：
```bash
go get github.com/starcat-app/starcat-api-kit@v0.3.0
go mod tidy
```

本地同时改 kit 与业务 API 时，在父目录临时 workspace（**勿提交 replace**）：
```bash
go work init ./starcat-api-kit ./starcat-xxx-api
```

## 代码与架构约束

- **向后兼容**：auth / envelope / tokenpool 语义变更须评估全部消费方（sharing、trending、weekly、wiki、recommend、discovery、history、collection）。
- **metrics 隐私**：只保留路由模板级聚合；禁止保存真实路径、Query、凭据、请求体、IP、User-Agent；排除 `/internal/metrics/*` 自身轮询。
- **日志脱敏**：API Key 显示 `sk-star****G6AE` 形式；禁止 `log.Printf("key=%s", key)`。
- 发版打 git tag（当前最新 v0.3.0）；`collection-api` 仍 pin v0.2.0，升级须协调。
- 各包须有 `_test.go` 覆盖核心行为；改接口先跑全量 `go test -race ./...`。

## 安全与数据边界

- 无 `.env`、无 secrets 文件；库本身不读 Fly secrets。
- 禁止在 kit 中硬编码 API Key 或 PAT 示例真实值。

## 部署与发布禁令

本库无部署。未经 dong4j 明确授权，禁止：`git push`、`git tag` 发版、未协调版本号强升所有消费方 go.mod。
