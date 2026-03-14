package clickhouse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

var (
	// ErrEmptyDSN is returned when clickhouse DSN is empty.
	ErrEmptyDSN = errors.New("analytics clickhouse dsn is required")
)

// Config defines clickhouse client configuration values.
type Config struct {
	// DSN defines clickhouse connection strings.
	DSN string
	// MaxOpenConns defines max open connections.
	MaxOpenConns int
	// MaxIdleConns defines max idle connections.
	MaxIdleConns int
	// ConnMaxLifetime defines connection max lifetime values.
	ConnMaxLifetime time.Duration
}

// Client defines clickhouse client behavior.
type Client struct {
	// db defines SQL driver handle dependencies.
	db *sql.DB
}

// NewClient creates clickhouse client dependencies.
func NewClient(cfg Config) (*Client, error) {
	dsn := strings.TrimSpace(cfg.DSN)
	if dsn == "" {
		return nil, ErrEmptyDSN
	}

	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, fmt.Errorf("open clickhouse connection: %w", err)
	}
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	return &Client{db: db}, nil
}

// Ping verifies clickhouse connectivity.
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.db == nil {
		return errors.New("clickhouse client is not initialized")
	}

	if err := c.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping clickhouse: %w", err)
	}

	return nil
}

// Close closes clickhouse client resources.
func (c *Client) Close() error {
	if c == nil || c.db == nil {
		return nil
	}

	return c.db.Close()
}
