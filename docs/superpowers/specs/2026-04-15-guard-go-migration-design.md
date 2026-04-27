# Guard-Go Migration Design

## Overview

Migrate the Guard security guardrail platform from Python (FastAPI) to Go (Gin) for improved performance. The Go version maintains 100% API compatibility with the Python version, allowing the existing Vue 3 frontend to work unchanged.

**Source:** `/Users/luzhipeng/projects/ynet/guard/` (Python)
**Target:** `/Users/luzhipeng/projects/ynet/guard-go/` (Go)

## Tech Stack

| Component | Choice | Reason |
|-----------|--------|--------|
| HTTP Framework | Gin | Mainstream, performant, large community |
| ORM | GORM | MySQL/OceanBase compatible, connection pooling |
| JSON | sonic | High-performance serialization (bytedance) |
| Redis | go-redis/v9 | Standalone + cluster support |
| Elasticsearch | olivere/elastic/v7 | Mature, dual-write support |
| Milvus | milvus-sdk-go | Vector search for KB matching |
| JWT | golang-jwt/jwt/v5 | Standard JWT library |
| ID Generation | oklog/ulid | ULID for request IDs and primary keys |
| Config | env + DB | Priority: DB > env > defaults (30s cache) |

## Project Structure

```
guard-go/
├── cmd/guard/main.go
├── internal/
│   ├── config/config.go           # Settings (env + DB hot-reload)
│   ├── middleware/
│   │   ├── auth.go                # JWT + API Key auth
│   │   ├── cors.go
│   │   └── logger.go
│   ├── model/                     # GORM models (17 tables)
│   │   ├── tenant.go
│   │   ├── user.go
│   │   ├── api_key.go
│   │   ├── knowledge_base.go
│   │   ├── kb_entry.go
│   │   ├── category.go
│   │   ├── audit_log.go
│   │   ├── operation_log.go
│   │   ├── system_config.go
│   │   ├── shield_category.go
│   │   ├── shield_product.go
│   │   ├── shield_business.go
│   │   ├── shield_policy_config.go
│   │   ├── shield_blocklist.go
│   │   ├── shield_blocklist_entry.go
│   │   ├── shield_business_blocklist.go
│   │   └── shield_audit_log.go
│   ├── handler/                   # Gin route handlers
│   │   ├── auth.go
│   │   ├── guard.go
│   │   ├── kb.go
│   │   ├── audit.go
│   │   ├── category.go
│   │   ├── user.go
│   │   ├── apikey.go
│   │   ├── tenant.go
│   │   ├── settings.go
│   │   └── shield/
│   │       ├── check.go
│   │       ├── category.go
│   │       ├── product.go
│   │       ├── business.go
│   │       ├── policy.go
│   │       ├── blocklist.go
│   │       └── audit.go
│   ├── service/                   # Business logic
│   │   ├── guard_engine.go        # Guard detection pipeline
│   │   ├── shield_engine.go       # Shield detection pipeline
│   │   ├── security_model.go      # AI model client (circuit breaker + LRU)
│   │   ├── embedding.go           # Embedding client (batch + Redis cache)
│   │   ├── knowledge_base.go      # Dual-channel KB matching (ES + Milvus)
│   │   ├── shield_blocklist.go    # Blocklist/allowlist checking
│   │   ├── config_service.go      # Dynamic config (DB > env > default)
│   │   ├── audit.go               # Guard audit logging
│   │   ├── shield_audit.go        # Shield audit logging
│   │   ├── user.go                # User auth & management
│   │   ├── tenant.go              # Multi-tenant management
│   │   ├── operation_log.go       # Operation tracking
│   │   └── import_task.go         # CSV bulk import
│   ├── store/                     # Data access layer
│   │   ├── mysql.go               # GORM init + connection pool
│   │   ├── redis.go               # go-redis (standalone/cluster)
│   │   ├── elasticsearch.go       # ES client (dual-write)
│   │   └── milvus.go              # Milvus vector DB
│   └── pkg/
│       ├── jwt.go                 # JWT create/verify
│       ├── hash.go                # bcrypt + SHA256
│       ├── ulid.go                # ULID generation
│       └── response.go           # Unified API response
├── deploy/
│   ├── Dockerfile
│   ├── docker-compose.yml
│   └── entrypoint.sh
├── migrations/
│   └── 001_init.sql               # Full schema (from Python alembic)
├── frontend/                      # Vue 3 frontend (copied from guard)
├── go.mod
└── go.sum
```

## Database Models (17 tables)

All tables maintain exact same schema as Python version for compatibility.

### Core
- `tenants` - Multi-tenant isolation
- `users` - User accounts (admin/operator/viewer/superadmin roles)
- `api_keys` - API key management (SHA256 hashed)
- `system_config` - Dynamic key-value configuration
- `operation_logs` - CRUD audit trail

### Guard
- `knowledge_bases` - KB collections (system + user)
- `kb_entries` - KB entries (keywords + semantic text)
- `categories` - Content categories (self-referential tree)
- `audit_logs` - Guard detection logs

### Shield (Eagle Shield)
- `shield_categories` - 3-level category hierarchy
- `shield_products` - Product definitions
- `shield_businesses` - Business configs (enable AI/KB/blocklist flags)
- `shield_policy_configs` - Risk level → action mapping (R1/R2/R3)
- `shield_blocklists` - Blocklist/allowlist definitions
- `shield_blocklist_entries` - Blocklist values
- `shield_business_blocklists` - Business↔blocklist many-to-many
- `shield_audit_logs` - Shield detection logs

## API Endpoints (100% compatible)

### Authentication (`/v1/auth`)
- POST `/login`, `/logout`, `/change-password`
- GET `/me`

### Guard Detection (`/v1/guard`)
- POST `/check` — content safety check (AI model + KB matching)

### Knowledge Base (`/v1/kb`)
- CRUD for knowledge bases and entries
- POST `/{kb_id}/entries/import` — CSV bulk import

### Shield Detection (`/v1/shield/check`)
- POST `/text` — text content detection

### Shield Management (`/v1/shield/`)
- CRUD for categories, products, businesses, policies, blocklists
- POST `/policies/batch` — batch policy creation

### Shield Audit (`/v1/shield/audit`)
- GET `/` — query detection logs
- GET `/stats` — audit statistics

### Guard Audit (`/v1/audit`)
- GET `/` — query guard logs (ES full-text search)
- GET `/stats/timeline`, `/stats/risk-distribution`, `/stats/top-blocked`

### Management APIs
- `/v1/users` — user CRUD
- `/v1/api_keys` — API key CRUD
- `/v1/tenants` — tenant CRUD
- `/v1/category` — category CRUD
- `/v1/settings` — system config get/set

### Health
- GET `/health` — returns `{"status": "ok"}`

## Performance Design

### Guard Detection Pipeline
```
Request → errgroup parallel:
  ├── goroutine 1: AI Model Check (with LRU cache + circuit breaker)
  └── goroutine 2: KB Matching
       ├── ES keyword search
       └── Milvus vector search
→ Merge results → Response
```

### Shield Detection Pipeline
```
Request → Load business config
→ Allowlist check (fast path exit)
→ errgroup parallel:
  ├── Blocklist check
  ├── AI Model check
  └── KB check
→ Policy rule matching → Response
```

### Key Optimizations
- **sync.Map LRU cache** for AI model results (1024 entries, 5min TTL)
- **Circuit breaker** on AI model API (3 failures → 60s cooldown)
- **Redis cache** for embeddings (1h TTL) and API keys
- **Connection pooling** via GORM (pool_size=10, max_overflow=20)
- **sonic JSON** for high-throughput serialization
- **sync.RWMutex** for config hot-reload (30s refresh)

## Authentication

Two methods (same as Python):
1. **JWT Bearer** — for web UI, `Authorization: Bearer <token>`
2. **API Key** — for programmatic access, `X-API-Key: <key>`

Multi-tenant isolation via `tenant_id` on all queries.

## Deployment

### Dockerfile (multi-stage)
- Stage 1: Node 22-alpine → build Vue frontend
- Stage 2: golang:1.24-alpine → build Go binary
- Stage 3: alpine → final image (Go binary + frontend static + migrations)

### Docker Compose
Same structure as Python version: guard-app + optional middleware containers.
