# gokit-gen

Code generation tool for go-kit projects. Provides database migration and GORM model generation.

## Install

```bash
go install github.com/tsopia/go-kit/cmd/gokit-gen@latest
```

## Usage

### Quick Start

```bash
# Auto-discover config, migrate up, then generate models
gokit-gen sync

# Or run separately
gokit-gen migrate up          # Run pending migrations
gokit-gen gen                 # Generate models from database
```

### Commands

| Command | Description |
|---------|-------------|
| `sync` | Run migrate up then gen (recommended) |
| `migrate up` | Execute pending migrations |
| `migrate down` | Rollback one migration |
| `migrate status` | Show migration status |
| `gen` | Generate GORM models |

### Flags

```
--dsn              Database DSN (skips auto-discovery)
--driver           Database driver: mysql, postgres
--migrate-source   Migrations directory (default: migrations)
--out              Output directory (default: internal/dal)
--tables           Comma-separated tables to generate
```

### Examples

```bash
# Use defaults
gokit-gen sync

# Custom output directory
gokit-gen sync --out ./pkg/model

# Generate specific tables
gokit-gen gen --tables user,order,product

# Manual connection (no auto-discovery)
gokit-gen sync --driver mysql --dsn "root:pass@tcp(localhost:3306)/mydb"

# Custom migrations path
gokit-gen migrate up --migrate-source ./db/migrations
```

## Configuration Auto-Discovery

`gokit-gen` tries to find database config from:

1. **Go source files** - Looks for `database.Config` in `main.go`, `cmd/*/main.go`
2. **.env file** - Reads `DATABASE_URL` or `DSN`
3. **docker-compose.yml** - Database service config (TBD)

If auto-discovery fails, provide `--dsn` and `--driver`.

## Default Conventions

| Item | Default | Override |
|------|---------|----------|
| Migrations | `migrations/` | `--migrate-source` |
| Generated code | `internal/dal/` | `--out` |
| Tables | All tables | `--tables` |

## dbmigrate Package

For programmatic use in your application:

```go
import "github.com/tsopia/go-kit/dbmigrate"

// Run migrations on startup
err := dbmigrate.Up(ctx, dbmigrate.Config{
    SourcePath: "migrations",
    Database:   db,
})
```

## Workflow

```bash
# 1. Create migration files
# migrations/001_create_users.up.sql
# migrations/001_create_users.down.sql

# 2. Run sync (migrate + gen)
gokit-gen sync

# 3. Use generated models in your code
# internal/dal/query/user.gen.go
```
