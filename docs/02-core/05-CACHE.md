# Cache & Redis

Mannaiah caches are accessed through the `cache.Store` interface, which is implemented by the
`redis` package. Cache consumers are not coupled to Redis — any backend that satisfies the interface
can be substituted.

## cache.Store Interface

| Method | Description |
|--------|-------------|
| `Ping(ctx)` | Health check |
| `Get(ctx, key)` | Retrieve a value by key |
| `Set(ctx, key, value, ttl)` | Store a value with expiry |
| `Delete(ctx, key)` | Remove a key |
| `Keys(ctx, pattern)` | List keys matching a glob pattern |
| `GetByPattern(ctx, pattern)` | Retrieve all key/value pairs matching a pattern |
| `Close()` | Release the underlying connection |

## Circuit Breaker

The Redis implementation wraps every operation with a configurable circuit breaker (Sony
`gobreaker`). When the failure threshold is reached, the breaker opens and operations return
`ErrUnavailable` immediately, preventing cascade failures while Redis is unreachable.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_URL` | `redis://localhost:6379/0` | Redis connection URL |
| `REDIS_USERNAME` | _(empty)_ | ACL username override |
| `REDIS_PASSWORD` | _(empty)_ | Password override |
| `REDIS_POOL_SIZE` | `20` | Max socket connections |
| `REDIS_MIN_IDLE_CONNS` | `5` | Min idle pooled connections |
| `REDIS_DIAL_TIMEOUT_MS` | `5000` | Connection dial timeout |
| `REDIS_READ_TIMEOUT_MS` | `3000` | Read operation timeout |
| `REDIS_WRITE_TIMEOUT_MS` | `3000` | Write operation timeout |
| `REDIS_SCAN_COUNT` | `200` | SCAN page size hint |
| `REDIS_BATCH_SIZE` | `200` | MGET batch size for pattern retrieval |
| `REDIS_CIRCUIT_BREAKER_ENABLED` | `true` | Enable circuit breaker |
| `REDIS_CIRCUIT_BREAKER_MAX_REQUESTS` | `1` | Half-open trial requests |
| `REDIS_CIRCUIT_BREAKER_INTERVAL_MS` | `60000` | Closed-state counter reset window |
| `REDIS_CIRCUIT_BREAKER_TIMEOUT_MS` | `30000` | Open-state cooldown window |
| `REDIS_CIRCUIT_BREAKER_FAILURE_THRESHOLD` | `5` | Failures before breaker opens |
