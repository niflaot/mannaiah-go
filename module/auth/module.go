package auth

import (
	authruntime "mannaiah/module/auth/runtime"

	"go.uber.org/zap"
)

// Authorizer defines authentication and authorization behavior required by module adapters.
type Authorizer = authruntime.Authorizer

// Module defines auth-module composition dependencies.
type Module = authruntime.Module

// New creates an auth module with JWT verification and scope authorization support.
func New(cfg Config, coreEnvironment string, logger *zap.Logger) (*Module, error) {
	return authruntime.New(cfg, coreEnvironment, logger)
}
