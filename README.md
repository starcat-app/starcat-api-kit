# Starcat API Kit

<!-- starcat-promo:start -->
<div align="center">
<a href="https://starcat.ink"><img src="https://raw.githubusercontent.com/starcat-app/starcat-pro/main/banner.webp" width="100%" alt="Starcat" /></a>

<p><strong>Shared Go kit for Starcat APIs: auth, envelopes, GitHub access, environment parsing, ping handlers, and token pools.</strong></p>
<p>Starcat is a native macOS app that turns GitHub Stars into a searchable, organized and AI-assisted local knowledge base. Version 1.4.0 includes README rendering, knowledge-base RAG, GitHub notifications, My Projects, library and repository insights, macOS desktop widgets, tags and private notes, release tracking, repository health signals, AI summaries, semantic search, browser plugins, Alfred / uTools / Raycast search integrations, and self-hostable support APIs.</p>

<a href="https://github.com/starcat-app/homebrew-starcat"><img src="https://img.shields.io/badge/Install%20with-Homebrew-FBBF24?style=for-the-badge&logo=homebrew&logoColor=white" width="220" alt="Install with Homebrew"/></a>
<br/>
<sub><a href="./README-ZH.md">中文说明</a></sub>
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

**Preferred install method:**

```bash
brew tap starcat-app/starcat
brew trust starcat-app/starcat
brew install --cask starcat
```

**Useful links:**

- Home and downloads: https://starcat.ink
- Mac App Store: search for Starcat for GitHub
- Current Direct build: https://starcat.ink/downloads/Starcat-1.4.0-arm64.dmg
- Public support and release notes: https://github.com/starcat-app/starcat-pro
- Starcat App Homebrew tap: https://github.com/starcat-app/homebrew-starcat
- CLI / MCP: [starcat-cli](https://github.com/starcat-app/starcat-cli) / [Homebrew tap](https://github.com/starcat-app/homebrew-starcat-cli)
- AI Agent Skill: https://github.com/starcat-app/starcat-skill
- Browser plugins: [Chrome](https://github.com/starcat-app/starcat-chrome-plugin) / [Safari](https://github.com/starcat-app/starcat-safari-plugin)
- Launcher integrations: [Alfred](https://github.com/starcat-app/starcat-alfred-workflow) / [uTools](https://github.com/starcat-app/starcat-utools-plugin) / [Raycast](https://github.com/starcat-app/starcat-raycast-extension)
- Documentation: https://github.com/starcat-app/starcat-docs
- Website source: https://github.com/starcat-app/starcat-site
- Localization: https://github.com/starcat-app/starcat-localization

**Self-hostable support APIs:**

- [starcat-sharing-api](https://github.com/starcat-app/starcat-sharing-api)
- [starcat-trending-api](https://github.com/starcat-app/starcat-trending-api)
- [starcat-weekly-api](https://github.com/starcat-app/starcat-weekly-api)
- [starcat-wiki-api](https://github.com/starcat-app/starcat-wiki-api)
- [starcat-recommend-api](https://github.com/starcat-app/starcat-recommend-api)
- [starcat-discovery-api](https://github.com/starcat-app/starcat-discovery-api)
<!-- starcat-promo:end -->

<sub><a href="./README-ZH.md">中文说明</a></sub>

Shared Go packages for Starcat APIs: Bearer auth, response envelopes, CORS, GitHub access, environment parsing, ping handlers, and token pools.

## Packages

| Package | Import path | Purpose |
|---------|-------------|---------|
| `auth` | `github.com/starcat-app/starcat-api-kit/auth` | Bearer API key middleware (`NewBearerAuth` / `NewNamedBearerAuth`) |
| `cors` | `github.com/starcat-app/starcat-api-kit/cors` | CORS + OPTIONS handling |
| `env` | `github.com/starcat-app/starcat-api-kit/env` | Shared environment parsing helpers |
| `envelope` | `github.com/starcat-app/starcat-api-kit/envelope` | Unified JSON response envelope (Meta is the field union) |
| `github` | `github.com/starcat-app/starcat-api-kit/github` | GitHub client and rate-limit handling |
| `httputil` | `github.com/starcat-app/starcat-api-kit/httputil` | Standard API ping handler |
| `tokenpool` | `github.com/starcat-app/starcat-api-kit/tokenpool` | GitHub PAT pool with quota-aware pick |

Individual API repos keep thin `internal/*` wrappers so existing import paths stay stable.

## Development

```bash
go test ./...
```

Local consumers should use a `replace` until the module is published:

```go
replace github.com/starcat-app/starcat-api-kit => ../starcat-api-kit
```

## Contributing

Read [CONTRIBUTING.md](./CONTRIBUTING.md) before opening a pull request.

## Security and support

Report vulnerabilities privately as described in [SECURITY.md](./SECURITY.md). Use
[SUPPORT.md](./SUPPORT.md) to choose the correct support channel.

## License

MIT. See [LICENSE](./LICENSE).
