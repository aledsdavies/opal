package lexer

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/core/types"
)

// Performance targets for instant feel
const (
	// Target: Process 5K lines in under 1ms
	MaxLinesTarget = 5000
	MaxLexingTime  = 1 * time.Millisecond
	MaxParsingTime = 2 * time.Millisecond // Including lexing + parsing

	// For comparison: Go compiler processes ~10K lines/ms
	// We want to be competitive with Go's speed
	LinesPerMsTarget = 5000
)

// generateLargeDevcmdFile creates a realistic devcmd file with the specified number of lines
func generateLargeDevcmdFile(lines int) string {
	var builder strings.Builder

	// Generate realistic devcmd content
	patterns := []string{
		"var PORT = 8080",
		"var HOST = \"localhost\"",
		"var DB_URL = \"postgres://localhost/myapp\"",
		"",
		"build: @timeout(30s) {",
		"    npm install",
		"    npm run build",
		"    npm run test",
		"}",
		"",
		"dev: @parallel {",
		"    @workdir(\"frontend\") { npm run dev }",
		"    @workdir(\"backend\") { go run main.go }",
		"}",
		"",
		"deploy: @confirm(\"Deploy to production?\") {",
		"    docker build -t myapp .",
		"    docker push myapp:latest",
		"    kubectl apply -f k8s/",
		"}",
		"",
		"test: @retry(3) {",
		"    go test ./...",
		"    npm test",
		"}",
		"",
		"clean: @log(\"Cleaning up...\") && rm -rf dist/",
		"",
		"watch-files: @when($ENV) {",
		"    development: @cmd(nodemon)",
		"    production: @cmd(pm2)",
		"    *: echo \"Unknown environment\"",
		"}",
		"",
		"handle-errors: @try {",
		"    main: ./deploy.sh",
		"    catch: ./rollback.sh",
		"    finally: ./cleanup.sh",
		"}",
	}

	currentLine := 0
	for currentLine < lines {
		for _, pattern := range patterns {
			if currentLine >= lines {
				break
			}
			builder.WriteString(pattern)
			builder.WriteString("\n")
			currentLine++
		}
	}

	return builder.String()
}

func TestPerformanceTargets(t *testing.T) {
	tests := []struct {
		name    string
		lines   int
		maxTime time.Duration
	}{
		{"Small file (100 lines)", 100, 50 * time.Microsecond},
		{"Medium file (1K lines)", 1000, 200 * time.Microsecond},
		{"Large file (5K lines)", 5000, MaxLexingTime},
		{"Very large file (10K lines)", 10000, 2 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := generateLargeDevcmdFile(tt.lines)

			start := time.Now()
			processor := NewProcessor(strings.NewReader(input))
			tokens := processor.AllTokens()
			elapsed := time.Since(start)

			if elapsed > tt.maxTime {
				t.Errorf("Lexing %d lines took %v, expected under %v (%.1fx slower than target)",
					tt.lines, elapsed, tt.maxTime, float64(elapsed)/float64(tt.maxTime))
			}

			// Verify we got reasonable number of tokens
			expectedMinTokens := tt.lines / 2 // Very conservative estimate
			if len(tokens) < expectedMinTokens {
				t.Errorf("Got %d tokens for %d lines, expected at least %d",
					len(tokens), tt.lines, expectedMinTokens)
			}

			// Calculate throughput
			linesPerMs := float64(tt.lines) / float64(elapsed.Nanoseconds()) * 1e6
			t.Logf("Processed %d lines in %v (%.0f lines/ms, target: %d lines/ms)",
				tt.lines, elapsed, linesPerMs, LinesPerMsTarget)

			if linesPerMs < float64(LinesPerMsTarget) {
				t.Logf("WARNING: Below target throughput (%.0f < %d lines/ms)",
					linesPerMs, LinesPerMsTarget)
			}
		})
	}
}

func BenchmarkLexingThroughput(b *testing.B) {
	sizes := []int{100, 500, 1000, 2000, 5000}

	for _, size := range sizes {
		input := generateLargeDevcmdFile(size)

		b.Run(fmt.Sprintf("%dlines", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				processor := NewProcessor(strings.NewReader(input))
				tokens := processor.AllTokens()
				_ = tokens
			}

			// Calculate and report throughput
			linesPerOp := float64(size)
			nsPerOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
			linesPerMs := linesPerOp / (nsPerOp / 1e6)

			b.ReportMetric(linesPerMs, "lines/ms")
			b.ReportMetric(float64(size), "lines")
		})
	}
}

func BenchmarkProcessorVsLegacyThroughput(b *testing.B) {
	input := generateLargeDevcmdFile(5000) // 5K line test

	b.Run("Legacy", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			lexer := New(strings.NewReader(input))
			tokens := lexer.TokenizeToSlice()
			_ = tokens
		}
	})

	b.Run("Processor", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			processor := NewProcessor(strings.NewReader(input))
			tokens := processor.AllTokens()
			_ = tokens
		}
	})
}

func BenchmarkStreamingVsBatch(b *testing.B) {
	input := generateLargeDevcmdFile(5000)

	b.Run("BatchAllTokens", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			processor := NewProcessor(strings.NewReader(input))
			tokens := processor.AllTokens()
			_ = tokens
		}
	})

	b.Run("StreamingNextToken", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			processor := NewProcessor(strings.NewReader(input))
			var tokens []types.Token

			for {
				token := processor.NextToken()
				tokens = append(tokens, token)
				if token.Type == types.EOF {
					break
				}
			}
			_ = tokens
		}
	})
}

func BenchmarkMemoryEfficiency(b *testing.B) {
	input := generateLargeDevcmdFile(5000)

	b.Run("LexingAllocations", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			processor := NewProcessor(strings.NewReader(input))
			tokens := processor.AllTokens()
			_ = tokens
		}

		// Target: < 1MB of allocations for 5K lines
		if b.N > 0 {
			totalBytes := testing.AllocsPerRun(1, func() {
				processor := NewProcessor(strings.NewReader(input))
				tokens := processor.AllTokens()
				_ = tokens
			})

			// Estimate bytes per allocation (rough calculation)
			bytesPerOp := totalBytes * 64 // Rough estimate of average allocation size
			if bytesPerOp > 1024*1024 {   // 1MB
				b.Logf("WARNING: High memory usage: ~%.1f KB/op (target: <1MB)", bytesPerOp/1024)
			} else {
				b.Logf("Memory usage: ~%.1f KB/op (target: <1MB)", bytesPerOp/1024)
			}
		}
	})
}

// Test real-world scenarios
func TestRealWorldPerformance(t *testing.T) {
	realWorldExamples := map[string]string{
		"Complex CI Pipeline": `
var DOCKER_IMAGE = "myapp:latest"
var KUBE_NAMESPACE = "production"

build: @timeout(300s) {
    @log("Building application...")
    docker build -t @var(DOCKER_IMAGE) .
    @log("Build complete")
}

test: @parallel(concurrency=4) {
    @workdir("backend") { go test ./... }
    @workdir("frontend") { npm test }
    @log("Integration tests") && npm run test:e2e
    docker run --rm @var(DOCKER_IMAGE) /app/health-check
}

deploy: @confirm("Deploy to production?") {
    @retry(3) {
        kubectl apply -f k8s/ --namespace=@var(KUBE_NAMESPACE)
    }
    @when($DEPLOYMENT_TYPE) {
        rolling: kubectl rollout status deployment/app
        blue-green: ./scripts/blue-green-deploy.sh
        *: echo "Unknown deployment type"
    }
}

rollback: @try {
    main: kubectl rollout undo deployment/app
    catch: @log("Rollback failed, manual intervention required", level="error")
    finally: @log("Rollback attempt completed")
}`,

		"Microservices Development": `
var SERVICES = "auth,api,worker,frontend"

dev: @parallel {
    @workdir("services/auth") { go run main.go --port=8001 }
    @workdir("services/api") { go run main.go --port=8002 }
    @workdir("services/worker") { go run main.go --port=8003 }
    @workdir("frontend") { npm run dev --port=3000 }
}

build-all: @parallel(concurrency=2) {
    @cmd(docker build -t auth services/auth)
    @cmd(docker build -t api services/api)
    @cmd(docker build -t worker services/worker)
    @cmd(docker build -t frontend frontend)
}

test-all: @timeout(600s) {
    @parallel {
        @workdir("services/auth") { go test ./... }
        @workdir("services/api") { go test ./... }
        @workdir("services/worker") { go test ./... }
        @workdir("frontend") { npm test }
    }
    @log("Integration tests") && docker-compose -f test/docker-compose.yml up --abort-on-container-exit
}`,
	}

	for name, content := range realWorldExamples {
		t.Run(name, func(t *testing.T) {
			lines := strings.Count(content, "\n")

			start := time.Now()
			processor := NewProcessor(strings.NewReader(content))
			tokens := processor.AllTokens()
			elapsed := time.Since(start)

			// Should be very fast for real-world files
			maxTime := 500 * time.Microsecond
			if elapsed > maxTime {
				t.Errorf("Real-world example '%s' (%d lines) took %v, expected under %v",
					name, lines, elapsed, maxTime)
			}

			// Verify reasonable token count
			if len(tokens) < lines {
				t.Errorf("Got %d tokens for %d lines, seems too low", len(tokens), lines)
			}

			t.Logf("Processed '%s': %d lines in %v (%d tokens)",
				name, lines, elapsed, len(tokens))
		})
	}
}

// Stress test with pathological cases
func TestPathologicalCases(t *testing.T) {
	cases := map[string]string{
		"Deeply nested decorators": strings.Repeat("@timeout(30s) { @retry(3) { @parallel { @confirm(\"OK?\") { ", 100) +
			"echo test" + strings.Repeat(" } } } }", 100),

		"Many variables": func() string {
			var b strings.Builder
			for i := 0; i < 1000; i++ {
				b.WriteString(fmt.Sprintf("var VAR_%d = \"value_%d\"\n", i, i))
			}
			return b.String()
		}(),

		"Long shell commands": strings.Repeat("build: npm install && npm run build && npm run test && docker build . && kubectl apply -f k8s/ && ", 200) + "echo done",

		"Complex pattern matching": func() string {
			var b strings.Builder
			b.WriteString("deploy: @when($ENV) {\n")
			for i := 0; i < 500; i++ {
				b.WriteString(fmt.Sprintf("    env_%d: ./deploy_env_%d.sh\n", i, i))
			}
			b.WriteString("    *: echo default\n}")
			return b.String()
		}(),
	}

	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			processor := NewProcessor(strings.NewReader(content))
			tokens := processor.AllTokens()
			elapsed := time.Since(start)

			// Even pathological cases should be fast
			maxTime := 2 * time.Millisecond
			if elapsed > maxTime {
				t.Errorf("Pathological case '%s' took %v, expected under %v", name, elapsed, maxTime)
			}

			t.Logf("Pathological case '%s': %v (%d tokens)", name, elapsed, len(tokens))
		})
	}
}
