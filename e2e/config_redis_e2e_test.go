package e2e_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"mannaiah/module/auth"
	coreconfig "mannaiah/module/core/config"
	coredatabase "mannaiah/module/core/database"
	coreredis "mannaiah/module/core/redis"
)

// TestConfigEnvAndRedisE2E verifies environment loading precedence and Redis runtime behavior end-to-end.
func TestConfigEnvAndRedisE2E(t *testing.T) {
	tracer := newStepTracer(t)

	tracer.Step("start miniredis instance")
	miniRedis := miniredis.RunT(t)

	tracer.Step("prepare temporary env file")
	envFile := writeTempEnvFile(t, fmt.Sprintf("CORE_PORT=7001\nDB_DRIVER=sqlite\nDB_DSN=file::memory:?cache=shared\nREDIS_URL=redis://%s/0\nREDIS_POOL_SIZE=20\nLOGTO_ISSUER=https://issuer.example\nLOGTO_AUDIENCE=https://api.mannaiah.e2e\n", miniRedis.Addr()))

	t.Setenv("CORE_PORT", "7011")
	t.Setenv("REDIS_POOL_SIZE", "50")

	var coreCfg coreconfig.Core
	var redisCfg coreredis.Config
	var dbCfg coredatabase.Config
	var authCfg auth.Config

	tracer.Step("load configuration with env overrides")
	if err := coreconfig.Load(envFile, tracer.logger, &coreCfg, &redisCfg, &dbCfg, &authCfg); err != nil {
		t.Fatalf("coreconfig.Load() error = %v", err)
	}
	if coreCfg.Port != 7011 {
		t.Fatalf("coreCfg.Port = %d, want %d", coreCfg.Port, 7011)
	}
	if redisCfg.PoolSize != 50 {
		t.Fatalf("redisCfg.PoolSize = %d, want %d", redisCfg.PoolSize, 50)
	}
	if authCfg.Issuer != "https://issuer.example" {
		t.Fatalf("authCfg.Issuer = %q, want %q", authCfg.Issuer, "https://issuer.example")
	}

	tracer.Step("initialize redis store")
	store, err := coreredis.New(redisCfg, tracer.logger)
	if err != nil {
		t.Fatalf("coreredis.New() error = %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	ctx := context.Background()

	tracer.Step("run redis ping")
	if err := store.Ping(ctx); err != nil {
		t.Fatalf("store.Ping() error = %v", err)
	}

	tracer.Step("write redis keys")
	if err := store.Set(ctx, "contacts:1", `{"name":"john"}`, time.Minute); err != nil {
		t.Fatalf("store.Set(contacts:1) error = %v", err)
	}
	if err := store.Set(ctx, "contacts:2", `{"name":"mary"}`, time.Minute); err != nil {
		t.Fatalf("store.Set(contacts:2) error = %v", err)
	}

	tracer.Step("read redis key")
	value, err := store.Get(ctx, "contacts:1")
	if err != nil {
		t.Fatalf("store.Get() error = %v", err)
	}
	if value != `{"name":"john"}` {
		t.Fatalf("store.Get() = %q, want %q", value, `{"name":"john"}`)
	}

	tracer.Step("scan redis keys by pattern")
	values, err := store.GetByPattern(ctx, "contacts:*")
	if err != nil {
		t.Fatalf("store.GetByPattern() error = %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("len(values) = %d, want %d", len(values), 2)
	}

	tracer.Step("delete redis key")
	deleted, err := store.Delete(ctx, "contacts:2")
	if err != nil {
		t.Fatalf("store.Delete() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want %d", deleted, 1)
	}

	tracer.Step("assert e2e trace logs")
	tracer.AssertStepCount(8)
}

// writeTempEnvFile creates a temporary environment file with the provided content.
func writeTempEnvFile(t *testing.T, content string) string {
	t.Helper()

	directory := t.TempDir()
	path := filepath.Join(directory, ".env")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	return path
}
