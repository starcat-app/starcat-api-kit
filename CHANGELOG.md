# Changelog

All notable changes to Starcat API Kit are documented here.

## [0.2.0] - 2026-08-07

### Added
- `github`：带 tokenpool 的 `GetRepo` / `GetReadme`、`RateLimitHandler`、中立 `Repo` DTO
- `github.Options.AllowAnonymous`：pool 无 token 时允许匿名 GetRepo（sharing 公开预览）
- `github.Repo` 补齐 `HTMLURL` / `IsTemplate`；`Client.SetHTTPClient` 测试钩子
- `httputil.HandlePingV1`：统一 `/api/v1/ping` envelope 响应
- `env`：`LookupRequired` / `CSV` / `OrDefault` / `DurationSeconds` / `RequiredCSV`

## [0.1.0] - 2026-08-07

### Added
- `auth` / `cors` / `envelope` / `tokenpool` 初版（供各业务 API 薄包装）

[0.2.0]: https://github.com/starcat-app/starcat-api-kit/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/starcat-app/starcat-api-kit/releases/tag/v0.1.0
