# go-zero-admin-pkg

`github.com/dwrui/go-zero-admin-pkg` 是 [go-zero-admin](https://github.com/dwrui/go-zero-admin) 的公共基础库，供各微服务与 API 网关复用。

> **版本说明**：当前主仓库通过 `go.work` 本地引用 `pkg/` 开发；线上部署请直接 `go get github.com/dwrui/go-zero-admin-pkg@<tag>`，**不要**在 `go.mod` 里写 `replace`。版本号待 API 稳定后统一打 tag。

## 目录结构

| 路径 | 说明 |
|------|------|
| `rpcclient/authcenter` | 鉴权 RPC 封装（CheckToken、字段权限等） |
| `rpcclient/configcenter` | 配置中心 RPC（分类配置、GetValue、DelFiles） |
| `rpcclient/bootstrap` | **推荐** RPC 客户端安全初始化（Etcd 未配不 panic） |
| `utils/db` | GORM 多库管理、表前缀 |
| `utils/cache` | Redis 缓存封装 |
| `utils/jwt` | JWT 签发与解析 |
| `utils/ga` | 类型转换、IP 获取等 |
| `utils/logsanitize` | 操作日志 JSON/Header 脱敏 |
| `datacenter` | 数据中心队列、存储抽象、设置模型（部分在 `service/admin/datacenter/pkg`） |

## RPC 客户端初始化（推荐）

微服务 `ServiceContext` 中**不要**无条件 `MustInit` + `GetClient()`，否则 Etcd 未配置或连接失败会直接 panic。

```go
import (
    "github.com/dwrui/go-zero-admin-pkg/rpcclient/bootstrap"
)

clients := bootstrap.InitOptionalClients(c.AuthEtcd, c.ConfigCenterEtcd, bootstrap.RpcTimeouts{
    Auth:         5000,
    ConfigCenter: 3000,
})

// clients.Auth / clients.Config 可能为 nil，业务层需判空或降级
```

`bootstrap.EtcdConfigured(etcd)` 判断 `Hosts` 与 `Key` 是否齐全。

### API 网关（admin-api）差异

`api/admin` 使用 goctl 生成的 `zrpc.MustNewClient`，**启动时强依赖**各下游 RPC 在 Etcd 可发现——这是 BFF 的预期行为（fail-fast）。若某下游未部署，网关进程会起不来，需在 `etc/admin-api.yaml` 配齐 Etcd 或拆分网关。

## configcenter 客户端

```go
import "github.com/dwrui/go-zero-admin-pkg/rpcclient/configcenter"

// 读取某分类下全部键值（带本地缓存，默认 2 分钟）
m, err := client.GetCategoryValues(ctx, businessId, "datacenter_settings")

// 删除 OSS 文件（FilesService.DelFile）
err := client.DelFiles(ctx, []uint64{fileId})

// 上传任务结果文件到 OSS（FilesService.UploadFile）
result, err := client.UploadFile(ctx, configcenter.UploadFileParams{
    BusinessID: 1,
    Filename:   "export.xlsx",
    ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    Data:       content,
    IsCommon:   true,
})
```

配置项通过配置中心 `ConfigItemService.GetValue` 拉取，**微服务不应直连配置库**。

## 日志脱敏

`utils/logsanitize` 供 admin-api 中间件使用：

- JSON 字段：`password`、`token`、`authorization` 等 → `***`
- Header：`Authorization`、`Cookie` 等 → `***`
- 大 body 截断（默认 8KB）
- 上传/导入路径可配置跳过 body 记录

## 本地开发

```bash
# 仓库根目录
go work sync
cd pkg && go build ./...
```

## 相关文档

- 主仓库 `docs/CODEGEN_EXPORT.md` — 代码生成与导出模式
- 主仓库 `docs/OBSERVABILITY.md` — 可观测性方案
- 主仓库 `docs/TESTING.md` — 测试策略
