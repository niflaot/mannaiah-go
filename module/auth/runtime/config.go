package runtime

// Config defines authentication and authorization runtime configuration.
type Config struct {
	// Issuer defines the Logto/OIDC issuer URL used for JWT issuer validation.
	Issuer string `mapstructure:"LOGTO_ISSUER"`
	// Audience defines the API audience used for JWT audience validation.
	Audience string `mapstructure:"LOGTO_AUDIENCE"`
	// DevAuthToken defines an optional static token used for development bypass.
	DevAuthToken string `mapstructure:"DEV_AUTH_TOKEN" default:""`
	// DevAuthScope defines optional scopes injected into the dev bypass principal.
	DevAuthScope string `mapstructure:"DEV_AUTH_SCOPE" default:""`
	// JWKSRateLimitPerMinute defines maximum JWKS fetch calls allowed per minute.
	JWKSRateLimitPerMinute int `mapstructure:"AUTH_JWKS_RATE_LIMIT_PER_MINUTE" default:"5"`
	// JWKSCacheTTLMS defines JWKS cache TTL in milliseconds.
	JWKSCacheTTLMS int `mapstructure:"AUTH_JWKS_CACHE_TTL_MS" default:"300000"`
	// JWKSHTTPTimeoutMS defines JWKS HTTP client timeout in milliseconds.
	JWKSHTTPTimeoutMS int `mapstructure:"AUTH_JWKS_HTTP_TIMEOUT_MS" default:"5000"`
}
