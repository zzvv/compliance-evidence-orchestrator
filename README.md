# 合规证据编排服务

这个服务面向制造项目中的材料合规审核。供应商证书、检测报告和声明材料按照项目与材料范围登记，随后组成审核批次，并在审核、回执、审计和通知之间保持可追踪的处理记录。

## 架构

- `cmd/server`：HTTP 服务与进程生命周期。
- `internal/domain`：证据、审核批次、回执、清单、风险和保留策略。
- `internal/application`：审核流程编排、查询、通知投递和报表聚合。
- `internal/repository`：当前提供线程安全的内存仓储，可替换为持久化实现。
- `internal/transport`：标准库 HTTP 路由与控制台页面。
- `internal/worker`：待发送通知的后台调度。

## 运行

需要 Go 1.25。

```bash
go test ./...
go run ./cmd/server
```

服务默认监听 `:8080`。`GET /healthz` 用于健康检查，访问根路径可查看轻量控制台。

## 接口概览

- `POST /v1/evidence` 登记材料证据。
- `POST /v1/batches` 创建并提交审核批次。
- `POST /v1/batches/{id}/start` 开始审核。
- `POST /v1/batches/{id}/decision` 记录通过或驳回决定。
- `POST /v1/batches/{id}/cancel` 取消尚未结束的批次。
- `GET /v1/batches/{id}` 查询批次、证据、回执和审计轨迹。
