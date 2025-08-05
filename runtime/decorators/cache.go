package decorators

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// CacheEntry represents a cached result with expiration
type CacheEntry struct {
	Value     interface{}
	CreatedAt time.Time
	ExpiresAt time.Time
	HitCount  int64
}

// IsExpired checks if the cache entry has expired
func (ce *CacheEntry) IsExpired() bool {
	return time.Now().After(ce.ExpiresAt)
}

// Cache provides thread-safe caching with TTL support
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	ttl     time.Duration
	maxSize int
}

// NewCache creates a new cache with specified TTL and max size
func NewCache(ttl time.Duration, maxSize int) *Cache {
	return &Cache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// Get retrieves a value from cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists || entry.IsExpired() {
		return nil, false
	}

	entry.HitCount++
	return entry.Value, true
}

// Set stores a value in cache
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clean expired entries if needed
	c.cleanExpiredLocked()

	// Evict oldest entries if cache is full
	if len(c.entries) >= c.maxSize {
		c.evictOldestLocked()
	}

	now := time.Now()
	c.entries[key] = &CacheEntry{
		Value:     value,
		CreatedAt: now,
		ExpiresAt: now.Add(c.ttl),
		HitCount:  0,
	}
}

// Clear removes all entries from cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*CacheEntry)
}

// Size returns the current number of entries in cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// cleanExpiredLocked removes expired entries (must hold write lock)
func (c *Cache) cleanExpiredLocked() {
	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}

// evictOldestLocked removes the oldest entry (must hold write lock)
func (c *Cache) evictOldestLocked() {
	if len(c.entries) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, entry := range c.entries {
		if first || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// Global caches for different types of operations
var (
	templateCache   = NewCache(30*time.Minute, 1000) // Templates live 30 minutes, max 1000 entries
	operationCache  = NewCache(15*time.Minute, 500)  // Operations live 15 minutes, max 500 entries
	validationCache = NewCache(5*time.Minute, 200)   // Validation results live 5 minutes, max 200 entries
	astCache        = NewCache(10*time.Minute, 300)  // AST conversions live 10 minutes, max 300 entries
)

// CacheKeyGenerator generates consistent cache keys for various inputs
type CacheKeyGenerator struct{}

// GenerateTemplateKey creates a cache key for template generation
func (ckg *CacheKeyGenerator) GenerateTemplateKey(decoratorName string, params []ast.NamedParameter, contentHash string) string {
	data := struct {
		Decorator string               `json:"decorator"`
		Params    []ast.NamedParameter `json:"params"`
		Content   string               `json:"content"`
	}{
		Decorator: decoratorName,
		Params:    params,
		Content:   contentHash,
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return "template:" + hex.EncodeToString(hash[:])
}

// GenerateOperationKey creates a cache key for operation conversion
func (ckg *CacheKeyGenerator) GenerateOperationKey(content []ast.CommandContent, contextHash string) string {
	data := struct {
		Content []ast.CommandContent `json:"content"`
		Context string               `json:"context"`
	}{
		Content: content,
		Context: contextHash,
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return "operation:" + hex.EncodeToString(hash[:])
}

// GenerateValidationKey creates a cache key for validation results
func (ckg *CacheKeyGenerator) GenerateValidationKey(decoratorName string, params []ast.NamedParameter) string {
	data := struct {
		Decorator string               `json:"decorator"`
		Params    []ast.NamedParameter `json:"params"`
	}{
		Decorator: decoratorName,
		Params:    params,
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return "validation:" + hex.EncodeToString(hash[:])
}

// GenerateASTKey creates a cache key for AST processing
func (ckg *CacheKeyGenerator) GenerateASTKey(content ast.CommandContent) string {
	jsonData, _ := json.Marshal(content)
	hash := sha256.Sum256(jsonData)
	return "ast:" + hex.EncodeToString(hash[:])
}

var keyGenerator = &CacheKeyGenerator{}

// CachedTemplateBuilder wraps TemplateBuilder with caching capabilities
type CachedTemplateBuilder struct {
	*TemplateBuilder
	decoratorName string
	params        []ast.NamedParameter
	enableCache   bool
}

// NewCachedTemplateBuilder creates a new cached template builder
func NewCachedTemplateBuilder(decoratorName string, params []ast.NamedParameter) *CachedTemplateBuilder {
	return &CachedTemplateBuilder{
		TemplateBuilder: NewTemplateBuilder(),
		decoratorName:   decoratorName,
		params:          params,
		enableCache:     true,
	}
}

// BuildTemplate builds a template with caching
func (ctb *CachedTemplateBuilder) BuildTemplate() (string, error) {
	if !ctb.enableCache {
		return ctb.TemplateBuilder.BuildTemplate()
	}

	// Generate cache key based on current builder state
	builderHash := ctb.generateBuilderHash()
	cacheKey := keyGenerator.GenerateTemplateKey(ctb.decoratorName, ctb.params, builderHash)

	// Check cache first
	if cached, found := templateCache.Get(cacheKey); found {
		if result, ok := cached.(string); ok {
			return result, nil
		}
	}

	// Build template if not cached
	result, err := ctb.TemplateBuilder.BuildTemplate()
	if err != nil {
		return "", err
	}

	// Cache the result
	templateCache.Set(cacheKey, result)

	return result, nil
}

// BuildCommandResultTemplate builds a CommandResult template with caching
func (ctb *CachedTemplateBuilder) BuildCommandResultTemplate() (string, error) {
	if !ctb.enableCache {
		return ctb.TemplateBuilder.BuildCommandResultTemplate()
	}

	// Generate cache key with "CommandResult" suffix to differentiate
	builderHash := ctb.generateBuilderHash()
	cacheKey := keyGenerator.GenerateTemplateKey(ctb.decoratorName+"_CR", ctb.params, builderHash)

	// Check cache first
	if cached, found := templateCache.Get(cacheKey); found {
		if result, ok := cached.(string); ok {
			return result, nil
		}
	}

	// Build template if not cached
	result, err := ctb.TemplateBuilder.BuildCommandResultTemplate()
	if err != nil {
		return "", err
	}

	// Cache the result
	templateCache.Set(cacheKey, result)

	return result, nil
}

// generateBuilderHash creates a hash representing the current builder state
func (ctb *CachedTemplateBuilder) generateBuilderHash() string {
	// This is a simplified hash - in practice, you'd want to hash the actual builder state
	data := struct {
		HasConcurrent bool  `json:"concurrent"`
		HasTimeout    bool  `json:"timeout"`
		HasRetry      bool  `json:"retry"`
		HasResource   bool  `json:"resource"`
		Timestamp     int64 `json:"timestamp"`
	}{
		HasConcurrent: false, // You'd determine this from builder state
		HasTimeout:    false,
		HasRetry:      false,
		HasResource:   false,
		Timestamp:     time.Now().Unix() / 3600, // Hour-based cache invalidation
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// CachedOperationConverter provides cached operation conversion
type CachedOperationConverter struct {
	enableCache bool
}

// NewCachedOperationConverter creates a new cached operation converter
func NewCachedOperationConverter() *CachedOperationConverter {
	return &CachedOperationConverter{
		enableCache: true,
	}
}

// ConvertCommandsToOperations converts commands to operations with caching
func (coc *CachedOperationConverter) ConvertCommandsToOperations(ctx execution.GeneratorContext, content []ast.CommandContent) ([]Operation, error) {
	if !coc.enableCache {
		return ConvertCommandsToOperations(ctx, content)
	}

	// Generate cache key
	contextHash := coc.generateContextHash(ctx)
	cacheKey := keyGenerator.GenerateOperationKey(content, contextHash)

	// Check cache first
	if cached, found := operationCache.Get(cacheKey); found {
		if result, ok := cached.([]Operation); ok {
			return result, nil
		}
	}

	// Convert if not cached
	result, err := ConvertCommandsToOperations(ctx, content)
	if err != nil {
		return nil, err
	}

	// Cache the result
	operationCache.Set(cacheKey, result)

	return result, nil
}

// generateContextHash creates a hash representing the generator context
func (coc *CachedOperationConverter) generateContextHash(ctx execution.GeneratorContext) string {
	// Simplified context hash - in practice, you'd hash relevant context properties
	data := struct {
		WorkingDir string `json:"workdir"`
		Timestamp  int64  `json:"timestamp"`
	}{
		WorkingDir: "/tmp",                   // You'd get this from context
		Timestamp:  time.Now().Unix() / 3600, // Hour-based cache invalidation
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// CacheManager provides utilities for managing all caches
type CacheManager struct{}

// ClearAllCaches clears all global caches
func (cm *CacheManager) ClearAllCaches() {
	templateCache.Clear()
	operationCache.Clear()
	validationCache.Clear()
	astCache.Clear()
}

// GetCacheStats returns statistics for all caches
func (cm *CacheManager) GetCacheStats() map[string]int {
	return map[string]int{
		"template_cache_size":   templateCache.Size(),
		"operation_cache_size":  operationCache.Size(),
		"validation_cache_size": validationCache.Size(),
		"ast_cache_size":        astCache.Size(),
	}
}

var cacheManager = &CacheManager{}

// GetCacheManager returns the global cache manager
func GetCacheManager() *CacheManager {
	return cacheManager
}
