# Starcat API Kit

<sub><a href="./README-ZH.md">中文说明</a></sub>

Shared Go packages for Starcat self-hosted APIs: Bearer auth, response envelope, CORS, and GitHub token pool.

## Packages

| Package | Import path | Purpose |
|---------|-------------|---------|
| `auth` | `github.com/starcat-app/starcat-api-kit/auth` | Bearer API key middleware (`NewBearerAuth` / `NewNamedBearerAuth`) |
| `cors` | `github.com/starcat-app/starcat-api-kit/cors` | CORS + OPTIONS handling |
| `envelope` | `github.com/starcat-app/starcat-api-kit/envelope` | Unified JSON response envelope (Meta is the field union) |
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
