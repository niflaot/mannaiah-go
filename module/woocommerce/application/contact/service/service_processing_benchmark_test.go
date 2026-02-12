package service

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/zap"
	"mannaiah/module/woocommerce/port"
)

// benchmarkTarget defines lightweight upsert behavior for processCommands benchmarks.
type benchmarkTarget struct{}

// UpsertByEmail returns a stable updated outcome for benchmark commands.
func (benchmarkTarget) UpsertByEmail(ctx context.Context, command port.ContactSyncCommand) (outcome port.UpsertOutcome, err error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	return port.UpsertOutcomeUpdated, nil
}

// BenchmarkProcessCommands measures throughput for worker-parallel command processing.
func BenchmarkProcessCommands(b *testing.B) {
	workerCounts := []int{1, 4, 8}
	commandCounts := []int{100, 1000}

	for _, workerCount := range workerCounts {
		for _, commandCount := range commandCounts {
			name := fmt.Sprintf("workers=%d/commands=%d", workerCount, commandCount)
			b.Run(name, func(b *testing.B) {
				commands := make([]port.ContactSyncCommand, commandCount)
				for index := 0; index < commandCount; index++ {
					commands[index] = port.ContactSyncCommand{
						Email: fmt.Sprintf("bench-%d@example.com", index),
					}
				}

				service := &ContactSyncService{
					target: benchmarkTarget{},
					logger: zap.NewNop(),
					cfg: SyncConfig{
						WorkerCount: workerCount,
					},
				}

				ctx := context.Background()
				b.ResetTimer()
				for index := 0; index < b.N; index++ {
					summary := &SyncSummary{}
					if err := service.processCommands(ctx, commands, summary); err != nil {
						b.Fatalf("processCommands() error = %v", err)
					}
				}
			})
		}
	}
}
