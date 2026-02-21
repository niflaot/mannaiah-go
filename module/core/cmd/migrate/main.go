package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	coreconfig "mannaiah/module/core/config"
	coredatabase "mannaiah/module/core/database"
	coredatabasemigration "mannaiah/module/core/database/migration"
	corelogger "mannaiah/module/core/logger"
)

// migrateOptions defines CLI options for dedicated migration command execution.
type migrateOptions struct {
	envFile      string
	operation    string
	steps        int
	all          bool
	forceVersion int
	respectEnv   bool
}

// main executes dedicated migration command lifecycle.
func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

// run executes migration command flow for provided arguments.
func run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}

	var coreCfg coreconfig.Core
	var dbCfg coredatabase.Config
	if loadErr := coreconfig.Load(opts.envFile, nil, &coreCfg, &dbCfg); loadErr != nil {
		return fmt.Errorf("load configuration: %w", loadErr)
	}

	logger, err := corelogger.New(coreCfg.Logging)
	if err != nil {
		return fmt.Errorf("build logger: %w", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	db, err := coredatabase.Open(dbCfg, logger)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("access sql db handle: %w", err)
	}
	defer func() {
		_ = sqlDB.Close()
	}()

	migrationCfg := coredatabasemigration.FromDatabaseConfig(dbCfg)
	if !opts.respectEnv {
		migrationCfg.Enabled = true
	}

	result, err := coredatabasemigration.Run(ctx, db, migrationCfg, coredatabasemigration.RunOptions{
		Operation:    coredatabasemigration.Operation(strings.ToLower(strings.TrimSpace(opts.operation))),
		Steps:        opts.steps,
		All:          opts.all,
		ForceVersion: opts.forceVersion,
	}, logger)
	if err != nil {
		return fmt.Errorf("execute migration operation: %w", err)
	}

	if result != nil {
		_, _ = fmt.Fprintf(stdout, "migration operation=%s version=%d dirty=%t\n", strings.ToLower(strings.TrimSpace(opts.operation)), result.Version, result.Dirty)
	}
	_, _ = fmt.Fprintln(stderr, "migration command completed")

	return nil
}

// parseOptions parses dedicated migration command options.
func parseOptions(args []string) (migrateOptions, error) {
	defaults := migrateOptions{
		envFile:      ".env",
		operation:    string(coredatabasemigration.OperationUp),
		steps:        0,
		all:          false,
		forceVersion: -1,
		respectEnv:   false,
	}

	flags := flag.NewFlagSet("migrate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&defaults.envFile, "env-file", defaults.envFile, "Path to .env file")
	flags.StringVar(&defaults.operation, "operation", defaults.operation, "Migration operation: up|down|version|force")
	flags.IntVar(&defaults.steps, "steps", defaults.steps, "Step count for up/down operations")
	flags.BoolVar(&defaults.all, "all", defaults.all, "Rollback all versions when operation=down")
	flags.IntVar(&defaults.forceVersion, "force-version", defaults.forceVersion, "Version for operation=force")
	flags.BoolVar(&defaults.respectEnv, "respect-env-enabled", defaults.respectEnv, "Respect DB_MIGRATIONS_ENABLED value from environment")
	if err := flags.Parse(args); err != nil {
		return migrateOptions{}, fmt.Errorf("parse migration command flags: %w", err)
	}

	operation := strings.ToLower(strings.TrimSpace(defaults.operation))
	switch operation {
	case string(coredatabasemigration.OperationUp), string(coredatabasemigration.OperationDown), string(coredatabasemigration.OperationVersion), string(coredatabasemigration.OperationForce):
	default:
		return migrateOptions{}, fmt.Errorf("invalid operation %q: expected up|down|version|force", defaults.operation)
	}
	defaults.operation = operation

	if operation != string(coredatabasemigration.OperationDown) && defaults.all {
		return migrateOptions{}, errors.New("--all is only valid when --operation=down")
	}
	if operation != string(coredatabasemigration.OperationForce) && defaults.forceVersion >= 0 {
		return migrateOptions{}, errors.New("--force-version is only valid when --operation=force")
	}
	if operation == string(coredatabasemigration.OperationForce) && defaults.forceVersion < 0 {
		return migrateOptions{}, errors.New("--force-version is required when --operation=force")
	}
	if operation == string(coredatabasemigration.OperationVersion) && (defaults.steps != 0 || defaults.all) {
		return migrateOptions{}, errors.New("--steps/--all are not valid when --operation=version")
	}

	return defaults, nil
}
