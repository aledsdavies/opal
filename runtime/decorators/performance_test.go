package decorators

import (
	"runtime"
	"testing"
	"time"
)

func TestPerformanceMonitoring(t *testing.T) {
	// Ensure monitoring is initially disabled
	DisablePerformanceMonitoring()

	// Test enabling/disabling
	if IsPerformanceMonitoringEnabled() {
		t.Error("Performance monitoring should be disabled initially")
	}

	EnablePerformanceMonitoring()
	if !IsPerformanceMonitoringEnabled() {
		t.Error("Performance monitoring should be enabled after EnablePerformanceMonitoring()")
	}

	DisablePerformanceMonitoring()
	if IsPerformanceMonitoringEnabled() {
		t.Error("Performance monitoring should be disabled after DisablePerformanceMonitoring()")
	}
}

func TestPerformanceTracking(t *testing.T) {
	EnablePerformanceMonitoring()
	defer DisablePerformanceMonitoring()

	ClearPerformanceMetrics()

	// Start tracking
	metrics := StartPerformanceTracking("test_decorator", "test_mode")
	if metrics == nil {
		t.Fatal("StartPerformanceTracking should return metrics when monitoring is enabled")
	}

	if metrics.DecoratorName != "test_decorator" {
		t.Errorf("Expected decorator name 'test_decorator', got '%s'", metrics.DecoratorName)
	}

	if metrics.ExecutionMode != "test_mode" {
		t.Errorf("Expected execution mode 'test_mode', got '%s'", metrics.ExecutionMode)
	}

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	// Finish tracking
	FinishPerformanceTracking(metrics, true, "")

	// Check that metrics were recorded
	allMetrics := GetPerformanceMetrics()
	if len(allMetrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(allMetrics))
	}

	metric := allMetrics[0]
	if metric.Success != true {
		t.Error("Expected success to be true")
	}

	if metric.Duration <= 0 {
		t.Error("Expected positive duration")
	}

	if metric.Duration < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", metric.Duration)
	}
}

func TestResourceLimiter(t *testing.T) {
	// Create a resource limiter with short timeout for testing
	limiter := NewResourceLimiter(1024, 100, 100*time.Millisecond)
	defer limiter.Cleanup()

	// Initially should not exceed limits
	if err := limiter.CheckResourceLimits(); err != nil {
		t.Errorf("Initial resource check should pass, got error: %v", err)
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should now exceed time limit
	if err := limiter.CheckResourceLimits(); err == nil {
		t.Error("Expected timeout error after waiting")
	}
}

func TestPerformanceOptimizedExecutor(t *testing.T) {
	EnablePerformanceMonitoring()
	defer DisablePerformanceMonitoring()

	ClearPerformanceMetrics()

	executor := NewPerformanceOptimizedExecutor("test", "test_mode", true)
	defer executor.Cleanup()

	// Test successful execution
	executed := false
	err := executor.Execute(func() error {
		executed = true
		return nil
	})
	if err != nil {
		t.Errorf("Execute should succeed, got error: %v", err)
	}

	if !executed {
		t.Error("Function should have been executed")
	}

	// Check that metrics were recorded
	metrics := GetPerformanceMetrics()
	if len(metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(metrics))
	}

	if !metrics[0].Success {
		t.Error("Expected execution to be marked as successful")
	}
}

func TestPerformanceOptimizedExecutorCaching(t *testing.T) {
	executor := NewPerformanceOptimizedExecutor("test", "test_mode", true)
	defer executor.Cleanup()

	// Test caching
	testValue := "test_value"
	executor.SetCachedOperation("test_key", testValue)

	cached, found := executor.GetCachedOperation("test_key")
	if !found {
		t.Error("Expected to find cached operation")
	}

	if cached != testValue {
		t.Errorf("Expected cached value '%s', got '%v'", testValue, cached)
	}

	// Test cache miss
	_, found = executor.GetCachedOperation("nonexistent_key")
	if found {
		t.Error("Expected cache miss for nonexistent key")
	}
}

func TestPerformanceOptimizedExecutorResourceLimits(t *testing.T) {
	EnablePerformanceMonitoring()
	defer DisablePerformanceMonitoring()

	executor := NewPerformanceOptimizedExecutor("test", "test_mode", false)
	defer executor.Cleanup()

	// Test execution that might exceed resource limits
	err := executor.Execute(func() error {
		// Simulate some work
		time.Sleep(1 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Errorf("Normal execution should succeed, got error: %v", err)
	}
}

func BenchmarkPerformanceTracking(b *testing.B) {
	EnablePerformanceMonitoring()
	defer DisablePerformanceMonitoring()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		metrics := StartPerformanceTracking("benchmark", "test")
		FinishPerformanceTracking(metrics, true, "")
	}
}

func BenchmarkPerformanceTrackingDisabled(b *testing.B) {
	DisablePerformanceMonitoring()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		metrics := StartPerformanceTracking("benchmark", "test")
		FinishPerformanceTracking(metrics, true, "")
	}
}

func BenchmarkResourceLimiterCheck(b *testing.B) {
	limiter := NewResourceLimiter(1024, 1000, 10*time.Minute)
	defer limiter.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		limiter.CheckResourceLimits()
	}
}

func TestMemoryTracking(t *testing.T) {
	EnablePerformanceMonitoring()
	defer DisablePerformanceMonitoring()

	ClearPerformanceMetrics()

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	metrics := StartPerformanceTracking("memory_test", "test")

	// Allocate some memory
	data := make([]byte, 1024*1024) // 1MB
	_ = data

	FinishPerformanceTracking(metrics, true, "")

	allMetrics := GetPerformanceMetrics()
	if len(allMetrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(allMetrics))
	}

	metric := allMetrics[0]

	// Check that memory usage was tracked
	if metric.MemoryUsageBefore <= 0 {
		t.Error("Expected positive memory usage before")
	}

	if metric.MemoryUsageAfter <= 0 {
		t.Error("Expected positive memory usage after")
	}

	// Memory allocated should be positive (we allocated 1MB)
	if metric.MemoryAllocated <= 0 {
		t.Error("Expected positive memory allocation")
	}
}

func TestGoroutineTracking(t *testing.T) {
	EnablePerformanceMonitoring()
	defer DisablePerformanceMonitoring()

	ClearPerformanceMetrics()

	metrics := StartPerformanceTracking("goroutine_test", "test")

	// Start some goroutines
	done := make(chan bool, 2)
	for i := 0; i < 2; i++ {
		go func() {
			time.Sleep(10 * time.Millisecond)
			done <- true
		}()
	}

	// Wait for goroutines to finish
	for i := 0; i < 2; i++ {
		<-done
	}

	FinishPerformanceTracking(metrics, true, "")

	allMetrics := GetPerformanceMetrics()
	if len(allMetrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(allMetrics))
	}

	metric := allMetrics[0]

	// Check that goroutine count was tracked
	if metric.GoroutinesBefore <= 0 {
		t.Error("Expected positive goroutine count before")
	}

	if metric.GoroutinesAfter <= 0 {
		t.Error("Expected positive goroutine count after")
	}
}
