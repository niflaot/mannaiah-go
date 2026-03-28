# Authentication & Authorization

The `auth` module provides stateless JWT verification and fine-grained permission checking for all
inbound HTTP requests. It integrates with [Logto](https://docs.logto.io) as the identity provider
using OIDC/JWKS, and exposes a route-level protect middleware consumed by every other module.

## Overview

- **Stateless JWT** — no session storage; every request carries a signed token.
- **RS256 / ES384** — asymmetric key algorithms only.
- **JWKS rotation** — public keys are fetched from the IdP, cached, and refreshed automatically.
- **Permission model** — `resource:action` scopes embedded in the token `scp` claim.

---

## Identity Provider: Logto

Logto is the OIDC provider. Its discovery document is fetched at startup to locate the JWKS
endpoint, issuer, and supported signing algorithms.

Current deployment: `https://auth.flockstore.co/oidc`

---

## Authentication Flow

```
Client                 Logto                  Mannaiah
  │                      │                       │
  │── POST /token ───────►│                       │
  │◄─ access_token ───────│                       │
  │                       │                       │
  │── GET /api/... ────────────────────────────►  │
  │        Authorization: Bearer <token>           │
  │                       │                       │
  │                       │  verify signature   ◄─┤
  │                       │  JWKS lookup        ◄─┤
  │                       │  check scopes       ◄─┤
  │                       │                       │
  │◄─ 200 / 401 / 403 ────────────────────────────│
```

The token must carry `scope=contact:view` (or any other required scope). Mannaiah validates:
1. Signature against the current JWKS
2. `iss` matches the configured issuer
3. `exp` is not in the past
4. `scp` contains all required permissions for the endpoint

---

## Authorization Model

Mannaiah uses a `resource:action` permission format.

### Wildcard Rule

A scope that ends in `:manage` implicitly covers all narrower actions for the same resource:

```
contact:manage  ⇒  covers  contact:view, contact:sync
```

### Permission Hierarchy

Certain higher-privilege actions also cover lower-privilege ones:

| Scope | Also Covers |
|-------|-------------|
| `product:edit` | `product:view` |
| `shipping:generate` | `shipping:quotations` |

### Cross-Domain Dependencies

Some scopes in one domain require the caller to also hold scopes in another domain:

| Scope | Requires |
|-------|----------|
| `order:view` | `contact:view`, `product:view` |
| `order:triage` | `contact:view`, `product:view` |
| `order:manage` | `contact:view`, `product:view` |
| `order:sync` | `contact:view`, `product:view` |

The `GET /users/malformation` endpoint (see below) surfaces tokens that are missing these
dependent scopes.

### hasPermission Logic

```go
func hasPermission(granted []string, required string) bool {
    for _, g := range granted {
        if g == required {
            return true
        }
        // wildcard: resource:manage covers all resource:* actions
        parts := strings.SplitN(required, ":", 2)
        if len(parts) == 2 && g == parts[0]+":manage" {
            return true
        }
        // explicit hierarchy (e.g. product:edit covers product:view)
        if covers, ok := PermissionCovers[g]; ok {
            for _, c := range covers {
                if c == required {
                    return true
                }
            }
        }
    }
    return false
}
```

---

## Scope Reference

### Contacts

| Scope | Description |
|-------|-------------|
| `contact:view` | Read contact profiles |
| `contact:sync` | Trigger contact sync from external sources |
| `contact:manage` | Full contact management (create, update, delete) |

### Orders

| Scope | Description | Requires |
|-------|-------------|----------|
| `order:view` | Read order details | `contact:view`, `product:view` |
| `order:triage` | Triage and flag orders | `contact:view`, `product:view` |
| `order:manage` | Full order management | `contact:view`, `product:view` |
| `order:sync` | Trigger order sync | `contact:view`, `product:view` |

### Products

| Scope | Description |
|-------|-------------|
| `product:view` | Read product catalog |
| `product:edit` | Edit product details (also covers `product:view`) |
| `product:manage` | Full product management |
| `product:tags` | Manage product tags and categories |

### Assets

| Scope | Description |
|-------|-------------|
| `assets:view` | Read media assets |
| `assets:manage` | Upload and manage media assets |

### Marketing

| Scope | Description |
|-------|-------------|
| `marketing:manage` | Full access to campaigns, segments, email, membership, syncrecord |

### Shipping

| Scope | Description |
|-------|-------------|
| `shipping:quotations` | Retrieve shipping quotes |
| `shipping:generate` | Generate labels (also covers `shipping:quotations`) |
| `shipping:manage` | Full shipping management |

### Recommended Scope Sets by Role

| Role | Scopes |
|------|--------|
| Read-only | `contact:view product:view order:view assets:view` |
| Operator | `contact:view product:view product:edit order:view order:triage` |
| Fulfillment | `contact:view product:view order:manage shipping:generate` |
| Admin | `contact:manage order:manage product:manage assets:manage marketing:manage shipping:manage` |

---

## JWKS Handling & Resilience

- JWKS are fetched from Logto's discovery document on startup.
- Keys are cached in memory with a configurable TTL.
- A background goroutine refreshes the key set before expiry.
- If a token arrives with an unknown `kid`, a one-shot refresh is triggered, rate-limited to
  prevent JWKS endpoint abuse.
- If the JWKS endpoint is unreachable, the last valid key set is used until TTL expires.
- HTTP timeout for JWKS fetches is configurable (`AUTH_JWKS_HTTP_TIMEOUT_MS`).

---

## HTTP Endpoints

### GET /check-auth

Verifies the bearer token and returns the resolved identity.

**Responses**

| Status | Description |
|--------|-------------|
| `200` | Token valid; body contains `{"sub":"...","scopes":["..."]}` |
| `401` | Missing or invalid token |
| `500` | JWKS fetch failure |

---

### GET /users/malformation

Detects tokens that hold order scopes but are missing the required cross-domain dependency scopes
(`contact:view` and/or `product:view`).

This endpoint is useful for auditing API clients that were granted order permissions before the
cross-domain dependency rule was introduced.

**Example response (malformed token detected)**

```json
{
  "status": "malformed",
  "issues": [
    {
      "scope": "order:manage",
      "missing": ["contact:view", "product:view"]
    }
  ]
}
```

**Example response (token is valid)**

```json
{
  "status": "ok"
}
```

---

## Using Auth in Other Modules

Import the auth port and call `protect()` in your route registration. The variadic signature
accepts one or more required permissions; **all** must be present for the request to proceed.

```go
func (h *Handler) RegisterRoutes(r core.Router) {
    // single permission
    r.GET("/orders", protect(h.listOrders, "order:view"))

    // multiple permissions required simultaneously
    r.POST("/orders/:id/ship", protect(h.shipOrder, "order:manage", "shipping:generate"))
}
```

---

## Development Bypass

Set `DEV_AUTH_ENABLED=false` together with `DEV_AUTH_SCOPE` to bypass JWT validation in local
environments:

```dotenv
DEV_AUTH_ENABLED=false
DEV_AUTH_SCOPE=contact:manage order:manage product:manage assets:manage marketing:manage shipping:manage
```

All requests will be treated as if the token contained the scopes listed in `DEV_AUTH_SCOPE`.

---

## Module Initialization Sequence

1. Load auth config from environment.
2. Fetch OIDC discovery document from `AUTH_ISSUER_URL`.
3. Extract JWKS URI from discovery document.
4. Fetch initial JWKS and populate key cache.
5. Start background JWKS refresh goroutine.
6. Register `protect()` middleware factory in the module registry.
7. Register `GET /check-auth` route.
8. Register `GET /users/malformation` route.
9. Provide `auth.Port` to dependent modules via core DI.
10. Module ready.

---

## Error Reference

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `ERR_MISSING_TOKEN` | `401` | Authorization header absent |
| `ERR_INVALID_TOKEN` | `401` | Token signature invalid or malformed |
| `ERR_TOKEN_EXPIRED` | `401` | Token `exp` claim is in the past |
| `ERR_ISSUER_MISMATCH` | `401` | Token `iss` does not match configured issuer |
| `ERR_INSUFFICIENT_SCOPE` | `403` | Token lacks a required permission |
| `ERR_JWKS_UNAVAILABLE` | `500` | JWKS endpoint unreachable and cache expired |

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `AUTH_ISSUER_URL` | ✓ | — | OIDC issuer URL (e.g. `https://auth.flockstore.co/oidc`) |
| `AUTH_AUDIENCE` | ✓ | — | Expected `aud` claim value |
| `AUTH_JWKS_CACHE_TTL_MS` | | `300000` | JWKS in-memory cache TTL (5 min) |
| `AUTH_JWKS_REFRESH_INTERVAL_MS` | | `240000` | Background refresh interval (4 min) |
| `AUTH_JWKS_HTTP_TIMEOUT_MS` | | `10000` | JWKS fetch HTTP timeout |
| `AUTH_JWKS_RATE_LIMIT_RPS` | | `1` | Unknown-kid refresh rate limit |
| `DEV_AUTH_ENABLED` | | `true` | When `false`, bypasses JWT validation |
| `DEV_AUTH_SCOPE` | | _(empty)_ | Space-separated scopes for bypass mode |

---

## Logto Setup Guide

When creating the Mannaiah API resource in Logto, define the following scopes. Add them to each
application's allowed scopes and include them in your token request.

```
contact:view
contact:sync
contact:manage
order:view
order:triage
order:manage
order:sync
product:view
product:edit
product:manage
product:tags
assets:view
assets:manage
marketing:manage
shipping:quotations
shipping:generate
shipping:manage
```

After defining scopes, assign them to roles in Logto and map roles to users or machine-to-machine
applications as required.

### Token Request Example

```http
POST /oidc/token
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials
&client_id=<client_id>
&client_secret=<client_secret>
&resource=https://api.flockstore.co
&scope=contact:view product:view order:view
```

The returned `access_token` will contain the granted scopes in the `scp` claim.
