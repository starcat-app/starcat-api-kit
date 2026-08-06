# Starcat API 共享工具包

<sub><a href="./README.md">English</a></sub>

Starcat 自建 API 共享 Go 包：Bearer 鉴权、响应 envelope、CORS 与 GitHub Token Pool。

## 包一览

| 包 | Import | 用途 |
|----|--------|------|
| `auth` | `github.com/starcat-app/starcat-api-kit/auth` | Bearer API Key 中间件 |
| `cors` | `github.com/starcat-app/starcat-api-kit/cors` | CORS / OPTIONS |
| `envelope` | `github.com/starcat-app/starcat-api-kit/envelope` | 统一 JSON envelope（Meta 为字段并集） |
| `tokenpool` | `github.com/starcat-app/starcat-api-kit/tokenpool` | GitHub PAT 池 |

各业务 API 通过薄 `internal/*` 别名包装本库，避免业务代码大面积改 import。

## 开发

```bash
go test ./...
```

发布前本地消费方使用：

```go
replace github.com/starcat-app/starcat-api-kit => ../starcat-api-kit
```

## 贡献 / 安全 / 支持

见 [CONTRIBUTING.md](./CONTRIBUTING.md)、[SECURITY.md](./SECURITY.md)、[SUPPORT.md](./SUPPORT.md)。

## License

MIT。见 [LICENSE](./LICENSE)。
