package e2e_test

import (
	"context"
	"testing"
	"time"

	coreconfig "mannaiah/module/core/config"
	corecron "mannaiah/module/core/cron"
)

// TestCronConfigSchedulerE2E verifies config loading and scheduled execution behavior end-to-end.
func TestCronConfigSchedulerE2E(t *testing.T) {
	tracer := newStepTracer(t)

	tracer.Step("prepare cron env file")
	envFile := writeTempEnvFile(t, "CRON_LOCATION=UTC\nCRON_WITH_SECONDS=false\n")
	t.Setenv("CRON_WITH_SECONDS", "true")

	var cronCfg corecron.Config

	tracer.Step("load cron configuration with env override")
	if err := coreconfig.Load(envFile, tracer.logger, &cronCfg); err != nil {
		t.Fatalf("coreconfig.Load() error = %v", err)
	}
	if cronCfg.Location != "UTC" {
		t.Fatalf("cronCfg.Location = %q, want %q", cronCfg.Location, "UTC")
	}
	if !cronCfg.WithSeconds {
		t.Fatalf("cronCfg.WithSeconds = false, want true")
	}

	tracer.Step("initialize abstract cron scheduler")
	scheduler, err := corecron.NewScheduler(cronCfg, tracer.logger)
	if err != nil {
		t.Fatalf("corecron.NewScheduler() error = %v", err)
	}

	runs := make(chan struct{}, 1)

	tracer.Step("register second-resolution cron job")
	if _, err := scheduler.AddFunc("*/1 * * * * *", func() {
		select {
		case runs <- struct{}{}:
		default:
		}
	}); err != nil {
		t.Fatalf("scheduler.AddFunc() error = %v", err)
	}

	tracer.Step("start scheduler and await execution")
	scheduler.Start()
	waitForCronRun(t, runs, 2*time.Second)

	tracer.Step("stop scheduler")
	stopContext, stopCancel := context.WithTimeout(context.Background(), time.Second)
	defer stopCancel()
	if err := scheduler.Stop(stopContext); err != nil {
		t.Fatalf("scheduler.Stop() error = %v", err)
	}

	tracer.Step("assert e2e trace logs")
	tracer.AssertStepCount(6)
}

// waitForCronRun waits for cron execution signals in E2E scenarios.
func waitForCronRun(t *testing.T, signal <-chan struct{}, timeout time.Duration) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(timeout):
		t.Fatalf("timeout waiting for cron execution")
	}
}
