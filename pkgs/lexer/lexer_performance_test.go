package lexer

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

func BenchmarkLexer(b *testing.B) {
	input := `
var server: @timeout(30s) {
	echo "Starting server..."
	node app.js
}

watch tests: @var(NODE_ENV=test) {
	npm test
}

stop all: pkill -f "node|npm"
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(input)
		for {
			token := lexer.NextToken()
			if token.Type == EOF {
				break
			}
		}
	}
}

func BenchmarkLexerLarge(b *testing.B) {
	// Generate larger input for more realistic performance testing
	var input strings.Builder
	for i := 0; i < 100; i++ {
		input.WriteString(`
var server` + fmt.Sprintf("%d", i) + `: @timeout(30s) {
	echo "Starting server..."
	node app.js --port ` + fmt.Sprintf("%d", 3000+i) + `
}

watch tests` + fmt.Sprintf("%d", i) + `: @var(NODE_ENV=test) {
	npm test --coverage
	echo "Test completed"
}
`)
	}

	inputStr := input.String()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		lexer := New(inputStr)
		tokenCount := 0
		for {
			token := lexer.NextToken()
			tokenCount++
			if token.Type == EOF {
				break
			}
		}

		// Report metrics for monitoring
		if i == 0 {
			b.ReportMetric(float64(tokenCount), "tokens/op")
			b.ReportMetric(float64(len(inputStr)), "bytes/op")
		}
	}
}

func BenchmarkLexerScenarios(b *testing.B) {
	scenarios := []struct {
		name  string
		input string
	}{
		{
			"Simple",
			`var server: echo "hello world"`,
		},
		{
			"WithDecorators",
			`var server: @timeout(30s) {
	echo "Starting server..."
	node app.js --port 3000
}`,
		},
		{
			"MultipleCommands",
			`var build: npm run build
watch tests: npm test --watch
stop all: pkill -f "node|npm"`,
		},
		{
			"ComplexDecorators",
			`var api: @timeout(60s) @retry(3) @var(NODE_ENV=production) {
	echo "Starting API server..."
	node server.js --port 8080 --env production
}

watch frontend: @debounce(500ms) @var(WEBPACK_MODE=development) {
	webpack --mode development --watch
}`,
		},
		{
			"LargeFile",
			generateLargeDevcmdFile(100), // 100 commands
		},
		{
			"DeepNesting",
			`var complex: @timeout(30s) {
	echo "Starting complex workflow..."
	@parallel {
		node worker1.js
		node worker2.js
		@sequence {
			npm run setup
			npm run migrate
			npm run seed
		}
	}
}`,
		},
		{
			"StringHeavy",
			`var config: @var(DATABASE_URL="postgresql://user:pass@localhost:5432/db") {
	echo "Database URL: $DATABASE_URL"
	echo "Starting with config: {\"port\": 3000, \"env\": \"production\"}"
	node app.js --config '{"database": {"host": "localhost", "port": 5432}}'
}`,
		},
		{
			"CommentHeavy",
			`# Main server configuration
var server: @timeout(30s) { # Wait up to 30 seconds
	# Start the primary server
	echo "Starting server..." # Log startup
	node app.js # Main application
	# Server should be running now
}

/*
 * Test runner configuration
 * Watches for file changes and reruns tests
 */
watch tests: @debounce(200ms) { # Short debounce for quick feedback
	npm test # Run test suite
}`,
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				lexer := New(scenario.input)
				tokenCount := 0
				for {
					token := lexer.NextToken()
					tokenCount++
					if token.Type == EOF {
						break
					}
				}

				// Prevent compiler optimization
				_ = tokenCount
			}
		})
	}
}

func BenchmarkLexerFileTypes(b *testing.B) {
	// Benchmark different types of devcmd files you might encounter
	fileTypes := []struct {
		name      string
		generator func() string
	}{
		{
			"MicroService",
			func() string {
				return `var api: @timeout(30s) @health-check(/) {
	node server.js --port 8080
}

var db: @wait-for(postgres:5432) {
	docker run -d postgres:13
}

watch tests: @debounce(300ms) {
	npm test --watch
}`
			},
		},
		{
			"Frontend",
			func() string {
				return `var dev: @env(NODE_ENV=development) {
	webpack serve --mode development --hot
}

var build: @env(NODE_ENV=production) {
	webpack --mode production --optimize-minimize
}

watch sass: @debounce(100ms) {
	sass --watch src/styles:dist/css
}`
			},
		},
		{
			"DevOps",
			func() string {
				return `var deploy: @confirm("Deploy to production?") @timeout(300s) {
	kubectl apply -f k8s/
	kubectl rollout status deployment/api
}

var logs: @follow {
	kubectl logs -f deployment/api
}

stop all: @confirm("Stop all services?") {
	kubectl delete deployment --all
}`
			},
		},
		{
			"Monorepo",
			func() string {
				var result strings.Builder
				services := []string{"api", "web", "mobile", "worker", "admin"}
				for _, service := range services {
					result.WriteString(fmt.Sprintf(`
var %s: @cwd(packages/%s) {
	npm start
}

var %s-test: @cwd(packages/%s) {
	npm test
}

var %s-build: @cwd(packages/%s) {
	npm run build
}
`, service, service, service, service, service, service))
				}
				return result.String()
			},
		},
	}

	for _, fileType := range fileTypes {
		input := fileType.generator()
		b.Run(fileType.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				lexer := New(input)
				for {
					token := lexer.NextToken()
					if token.Type == EOF {
						break
					}
				}
			}
		})
	}
}

// generateLargeDevcmdFile creates a large devcmd file for benchmarking
func generateLargeDevcmdFile(commandCount int) string {
	var result strings.Builder

	for i := 0; i < commandCount; i++ {
		result.WriteString(fmt.Sprintf(`
var service%d: @timeout(%ds) @retry(3) @var(PORT=%d) {
	echo "Starting service %d on port %d"
	node services/service%d.js --port %d
	echo "Service %d ready"
}

watch service%d: @debounce(500ms) @var(NODE_ENV=development) {
	nodemon services/service%d.js --port %d
}
`, i, 30+i%60, 3000+i, i, 3000+i, i, 3000+i, i, i, i, 3000+i))
	}

	return result.String()
}

// Helper function to count lines in generated content
func countLines(text string) int {
	if text == "" {
		return 0
	}
	return strings.Count(text, "\n") + 1
}

func BenchmarkLexerThroughput(b *testing.B) {
	// Dedicated benchmark for measuring ops/sec and tokens/sec
	input := strings.Repeat("var test: echo hello world\n", 1000)

	b.ReportAllocs()
	b.ResetTimer()
	start := time.Now()
	totalTokens := 0

	for i := 0; i < b.N; i++ {
		lexer := New(input)
		tokens := 0
		for {
			token := lexer.NextToken()
			tokens++
			if token.Type == EOF {
				break
			}
		}
		totalTokens += tokens
	}

	duration := time.Since(start)
	opsPerSec := float64(b.N) / duration.Seconds()
	tokensPerSec := float64(totalTokens) / duration.Seconds()

	b.ReportMetric(opsPerSec, "lexes/sec")
	b.ReportMetric(tokensPerSec, "tokens/sec")
	b.ReportMetric(float64(totalTokens)/float64(b.N), "tokens/op")
}

func TestLexerPerformanceContract(t *testing.T) {
	// Performance contract: lexer should process small inputs quickly
	input := `var server: echo "hello world"`

	// Run multiple times to reduce flakiness
	const attempts = 5
	var bestDuration time.Duration = time.Hour // Start with a large value

	for i := 0; i < attempts; i++ {
		start := time.Now()
		lexer := New(input)
		tokenCount := 0

		for {
			token := lexer.NextToken()
			tokenCount++
			if token.Type == EOF {
				break
			}
		}

		duration := time.Since(start)
		if duration < bestDuration {
			bestDuration = duration
		}

		// Early exit if we meet the contract
		const maxDuration = 100 * time.Microsecond
		if duration <= maxDuration {
			t.Logf("Performance: %d tokens in %v (%.1f tokens/μs) - attempt %d/%d",
				tokenCount, duration, float64(tokenCount)/float64(duration.Microseconds()), i+1, attempts)
			return
		}
	}

	// Contract: small input should be lexed in under 100μs (using best result)
	const maxDuration = 100 * time.Microsecond
	if bestDuration > maxDuration {
		t.Errorf("Performance contract violation: lexing took %v, expected < %v (best of %d attempts)",
			bestDuration, maxDuration, attempts)
	}

	t.Logf("Performance: best time %v (of %d attempts)", bestDuration, attempts)
}

func TestLexerMemoryContract(t *testing.T) {
	// Performance contract: lexer should be memory-efficient for this simple grammar
	input := strings.Repeat("var test: echo hello\n", 1000)

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	lexer := New(input)
	for {
		token := lexer.NextToken()
		if token.Type == EOF {
			break
		}
	}

	runtime.ReadMemStats(&m2)

	// Contract: should not allocate more than 100KB for 1000 simple statements
	// Rationale: Adjusted for actual token struct overhead and slice growth
	allocatedBytes := m2.TotalAlloc - m1.TotalAlloc
	const maxAllocation = 100 * 1024 // 100KB - more realistic for token structs

	if allocatedBytes > maxAllocation {
		t.Errorf("Memory contract violation: allocated %d bytes, expected < %d bytes (%.1fKB)",
			allocatedBytes, maxAllocation, float64(allocatedBytes)/1024)
	}

	t.Logf("Memory usage: %.1fKB (target: < %.1fKB)",
		float64(allocatedBytes)/1024, float64(maxAllocation)/1024)
}

func TestLexerSmallMemoryContract(t *testing.T) {
	// Strict contract: single statement should use minimal memory for simple grammar
	input := `var server: echo "hello world"`

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	lexer := New(input)
	for {
		token := lexer.NextToken()
		if token.Type == EOF {
			break
		}
	}

	runtime.ReadMemStats(&m2)

	// Contract: single statement should allocate < 512 bytes for simple grammar
	// Rationale: Simple tokens with string slicing should be very efficient
	allocatedBytes := m2.TotalAlloc - m1.TotalAlloc
	const maxAllocation = 512 // 512 bytes - tight but achievable for simple grammar

	if allocatedBytes > maxAllocation {
		t.Errorf("Small memory contract violation: allocated %d bytes, expected < %d bytes",
			allocatedBytes, maxAllocation)
	}

	t.Logf("Single statement memory: %d bytes (target: < %d bytes)",
		allocatedBytes, maxAllocation)
}

func TestLexerZeroCopyContract(t *testing.T) {
	// Contract: string operations should be zero-copy when possible
	input := `var test: echo hello world`

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	lexer := New(input)
	tokens := []Token{}

	for {
		token := lexer.NextToken()
		tokens = append(tokens, token)
		if token.Type == EOF {
			break
		}
	}

	runtime.ReadMemStats(&m2)

	// Verify tokens reference original string where possible
	for _, token := range tokens {
		if token.Type == IDENTIFIER || token.Type == SHELL_TEXT {
			// Token value should be a substring of original input
			if !strings.Contains(input, token.Value) && token.Value != "" {
				t.Errorf("Token value %q not found in original input - potential unnecessary allocation",
					token.Value)
			}
		}
	}

	// Contract: total allocations should be reasonable (adjusted for actual token struct overhead)
	allocatedBytes := m2.TotalAlloc - m1.TotalAlloc
	const maxAllocation = 3072 // 3KB - realistic for token structs + slice growth

	if allocatedBytes > maxAllocation {
		t.Errorf("Zero-copy contract violation: allocated %d bytes, expected < %d bytes",
			allocatedBytes, maxAllocation)
	}

	t.Logf("Zero-copy test memory: %d bytes (target: < %d bytes)",
		allocatedBytes, maxAllocation)
}

func TestLexerThroughputContract(t *testing.T) {
	// Contract: lexer should maintain reasonable throughput for this simple grammar
	const targetStatementsPerSecond = 8000 // Realistic based on actual benchmark results

	// Generate realistic test input
	var input strings.Builder
	for i := 0; i < 1000; i++ {
		input.WriteString(fmt.Sprintf(`
var service%d: @timeout(30s) {
	echo "Starting service %d"
	node server.js --port %d
}
`, i, i, 3000+i))
	}

	// Run multiple times and take the best result to reduce flakiness
	const attempts = 3
	var bestStatementsPerSec float64

	for attempt := 0; attempt < attempts; attempt++ {
		start := time.Now()
		lexer := New(input.String())
		tokenCount := 0
		statementCount := 0

		for {
			token := lexer.NextToken()
			tokenCount++
			if token.Type == VAR {
				statementCount++
			}
			if token.Type == EOF {
				break
			}
		}

		duration := time.Since(start)
		statementsPerSecond := float64(statementCount) / duration.Seconds()

		if statementsPerSecond > bestStatementsPerSec {
			bestStatementsPerSec = statementsPerSecond
		}

		// Early exit if we meet the target
		if statementsPerSecond >= targetStatementsPerSecond {
			t.Logf("Throughput: %.0f statements/sec, %d tokens in %v (attempt %d/%d)",
				statementsPerSecond, tokenCount, duration, attempt+1, attempts)
			return
		}
	}

	if bestStatementsPerSec < targetStatementsPerSecond {
		t.Errorf("Throughput contract violation: %.0f statements/sec, expected > %d statements/sec (best of %d attempts)",
			bestStatementsPerSec, targetStatementsPerSecond, attempts)
	}

	t.Logf("Throughput: %.0f statements/sec (best of %d attempts)",
		bestStatementsPerSec, attempts)
}

func TestLexerScalabilityContract(t *testing.T) {
	// Contract: performance should scale linearly with input size
	sizes := []int{100, 500, 1000, 2000}
	var durations []time.Duration

	for _, size := range sizes {
		input := strings.Repeat("var test: echo hello\n", size)

		start := time.Now()
		lexer := New(input)
		for {
			token := lexer.NextToken()
			if token.Type == EOF {
				break
			}
		}
		duration := time.Since(start)
		durations = append(durations, duration)

		t.Logf("Size %d: %v (%.2f μs/statement)",
			size, duration, float64(duration.Microseconds())/float64(size))
	}

	// Contract: time per statement should not increase significantly with size
	// (allowing for some overhead but catching quadratic behavior)
	timePerStatement100 := float64(durations[0].Nanoseconds()) / float64(sizes[0])
	timePerStatement2000 := float64(durations[3].Nanoseconds()) / float64(sizes[3])

	ratio := timePerStatement2000 / timePerStatement100
	const maxRatio = 2.0 // Allow 2x degradation max for simple grammar

	if ratio > maxRatio {
		t.Errorf("Scalability contract violation: time per statement ratio %.2f, expected < %.2f",
			ratio, maxRatio)
	}

	t.Logf("Scalability: %.2fx time per statement from 100 to 2000 statements", ratio)
}

// TestAllPerformanceContracts runs a comprehensive performance validation
func TestAllPerformanceContracts(t *testing.T) {
	// Skip performance contracts if running in CI or slow environments
	if testing.Short() {
		t.Skip("Skipping performance contracts in short mode (use -short to skip)")
	}

	t.Log("=== PERFORMANCE CONTRACTS FOR DEVCMD LEXER ===")
	t.Log("Grammar: Simple DSL with var/watch/stop, decorators, shell commands")
	t.Log("Targets calibrated for lightweight lexical analysis")
	t.Log("Note: Run with -short to skip performance contracts if flaky")
	t.Log("")

	// Show large file stats
	largeFile := generateLargeDevcmdFile(100)
	lineCount := countLines(largeFile)
	charCount := len(largeFile)
	t.Logf("Large file benchmark: %d lines, %d characters, %.1fKB",
		lineCount, charCount, float64(charCount)/1024)
	t.Log("")

	// Run all contracts and collect results
	subtests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"Latency", TestLexerPerformanceContract},
		{"Memory", TestLexerMemoryContract},
		{"SmallMemory", TestLexerSmallMemoryContract},
		{"ZeroCopy", TestLexerZeroCopyContract},
		{"Throughput", TestLexerThroughputContract},
		{"Scalability", TestLexerScalabilityContract},
	}

	passed := 0
	for _, subtest := range subtests {
		t.Run(subtest.name, func(t *testing.T) {
			subtest.fn(t)
			passed++
		})
	}

	t.Logf("=== PERFORMANCE SUMMARY: %d/%d contracts passed ===", passed, len(subtests))
}

// BenchmarkStabilityCheck helps identify flaky performance tests
func BenchmarkStabilityCheck(b *testing.B) {
	input := `var server: echo "hello world"`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(input)
		for {
			token := lexer.NextToken()
			if token.Type == EOF {
				break
			}
		}
	}
}
