package jwt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/MicahParks/keyfunc/v3"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"
	"mannaiah/module/auth/domain"
	"mannaiah/module/auth/port"
)

var (
	// ErrInvalidConfig is returned when verifier configuration is invalid.
	ErrInvalidConfig = errors.New("jwt verifier config is invalid")
	// ErrInvalidToken is returned when JWT token validation fails.
	ErrInvalidToken = errors.New("jwt token is invalid")
)

// Config defines verifier runtime configuration.
type Config struct {
	// Issuer defines expected token issuer claim.
	Issuer string
	// Audience defines expected token audience claim.
	Audience string
	// JWKSURL defines JWKS endpoint used to fetch public keys.
	JWKSURL string
	// RateLimitPerMinute defines maximum unknown-kid refresh calls per minute.
	RateLimitPerMinute int
	// CacheTTL defines refresh interval for background JWKS refresh.
	CacheTTL time.Duration
	// HTTPTimeout defines JWKS HTTP request timeout.
	HTTPTimeout time.Duration
	// Algorithm defines the single allowed JWT signing algorithm. Defaults to RS256 when empty.
	Algorithm string
}

// Verifier defines a JWKS-backed token verifier implementation.
type Verifier struct {
	// issuer defines expected JWT issuer.
	issuer string
	// audience defines expected JWT audience.
	audience string
	// algorithm defines the single allowed JWT signing algorithm.
	algorithm string
	// keyfunc defines JWKS-backed signing-key resolution.
	keyfunc keyfunc.Keyfunc
}

var (
	// _ ensures Verifier satisfies token verifier contracts.
	_ port.TokenVerifier = (*Verifier)(nil)
)

// NewVerifier creates a new JWKS-backed token verifier.
func NewVerifier(cfg Config) (*Verifier, error) {
	issuer := strings.TrimSpace(cfg.Issuer)
	audience := strings.TrimSpace(cfg.Audience)
	jwksURL := strings.TrimSpace(cfg.JWKSURL)
	if issuer == "" || audience == "" || jwksURL == "" {
		return nil, ErrInvalidConfig
	}

	rateLimitPerMinute := cfg.RateLimitPerMinute
	if rateLimitPerMinute <= 0 {
		rateLimitPerMinute = 5
	}

	httpTimeout := cfg.HTTPTimeout
	if httpTimeout <= 0 {
		httpTimeout = 5 * time.Second
	}

	refreshInterval := cfg.CacheTTL
	if refreshInterval <= 0 {
		refreshInterval = 5 * time.Minute
	}

	limiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(rateLimitPerMinute)), 1)

	storage, err := jwkset.NewStorageFromHTTP(jwksURL, jwkset.HTTPClientStorageOptions{
		Ctx:                       context.Background(),
		HTTPTimeout:               httpTimeout,
		NoErrorReturnFirstHTTPReq: true,
		RefreshInterval:           refreshInterval,
	})
	if err != nil {
		return nil, fmt.Errorf("create jwks storage: %w", err)
	}

	httpClient, err := jwkset.NewHTTPClient(jwkset.HTTPClientOptions{
		HTTPURLs: map[string]jwkset.Storage{
			jwksURL: storage,
		},
		RateLimitWaitMax:  httpTimeout,
		RefreshUnknownKID: limiter,
	})
	if err != nil {
		return nil, fmt.Errorf("create jwks client: %w", err)
	}

	resolvedKeyfunc, err := keyfunc.New(keyfunc.Options{Storage: httpClient})
	if err != nil {
		return nil, fmt.Errorf("create keyfunc resolver: %w", err)
	}

	algorithm := strings.TrimSpace(cfg.Algorithm)
	if algorithm == "" {
		algorithm = "RS256"
	}

	return &Verifier{
		issuer:    issuer,
		audience:  audience,
		algorithm: algorithm,
		keyfunc:   resolvedKeyfunc,
	}, nil
}

// Verify validates JWT tokens and returns normalized claims.
func (v *Verifier) Verify(ctx context.Context, token string) (*domain.Claims, error) {
	parsed, err := jwtlib.Parse(token, v.keyfunc.KeyfuncCtx(ctx),
		jwtlib.WithIssuer(v.issuer),
		jwtlib.WithAudience(v.audience),
		jwtlib.WithValidMethods([]string{v.algorithm}),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if parsed == nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(jwtlib.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return mapClaims(claims), nil
}

// mapClaims maps JWT map claims into normalized domain claims.
func mapClaims(claims jwtlib.MapClaims) *domain.Claims {
	copied := map[string]any{}
	for key, value := range claims {
		copied[key] = value
	}

	return &domain.Claims{
		Subject:  readStringClaim(claims, "sub"),
		Issuer:   readStringClaim(claims, "iss"),
		Audience: readAudienceClaim(claims["aud"]),
		Scope:    readStringClaim(claims, "scope"),
		Raw:      copied,
	}
}

// readStringClaim reads a string claim value from map claims.
func readStringClaim(claims jwtlib.MapClaims, key string) string {
	value, _ := claims[key].(string)
	return strings.TrimSpace(value)
}

// readAudienceClaim maps token audience claim values to string slices.
func readAudienceClaim(raw any) []string {
	switch typed := raw.(type) {
	case string:
		value := strings.TrimSpace(typed)
		if value == "" {
			return nil
		}
		return []string{value}
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			stringValue, ok := item.(string)
			if ok {
				value := strings.TrimSpace(stringValue)
				if value != "" {
					result = append(result, value)
				}
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result
	default:
		return nil
	}
}
