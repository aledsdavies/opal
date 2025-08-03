package decorators

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// TraceEvent represents a single event in the execution trace
type TraceEvent struct {
	ID           string                 `json:"id"`
	ParentID     string                 `json:"parent_id,omitempty"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Decorator    string                 `json:"decorator,omitempty"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time,omitempty"`
	Duration     time.Duration          `json:"duration,omitempty"`
	Status       string                 `json:"status"` // started, completed, failed
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	StackTrace   []string               `json:"stack_trace,omitempty"`
	MemoryBefore int64                  `json:"memory_before,omitempty"`
	MemoryAfter  int64                  `json:"memory_after,omitempty"`
	GoroutineID  int64                  `json:"goroutine_id,omitempty"`
}

// ExecutionTrace represents a complete execution trace
type ExecutionTrace struct {
	ID         string       `json:"id"`
	StartTime  time.Time    `json:"start_time"`
	EndTime    time.Time    `json:"end_time"`
	Duration   time.Duration `json:"duration"`
	Events     []TraceEvent `json:"events"`
	TotalEvents int         `json:"total_events"`
	Success    bool         `json:"success"`
	RootEvent  string       `json:"root_event"`
}

// ExecutionTracer manages execution tracing
type ExecutionTracer struct {
	mu       sync.RWMutex
	enabled  bool
	traces   map[string]*ExecutionTrace
	events   map[string]*TraceEvent
	maxTraces int
	logger   *Logger
}

// NewExecutionTracer creates a new execution tracer
func NewExecutionTracer() *ExecutionTracer {
	return &ExecutionTracer{
		enabled:   false,
		traces:    make(map[string]*ExecutionTrace),
		events:    make(map[string]*TraceEvent),
		maxTraces: 100, // Keep last 100 traces
		logger:    GetLogger("tracer"),
	}
}

// Enable enables execution tracing
func (et *ExecutionTracer) Enable() {
	et.mu.Lock()
	defer et.mu.Unlock()
	et.enabled = true
	et.logger.Info("Execution tracing enabled")
}

// Disable disables execution tracing
func (et *ExecutionTracer) Disable() {
	et.mu.Lock()
	defer et.mu.Unlock()
	et.enabled = false
	et.logger.Info("Execution tracing disabled")
}

// IsEnabled returns whether tracing is enabled
func (et *ExecutionTracer) IsEnabled() bool {
	et.mu.RLock()
	defer et.mu.RUnlock()
	return et.enabled
}

// StartTrace begins a new execution trace
func (et *ExecutionTracer) StartTrace(traceID, name string) *ExecutionTrace {
	if !et.IsEnabled() {
		return nil
	}
	
	et.mu.Lock()
	defer et.mu.Unlock()
	
	trace := &ExecutionTrace{
		ID:        traceID,
		StartTime: time.Now(),
		Events:    make([]TraceEvent, 0),
		Success:   false,
	}
	
	// Evict old traces if needed
	if len(et.traces) >= et.maxTraces {
		et.evictOldestTrace()
	}
	
	et.traces[traceID] = trace
	et.logger.Debugf("Started trace %s: %s", traceID, name)
	
	return trace
}

// FinishTrace completes an execution trace
func (et *ExecutionTracer) FinishTrace(traceID string, success bool) {
	if !et.IsEnabled() {
		return
	}
	
	et.mu.Lock()
	defer et.mu.Unlock()
	
	if trace, exists := et.traces[traceID]; exists {
		trace.EndTime = time.Now()
		trace.Duration = trace.EndTime.Sub(trace.StartTime)
		trace.Success = success
		trace.TotalEvents = len(trace.Events)
		
		et.logger.Debugf("Finished trace %s: success=%v duration=%v events=%d", 
			traceID, success, trace.Duration, trace.TotalEvents)
	}
}

// StartEvent begins a new trace event
func (et *ExecutionTracer) StartEvent(traceID, eventID, parentID, eventType, name, decorator string, metadata map[string]interface{}) *TraceEvent {
	if !et.IsEnabled() {
		return nil
	}
	
	et.mu.Lock()
	defer et.mu.Unlock()
	
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	event := &TraceEvent{
		ID:           eventID,
		ParentID:     parentID,
		Type:         eventType,
		Name:         name,
		Decorator:    decorator,
		StartTime:    time.Now(),
		Status:       "started",
		Metadata:     metadata,
		MemoryBefore: int64(memStats.Alloc),
		GoroutineID:  getGoroutineID(),
	}
	
	// Add stack trace for debugging (always add for tracing purposes)
	event.StackTrace = getStackTrace()
	
	et.events[eventID] = event
	
	// Add to trace if it exists
	if trace, exists := et.traces[traceID]; exists {
		trace.Events = append(trace.Events, *event)
		if parentID == "" {
			trace.RootEvent = eventID
		}
	}
	
	et.logger.Tracef("Started event %s in trace %s: %s", eventID, traceID, name)
	
	return event
}

// FinishEvent completes a trace event
func (et *ExecutionTracer) FinishEvent(eventID string, success bool, err error) {
	if !et.IsEnabled() {
		return
	}
	
	et.mu.Lock()
	defer et.mu.Unlock()
	
	if event, exists := et.events[eventID]; exists {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		
		event.EndTime = time.Now()
		event.Duration = event.EndTime.Sub(event.StartTime)
		event.MemoryAfter = int64(memStats.Alloc)
		
		if success {
			event.Status = "completed"
		} else {
			event.Status = "failed"
			if err != nil {
				event.Error = err.Error()
			}
		}
		
		et.logger.Tracef("Finished event %s: status=%s duration=%v", 
			eventID, event.Status, event.Duration)
	}
}

// GetTrace returns a trace by ID
func (et *ExecutionTracer) GetTrace(traceID string) *ExecutionTrace {
	et.mu.RLock()
	defer et.mu.RUnlock()
	
	if trace, exists := et.traces[traceID]; exists {
		// Return a copy to prevent race conditions
		traceCopy := *trace
		traceCopy.Events = make([]TraceEvent, len(trace.Events))
		copy(traceCopy.Events, trace.Events)
		return &traceCopy
	}
	
	return nil
}

// GetAllTraces returns all traces
func (et *ExecutionTracer) GetAllTraces() []ExecutionTrace {
	et.mu.RLock()
	defer et.mu.RUnlock()
	
	traces := make([]ExecutionTrace, 0, len(et.traces))
	for _, trace := range et.traces {
		traceCopy := *trace
		traceCopy.Events = make([]TraceEvent, len(trace.Events))
		copy(traceCopy.Events, trace.Events)
		traces = append(traces, traceCopy)
	}
	
	return traces
}

// ClearTraces removes all traces
func (et *ExecutionTracer) ClearTraces() {
	et.mu.Lock()
	defer et.mu.Unlock()
	
	et.traces = make(map[string]*ExecutionTrace)
	et.events = make(map[string]*TraceEvent)
	et.logger.Info("Cleared all traces")
}

// ExportTrace exports a trace to JSON
func (et *ExecutionTracer) ExportTrace(traceID string) (string, error) {
	trace := et.GetTrace(traceID)
	if trace == nil {
		return "", fmt.Errorf("trace not found: %s", traceID)
	}
	
	data, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal trace: %w", err)
	}
	
	return string(data), nil
}

// GetTraceStatistics returns statistics about traces
func (et *ExecutionTracer) GetTraceStatistics() map[string]interface{} {
	et.mu.RLock()
	defer et.mu.RUnlock()
	
	totalTraces := len(et.traces)
	successfulTraces := 0
	failedTraces := 0
	totalEvents := 0
	totalDuration := time.Duration(0)
	
	for _, trace := range et.traces {
		if trace.Success {
			successfulTraces++
		} else {
			failedTraces++
		}
		totalEvents += len(trace.Events)
		totalDuration += trace.Duration
	}
	
	avgDuration := time.Duration(0)
	if totalTraces > 0 {
		avgDuration = totalDuration / time.Duration(totalTraces)
	}
	
	return map[string]interface{}{
		"total_traces":      totalTraces,
		"successful_traces": successfulTraces,
		"failed_traces":     failedTraces,
		"total_events":      totalEvents,
		"total_duration":    totalDuration,
		"average_duration":  avgDuration,
		"enabled":           et.enabled,
	}
}

// evictOldestTrace removes the oldest trace to make room for new ones
func (et *ExecutionTracer) evictOldestTrace() {
	var oldestID string
	var oldestTime time.Time
	first := true
	
	for id, trace := range et.traces {
		if first || trace.StartTime.Before(oldestTime) {
			oldestID = id
			oldestTime = trace.StartTime
			first = false
		}
	}
	
	if oldestID != "" {
		delete(et.traces, oldestID)
		et.logger.Debugf("Evicted oldest trace: %s", oldestID)
	}
}

// getGoroutineID returns the current goroutine ID
func getGoroutineID() int64 {
	// This is a simplified implementation
	// In practice, you might want to use a more robust method
	return int64(runtime.NumGoroutine())
}

// TracingDecorator wraps a decorator with execution tracing
type TracingDecorator struct {
	underlying BlockDecorator
	tracer     *ExecutionTracer
	logger     *Logger
}

// NewTracingDecorator creates a new tracing decorator wrapper
func NewTracingDecorator(underlying BlockDecorator, tracer *ExecutionTracer) *TracingDecorator {
	return &TracingDecorator{
		underlying: underlying,
		tracer:     tracer,
		logger:     GetLogger("tracing"),
	}
}

// Name returns the underlying decorator name
func (td *TracingDecorator) Name() string {
	return td.underlying.Name()
}

// Description returns the underlying decorator description
func (td *TracingDecorator) Description() string {
	return td.underlying.Description()
}

// ParameterSchema returns the underlying decorator parameter schema
func (td *TracingDecorator) ParameterSchema() []ParameterSchema {
	return td.underlying.ParameterSchema()
}

// ImportRequirements returns the underlying decorator import requirements
func (td *TracingDecorator) ImportRequirements() ImportRequirement {
	return td.underlying.ImportRequirements()
}

// ExecuteInterpreter executes with tracing in interpreter mode
func (td *TracingDecorator) ExecuteInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	if !td.tracer.IsEnabled() {
		return td.underlying.ExecuteInterpreter(ctx, params, content)
	}
	
	traceID := generateTraceID()
	eventID := generateEventID()
	
	td.tracer.StartTrace(traceID, fmt.Sprintf("Interpreter: @%s", td.underlying.Name()))
	defer td.tracer.FinishTrace(traceID, false) // Will be updated on success
	
	metadata := map[string]interface{}{
		"mode":        "interpreter",
		"params":      len(params),
		"content":     len(content),
		"working_dir": getWorkingDir(ctx),
	}
	
	td.tracer.StartEvent(traceID, eventID, "", "decorator_execution", 
		fmt.Sprintf("@%s interpreter execution", td.underlying.Name()), 
		td.underlying.Name(), metadata)
	
	result := td.underlying.ExecuteInterpreter(ctx, params, content)
	
	success := result.Error == nil
	td.tracer.FinishEvent(eventID, success, result.Error)
	td.tracer.FinishTrace(traceID, success)
	
	return result
}

// ExecuteGenerator executes with tracing in generator mode
func (td *TracingDecorator) ExecuteGenerator(ctx execution.GeneratorContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	if !td.tracer.IsEnabled() {
		return td.underlying.ExecuteGenerator(ctx, params, content)
	}
	
	traceID := generateTraceID()
	eventID := generateEventID()
	
	td.tracer.StartTrace(traceID, fmt.Sprintf("Generator: @%s", td.underlying.Name()))
	defer td.tracer.FinishTrace(traceID, false) // Will be updated on success
	
	metadata := map[string]interface{}{
		"mode":    "generator",
		"params":  len(params),
		"content": len(content),
	}
	
	td.tracer.StartEvent(traceID, eventID, "", "decorator_execution", 
		fmt.Sprintf("@%s generator execution", td.underlying.Name()), 
		td.underlying.Name(), metadata)
	
	result := td.underlying.ExecuteGenerator(ctx, params, content)
	
	success := result.Error == nil
	td.tracer.FinishEvent(eventID, success, result.Error)
	td.tracer.FinishTrace(traceID, success)
	
	return result
}

// ExecutePlan executes with tracing in plan mode
func (td *TracingDecorator) ExecutePlan(ctx execution.PlanContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	if !td.tracer.IsEnabled() {
		return td.underlying.ExecutePlan(ctx, params, content)
	}
	
	traceID := generateTraceID()
	eventID := generateEventID()
	
	td.tracer.StartTrace(traceID, fmt.Sprintf("Plan: @%s", td.underlying.Name()))
	defer td.tracer.FinishTrace(traceID, false) // Will be updated on success
	
	metadata := map[string]interface{}{
		"mode":    "plan",
		"params":  len(params),
		"content": len(content),
	}
	
	td.tracer.StartEvent(traceID, eventID, "", "decorator_execution", 
		fmt.Sprintf("@%s plan execution", td.underlying.Name()), 
		td.underlying.Name(), metadata)
	
	result := td.underlying.ExecutePlan(ctx, params, content)
	
	success := result.Error == nil
	td.tracer.FinishEvent(eventID, success, result.Error)
	td.tracer.FinishTrace(traceID, success)
	
	return result
}

// Helper functions

func generateTraceID() string {
	return fmt.Sprintf("trace_%d", time.Now().UnixNano())
}

func generateEventID() string {
	return fmt.Sprintf("event_%d", time.Now().UnixNano())
}

func getWorkingDir(ctx execution.InterpreterContext) string {
	// This would need to be implemented based on the actual context interface
	return "/unknown"
}

// Global tracer instance
var globalTracer = NewExecutionTracer()

// GetGlobalTracer returns the global execution tracer
func GetGlobalTracer() *ExecutionTracer {
	return globalTracer
}

// EnableTracing enables global execution tracing
func EnableTracing() {
	globalTracer.Enable()
}

// DisableTracing disables global execution tracing
func DisableTracing() {
	globalTracer.Disable()
}

// IsTracingEnabled returns whether global tracing is enabled
func IsTracingEnabled() bool {
	return globalTracer.IsEnabled()
}