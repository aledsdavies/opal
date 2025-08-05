package decorators

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// DiagnosticLevel represents the level of diagnostic information
type DiagnosticLevel int

const (
	DiagnosticBasic DiagnosticLevel = iota
	DiagnosticDetailed
	DiagnosticVerbose
)

// DiagnosticReport contains comprehensive diagnostic information
type DiagnosticReport struct {
	Timestamp       time.Time            `json:"timestamp"`
	Level           DiagnosticLevel      `json:"level"`
	SystemInfo      SystemInfo           `json:"system_info"`
	RuntimeInfo     RuntimeInfo          `json:"runtime_info"`
	PerformanceInfo PerformanceInfo      `json:"performance_info"`
	ErrorInfo       ErrorInfo            `json:"error_info,omitempty"`
	DecoratorsInfo  DecoratorDiagnostics `json:"decorators_info"`
	SecurityInfo    SecurityDiagnostics  `json:"security_info"`
	CacheInfo       CacheDiagnostics     `json:"cache_info"`
	TraceInfo       TraceDiagnostics     `json:"trace_info"`
	Recommendations []string             `json:"recommendations"`
}

// SystemInfo contains system-level diagnostic information
type SystemInfo struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	NumCPU       int    `json:"num_cpu"`
	GoVersion    string `json:"go_version"`
	WorkingDir   string `json:"working_dir"`
	Hostname     string `json:"hostname"`
	PID          int    `json:"pid"`
	PPID         int    `json:"ppid"`
	UID          int    `json:"uid"`
	GID          int    `json:"gid"`
}

// RuntimeInfo contains Go runtime diagnostic information
type RuntimeInfo struct {
	NumGoroutine int         `json:"num_goroutine"`
	NumCgoCall   int64       `json:"num_cgo_call"`
	MemStats     MemoryStats `json:"mem_stats"`
	GCStats      GCStats     `json:"gc_stats"`
	BuildInfo    BuildInfo   `json:"build_info"`
}

// MemoryStats contains memory-related statistics
type MemoryStats struct {
	Alloc      uint64 `json:"alloc_bytes"`
	TotalAlloc uint64 `json:"total_alloc_bytes"`
	Sys        uint64 `json:"sys_bytes"`
	Lookups    uint64 `json:"lookups"`
	Mallocs    uint64 `json:"mallocs"`
	Frees      uint64 `json:"frees"`
	HeapAlloc  uint64 `json:"heap_alloc_bytes"`
	HeapSys    uint64 `json:"heap_sys_bytes"`
	HeapIdle   uint64 `json:"heap_idle_bytes"`
	HeapInuse  uint64 `json:"heap_inuse_bytes"`
	StackInuse uint64 `json:"stack_inuse_bytes"`
	StackSys   uint64 `json:"stack_sys_bytes"`
	NumGC      uint32 `json:"num_gc"`
}

// GCStats contains garbage collection statistics
type GCStats struct {
	LastGC       time.Time     `json:"last_gc"`
	NextGC       uint64        `json:"next_gc_bytes"`
	PauseTotal   time.Duration `json:"pause_total"`
	PauseAvg     time.Duration `json:"pause_avg"`
	GCCPUPercent float64       `json:"gc_cpu_percent"`
}

// BuildInfo contains build-related information
type BuildInfo struct {
	GoVersion string            `json:"go_version"`
	Path      string            `json:"path"`
	Main      ModuleInfo        `json:"main"`
	Deps      []ModuleInfo      `json:"deps"`
	Settings  map[string]string `json:"settings"`
}

// ModuleInfo contains module information
type ModuleInfo struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	Sum     string `json:"sum"`
}

// PerformanceInfo contains performance-related diagnostics
type PerformanceInfo struct {
	Enabled              bool                   `json:"enabled"`
	TotalExecutions      int                    `json:"total_executions"`
	SuccessfulExecutions int                    `json:"successful_executions"`
	FailedExecutions     int                    `json:"failed_executions"`
	AverageExecutionTime time.Duration          `json:"average_execution_time"`
	TotalExecutionTime   time.Duration          `json:"total_execution_time"`
	PeakMemoryUsage      int64                  `json:"peak_memory_usage"`
	ResourceLimitHits    int                    `json:"resource_limit_hits"`
	TopDecorators        []DecoratorPerformance `json:"top_decorators"`
}

// DecoratorPerformance contains performance info for a specific decorator
type DecoratorPerformance struct {
	Name            string        `json:"name"`
	ExecutionCount  int           `json:"execution_count"`
	TotalTime       time.Duration `json:"total_time"`
	AverageTime     time.Duration `json:"average_time"`
	SuccessRate     float64       `json:"success_rate"`
	PeakMemoryUsage int64         `json:"peak_memory_usage"`
}

// ErrorInfo contains error-related diagnostic information
type ErrorInfo struct {
	HasErrors     bool           `json:"has_errors"`
	TotalErrors   int            `json:"total_errors"`
	ErrorTypes    map[string]int `json:"error_types"`
	RecentErrors  []ErrorDetail  `json:"recent_errors"`
	ErrorPatterns []ErrorPattern `json:"error_patterns"`
}

// ErrorDetail contains details about a specific error
type ErrorDetail struct {
	Timestamp  time.Time `json:"timestamp"`
	Decorator  string    `json:"decorator"`
	Message    string    `json:"message"`
	StackTrace []string  `json:"stack_trace"`
	Context    string    `json:"context"`
}

// ErrorPattern represents a recurring error pattern
type ErrorPattern struct {
	Pattern    string    `json:"pattern"`
	Count      int       `json:"count"`
	LastSeen   time.Time `json:"last_seen"`
	Decorators []string  `json:"decorators"`
	Suggestion string    `json:"suggestion"`
}

// DecoratorDiagnostics contains decorator-specific diagnostic information
type DecoratorDiagnostics struct {
	RegisteredDecorators int                        `json:"registered_decorators"`
	ActiveDecorators     []string                   `json:"active_decorators"`
	DecoratorDetails     map[string]DecoratorDetail `json:"decorator_details"`
}

// DecoratorDetail contains detailed information about a decorator
type DecoratorDetail struct {
	Name               string        `json:"name"`
	Description        string        `json:"description"`
	ParameterCount     int           `json:"parameter_count"`
	ExecutionCount     int           `json:"execution_count"`
	LastExecuted       time.Time     `json:"last_executed"`
	AverageTime        time.Duration `json:"average_time"`
	SuccessRate        float64       `json:"success_rate"`
	ImportRequirements []string      `json:"import_requirements"`
}

// SecurityDiagnostics contains security-related diagnostic information
type SecurityDiagnostics struct {
	ValidationEnabled  bool                `json:"validation_enabled"`
	TotalValidations   int                 `json:"total_validations"`
	FailedValidations  int                 `json:"failed_validations"`
	SecurityViolations int                 `json:"security_violations"`
	ViolationTypes     map[string]int      `json:"violation_types"`
	RecentViolations   []SecurityViolation `json:"recent_violations"`
	SanitizationStats  SanitizationStats   `json:"sanitization_stats"`
}

// SecurityViolation represents a security violation
type SecurityViolation struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Decorator string    `json:"decorator"`
	Parameter string    `json:"parameter"`
	Value     string    `json:"value"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
}

// SanitizationStats contains sanitization statistics
type SanitizationStats struct {
	PathSanitizations    int `json:"path_sanitizations"`
	CommandSanitizations int `json:"command_sanitizations"`
	EnvVarSanitizations  int `json:"env_var_sanitizations"`
}

// CacheDiagnostics contains cache-related diagnostic information
type CacheDiagnostics struct {
	Enabled     bool                  `json:"enabled"`
	CacheStats  map[string]CacheStats `json:"cache_stats"`
	TotalHits   int                   `json:"total_hits"`
	TotalMisses int                   `json:"total_misses"`
	HitRate     float64               `json:"hit_rate"`
	MemoryUsage int64                 `json:"memory_usage_bytes"`
}

// CacheStats contains statistics for a specific cache
type CacheStats struct {
	Size      int     `json:"size"`
	MaxSize   int     `json:"max_size"`
	Hits      int     `json:"hits"`
	Misses    int     `json:"misses"`
	HitRate   float64 `json:"hit_rate"`
	Evictions int     `json:"evictions"`
	TTL       string  `json:"ttl"`
}

// TraceDiagnostics contains trace-related diagnostic information
type TraceDiagnostics struct {
	Enabled       bool    `json:"enabled"`
	TotalTraces   int     `json:"total_traces"`
	ActiveTraces  int     `json:"active_traces"`
	TotalEvents   int     `json:"total_events"`
	AverageEvents float64 `json:"average_events_per_trace"`
	TraceSuccess  float64 `json:"trace_success_rate"`
	MemoryUsage   int64   `json:"memory_usage_bytes"`
}

// DiagnosticsCollector collects and analyzes diagnostic information
type DiagnosticsCollector struct {
	mu           sync.RWMutex
	enabled      bool
	level        DiagnosticLevel
	errorHistory []ErrorDetail
	violations   []SecurityViolation
	maxHistory   int
	logger       *Logger
}

// NewDiagnosticsCollector creates a new diagnostics collector
func NewDiagnosticsCollector() *DiagnosticsCollector {
	return &DiagnosticsCollector{
		enabled:      true,
		level:        DiagnosticBasic,
		errorHistory: make([]ErrorDetail, 0),
		violations:   make([]SecurityViolation, 0),
		maxHistory:   100,
		logger:       GetLogger("diagnostics"),
	}
}

// Enable enables diagnostics collection
func (dc *DiagnosticsCollector) Enable() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.enabled = true
	dc.logger.Info("Diagnostics collection enabled")
}

// Disable disables diagnostics collection
func (dc *DiagnosticsCollector) Disable() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.enabled = false
	dc.logger.Info("Diagnostics collection disabled")
}

// SetLevel sets the diagnostic level
func (dc *DiagnosticsCollector) SetLevel(level DiagnosticLevel) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.level = level
	dc.logger.Infof("Diagnostic level set to %d", level)
}

// RecordError records an error for diagnostic purposes
func (dc *DiagnosticsCollector) RecordError(decorator, message string, stackTrace []string, context string) {
	if !dc.enabled {
		return
	}

	dc.mu.Lock()
	defer dc.mu.Unlock()

	error := ErrorDetail{
		Timestamp:  time.Now(),
		Decorator:  decorator,
		Message:    message,
		StackTrace: stackTrace,
		Context:    context,
	}

	dc.errorHistory = append(dc.errorHistory, error)

	// Keep only recent errors
	if len(dc.errorHistory) > dc.maxHistory {
		dc.errorHistory = dc.errorHistory[len(dc.errorHistory)-dc.maxHistory:]
	}

	dc.logger.Debugf("Recorded error for %s: %s", decorator, message)
}

// RecordSecurityViolation records a security violation
func (dc *DiagnosticsCollector) RecordSecurityViolation(violationType, decorator, parameter, value, severity, message string) {
	if !dc.enabled {
		return
	}

	dc.mu.Lock()
	defer dc.mu.Unlock()

	violation := SecurityViolation{
		Timestamp: time.Now(),
		Type:      violationType,
		Decorator: decorator,
		Parameter: parameter,
		Value:     value,
		Severity:  severity,
		Message:   message,
	}

	dc.violations = append(dc.violations, violation)

	// Keep only recent violations
	if len(dc.violations) > dc.maxHistory {
		dc.violations = dc.violations[len(dc.violations)-dc.maxHistory:]
	}

	dc.logger.Warnf("Security violation recorded: %s in %s", violationType, decorator)
}

// GenerateReport generates a comprehensive diagnostic report
func (dc *DiagnosticsCollector) GenerateReport() *DiagnosticReport {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	report := &DiagnosticReport{
		Timestamp:       time.Now(),
		Level:           dc.level,
		SystemInfo:      dc.collectSystemInfo(),
		RuntimeInfo:     dc.collectRuntimeInfo(),
		PerformanceInfo: dc.collectPerformanceInfo(),
		ErrorInfo:       dc.collectErrorInfo(),
		DecoratorsInfo:  dc.collectDecoratorsInfo(),
		SecurityInfo:    dc.collectSecurityInfo(),
		CacheInfo:       dc.collectCacheInfo(),
		TraceInfo:       dc.collectTraceInfo(),
		Recommendations: dc.generateRecommendations(),
	}

	return report
}

// collectSystemInfo collects system-level information
func (dc *DiagnosticsCollector) collectSystemInfo() SystemInfo {
	hostname, _ := os.Hostname()
	workingDir, _ := os.Getwd()

	return SystemInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		NumCPU:       runtime.NumCPU(),
		GoVersion:    runtime.Version(),
		WorkingDir:   workingDir,
		Hostname:     hostname,
		PID:          os.Getpid(),
		PPID:         os.Getppid(),
		UID:          os.Getuid(),
		GID:          os.Getgid(),
	}
}

// collectRuntimeInfo collects Go runtime information
func (dc *DiagnosticsCollector) collectRuntimeInfo() RuntimeInfo {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	buildInfo := dc.collectBuildInfo()

	return RuntimeInfo{
		NumGoroutine: runtime.NumGoroutine(),
		NumCgoCall:   runtime.NumCgoCall(),
		MemStats:     dc.convertMemStats(&memStats),
		GCStats:      dc.convertGCStats(&memStats),
		BuildInfo:    buildInfo,
	}
}

// collectBuildInfo collects build information
func (dc *DiagnosticsCollector) collectBuildInfo() BuildInfo {
	info := BuildInfo{
		GoVersion: runtime.Version(),
		Deps:      make([]ModuleInfo, 0),
		Settings:  make(map[string]string),
	}

	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		info.Path = buildInfo.Path
		info.Main = ModuleInfo{
			Path:    buildInfo.Main.Path,
			Version: buildInfo.Main.Version,
			Sum:     buildInfo.Main.Sum,
		}

		for _, dep := range buildInfo.Deps {
			info.Deps = append(info.Deps, ModuleInfo{
				Path:    dep.Path,
				Version: dep.Version,
				Sum:     dep.Sum,
			})
		}

		for _, setting := range buildInfo.Settings {
			info.Settings[setting.Key] = setting.Value
		}
	}

	return info
}

// convertMemStats converts runtime.MemStats to our MemoryStats
func (dc *DiagnosticsCollector) convertMemStats(memStats *runtime.MemStats) MemoryStats {
	return MemoryStats{
		Alloc:      memStats.Alloc,
		TotalAlloc: memStats.TotalAlloc,
		Sys:        memStats.Sys,
		Lookups:    memStats.Lookups,
		Mallocs:    memStats.Mallocs,
		Frees:      memStats.Frees,
		HeapAlloc:  memStats.HeapAlloc,
		HeapSys:    memStats.HeapSys,
		HeapIdle:   memStats.HeapIdle,
		HeapInuse:  memStats.HeapInuse,
		StackInuse: memStats.StackInuse,
		StackSys:   memStats.StackSys,
		NumGC:      memStats.NumGC,
	}
}

// convertGCStats converts GC-related stats
func (dc *DiagnosticsCollector) convertGCStats(memStats *runtime.MemStats) GCStats {
	gcStats := GCStats{
		NextGC:       memStats.NextGC,
		PauseTotal:   time.Duration(memStats.PauseTotalNs),
		GCCPUPercent: memStats.GCCPUFraction * 100,
	}

	if memStats.NumGC > 0 {
		gcStats.LastGC = time.Unix(0, int64(memStats.LastGC))
		gcStats.PauseAvg = gcStats.PauseTotal / time.Duration(memStats.NumGC)
	}

	return gcStats
}

// collectPerformanceInfo collects performance-related information
func (dc *DiagnosticsCollector) collectPerformanceInfo() PerformanceInfo {
	metrics := GetPerformanceMetrics()

	info := PerformanceInfo{
		Enabled:         IsPerformanceMonitoringEnabled(),
		TotalExecutions: len(metrics),
		TopDecorators:   make([]DecoratorPerformance, 0),
	}

	if len(metrics) > 0 {
		totalTime := time.Duration(0)
		successful := 0
		failed := 0
		var peakMemory int64

		decoratorStats := make(map[string]*DecoratorPerformance)

		for _, metric := range metrics {
			totalTime += metric.Duration
			if metric.Success {
				successful++
			} else {
				failed++
			}

			if metric.MemoryAllocated > peakMemory {
				peakMemory = metric.MemoryAllocated
			}

			// Update decorator-specific stats
			if _, exists := decoratorStats[metric.DecoratorName]; !exists {
				decoratorStats[metric.DecoratorName] = &DecoratorPerformance{
					Name: metric.DecoratorName,
				}
			}

			decoratorStats[metric.DecoratorName].ExecutionCount++
			decoratorStats[metric.DecoratorName].TotalTime += metric.Duration
			if metric.Success {
				decoratorStats[metric.DecoratorName].SuccessRate++
			}
			if metric.MemoryAllocated > decoratorStats[metric.DecoratorName].PeakMemoryUsage {
				decoratorStats[metric.DecoratorName].PeakMemoryUsage = metric.MemoryAllocated
			}
		}

		info.SuccessfulExecutions = successful
		info.FailedExecutions = failed
		info.TotalExecutionTime = totalTime
		info.AverageExecutionTime = totalTime / time.Duration(len(metrics))
		info.PeakMemoryUsage = peakMemory

		// Calculate success rates and average times for decorators
		for _, decoratorStat := range decoratorStats {
			decoratorStat.AverageTime = decoratorStat.TotalTime / time.Duration(decoratorStat.ExecutionCount)
			decoratorStat.SuccessRate = decoratorStat.SuccessRate / float64(decoratorStat.ExecutionCount) * 100
			info.TopDecorators = append(info.TopDecorators, *decoratorStat)
		}
	}

	return info
}

// collectErrorInfo collects error-related information
func (dc *DiagnosticsCollector) collectErrorInfo() ErrorInfo {
	errorTypes := make(map[string]int)
	patterns := make(map[string]*ErrorPattern)

	for _, err := range dc.errorHistory {
		// Categorize error types
		if strings.Contains(err.Message, "timeout") {
			errorTypes["timeout"]++
		} else if strings.Contains(err.Message, "permission") {
			errorTypes["permission"]++
		} else if strings.Contains(err.Message, "not found") {
			errorTypes["not_found"]++
		} else if strings.Contains(err.Message, "validation") {
			errorTypes["validation"]++
		} else {
			errorTypes["other"]++
		}

		// Identify patterns
		pattern := dc.extractErrorPattern(err.Message)
		if _, exists := patterns[pattern]; !exists {
			patterns[pattern] = &ErrorPattern{
				Pattern:    pattern,
				Count:      0,
				Decorators: make([]string, 0),
			}
		}
		patterns[pattern].Count++
		patterns[pattern].LastSeen = err.Timestamp
		patterns[pattern].Decorators = append(patterns[pattern].Decorators, err.Decorator)
	}

	// Convert pattern map to slice
	errorPatterns := make([]ErrorPattern, 0, len(patterns))
	for _, pattern := range patterns {
		errorPatterns = append(errorPatterns, *pattern)
	}

	return ErrorInfo{
		HasErrors:     len(dc.errorHistory) > 0,
		TotalErrors:   len(dc.errorHistory),
		ErrorTypes:    errorTypes,
		RecentErrors:  dc.getRecentErrors(10),
		ErrorPatterns: errorPatterns,
	}
}

// extractErrorPattern extracts a pattern from an error message
func (dc *DiagnosticsCollector) extractErrorPattern(message string) string {
	// Simplified pattern extraction
	message = strings.ToLower(message)
	if strings.Contains(message, "timeout") {
		return "timeout_error"
	} else if strings.Contains(message, "permission denied") {
		return "permission_error"
	} else if strings.Contains(message, "not found") {
		return "not_found_error"
	} else if strings.Contains(message, "validation failed") {
		return "validation_error"
	}
	return "generic_error"
}

// getRecentErrors returns the most recent errors
func (dc *DiagnosticsCollector) getRecentErrors(count int) []ErrorDetail {
	if len(dc.errorHistory) <= count {
		return dc.errorHistory
	}
	return dc.errorHistory[len(dc.errorHistory)-count:]
}

// collectDecoratorsInfo collects decorator-related information
func (dc *DiagnosticsCollector) collectDecoratorsInfo() DecoratorDiagnostics {
	return DecoratorDiagnostics{
		RegisteredDecorators: 0, // Would need access to decorator registry
		ActiveDecorators:     make([]string, 0),
		DecoratorDetails:     make(map[string]DecoratorDetail),
	}
}

// collectSecurityInfo collects security-related information
func (dc *DiagnosticsCollector) collectSecurityInfo() SecurityDiagnostics {
	violationTypes := make(map[string]int)

	for _, violation := range dc.violations {
		violationTypes[violation.Type]++
	}

	return SecurityDiagnostics{
		ValidationEnabled:  true, // Would check actual security settings
		TotalValidations:   0,    // Would need access to validation stats
		FailedValidations:  0,
		SecurityViolations: len(dc.violations),
		ViolationTypes:     violationTypes,
		RecentViolations:   dc.getRecentViolations(10),
		SanitizationStats:  SanitizationStats{}, // Would need access to sanitization stats
	}
}

// getRecentViolations returns the most recent security violations
func (dc *DiagnosticsCollector) getRecentViolations(count int) []SecurityViolation {
	if len(dc.violations) <= count {
		return dc.violations
	}
	return dc.violations[len(dc.violations)-count:]
}

// collectCacheInfo collects cache-related information
func (dc *DiagnosticsCollector) collectCacheInfo() CacheDiagnostics {
	cacheManager := GetCacheManager()
	stats := cacheManager.GetCacheStats()

	return CacheDiagnostics{
		Enabled:     true,                        // Caching is always enabled
		CacheStats:  make(map[string]CacheStats), // Would need detailed cache stats
		TotalHits:   0,
		TotalMisses: 0,
		HitRate:     0.0,
		MemoryUsage: int64(stats["template_cache_size"]), // Use actual stats
	}
}

// collectTraceInfo collects trace-related information
func (dc *DiagnosticsCollector) collectTraceInfo() TraceDiagnostics {
	tracer := GetGlobalTracer()
	stats := tracer.GetTraceStatistics()

	return TraceDiagnostics{
		Enabled:       tracer.IsEnabled(),
		TotalTraces:   stats["total_traces"].(int),
		ActiveTraces:  0, // Would need to track active traces
		TotalEvents:   stats["total_events"].(int),
		AverageEvents: 0.0,
		TraceSuccess:  0.0,
		MemoryUsage:   0,
	}
}

// generateRecommendations generates optimization recommendations
func (dc *DiagnosticsCollector) generateRecommendations() []string {
	recommendations := make([]string, 0)

	// Analyze performance metrics
	metrics := GetPerformanceMetrics()
	if len(metrics) > 0 {
		avgDuration := time.Duration(0)
		for _, metric := range metrics {
			avgDuration += metric.Duration
		}
		avgDuration /= time.Duration(len(metrics))

		if avgDuration > 5*time.Second {
			recommendations = append(recommendations, "Consider using @parallel decorator for long-running operations")
		}
	}

	// Analyze error patterns
	if len(dc.errorHistory) > 0 {
		timeoutErrors := 0
		for _, err := range dc.errorHistory {
			if strings.Contains(err.Message, "timeout") {
				timeoutErrors++
			}
		}

		if float64(timeoutErrors)/float64(len(dc.errorHistory)) > 0.1 {
			recommendations = append(recommendations, "High timeout error rate - consider increasing timeout values or using @retry decorator")
		}
	}

	// Check memory usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	if memStats.Alloc > 100*1024*1024 { // 100MB
		recommendations = append(recommendations, "High memory usage detected - consider optimizing resource usage")
	}

	// Check security violations
	if len(dc.violations) > 0 {
		recommendations = append(recommendations, "Security violations detected - review parameter validation and sanitization")
	}

	return recommendations
}

// ExportReport exports a diagnostic report to JSON
func (dc *DiagnosticsCollector) ExportReport() (string, error) {
	report := dc.GenerateReport()
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal diagnostic report: %w", err)
	}

	return string(data), nil
}

// Global diagnostics collector
var globalDiagnosticsCollector = NewDiagnosticsCollector()

// GetGlobalDiagnosticsCollector returns the global diagnostics collector
func GetGlobalDiagnosticsCollector() *DiagnosticsCollector {
	return globalDiagnosticsCollector
}

// EnableDiagnostics enables global diagnostics collection
func EnableDiagnostics() {
	globalDiagnosticsCollector.Enable()
}

// DisableDiagnostics disables global diagnostics collection
func DisableDiagnostics() {
	globalDiagnosticsCollector.Disable()
}

// SetDiagnosticLevel sets the global diagnostic level
func SetDiagnosticLevel(level DiagnosticLevel) {
	globalDiagnosticsCollector.SetLevel(level)
}

// RecordError records an error globally
func RecordError(decorator, message string, stackTrace []string, context string) {
	globalDiagnosticsCollector.RecordError(decorator, message, stackTrace, context)
}

// RecordSecurityViolation records a security violation globally
func RecordSecurityViolation(violationType, decorator, parameter, value, severity, message string) {
	globalDiagnosticsCollector.RecordSecurityViolation(violationType, decorator, parameter, value, severity, message)
}

// GenerateDiagnosticReport generates a global diagnostic report
func GenerateDiagnosticReport() *DiagnosticReport {
	return globalDiagnosticsCollector.GenerateReport()
}

// ExportDiagnosticReport exports a global diagnostic report to JSON
func ExportDiagnosticReport() (string, error) {
	return globalDiagnosticsCollector.ExportReport()
}
