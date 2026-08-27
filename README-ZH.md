# Starcat API 共享工具包

<!-- starcat-promo:start -->
<div align="center">
<a href="https://starcat.ink"><img src="https://raw.githubusercontent.com/starcat-app/starcat-pro/main/banner.webp" width="100%" alt="Starcat" /></a>

<p><strong>这是 Starcat API 共享的鉴权、envelope、GitHub、env、ping 与 Token Pool 工具包。</strong></p>
<p>Starcat 是一款原生 macOS 应用，可以把 GitHub Stars 变成可搜索、可整理、可用 AI 追问的本地知识库。当前 1.4.0 支持 README 渲染、知识库 RAG、GitHub 通知、我的项目、全局与仓库洞察、macOS 桌面小组件、标签与私有笔记、Release 追踪、仓库健康度、AI 摘要、语义搜索、浏览器插件，以及 Alfred / uTools / Raycast 外部搜索，并提供多个可自部署 API。</p>

<a href="https://github.com/starcat-app/homebrew-starcat"><img src="https://img.shields.io/badge/Install%20with-Homebrew-FBBF24?style=for-the-badge&logo=homebrew&logoColor=white" width="220" alt="Install with Homebrew"/></a>
<br/>
<sub><a href="./README.md">English</a></sub>
</div>

<div align="center">
<a href="https://starcat.ink"><img src="https://img.shields.io/badge/website-starcat.ink-38BDF8?style=flat&color=blue" alt="website"/></a>
<a href="https://github.com/starcat-app/starcat-pro"><img src="https://img.shields.io/badge/support-starcat--pro-lightgrey.svg?style=flat&color=blue" alt="support"/></a>
<a href="https://github.com/starcat-app/homebrew-starcat"><img src="https://img.shields.io/badge/install-homebrew-lightgrey.svg?style=flat&color=blue" alt="homebrew"/></a>
<a href="https://github.com/starcat-app/starcat-localization"><img src="https://img.shields.io/badge/localization-open-lightgrey.svg?style=flat&color=blue" alt="localization"/></a>
</div>

<div align="center">
<img width="900" src="https://raw.githubusercontent.com/starcat-app/starcat-pro/main/main.webp" alt="Starcat main window"/>
</div>

**首选 Homebrew 安装：**

```bash
brew tap starcat-app/starcat
brew trust starcat-app/starcat
brew install --cask starcat
```

**相关链接：**

- 官网与下载: https://starcat.ink
- Mac App Store: 搜索 Starcat for GitHub
- 当前 Direct 版本: https://starcat.ink/downloads/Starcat-1.4.0-arm64.dmg
- 公开支持与发布说明: https://github.com/starcat-app/starcat-pro
- Starcat App Homebrew tap: https://github.com/starcat-app/homebrew-starcat
- CLI / MCP: [starcat-cli](https://github.com/starcat-app/starcat-cli) / [Homebrew tap](https://github.com/starcat-app/homebrew-starcat-cli)
- AI Agent Skill: https://github.com/starcat-app/starcat-skill
- 浏览器插件: [Chrome](https://github.com/starcat-app/starcat-chrome-plugin) / [Safari](https://github.com/starcat-app/starcat-safari-plugin)
- 启动器集成: [Alfred](https://github.com/starcat-app/starcat-alfred-workflow) / [uTools](https://github.com/starcat-app/starcat-utools-plugin) / [Raycast](https://github.com/starcat-app/starcat-raycast-extension)
- 官方文档: https://github.com/starcat-app/starcat-docs
- 官网源码: https://github.com/starcat-app/starcat-site
- 本地化: https://github.com/starcat-app/starcat-localization

**可自部署支撑 API：**

- [starcat-sharing-api](https://github.com/starcat-app/starcat-sharing-api)
- [starcat-trending-api](https://github.com/starcat-app/starcat-trending-api)
- [starcat-weekly-api](https://github.com/starcat-app/starcat-weekly-api)
- [starcat-wiki-api](https://github.com/starcat-app/starcat-wiki-api)
- [starcat-recommend-api](https://github.com/starcat-app/starcat-recommend-api)
- [starcat-discovery-api](https://github.com/starcat-app/starcat-discovery-api)
<!-- starcat-promo:end -->

<sub><a href="./README.md">English</a></sub>

Starcat API 共享 Go 包：Bearer 鉴权、响应 envelope、CORS、GitHub 访问、环境变量解析、ping handler、Token Pool 与隐私安全的请求指标。

## 包一览

| 包 | Import | 用途 |
|----|--------|------|
| `auth` | `github.com/starcat-app/starcat-api-kit/auth` | Bearer API Key 中间件 |
| `cors` | `github.com/starcat-app/starcat-api-kit/cors` | CORS / OPTIONS |
| `env` | `github.com/starcat-app/starcat-api-kit/env` | 共享环境变量解析工具 |
| `envelope` | `github.com/starcat-app/starcat-api-kit/envelope` | 统一 JSON envelope（Meta 为字段并集） |
| `github` | `github.com/starcat-app/starcat-api-kit/github` | GitHub Client 与 Rate Limit 处理 |
| `httputil` | `github.com/starcat-app/starcat-api-kit/httputil` | 标准 API ping handler |
| `metrics` | `github.com/starcat-app/starcat-api-kit/metrics` | 路由调用指标、SQLite 聚合与受控查询接口 |
| `tokenpool` | `github.com/starcat-app/starcat-api-kit/tokenpool` | GitHub PAT 池 |

各业务 API 通过薄 `internal/*` 别名包装本库，避免业务代码大面积改 import。

### 请求指标

`metrics.Collector` 包装根 `http.Handler`，只在内存中保留匹配后的路由模板，并把分钟、小时、天
聚合写入独立 SQLite。各服务把 `metrics.Handler` 方法挂在现有 Bearer 鉴权之后。指标不会保存真实
路径、Query、凭据、请求体、IP 或 User-Agent，且排除 `/internal/metrics/*`，避免控制台轮询污染数据。

## 开发

```bash
go test ./...
```

业务 API 应依赖已发布的模块版本：

```bash
go get github.com/starcat-app/starcat-api-kit@v0.3.0
```

需要同时修改 kit 和业务 API 时，在它们的父目录创建临时 Go workspace：

```bash
go work init ./starcat-api-kit ./starcat-wiki-api
```

不要把指向兄弟目录的 `replace` 提交到业务模块；独立 clone 和 CI runner
中不存在该兄弟路径。

## 贡献 / 安全 / 支持

见 [CONTRIBUTING.md](./CONTRIBUTING.md)、[SECURITY.md](./SECURITY.md)、[SUPPORT.md](./SUPPORT.md)。

## License

MIT。见 [LICENSE](./LICENSE)。
