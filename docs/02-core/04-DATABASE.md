# Database

The `database` package provides GORM connection management, a shared base model, a generic CRUD
service, and a versioned migration runner.

## Base Model

All persistent entities embed `database.Model`:

```go
type Model struct {
    ID        uint           `gorm:"primaryKey"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"` // soft-delete
}
```

## Generic CRUD Service

`Service[T]` is a generic implementation of `CRUDService[T]` that covers standard persistence
operations without boilerplate:

| Method | Description |
|--------|-------------|
| `Create(ctx, entity)` | Insert a new record |
| `Read(ctx, id)` | Fetch by primary key |
| `Find(ctx, query)` | List with filters, ordering, pagination |
| `Paginate(ctx, query)` | Paginated list with total counts |
| `Update(ctx, id, updates)` | Partial update by primary key |
| `Delete(ctx, id)` | Soft-delete by primary key |

`Query` carries dynamic filter fields: `Where`, `Order`, `Limit`, `Offset`, `Preloads`,
`ExcludeIDs`, `Page`, and `PageSize`.

## Migrations

Migration files are embedded SQL scripts versioned with numeric prefixes (`0001_*.up.sql` /
`0001_*.down.sql`). They run automatically on startup when `DB_MIGRATIONS_ENABLED=true` or via
the standalone `cmd/migrate` CLI.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_DRIVER` | `sqlite` | Database driver: `sqlite`, `postgres`, `mysql` |
| `DB_DSN` | `file::memory:?cache=shared` | Connection string |
| `DB_MAX_OPEN_CONNS` | `25` | Max open connections |
| `DB_MAX_IDLE_CONNS` | `5` | Max idle connections |
| `DB_CONN_MAX_LIFETIME_MS` | `600000` | Max connection lifetime |
| `DB_CONN_MAX_IDLE_TIME_MS` | `300000` | Max idle connection lifetime |
| `DB_GORM_LOG_LEVEL` | `warn` | GORM log level: `silent`, `error`, `warn`, `info` |
| `DB_GORM_SLOW_QUERY_THRESHOLD_MS` | `200` | Slow-query warning threshold |
| `DB_MIGRATIONS_ENABLED` | `true` | Run pending migrations on startup |
| `DB_MIGRATIONS_TABLE` | `schema_migrations` | Migration state table name |
| `DB_MIGRATIONS_TIMEOUT_MS` | `30000` | Migration execution timeout |
