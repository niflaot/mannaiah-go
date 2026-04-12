package runtime

// Config defines product runtime feature configuration.
type Config struct {
	// StorefrontNavigationEnabled controls storefront navigation caching and scheduled regeneration.
	StorefrontNavigationEnabled bool `mapstructure:"PRODUCTS_STOREFRONT_NAVIGATION_ENABLED" default:"true"`
	// StorefrontNavigationRealm defines the datasheet realm used for navigation mapping.
	StorefrontNavigationRealm string `mapstructure:"PRODUCTS_STOREFRONT_NAVIGATION_REALM" default:"default"`
	// StorefrontNavigationRefreshHours defines the storefront navigation regeneration interval in hours.
	StorefrontNavigationRefreshHours int `mapstructure:"PRODUCTS_STOREFRONT_NAVIGATION_REFRESH_HOURS" default:"12"`
	// StorefrontNavigationCacheMultiplier defines the cache TTL multiplier relative to the refresh interval.
	StorefrontNavigationCacheMultiplier int `mapstructure:"PRODUCTS_STOREFRONT_NAVIGATION_CACHE_MULTIPLIER" default:"2"`
	// StorefrontNavigationFailureExtensionHours defines stale-cache extension duration after regeneration failures.
	StorefrontNavigationFailureExtensionHours int `mapstructure:"PRODUCTS_STOREFRONT_NAVIGATION_FAILURE_EXTENSION_HOURS" default:"12"`
	// StorefrontNavigationCacheKey defines the cache key used for storefront navigation snapshots.
	StorefrontNavigationCacheKey string `mapstructure:"PRODUCTS_STOREFRONT_NAVIGATION_CACHE_KEY" default:"products:storefront:navigation:default"`
	// StorefrontNavigationRegenerationTimeoutSeconds defines regeneration timeout values for cron and mutation-triggered refreshes.
	StorefrontNavigationRegenerationTimeoutSeconds int `mapstructure:"PRODUCTS_STOREFRONT_NAVIGATION_REGENERATION_TIMEOUT_SECONDS" default:"30"`
}
