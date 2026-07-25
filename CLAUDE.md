# ima_cli-go

Go CLI for IMA notes and knowledge base management.

## Commands

- Knowledge base: `ima list|info|browse|search|upload|url|get-media`
- Notes: `ima notes list-notebooks|list|search|get|create|append`
- Alias: `ima alias add|list|remove`

## Conventions

- Zero external dependencies — stdlib only
- Chinese error messages via `fmt.Errorf`
- `flag` stdlib for CLI, no cobra
- JSON output for data commands
- `internal/ima/` — HTTP client + types
- `internal/notes/` — notes API
- `internal/kb/` — kb API + COS upload
- `internal/config/` — credential loading
- `internal/alias/` — kb-id alias system
- Credentials: env vars `IMA_OPENAPI_CLIENTID`/`IMA_OPENAPI_APIKEY` → `~/.config/ima/` files
- File upload flow: create_media → COS PUT → add_knowledge
- `--folder-id` flag available for browse/upload/url/notes list

## Build

```bash
# 开发版本
go build -o ima .

# 发布版本（注入版本号）
go build -ldflags "-X main.Version=1.0.0" -o ima .
```
