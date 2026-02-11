# Cron Package

`cron` provides a provider-agnostic scheduler interface backed by `robfig/cron` for in-process periodic jobs.

## Features
- Decoupled scheduler contract (`cron.Scheduler`) for module-level orchestration.
- Config-driven timezone and seconds precision support.
- Panic-safe job execution with Zap error logs.
- Graceful shutdown with context-aware wait semantics.

## Usage Rules
- Load `cron.Config` through the shared config loader.
- Depend on `cron.Scheduler` in modules that need scheduled behavior.
- Keep business logic outside scheduler setup; schedule application/use-case entrypoints.

## Key Methods / Endpoints / Events
- Methods:
  - `cron.New(cfg, providedLogger)`
  - `cron.NewScheduler(cfg, providedLogger)`
  - `cron.JobFunc.Run()`
  - `(*cron.Service).Add(spec, job)`
  - `(*cron.Service).AddFunc(spec, job)`
  - `(*cron.Service).Remove(id)`
  - `(*cron.Service).Entries()`
  - `(*cron.Service).Start()`
  - `(*cron.Service).Run()`
  - `(*cron.Service).Stop(ctx)`
- Endpoints: none in this package.
- Events: panic recovery emits `cron job panic recovered` through Zap logger records.
