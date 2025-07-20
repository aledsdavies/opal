package engine

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/pkgs/decorators"
	"github.com/aledsdavies/devcmd/pkgs/parser"
)

// BenchmarkExecutionEngine_SmallConfig benchmarks execution of a small configuration
func BenchmarkExecutionEngine_SmallConfig(b *testing.B) {
	content := `
var PORT = "3000"
var ENV = "development"

build: go build -o app ./cmd
test: go test ./...
serve: python -m http.server @var(PORT)
clean: rm -rf dist/
`

	program, err := parser.Parse(strings.NewReader(content))
	if err != nil {
		b.Fatalf("Failed to parse content: %v", err)
	}

	b.ResetTimer()
	b.Run("interpreter_mode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(InterpreterMode, ctx)
			_, err := engine.Execute(program)
			if err != nil {
				b.Fatalf("Execution failed: %v", err)
			}
		}
	})

	b.Run("generator_mode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(GeneratorMode, ctx)
			_, err := engine.Execute(program)
			if err != nil {
				b.Fatalf("Code generation failed: %v", err)
			}
		}
	})
}

// BenchmarkExecutionEngine_MediumConfig benchmarks execution of a medium-sized configuration
func BenchmarkExecutionEngine_MediumConfig(b *testing.B) {
	var content strings.Builder

	// Generate medium-sized config (20 variables, 50 commands)
	for i := 0; i < 20; i++ {
		content.WriteString(fmt.Sprintf("var VAR%d = \"value%d\"\n", i, i))
	}

	for i := 0; i < 50; i++ {
		switch i % 3 {
		case 0:
			content.WriteString(fmt.Sprintf("cmd%d: echo \"Command %d with @var(VAR%d)\"\n", i, i, i%20))
		case 1:
			content.WriteString(fmt.Sprintf("cmd%d: @timeout(duration=30s) { echo \"Timeout command %d\" }\n", i, i))
		case 2:
			content.WriteString(fmt.Sprintf("cmd%d: @parallel { echo \"Parallel %d\" }\n", i, i))
		}
	}

	program, err := parser.Parse(strings.NewReader(content.String()))
	if err != nil {
		b.Fatalf("Failed to parse content: %v", err)
	}

	b.ResetTimer()
	b.Run("interpreter_mode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(InterpreterMode, ctx)
			_, err := engine.Execute(program)
			if err != nil {
				b.Fatalf("Execution failed: %v", err)
			}
		}
	})

	b.Run("generator_mode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(GeneratorMode, ctx)
			_, err := engine.Execute(program)
			if err != nil {
				b.Fatalf("Code generation failed: %v", err)
			}
		}
	})
}

// BenchmarkExecutionEngine_LargeConfig benchmarks execution of a large configuration
func BenchmarkExecutionEngine_LargeConfig(b *testing.B) {
	var content strings.Builder

	// Generate large config (100 variables, 200 commands)
	for i := 0; i < 100; i++ {
		content.WriteString(fmt.Sprintf("var VAR%d = \"value%d\"\n", i, i))
	}

	for i := 0; i < 200; i++ {
		switch i % 4 {
		case 0:
			content.WriteString(fmt.Sprintf("cmd%d: echo \"Command %d with @var(VAR%d)\"\n", i, i, i%100))
		case 1:
			content.WriteString(fmt.Sprintf("cmd%d: @timeout(duration=30s) { echo \"Timeout command %d\" }\n", i, i))
		case 2:
			content.WriteString(fmt.Sprintf("cmd%d: @parallel { echo \"Parallel %d\" }\n", i, i))
		case 3:
			content.WriteString(fmt.Sprintf("cmd%d: @retry(attempts=2) { echo \"Retry command %d\" }\n", i, i))
		}
	}

	program, err := parser.Parse(strings.NewReader(content.String()))
	if err != nil {
		b.Fatalf("Failed to parse content: %v", err)
	}

	b.ResetTimer()
	b.Run("interpreter_mode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(InterpreterMode, ctx)
			_, err := engine.Execute(program)
			if err != nil {
				b.Fatalf("Execution failed: %v", err)
			}
		}
	})

	b.Run("generator_mode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(GeneratorMode, ctx)
			_, err := engine.Execute(program)
			if err != nil {
				b.Fatalf("Code generation failed: %v", err)
			}
		}
	})
}

// BenchmarkExecutionEngine_DecoratorIntensive benchmarks decorator-heavy configurations
func BenchmarkExecutionEngine_DecoratorIntensive(b *testing.B) {
	content := `
var TIMEOUT = "30s"
var RETRIES = "3"

# Nested decorators
complex1: @retry(attempts=@var(RETRIES), delay=1s) {
	@timeout(duration=@var(TIMEOUT)) {
		@parallel {
			echo "Task 1"
			echo "Task 2"
			echo "Task 3"
		}
	}
}

complex2: @timeout(duration=1m) {
	@retry(attempts=2) {
		@parallel {
			make build
			make test
			make lint
		}
	}
}

complex3: @when(condition="ENV=prod") {
	@timeout(duration=5m) {
		@parallel {
			docker build -t app:latest .
			docker push app:latest
		}
	}
}

complex4: @try {
	@parallel {
		@timeout(duration=30s) { make frontend }
		@timeout(duration=45s) { make backend }
		@timeout(duration=60s) { make integration-tests }
	}
}
`

	program, err := parser.Parse(strings.NewReader(content))
	if err != nil {
		b.Fatalf("Failed to parse content: %v", err)
	}

	b.ResetTimer()
	b.Run("interpreter_mode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(InterpreterMode, ctx)
			_, err := engine.Execute(program)
			if err != nil {
				b.Fatalf("Execution failed: %v", err)
			}
		}
	})

	b.Run("generator_mode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(GeneratorMode, ctx)
			_, err := engine.Execute(program)
			if err != nil {
				b.Fatalf("Code generation failed: %v", err)
			}
		}
	})
}

// BenchmarkExecutionEngine_ParsingOverhead measures parsing vs execution overhead
func BenchmarkExecutionEngine_ParsingOverhead(b *testing.B) {
	content := `
var USER = "admin"
var PORT = "8080"

build: go build -o app ./cmd
test: go test ./...
serve: python -m http.server @var(PORT)
deploy: kubectl apply -f k8s/
clean: rm -rf dist/
`

	b.Run("parse_only", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := parser.Parse(strings.NewReader(content))
			if err != nil {
				b.Fatalf("Parse failed: %v", err)
			}
		}
	})

	b.Run("parse_and_execute", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			program, err := parser.Parse(strings.NewReader(content))
			if err != nil {
				b.Fatalf("Parse failed: %v", err)
			}

			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(InterpreterMode, ctx)
			_, err = engine.Execute(program)
			if err != nil {
				b.Fatalf("Execution failed: %v", err)
			}
		}
	})

	// Pre-parsed benchmark
	program, err := parser.Parse(strings.NewReader(content))
	if err != nil {
		b.Fatalf("Failed to parse content: %v", err)
	}

	b.Run("execute_only_preparsed", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(InterpreterMode, ctx)
			_, err := engine.Execute(program)
			if err != nil {
				b.Fatalf("Execution failed: %v", err)
			}
		}
	})
}

// BenchmarkExecutionEngine_MemoryUsage measures memory allocation patterns
func BenchmarkExecutionEngine_MemoryUsage(b *testing.B) {
	content := `
var BASE_URL = "https://api.example.com"
var VERSION = "v1.2.3"

test_api: curl -f @var(BASE_URL)/health
deploy: @timeout(duration=5m) {
	docker build -t app:@var(VERSION) .
	docker push app:@var(VERSION)
	kubectl set image deployment/app app=app:@var(VERSION)
}
monitor: @parallel {
	kubectl logs -f deployment/app
	watch kubectl get pods
}
`

	program, err := parser.Parse(strings.NewReader(content))
	if err != nil {
		b.Fatalf("Failed to parse content: %v", err)
	}

	b.ResetTimer()
	b.Run("memory_allocation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(InterpreterMode, ctx)
			_, err := engine.Execute(program)
			if err != nil {
				b.Fatalf("Execution failed: %v", err)
			}
		}
	})
}

// BenchmarkExecutionEngine_ConcurrentExecution benchmarks concurrent engine usage
func BenchmarkExecutionEngine_ConcurrentExecution(b *testing.B) {
	content := `
var WORKER_ID = "worker"

task: echo "Processing @var(WORKER_ID)"
parallel_task: @parallel {
	echo "Subtask 1 for @var(WORKER_ID)"
	echo "Subtask 2 for @var(WORKER_ID)"
}
`

	program, err := parser.Parse(strings.NewReader(content))
	if err != nil {
		b.Fatalf("Failed to parse content: %v", err)
	}

	b.ResetTimer()
	b.Run("concurrent_execution", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ctx := decorators.NewExecutionContext(context.Background(), program)
				engine := New(InterpreterMode, ctx)
				_, err := engine.Execute(program)
				if err != nil {
					b.Fatalf("Execution failed: %v", err)
				}
			}
		})
	})
}

// BenchmarkExecutionEngine_GeneratedCodeSize measures generated code size and complexity
func BenchmarkExecutionEngine_GeneratedCodeSize(b *testing.B) {
	sizes := []struct {
		name     string
		varCount int
		cmdCount int
	}{
		{"small", 5, 10},
		{"medium", 25, 50},
		{"large", 100, 200},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			var content strings.Builder

			// Generate variables
			for i := 0; i < size.varCount; i++ {
				content.WriteString(fmt.Sprintf("var VAR%d = \"value%d\"\n", i, i))
			}

			// Generate commands
			for i := 0; i < size.cmdCount; i++ {
				content.WriteString(fmt.Sprintf("cmd%d: echo \"Command %d with @var(VAR%d)\"\n", i, i, i%size.varCount))
			}

			program, err := parser.Parse(strings.NewReader(content.String()))
			if err != nil {
				b.Fatalf("Failed to parse content: %v", err)
			}

			b.ResetTimer()
			b.ReportAllocs()

			var totalCodeSize int64
			for i := 0; i < b.N; i++ {
				ctx := decorators.NewExecutionContext(context.Background(), program)
				engine := New(GeneratorMode, ctx)
				result, err := engine.Execute(program)
				if err != nil {
					b.Fatalf("Code generation failed: %v", err)
				}

				genResult := result.(*GenerationResult)
				totalCodeSize += int64(len(genResult.String()))
			}

			b.ReportMetric(float64(totalCodeSize)/float64(b.N), "bytes/op")
		})
	}
}

// BenchmarkExecutionEngine_RealWorldScenarios benchmarks realistic development scenarios
func BenchmarkExecutionEngine_RealWorldScenarios(b *testing.B) {
	scenarios := map[string]string{
		"web_frontend": `
var NODE_ENV = "development"
var PORT = "3000"

install: npm install
build: npm run build
dev: npm run dev
test: npm test
lint: npm run lint
clean: rm -rf dist/ node_modules/
`,
		"go_backend": `
var GO_VERSION = "1.24"
var BINARY = "server"

deps: go mod download
build: go build -o @var(BINARY) ./cmd
test: go test -v ./...
lint: golangci-lint run
clean: rm -f @var(BINARY)
`,
		"docker_compose": `
var COMPOSE_FILE = "docker-compose.yml"

up: docker-compose -f @var(COMPOSE_FILE) up -d
down: docker-compose -f @var(COMPOSE_FILE) down
logs: docker-compose -f @var(COMPOSE_FILE) logs -f
build: docker-compose -f @var(COMPOSE_FILE) build
`,
		"kubernetes": `
var NAMESPACE = "default"
var IMAGE_TAG = "latest"

deploy: kubectl apply -f k8s/ -n @var(NAMESPACE)
status: kubectl get pods -n @var(NAMESPACE)
logs: kubectl logs -f deployment/app -n @var(NAMESPACE)
scale: kubectl scale deployment/app --replicas=3 -n @var(NAMESPACE)
`,
	}

	for name, content := range scenarios {
		b.Run(name, func(b *testing.B) {
			program, err := parser.Parse(strings.NewReader(content))
			if err != nil {
				b.Fatalf("Failed to parse %s scenario: %v", name, err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx := decorators.NewExecutionContext(context.Background(), program)
				engine := New(InterpreterMode, ctx)
				_, err := engine.Execute(program)
				if err != nil {
					b.Fatalf("%s scenario execution failed: %v", name, err)
				}
			}
		})
	}
}

// TestExecutionEngine_PerformanceRegression tests for performance regressions
func TestExecutionEngine_PerformanceRegression(t *testing.T) {
	content := `
var ENV = "test"
var TIMEOUT = "30s"

# Moderately complex scenario
complex_workflow: @timeout(duration=30s) {
	@parallel {
		echo "Task 1 in @var(ENV)"
		echo "Task 2 in @var(ENV)"
		echo "Task 3 in @var(ENV)"
	}
}
`

	program, err := parser.Parse(strings.NewReader(content))
	if err != nil {
		t.Fatalf("Failed to parse content: %v", err)
	}

	// Performance baseline - execution should complete within reasonable time
	const maxExecutionTime = 100 * time.Millisecond
	const iterations = 10

	start := time.Now()
	for i := 0; i < iterations; i++ {
		ctx := decorators.NewExecutionContext(context.Background(), program)
		engine := New(InterpreterMode, ctx)
		_, err := engine.Execute(program)
		if err != nil {
			t.Fatalf("Execution failed on iteration %d: %v", i, err)
		}
	}
	duration := time.Since(start)

	averageTime := duration / iterations
	if averageTime > maxExecutionTime {
		t.Errorf("Performance regression detected: average execution time %v exceeds baseline %v",
			averageTime, maxExecutionTime)
	}

	t.Logf("Performance test passed: average execution time %v (baseline: %v)",
		averageTime, maxExecutionTime)
}
