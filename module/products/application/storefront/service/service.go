package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	corecache "mannaiah/module/core/cache"
	categorydomain "mannaiah/module/products/domain/category"
	productdomain "mannaiah/module/products/domain/product"
	storefrontdomain "mannaiah/module/products/domain/storefront"

	"go.uber.org/zap"
	"golang.org/x/text/unicode/norm"
)

var (
	// ErrNilSource is returned when navigation data sources are missing.
	ErrNilSource = errors.New("storefront navigation source must not be nil")
)

// Source defines category/product navigation source behavior.
type Source interface {
	// Tree returns all root categories ordered from oldest to newest.
	Tree(ctx context.Context) ([]*categorydomain.Category, error)
	// Children returns all direct children of the provided category ordered from oldest to newest.
	Children(ctx context.Context, parentID string) ([]*categorydomain.Category, error)
	// ListProducts returns all products visible under the provided category ordered from oldest to newest.
	ListProducts(ctx context.Context, categoryID string) ([]*productdomain.Product, error)
}

// Config defines storefront navigation caching and mapping behavior.
type Config struct {
	// Enabled reports whether scheduled refresh and cache persistence are active.
	Enabled bool
	// Realm defines the datasheet realm used for storefront mapping.
	Realm string
	// CacheKey defines the Redis key used for the cached navigation snapshot.
	CacheKey string
	// RefreshInterval defines the intended navigation regeneration cadence.
	RefreshInterval time.Duration
	// CacheTTL defines how long cached navigation snapshots stay in Redis.
	CacheTTL time.Duration
	// FailureExtensionTTL defines how long stale snapshots stay alive after regeneration failures.
	FailureExtensionTTL time.Duration
	// CollectionBasePath defines the base storefront collection path.
	CollectionBasePath string
	// ProductBasePath defines the base storefront product path.
	ProductBasePath string
	// RegenerationTimeout defines the timeout used for background-triggered refreshes.
	RegenerationTimeout time.Duration
}

// DefaultConfig returns the default storefront navigation configuration.
func DefaultConfig() Config {
	refreshInterval := 12 * time.Hour

	return Config{
		Enabled:             true,
		Realm:               "default",
		CacheKey:            "products:storefront:navigation:default",
		RefreshInterval:     refreshInterval,
		CacheTTL:            24 * time.Hour,
		FailureExtensionTTL: refreshInterval,
		CollectionBasePath:  "/collections",
		ProductBasePath:     "/product",
		RegenerationTimeout: 30 * time.Second,
	}
}

// Service defines storefront navigation access behavior.
type Service interface {
	// Get resolves storefront navigation from cache or regenerates it on cache miss.
	Get(ctx context.Context) (*storefrontdomain.Navigation, error)
	// Regenerate rebuilds storefront navigation and persists the latest cache snapshot.
	Regenerate(ctx context.Context) (*storefrontdomain.Navigation, error)
	// TriggerRefresh requests a fail-open background regeneration after product/category mutations.
	TriggerRefresh(ctx context.Context)
}

// NavigationService implements storefront navigation access behavior.
type NavigationService struct {
	// source defines category and product navigation data dependencies.
	source Source
	// store defines optional cache persistence dependencies.
	store corecache.Store
	// config defines storefront navigation runtime configuration.
	config Config
	// logger defines structured logging dependencies.
	logger *zap.Logger
	// mutex guards concurrent regeneration attempts.
	mutex sync.Mutex
}

var (
	// _ ensures NavigationService satisfies Service contracts.
	_ Service = (*NavigationService)(nil)
)

// NewService creates storefront navigation services.
func NewService(source Source, store corecache.Store, cfg Config, providedLogger *zap.Logger) (*NavigationService, error) {
	if source == nil {
		return nil, ErrNilSource
	}

	resolvedConfig := normalizeConfig(cfg)

	return &NavigationService{
		source: source,
		store:  store,
		config: resolvedConfig,
		logger: resolveLogger(providedLogger),
	}, nil
}

// Get resolves storefront navigation from cache when available.
func (s *NavigationService) Get(ctx context.Context) (*storefrontdomain.Navigation, error) {
	if s == nil {
		return nil, ErrNilSource
	}

	if s.store != nil {
		cached, err := s.store.Get(ctx, s.config.CacheKey)
		if err == nil && strings.TrimSpace(cached) != "" {
			var snapshot storefrontdomain.Navigation
			if jsonErr := json.Unmarshal([]byte(cached), &snapshot); jsonErr == nil {
				return &snapshot, nil
			}
		}
	}

	return s.Regenerate(ctx)
}

// Regenerate rebuilds storefront navigation and updates the cached snapshot.
func (s *NavigationService) Regenerate(ctx context.Context) (*storefrontdomain.Navigation, error) {
	if s == nil {
		return nil, ErrNilSource
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	snapshot, err := s.build(ctx)
	if err != nil {
		s.extendCachedSnapshot(ctx)
		return nil, err
	}

	if persistErr := s.persist(ctx, snapshot, s.config.CacheTTL); persistErr != nil {
		return nil, persistErr
	}

	return snapshot, nil
}

// TriggerRefresh requests a fail-open background regeneration after mutations.
func (s *NavigationService) TriggerRefresh(ctx context.Context) {
	if s == nil || !s.config.Enabled {
		return
	}

	regenerationCtx := context.Background()
	if timeout := s.config.RegenerationTimeout; timeout > 0 {
		var cancel context.CancelFunc
		regenerationCtx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	}

	if ctx != nil {
		select {
		case <-ctx.Done():
		default:
		}
	}

	if _, err := s.Regenerate(regenerationCtx); err != nil {
		s.logger.Warn("storefront navigation refresh failed", zap.Error(err))
	}
}

// build resolves the latest storefront navigation tree from categories and products.
func (s *NavigationService) build(ctx context.Context) (*storefrontdomain.Navigation, error) {
	roots, err := s.source.Tree(ctx)
	if err != nil {
		return nil, fmt.Errorf("load storefront root categories: %w", err)
	}

	categories := make([]storefrontdomain.CategoryNode, 0, len(roots))
	for _, root := range roots {
		node, mapErr := s.mapCategory(ctx, root, nil)
		if mapErr != nil {
			return nil, mapErr
		}
		categories = append(categories, node)
	}

	return &storefrontdomain.Navigation{
		Realm:       s.config.Realm,
		GeneratedAt: time.Now().UTC(),
		Categories:  categories,
	}, nil
}

// mapCategory maps one category aggregate into a storefront navigation node.
func (s *NavigationService) mapCategory(
	ctx context.Context,
	category *categorydomain.Category,
	ancestorSlugs []string,
) (storefrontdomain.CategoryNode, error) {
	if category == nil {
		return storefrontdomain.CategoryNode{}, nil
	}

	slug := slugify(strings.TrimSpace(category.Name))
	pathSegments := append(append([]string{}, ancestorSlugs...), slug)
	children, err := s.source.Children(ctx, category.ID)
	if err != nil {
		return storefrontdomain.CategoryNode{}, fmt.Errorf("load storefront category children: %w", err)
	}

	childNodes := make([]storefrontdomain.CategoryNode, 0, len(children))
	for _, child := range children {
		node, mapErr := s.mapCategory(ctx, child, pathSegments)
		if mapErr != nil {
			return storefrontdomain.CategoryNode{}, mapErr
		}
		childNodes = append(childNodes, node)
	}

	products, err := s.source.ListProducts(ctx, category.ID)
	if err != nil {
		return storefrontdomain.CategoryNode{}, fmt.Errorf("load storefront category products: %w", err)
	}

	productNodes := make([]storefrontdomain.ProductNode, 0, len(products))
	for _, product := range products {
		node, ok := s.mapProduct(product)
		if !ok {
			continue
		}
		productNodes = append(productNodes, node)
	}

	sort.SliceStable(productNodes, func(left, right int) bool {
		if productNodes[left].CreatedAt.Equal(productNodes[right].CreatedAt) {
			return productNodes[left].ID < productNodes[right].ID
		}

		return productNodes[left].CreatedAt.Before(productNodes[right].CreatedAt)
	})

	return storefrontdomain.CategoryNode{
		ID:        strings.TrimSpace(category.ID),
		Name:      strings.TrimSpace(category.Name),
		Slug:      slug,
		Path:      joinPath(s.config.CollectionBasePath, pathSegments...),
		CreatedAt: category.CreatedAt,
		UpdatedAt: category.UpdatedAt,
		Products:  productNodes,
		Children:  childNodes,
	}, nil
}

// mapProduct maps one product aggregate into a storefront navigation node.
func (s *NavigationService) mapProduct(product *productdomain.Product) (storefrontdomain.ProductNode, bool) {
	if product == nil {
		return storefrontdomain.ProductNode{}, false
	}

	datasheet, ok := defaultRealmDatasheet(product.Datasheets, s.config.Realm)
	if !ok {
		return storefrontdomain.ProductNode{}, false
	}

	name := strings.TrimSpace(datasheet.Name)
	if name == "" {
		return storefrontdomain.ProductNode{}, false
	}

	slugValue, fullPath := resolveProductPath(datasheet.Attributes, name, s.config.ProductBasePath)

	return storefrontdomain.ProductNode{
		ID:        strings.TrimSpace(product.ID),
		SKU:       strings.TrimSpace(product.SKU),
		Name:      name,
		Slug:      slugValue,
		Path:      fullPath,
		CreatedAt: product.CreatedAt,
		UpdatedAt: product.UpdatedAt,
	}, true
}

// persist encodes and stores the navigation snapshot when cache is available.
func (s *NavigationService) persist(ctx context.Context, snapshot *storefrontdomain.Navigation, ttl time.Duration) error {
	if s.store == nil || snapshot == nil {
		return nil
	}

	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal storefront navigation snapshot: %w", err)
	}
	if err := s.store.Set(ctx, s.config.CacheKey, string(encoded), ttl); err != nil {
		return fmt.Errorf("persist storefront navigation snapshot: %w", err)
	}

	return nil
}

// extendCachedSnapshot re-writes the current cached value to extend stale availability on failures.
func (s *NavigationService) extendCachedSnapshot(ctx context.Context) {
	if s.store == nil {
		return
	}

	cached, err := s.store.Get(ctx, s.config.CacheKey)
	if err != nil || strings.TrimSpace(cached) == "" {
		return
	}
	if setErr := s.store.Set(ctx, s.config.CacheKey, cached, s.config.FailureExtensionTTL); setErr != nil {
		s.logger.Warn("extend stale storefront navigation snapshot failed", zap.Error(setErr))
	}
}

// normalizeConfig resolves default storefront navigation configuration values.
func normalizeConfig(cfg Config) Config {
	defaults := DefaultConfig()

	if strings.TrimSpace(cfg.Realm) == "" {
		cfg.Realm = defaults.Realm
	}
	if strings.TrimSpace(cfg.CacheKey) == "" {
		cfg.CacheKey = defaults.CacheKey
	}
	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = defaults.RefreshInterval
	}
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = defaults.CacheTTL
	}
	if cfg.FailureExtensionTTL <= 0 {
		cfg.FailureExtensionTTL = defaults.FailureExtensionTTL
	}
	if strings.TrimSpace(cfg.CollectionBasePath) == "" {
		cfg.CollectionBasePath = defaults.CollectionBasePath
	}
	if strings.TrimSpace(cfg.ProductBasePath) == "" {
		cfg.ProductBasePath = defaults.ProductBasePath
	}
	if cfg.RegenerationTimeout <= 0 {
		cfg.RegenerationTimeout = defaults.RegenerationTimeout
	}

	return cfg
}

// resolveLogger resolves nil loggers to no-op defaults.
func resolveLogger(providedLogger *zap.Logger) *zap.Logger {
	if providedLogger != nil {
		return providedLogger
	}

	return zap.NewNop()
}

// defaultRealmDatasheet resolves the datasheet matching the configured realm.
func defaultRealmDatasheet(datasheets []productdomain.Datasheet, realm string) (productdomain.Datasheet, bool) {
	trimmedRealm := strings.TrimSpace(realm)
	for _, datasheet := range datasheets {
		if strings.EqualFold(strings.TrimSpace(datasheet.Realm), trimmedRealm) {
			return datasheet, true
		}
	}

	return productdomain.Datasheet{}, false
}

// resolveProductPath resolves the mapped product slug and storefront path.
func resolveProductPath(attributes map[string]any, fallbackName string, basePath string) (string, string) {
	rawURL := strings.TrimSpace(stringAttribute(attributes, "storefronturl"))
	if rawURL == "" {
		slugValue := slugify(fallbackName)
		return slugValue, joinPath(basePath, slugValue)
	}

	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		lastSegment := slugify(path.Base(rawURL))
		if lastSegment == "" {
			lastSegment = slugify(fallbackName)
		}

		return lastSegment, rawURL
	}

	if strings.HasPrefix(rawURL, "/") {
		lastSegment := slugify(path.Base(rawURL))
		if lastSegment == "" {
			lastSegment = slugify(fallbackName)
		}

		return lastSegment, rawURL
	}

	slugValue := slugify(rawURL)
	if slugValue == "" {
		slugValue = slugify(fallbackName)
	}

	return slugValue, joinPath(basePath, slugValue)
}

// stringAttribute resolves one string attribute value from datasheet attributes.
func stringAttribute(attributes map[string]any, key string) string {
	if len(attributes) == 0 {
		return ""
	}

	value, ok := attributes[key]
	if !ok {
		return ""
	}

	switch resolved := value.(type) {
	case string:
		return strings.TrimSpace(resolved)
	case fmt.Stringer:
		return strings.TrimSpace(resolved.String())
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", resolved))
	}
}

// joinPath resolves one storefront path from a base path and one or more segments.
func joinPath(basePath string, segments ...string) string {
	cleanSegments := make([]string, 0, len(segments)+1)
	cleanBase := strings.TrimSpace(basePath)
	if cleanBase != "" {
		cleanSegments = append(cleanSegments, strings.Trim(cleanBase, "/"))
	}
	for _, segment := range segments {
		trimmed := strings.Trim(strings.TrimSpace(segment), "/")
		if trimmed == "" {
			continue
		}
		cleanSegments = append(cleanSegments, trimmed)
	}
	if len(cleanSegments) == 0 {
		return "/"
	}

	return "/" + path.Join(cleanSegments...)
}

// slugify normalizes storefront route segments into lowercase dashed values.
func slugify(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}

	builder := strings.Builder{}
	lastWasDash := false
	for _, current := range norm.NFD.String(trimmed) {
		if unicode.Is(unicode.Mn, current) {
			continue
		}
		if (current >= 'a' && current <= 'z') || (current >= '0' && current <= '9') {
			builder.WriteRune(current)
			lastWasDash = false
			continue
		}
		if lastWasDash {
			continue
		}
		builder.WriteRune('-')
		lastWasDash = true
	}

	return strings.Trim(builder.String(), "-")
}
