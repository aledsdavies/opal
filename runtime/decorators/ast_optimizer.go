package decorators

import (
	"fmt"
	"strings"
	"sync"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// ASTOptimizer provides optimized AST traversal and processing
type ASTOptimizer struct {
	cache           sync.Map // Cache for processed AST nodes
	flattenedCache  sync.Map // Cache for flattened command sequences
	enableCaching   bool
	enableFlattening bool
}

// NewASTOptimizer creates a new AST optimizer
func NewASTOptimizer() *ASTOptimizer {
	return &ASTOptimizer{
		enableCaching:   true,
		enableFlattening: true,
	}
}

// OptimizedOperation represents an optimized operation with metadata
type OptimizedOperation struct {
	Operation
	Depth           int
	CanParallelize  bool
	EstimatedTime   int64 // Estimated execution time in milliseconds
	ResourceUsage   int   // Estimated resource usage (arbitrary units)
	Dependencies    []string // List of dependencies this operation has
}

// FlattenedCommandSequence represents a flattened view of nested commands
type FlattenedCommandSequence struct {
	Commands    []OptimizedOperation
	TotalDepth  int
	Parallelizable bool
	EstimatedTotalTime int64
}

// OptimizeCommandSequence analyzes and optimizes a sequence of commands
func (ao *ASTOptimizer) OptimizeCommandSequence(ctx execution.GeneratorContext, content []ast.CommandContent) (*FlattenedCommandSequence, error) {
	if !ao.enableFlattening {
		// Fall back to basic conversion
		operations, err := ConvertCommandsToOperations(ctx, content)
		if err != nil {
			return nil, err
		}
		
		optimized := make([]OptimizedOperation, len(operations))
		for i, op := range operations {
			optimized[i] = OptimizedOperation{
				Operation: op,
				Depth:     0,
				CanParallelize: true,
				EstimatedTime: 1000, // Default 1 second
				ResourceUsage: 1,
			}
		}
		
		return &FlattenedCommandSequence{
			Commands: optimized,
			TotalDepth: 1,
			Parallelizable: true,
			EstimatedTotalTime: int64(len(operations) * 1000),
		}, nil
	}
	
	// Generate cache key for the command sequence
	cacheKey := ao.generateSequenceKey(content)
	if cached, found := ao.flattenedCache.Load(cacheKey); found {
		if result, ok := cached.(*FlattenedCommandSequence); ok {
			return result, nil
		}
	}
	
	// Perform deep analysis and optimization
	result, err := ao.analyzeAndOptimize(ctx, content, 0)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	ao.flattenedCache.Store(cacheKey, result)
	
	return result, nil
}

// analyzeAndOptimize performs deep analysis of command structure
func (ao *ASTOptimizer) analyzeAndOptimize(ctx execution.GeneratorContext, content []ast.CommandContent, depth int) (*FlattenedCommandSequence, error) {
	sequence := &FlattenedCommandSequence{
		Commands:       make([]OptimizedOperation, 0),
		TotalDepth:     depth,
		Parallelizable: true,
		EstimatedTotalTime: 0,
	}
	
	for _, cmd := range content {
		switch c := cmd.(type) {
		case *ast.ShellContent:
			op, err := ao.optimizeShellContent(ctx, c, depth)
			if err != nil {
				return nil, err
			}
			sequence.Commands = append(sequence.Commands, *op)
			sequence.EstimatedTotalTime += op.EstimatedTime
			
		case *ast.BlockDecorator:
			nestedOps, err := ao.optimizeBlockDecorator(ctx, c, depth+1)
			if err != nil {
				return nil, err
			}
			sequence.Commands = append(sequence.Commands, nestedOps...)
			
			// Update sequence metadata based on nested operations
			for _, op := range nestedOps {
				sequence.EstimatedTotalTime += op.EstimatedTime
				if !op.CanParallelize {
					sequence.Parallelizable = false
				}
				if op.Depth > sequence.TotalDepth {
					sequence.TotalDepth = op.Depth
				}
			}
		}
	}
	
	return sequence, nil
}

// optimizeShellContent optimizes a shell command
func (ao *ASTOptimizer) optimizeShellContent(ctx execution.GeneratorContext, shell *ast.ShellContent, depth int) (*OptimizedOperation, error) {
	// Extract shell command text
	var commandText strings.Builder
	for _, part := range shell.Parts {
		switch p := part.(type) {
		case *ast.TextPart:
			commandText.WriteString(p.Text)
		// Note: VariablePart doesn't exist in this AST structure
		// Variables are handled at a different level
		default:
			commandText.WriteString(part.String())
		}
	}
	
	cmdStr := strings.TrimSpace(commandText.String())
	
	// Analyze command characteristics
	canParallelize := ao.analyzeParallelizability(cmdStr)
	estimatedTime := ao.estimateExecutionTime(cmdStr)
	resourceUsage := ao.estimateResourceUsage(cmdStr)
	dependencies := ao.extractDependencies(cmdStr)
	
	// Generate proper shell execution code using the template system
	shellBuilder := execution.NewShellCodeBuilder(ctx)
	executionCode, err := shellBuilder.GenerateShellCodeWithReturn(shell)
	if err != nil {
		return nil, fmt.Errorf("failed to generate shell execution code: %w", err)
	}
	
	operation := Operation{
		Code: executionCode,
	}
	
	return &OptimizedOperation{
		Operation:      operation,
		Depth:          depth,
		CanParallelize: canParallelize,
		EstimatedTime:  estimatedTime,
		ResourceUsage:  resourceUsage,
		Dependencies:   dependencies,
	}, nil
}

// optimizeBlockDecorator optimizes a block decorator
func (ao *ASTOptimizer) optimizeBlockDecorator(ctx execution.GeneratorContext, block *ast.BlockDecorator, depth int) ([]OptimizedOperation, error) {
	// Analyze decorator characteristics
	decoratorAnalysis := ao.analyzeDecorator(block.Name, block.Args)
	
	// Recursively optimize nested content
	nestedSequence, err := ao.analyzeAndOptimize(ctx, block.Content, depth)
	if err != nil {
		return nil, err
	}
	
	// Apply decorator-specific optimizations
	operations := ao.applyDecoratorOptimizations(block.Name, decoratorAnalysis, nestedSequence.Commands)
	
	return operations, nil
}

// analyzeParallelizability determines if a command can be run in parallel
func (ao *ASTOptimizer) analyzeParallelizability(command string) bool {
	// Commands that modify global state or depend on previous commands cannot be parallelized
	nonParallelPatterns := []string{
		"cd ", "export ", "source ", ".", ">>", "&&", "||", ";",
		"mkdir -p", "rm -rf", "mv ", "cp -r",
	}
	
	cmdLower := strings.ToLower(command)
	for _, pattern := range nonParallelPatterns {
		if strings.Contains(cmdLower, pattern) {
			return false
		}
	}
	
	return true
}

// estimateExecutionTime estimates how long a command will take to execute
func (ao *ASTOptimizer) estimateExecutionTime(command string) int64 {
	cmdLower := strings.ToLower(command)
	
	// Fast commands (< 1 second)
	if strings.HasPrefix(cmdLower, "echo ") || strings.HasPrefix(cmdLower, "pwd") || 
	   strings.HasPrefix(cmdLower, "whoami") || strings.HasPrefix(cmdLower, "date") {
		return 100 // 100ms
	}
	
	// Medium commands (1-5 seconds)
	if strings.Contains(cmdLower, "ls ") || strings.Contains(cmdLower, "cat ") ||
	   strings.Contains(cmdLower, "grep ") {
		return 1000 // 1 second
	}
	
	// Slow commands (5+ seconds)
	if strings.Contains(cmdLower, "find ") || strings.Contains(cmdLower, "docker ") ||
	   strings.Contains(cmdLower, "npm ") || strings.Contains(cmdLower, "go build") ||
	   strings.Contains(cmdLower, "make ") {
		return 10000 // 10 seconds
	}
	
	// Very slow commands (30+ seconds)
	if strings.Contains(cmdLower, "apt-get ") || strings.Contains(cmdLower, "yum ") ||
	   strings.Contains(cmdLower, "pip install") || strings.Contains(cmdLower, "cargo build") {
		return 30000 // 30 seconds
	}
	
	// Default estimate
	return 2000 // 2 seconds
}

// estimateResourceUsage estimates resource usage (CPU, memory, I/O)
func (ao *ASTOptimizer) estimateResourceUsage(command string) int {
	cmdLower := strings.ToLower(command)
	
	// Low resource usage
	if strings.HasPrefix(cmdLower, "echo ") || strings.HasPrefix(cmdLower, "pwd") {
		return 1
	}
	
	// Medium resource usage
	if strings.Contains(cmdLower, "grep ") || strings.Contains(cmdLower, "sed ") ||
	   strings.Contains(cmdLower, "awk ") {
		return 3
	}
	
	// High resource usage
	if strings.Contains(cmdLower, "docker ") || strings.Contains(cmdLower, "go build") ||
	   strings.Contains(cmdLower, "npm ") || strings.Contains(cmdLower, "make ") {
		return 8
	}
	
	// Very high resource usage
	if strings.Contains(cmdLower, "find / ") || strings.Contains(cmdLower, "dd ") ||
	   strings.Contains(cmdLower, "tar ") {
		return 10
	}
	
	return 5 // Default medium usage
}

// extractDependencies identifies what this command depends on
func (ao *ASTOptimizer) extractDependencies(command string) []string {
	dependencies := make([]string, 0)
	cmdLower := strings.ToLower(command)
	
	// File system dependencies
	if strings.Contains(cmdLower, "cd ") {
		dependencies = append(dependencies, "filesystem")
	}
	
	// Network dependencies
	if strings.Contains(cmdLower, "curl ") || strings.Contains(cmdLower, "wget ") ||
	   strings.Contains(cmdLower, "git clone") || strings.Contains(cmdLower, "npm install") {
		dependencies = append(dependencies, "network")
	}
	
	// Docker dependencies
	if strings.Contains(cmdLower, "docker ") {
		dependencies = append(dependencies, "docker")
	}
	
	// Package manager dependencies
	if strings.Contains(cmdLower, "apt-get ") || strings.Contains(cmdLower, "yum ") ||
	   strings.Contains(cmdLower, "pip ") {
		dependencies = append(dependencies, "package-manager")
	}
	
	return dependencies
}

// analyzeDecorator analyzes decorator characteristics for optimization
func (ao *ASTOptimizer) analyzeDecorator(decoratorName string, params []ast.NamedParameter) map[string]interface{} {
	analysis := make(map[string]interface{})
	analysis["name"] = decoratorName
	analysis["can_optimize"] = true
	
	switch decoratorName {
	case "parallel":
		analysis["creates_concurrency"] = true
		analysis["resource_multiplier"] = ast.GetIntParam(params, "concurrency", 2)
		
	case "timeout":
		analysis["has_time_limit"] = true
		analysis["max_time"] = ast.GetDurationParam(params, "duration", 0)
		
	case "retry":
		analysis["can_fail"] = true
		analysis["max_attempts"] = ast.GetIntParam(params, "attempts", 3)
		analysis["resource_multiplier"] = ast.GetIntParam(params, "attempts", 3)
		
	case "workdir":
		analysis["changes_context"] = true
		analysis["filesystem_dependent"] = true
		
	default:
		analysis["can_optimize"] = false
	}
	
	return analysis
}

// applyDecoratorOptimizations applies decorator-specific optimizations
func (ao *ASTOptimizer) applyDecoratorOptimizations(decoratorName string, analysis map[string]interface{}, operations []OptimizedOperation) []OptimizedOperation {
	switch decoratorName {
	case "parallel":
		// For parallel decorators, mark operations as potentially concurrent
		for i := range operations {
			if operations[i].CanParallelize {
				operations[i].EstimatedTime = operations[i].EstimatedTime / 2 // Assume 50% time reduction from parallelization
			}
		}
		
	case "retry":
		// For retry decorators, multiply resource usage by attempt count
		multiplier := 1
		if mult, ok := analysis["resource_multiplier"].(int); ok {
			multiplier = mult
		}
		for i := range operations {
			operations[i].ResourceUsage *= multiplier
			operations[i].EstimatedTime = (operations[i].EstimatedTime * int64(multiplier)) / 2 // Assume some operations succeed earlier
		}
		
	case "timeout":
		// For timeout decorators, cap the estimated time
		if maxTime, ok := analysis["max_time"].(int64); ok && maxTime > 0 {
			for i := range operations {
				if operations[i].EstimatedTime > maxTime {
					operations[i].EstimatedTime = maxTime
				}
			}
		}
	}
	
	return operations
}

// generateSequenceKey generates a cache key for a command sequence
func (ao *ASTOptimizer) generateSequenceKey(content []ast.CommandContent) string {
	// Simple hash based on content structure
	var builder strings.Builder
	for i, cmd := range content {
		if i > 0 {
			builder.WriteString("|")
		}
		switch c := cmd.(type) {
		case *ast.ShellContent:
			builder.WriteString("shell:")
			for _, part := range c.Parts {
				if text, ok := part.(*ast.TextPart); ok {
					builder.WriteString(text.Text)
				}
			}
		case *ast.BlockDecorator:
			builder.WriteString("block:" + c.Name)
		}
	}
	return builder.String()
}

// GetOptimizationRecommendations provides optimization recommendations for a command sequence
func (ao *ASTOptimizer) GetOptimizationRecommendations(sequence *FlattenedCommandSequence) []string {
	recommendations := make([]string, 0)
	
	// Check for parallelization opportunities
	parallelizableCount := 0
	totalOperations := len(sequence.Commands)
	
	for _, op := range sequence.Commands {
		if op.CanParallelize {
			parallelizableCount++
		}
	}
	
	if parallelizableCount > 1 && float64(parallelizableCount)/float64(totalOperations) > 0.5 {
		recommendations = append(recommendations, 
			fmt.Sprintf("Consider using @parallel decorator - %d out of %d operations can be parallelized", 
				parallelizableCount, totalOperations))
	}
	
	// Check for timeout opportunities
	totalTime := sequence.EstimatedTotalTime
	if totalTime > 60000 { // More than 1 minute
		recommendations = append(recommendations, 
			fmt.Sprintf("Consider using @timeout decorator - estimated execution time is %d seconds", 
				totalTime/1000))
	}
	
	// Check for retry opportunities
	highResourceOps := 0
	for _, op := range sequence.Commands {
		if op.ResourceUsage > 7 {
			highResourceOps++
		}
	}
	
	if highResourceOps > 0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("Consider using @retry decorator for %d high-resource operations that might fail", 
				highResourceOps))
	}
	
	return recommendations
}

// Global AST optimizer instance
var globalASTOptimizer = NewASTOptimizer()

// GetASTOptimizer returns the global AST optimizer
func GetASTOptimizer() *ASTOptimizer {
	return globalASTOptimizer
}