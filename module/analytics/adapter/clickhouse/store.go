package clickhouse

import "context"

// StoreAdapter defines clickhouse-backed analytics store behavior.
type StoreAdapter struct {
	// client defines clickhouse client dependencies.
	client *Client
}

// NewStoreAdapter creates clickhouse-backed analytics store adapters.
func NewStoreAdapter(client *Client) *StoreAdapter {
	return &StoreAdapter{client: client}
}

// Ping verifies analytics backend connectivity.
func (s *StoreAdapter) Ping(ctx context.Context) error {
	if s == nil || s.client == nil {
		return nil
	}

	return s.client.Ping(ctx)
}
