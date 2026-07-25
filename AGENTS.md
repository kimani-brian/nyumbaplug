# NyumbaPlug — AGENTS.md

## Quick start

```bash
cp .env.example .env
createdb -p 5434 kenyahouses
psql -p 5434 -d kenyahouses -f migrations/000001_init_schema.up.sql
go run cmd/seed/main.go           # admin: +254700000000 / AdminPass123!
go run cmd/server/main.go         # :8081
```

## Environment quirks

- Postgres runs on **port 5434**, not 5432. The `.env` uses Unix socket peer auth (`host=/var/run/postgresql`). If connecting via TCP you need `localhost:5434`.
- Server port is **8081** (8080 is occupied by Odoo on this machine).
- `migrate` CLI must be built with `-tags 'postgres'` or it errors with "unknown driver". If that fails, run the `.up.sql` file directly with psql.

## Commands

| What | Command |
|------|---------|
| Run server | `go run cmd/server/main.go` |
| Seed admin | `go run cmd/seed/main.go` |
| Unit tests (no DB) | `go test ./tests/unit -v` |
| Integration tests | `TEST_DATABASE_URL="..." go test ./tests/integration -v` |
| Build everything | `go build ./...` |
| All tests | `go test ./...` (integration skips if no DB) |

## Architecture

```
chi Router (:8081) → middleware (JWT → RequireRole) → Handler → Service → Repository → PostgreSQL
```

- All routes are under `/api/v1`. Chi route params use curly braces (`{id}`, `{landlord_id}`).
- Three roles: `admin`, `landlord`, `tenant`. Constants in `internal/domain/models.go`.
- JWT claims: `user_id` (UUID string), `role` (string), `exp` (24h).
- Context keys for middleware: `middleware.UserIDKey` (`"user_id"`), `middleware.RoleKey` (`"role"`).

## Critical business rules enforced in TWO layers

| Rule | Service enforcement | SQL enforcement |
|------|-------------------|-----------------|
| 2: Revoked landlords hidden | `property_service.go` | `SearchVerifiedProperties` joins `landlord_profiles WHERE verification_status = 'verified'` |
| 3: Contact only for vacant+verified | `property_service.go` | `GetUnitContactDetails` checks status + verification after query |

Never remove the SQL-layer enforcement — it's the definitive gate.

## Repository pattern

- `Repository` interface in `internal/repository/postgres.go:14`
- `PostgresRepo` stores a `DBTX` interface (satisfied by both `*sql.DB` and `*sql.Tx`)
- Integration tests inject `*sql.Tx` via `repository.NewPostgresRepo(tx)`, then roll back
- Constructor: `repository.NewPostgresRepo(dbtx)` — pass `*sql.DB` for production, `*sql.Tx` for test transactions

## Testing quirks

- Integration tests require a real Postgres. Set `TEST_DATABASE_URL`. Skip if unreachable (via `t.Skip`).
- Integration tests use `tx.Rollback()` for isolation — no cleanup needed.
- Unit tests in `tests/unit/` use testify mocks (`MockRepo` implements `Repository`). No DB needed.
- Test helpers: `registerUser()`, `doRequest()` in `tests/integration/integration_test.go`.

## Module path

`github.com/kenya-houses/backend` — all imports use this, not the directory name.

## UUIDs

All IDs are `uuid.UUID` generated server-side with `github.com/google/uuid`. Never send UUIDs from the client for creation — only for referencing existing resources in URL params.

## API at a glance

All endpoints documented in `API.md` and `README.md`. Key: landlord endpoints require `verification_status=verified` (enforced in service), admin endpoints write audit logs.
