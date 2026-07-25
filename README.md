# NyumbaPlug — Kenyan House-Hunting Platform

A Go/PostgreSQL backend for the Kenyan rental market. Core mission: **reduce rental scams** by gating tenant–landlord contact behind admin-verified landlord/caretaker accounts.

---

## Tech Stack

| Layer      | Choice                     | Rationale                                       |
|------------|----------------------------|--------------------------------------------------|
| Language   | Go 1.22                    | Concurrency, fast compilation, excellent stdlib  |
| Router     | chi v5                     | Lightweight, stdlib-compatible, middleware-friendly |
| Database   | PostgreSQL                 | UUID support, CHECK constraints, JSON, reliability |
| DB access  | Raw `database/sql` + `lib/pq` | Full control over queries; business rules enforced in SQL |
| Migrations | golang-migrate (files)     | Declarative, env-agnostic, widely adopted        |
| Auth       | JWT (HS256, access tokens) | Stateless auth; refresh tokens skipped for v1    |
| Hashing    | bcrypt                     | Industry standard for passwords                  |
| Testing    | `testing` + testify        | Table-driven tests, mocks, assertions            |
| Config     | `.env` / `godotenv`        | No hardcoded secrets                             |

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        HTTP Client                          │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                        chi Router                           │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────────┐  │
│  │ Middleware   │  │ Logger/      │  │ CORS              │  │
│  │ Stack       │  │ Recoverer    │  │                   │  │
│  └──────┬──────┘  └──────────────┘  └───────────────────┘  │
│         │                                                     │
│  ┌──────▼────────────────────────────────────────────────┐   │
│  │                  JWT Auth Middleware                   │   │
│  │         (Extracts user_id, role from token)            │   │
│  └──────┬────────────────────────────────────────────────┘   │
│         │                                                     │
│  ┌──────▼────────────────────────────────────────────────┐   │
│  │              RequireRole Middleware                     │   │
│  │         ("admin" / "landlord" / "tenant")               │   │
│  └──────┬────────────────────────────────────────────────┘   │
│         │                                                     │
│  ┌──────▼────────────────────────────────────────────────┐   │
│  │                    Handlers Layer                      │   │
│  │   auth_handler / admin_handler / landlord_handler /   │   │
│  │                 property_handler                       │   │
│  │   (HTTP concerns: parse request, write response)       │   │
│  └──────┬────────────────────────────────────────────────┘   │
│         │                                                     │
│  ┌──────▼────────────────────────────────────────────────┐   │
│  │                    Service Layer                       │   │
│  │   auth_service / admin_service / landlord_service /   │   │
│  │                property_service                        │   │
│  │   (Business rules, orchestration, validation)          │   │
│  └──────┬────────────────────────────────────────────────┘   │
│         │                                                     │
│  ┌──────▼────────────────────────────────────────────────┐   │
│  │                   Repository Layer                     │   │
│  │  postgres.go (Repository interface + PostgresRepo)    │   │
│  │  (Raw SQL, data access, rule enforcement in queries)  │   │
│  └──────┬────────────────────────────────────────────────┘   │
│         │                                                     │
└─────────┼─────────────────────────────────────────────────────┘
          │
┌─────────▼─────────────────────────────────────────────────────┐
│                       PostgreSQL                               │
│  users / landlord_profiles / tenant_profiles / properties /   │
│  units / property_reports / admin_audit_log                    │
└───────────────────────────────────────────────────────────────┘
```

### Layered Design

- **Handler** — HTTP concerns only: parse request, decode JSON, call service, encode response.
- **Service** — Business logic & rule enforcement. No knowledge of HTTP or database internals.
- **Repository** — Data access via interface. Business rule enforcement *also* lives at the SQL level for critical rules (e.g., tenant queries never see revoked-landlord data even if the service layer is bypassed).

---

## Data Model

```
users
├─ id UUID PK
├─ role TEXT CHECK (admin|landlord|tenant)
├─ phone TEXT UNIQUE NOT NULL
├─ email TEXT UNIQUE
├─ password_hash TEXT NOT NULL
└─ created_at TIMESTAMPTZ

landlord_profiles
├─ id UUID PK
├─ user_id UUID FK → users
├─ national_id_number TEXT NOT NULL
├─ id_document_url TEXT
├─ is_caretaker BOOLEAN DEFAULT false
├─ authorized_by_landlord_id UUID FK → landlord_profiles (for caretakers)
├─ verification_status TEXT CHECK (pending|verified|revoked)
├─ verified_by UUID FK → users (admin)
├─ verified_at TIMESTAMPTZ
├─ revoked_at TIMESTAMPTZ
├─ revoke_reason TEXT
└─ created_at TIMESTAMPTZ

tenant_profiles
├─ id UUID PK
├─ user_id UUID FK → users
└─ created_at TIMESTAMPTZ

properties
├─ id UUID PK
├─ landlord_id UUID FK → landlord_profiles
├─ name TEXT
├─ location TEXT
├─ address TEXT
├─ description TEXT
└─ created_at TIMESTAMPTZ

units
├─ id UUID PK
├─ property_id UUID FK → properties
├─ unit_label TEXT (e.g. "1A")
├─ bedrooms INT
├─ unit_type TEXT CHECK (studio|1br|2br|3br|other)
├─ rent_amount NUMERIC(12,2)
├─ status TEXT CHECK (vacant|occupied|reserved|maintenance) DEFAULT 'vacant'
└─ created_at TIMESTAMPTZ

property_reports
├─ id UUID PK
├─ property_id UUID FK → properties
├─ reported_by UUID FK → tenant_profiles
├─ reason TEXT
├─ resolved BOOLEAN DEFAULT false
└─ created_at TIMESTAMPTZ

admin_audit_log
├─ id UUID PK
├─ admin_id UUID FK → users
├─ action TEXT
├─ target_type TEXT
├─ target_id UUID
├─ reason TEXT
└─ created_at TIMESTAMPTZ
```

---

## Business Rules

1. **Verified-gated property/unit creation** — A landlord cannot create properties or units until `verification_status = 'verified'`. Enforced in `landlord_service.go`.

2. **Revoked landlords disappear from tenant queries** — When an admin revokes a landlord (`verification_status = 'revoked'`), all their properties and units stop appearing in tenant-facing `GET /properties` and `GET /properties/:id` immediately. Enforced at the **SQL query layer** (`SearchVerifiedProperties`, `GetVerifiedPropertyByID`) — not by a soft flag the frontend can ignore.

3. **Contact info gating** — `GET /units/:id/contact` returns landlord contact ONLY when: the unit's parent property's landlord is `verified` **AND** the unit's status is `vacant`. Non-vacant units remain browsable but expose no contact details. Enforced in `GetUnitContactDetails` SQL query + code check.

4. **Audit trail** — Every admin verify/revoke action writes a row to `admin_audit_log`. Enforced in `admin_service.go`.

5. **Caretaker authorization chain** — A caretaker (`is_caretaker = true`) must have a non-null `authorized_by_landlord_id` pointing to an already-verified landlord before an admin can approve them. Enforced in `admin_service.go`.

6. **Property reporting** — Tenants can report a property. Reports do not auto-hide anything (manual admin decision). Admins query and resolve reports.

---

## API Overview

### Auth
| Method | Path | Role | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/register` | Public | Create user + role-specific profile |
| POST | `/api/v1/auth/login` | Public | Returns JWT token |

### Admin
| Method | Path | Role | Description |
|--------|------|------|-------------|
| GET | `/api/v1/admin/verifications?status=` | Admin | List landlord profiles by status |
| POST | `/api/v1/admin/verifications/:landlord_id/approve` | Admin | Approve a landlord/caretaker |
| POST | `/api/v1/admin/verifications/:landlord_id/revoke` | Admin | Revoke with reason |
| GET | `/api/v1/admin/reports?resolved=` | Admin | List property reports |
| POST | `/api/v1/admin/reports/:id/resolve` | Admin | Mark report resolved |
| GET | `/api/v1/admin/audit-log` | Admin | Full audit trail |

### Landlord
| Method | Path | Role | Description |
|--------|------|------|-------------|
| GET | `/api/v1/landlord/me` | Landlord | Own profile + verification status |
| POST | `/api/v1/landlord/properties` | Landlord | Create property (requires verified) |
| GET | `/api/v1/landlord/properties` | Landlord | Own properties |
| POST | `/api/v1/landlord/properties/:id/units` | Landlord | Add unit to property |
| PATCH | `/api/v1/landlord/units/:id` | Landlord | Update unit status/details |

### Public / Tenant
| Method | Path | Role | Description |
|--------|------|------|-------------|
| GET | `/api/v1/properties?location=&bedrooms=&unit_type=` | Public | Browse verified-landlord properties |
| GET | `/api/v1/properties/:id` | Public | Property detail with units |
| GET | `/api/v1/units/:id/contact` | Public | Landlord contact (vacant + verified only) |
| POST | `/api/v1/properties/:id/report` | Tenant | Report a property |

---

## Workflows

### Tenant Flow
```
Register ──→ Browse properties (verified landlords only)
               │
               ├── View property detail + unit statuses
               │
               └── Request contact on vacant unit
                     → 200: phone/email returned
                     → 403: occupied/maintenance/reserved or landlord revoked
```

### Landlord Flow
```
Register (verification_status = 'pending')
   │
   └── Wait for admin approval
          │
          ├── Approved (verified)
          │     └── Create properties → add units → manage statuses
          │
          └── Revoked
                └── All properties hidden from tenant queries
```

### Admin Verification Flow
```
Landlord registers ──→ Admin GETs /verifications?status=pending
                         │
                         ├── POST /verifications/:id/approve
                         │   ├── Checks caretaker → authorizer chain (Rule 5)
                         │   ├── Sets verification_status = 'verified'
                         │   └── Writes audit log (Rule 4)
                         │
                         └── POST /verifications/:id/revoke
                             ├── Sets verification_status = 'revoked'
                             ├── Properties vanish from tenant queries (Rule 2)
                             └── Writes audit log (Rule 4)
```

---

## Project Structure

```
.
├── .env.example
├── API.md
├── README.md
├── cmd/
│   ├── seed/
│   │   └── main.go              # Admin account seeder
│   └── server/
│       └── main.go               # HTTP server entrypoint
├── go.mod
├── internal/
│   ├── config/
│   │   └── config.go             # Env-based configuration
│   ├── domain/
│   │   ├── errors.go             # Domain errors
│   │   └── models.go             # Entities + DTOs
│   ├── handlers/
│   │   ├── admin_handler.go
│   │   ├── auth_handler.go
│   │   ├── landlord_handler.go
│   │   └── property_handler.go
│   ├── middleware/
│   │   └── auth.go               # JWT auth + role guard
│   ├── repository/
│   │   └── postgres.go           # Repository interface + Postgres implementation
│   └── service/
│       ├── admin_service.go
│       ├── auth_service.go
│       ├── landlord_service.go
│       └── property_service.go
├── migrations/
│   ├── 000001_init_schema.down.sql
│   └── 000001_init_schema.up.sql
└── tests/
    ├── integration/
    │   └── integration_test.go   # End-to-end Postgres tests
    └── unit/
        ├── service_test.go       # Mocked service-layer tests (Rules 1, 3, 5)
        └── validation_test.go    # Table-driven handler input validation
```

---

## Getting Started

### Prerequisites

- Go 1.22+
- PostgreSQL 14+
- golang-migrate CLI (`brew install golang-migrate` or `go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest`)

### Setup

```bash
# 1. Clone and enter project
git clone <repo> && cd kenya-houses

# 2. Copy env config
cp .env.example .env
# Edit .env with your database credentials

# 3. Create database
createdb kenyahouses

# 4. Run migrations
migrate -path migrations -database "$DATABASE_URL" up

# 5. Seed an admin account
go run cmd/seed/main.go
# Creates: Phone (+254700000000), Password (AdminPass123!)

# 6. Run server
go run cmd/server/main.go
# Server starts on :8080
```

### Testing

```bash
# Unit tests (no database required)
go test ./tests/unit -v

# Integration tests (requires a running PostgreSQL)
# Set TEST_DATABASE_URL in .env if different from default
go test ./tests/integration -v
```

Integration tests use a real PostgreSQL instance (default: `postgres://postgres:postgres@localhost:5432/kenyahouses_test?sslmode=disable`). If the DB is unreachable, tests are skipped via `t.Skip`.

---

## Key Design Decisions

### Why `database/sql` + raw queries instead of GORM/sqlc?
The business rules require precise SQL-level enforcement (e.g., joining `properties` with `landlord_profiles` to filter by `verification_status` in tenant queries). Raw SQL gives full control over these joins and avoids ORM magic that could accidentally leak data. The Repository interface still allows swapping implementations.

### Why chi over Gin?
chi has a smaller dependency footprint, composes natively with `net/http` handlers, and its middleware chaining is more explicit. Both are excellent choices; chi was selected for its adherence to the Go stdlib philosophy.

### Refresh tokens
Access tokens expire in 24 hours. Refresh tokens are not implemented in v1 — they would slot into `POST /api/v1/auth/refresh` with a longer-lived signed token stored alongside the access token.

### Business rules at both layers
Rules 2 and 3 are enforced **both** in the service layer (as a defense-in-depth check) and in the repository SQL queries (as the definitive gate). This ensures that even if a handler or service is misconfigured, the SQL layer prevents data leaks.
