package decorators

import (
	"context"
	"runtime"
	"sync"
	"time"
)

// PerformanceMetrics tracks performance data for decorator execution
type PerformanceMetrics struct {
	mu                sync.RWMutex
	DecoratorName     string
	ExecutionMode     string
	StartTime         time.Time
	EndTime           time.Time
	Duration          time.Duration
	MemoryUsageBefore int64
	MemoryUsageAfter  int64
	MemoryAllocated   int64
	GoroutinesBefore  int
	GoroutinesAfter   int
	Success           bool
	ErrorMessage      string
}

// PerformanceMonitor tracks performance across all decorator executions
type PerformanceMonitor struct {
	mu      sync.RWMutex
	metrics []PerformanceMetrics
	enabled bool
}

var globalPerformanceMonitor = &PerformanceMonitor{
	metrics: make([]PerformanceMetrics, 0),
	enabled: false, // Disabled by default for production performance
}

// EnablePerformanceMonitoring enables performance tracking globally
func EnablePerformanceMonitoring() {
	globalPerformanceMonitor.mu.Lock()
	defer globalPerformanceMonitor.mu.Unlock()
	globalPerformanceMonitor.enabled = true
}

// DisablePerformanceMonitoring disables performance tracking
func DisablePerformanceMonitoring() {
	globalPerformanceMonitor.mu.Lock()
	defer globalPerformanceMonitor.mu.Unlock()
	globalPerformanceMonitor.enabled = false
}

// IsPerformanceMonitoringEnabled returns whether monitoring is active
func IsPerformanceMonitoringEnabled() bool {
	globalPerformanceMonitor.mu.RLock()
	defer globalPerformanceMonitor.mu.RUnlock()
	return globalPerformanceMonitor.enabled
}

// StartPerformanceTracking begins tracking performance for a decorator execution
func StartPerformanceTracking(decoratorName, executionMode string) *PerformanceMetrics {
	if !IsPerformanceMonitoringEnabled() {
		return nil
	}
	
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	return &PerformanceMetrics{
		DecoratorName:     decoratorName,
		ExecutionMode:     executionMode,
		StartTime:         time.Now(),
		MemoryUsageBefore: int64(memStats.Alloc),
		GoroutinesBefore:  runtime.NumGoroutine(),
	}
}

// FinishPerformanceTracking completes performance tracking and records the metrics
func FinishPerformanceTracking(metrics *PerformanceMetrics, success bool, errorMessage string) {
	if metrics == nil || !IsPerformanceMonitoringEnabled() {
		return
	}
	
	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	metrics.EndTime = time.Now()
	metrics.Duration = metrics.EndTime.Sub(metrics.StartTime)
	metrics.MemoryUsageAfter = int64(memStats.Alloc)
	metrics.MemoryAllocated = metrics.MemoryUsageAfter - metrics.MemoryUsageBefore
	metrics.GoroutinesAfter = runtime.NumGoroutine()
	metrics.Success = success
	metrics.ErrorMessage = errorMessage
	
	// Record in global monitor
	globalPerformanceMonitor.mu.Lock()
	globalPerformanceMonitor.metrics = append(globalPerformanceMonitor.metrics, *metrics)
	globalPerformanceMonitor.mu.Unlock()
}

// GetPerformanceMetrics returns all recorded performance metrics
func GetPerformanceMetrics() []PerformanceMetrics {
	globalPerformanceMonitor.mu.RLock()
	defer globalPerformanceMonitor.mu.RUnlock()
	
	// Return a copy to prevent race conditions
	result := make([]PerformanceMetrics, len(globalPerformanceMonitor.metrics))
	copy(result, globalPerformanceMonitor.metrics)
	return result
}

// ClearPerformanceMetrics clears all recorded metrics
func ClearPerformanceMetrics() {
	globalPerformanceMonitor.mu.Lock()
	defer globalPerformanceMonitor.mu.Unlock()
	globalPerformanceMonitor.metrics = globalPerformanceMonitor.metrics[:0]
}

// ResourceLimiter enforces resource limits during decorator execution
type ResourceLimiter struct {
	MaxMemoryMB      int64
	MaxGoroutines    int
	MaxExecutionTime time.Duration
	ctx              context.Context
	cancel           context.CancelFunc
}

// NewResourceLimiter creates a new resource limiter with specified limits
func NewResourceLimiter(maxMemoryMB int64, maxGoroutines int, maxExecutionTime time.Duration) *ResourceLimiter {
	ctx, cancel := context.WithTimeout(context.Background(), maxExecutionTime)
	
	return &ResourceLimiter{
		MaxMemoryMB:      maxMemoryMB,
		MaxGoroutines:    maxGoroutines,
		MaxExecutionTime: maxExecutionTime,
		ctx:              ctx,
		cancel:           cancel,
	}
}

// CheckResourceLimits verifies that current resource usage is within limits
func (rl *ResourceLimiter) CheckResourceLimits() error {
	// Check context timeout
	select {
	case <-rl.ctx.Done():
		return rl.ctx.Err()
	default:
	}
	
	// Check memory usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	currentMemoryMB := int64(memStats.Alloc) / (1024 * 1024)
	
	if rl.MaxMemoryMB > 0 && currentMemoryMB > rl.MaxMemoryMB {
		return context.DeadlineExceeded // Reuse existing error type
	}
	
	// Check goroutine count
	currentGoroutines := runtime.NumGoroutine()
	if rl.MaxGoroutines > 0 && currentGoroutines > rl.MaxGoroutines {
		return context.DeadlineExceeded
	}
	
	return nil
}

// Context returns the resource limiter's context for cancellation
func (rl *ResourceLimiter) Context() context.Context {
	return rl.ctx
}

// Cleanup cancels the resource limiter context
func (rl *ResourceLimiter) Cleanup() {
	if rl.cancel != nil {
		rl.cancel()
	}
}

// PerformanceOptimizedExecutor provides optimized execution with resource monitoring
type PerformanceOptimizedExecutor struct {
	limiter         *ResourceLimiter
	metrics         *PerformanceMetrics
	decoratorName   string
	executionMode   string
	enableCaching   bool
	operationCache  sync.Map // Cache for repeated operations
}

// NewPerformanceOptimizedExecutor creates a new performance-optimized executor
func NewPerformanceOptimizedExecutor(decoratorName, executionMode string, enableCaching bool) *PerformanceOptimizedExecutor {
	// Set reasonable default limits
	limiter := NewResourceLimiter(
		512,           // 512MB max memory
		1000,          // 1000 max goroutines
		10*time.Minute, // 10 minute max execution time
	)
	
	metrics := StartPerformanceTracking(decoratorName, executionMode)
	
	return &PerformanceOptimizedExecutor{
		limiter:       limiter,
		metrics:       metrics,
		decoratorName: decoratorName,
		executionMode: executionMode,
		enableCaching: enableCaching,
	}
}

// Execute runs a function with performance monitoring and resource limits
func (poe *PerformanceOptimizedExecutor) Execute(fn func() error) error {
	defer poe.Cleanup()
	
	// Check resource limits before execution
	if err := poe.limiter.CheckResourceLimits(); err != nil {
		FinishPerformanceTracking(poe.metrics, false, "Resource limit exceeded before execution")
		return err
	}
	
	// Execute the function
	err := fn()
	
	// Check resource limits after execution
	if resourceErr := poe.limiter.CheckResourceLimits(); resourceErr != nil {
		FinishPerformanceTracking(poe.metrics, false, "Resource limit exceeded during execution")
		return resourceErr
	}
	
	// Record performance metrics
	FinishPerformanceTracking(poe.metrics, err == nil, func() string {
		if err != nil {
			return err.Error()
		}
		return ""
	}())
	
	return err
}

// GetCachedOperation retrieves a cached operation result
func (poe *PerformanceOptimizedExecutor) GetCachedOperation(key string) (interface{}, bool) {
	if !poe.enableCaching {
		return nil, false
	}
	return poe.operationCache.Load(key)
}

// SetCachedOperation stores an operation result in cache
func (poe *PerformanceOptimizedExecutor) SetCachedOperation(key string, value interface{}) {
	if !poe.enableCaching {
		return
	}
	poe.operationCache.Store(key, value)
}

// Cleanup releases resources used by the executor
func (poe *PerformanceOptimizedExecutor) Cleanup() {
	if poe.limiter != nil {
		poe.limiter.Cleanup()
	}
}