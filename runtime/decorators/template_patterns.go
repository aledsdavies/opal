package decorators

// Generic Go code patterns for resource management that any decorator can use
// These are building blocks, not decorator-specific templates

const (
	// ConcurrentExecutionPattern - Generic pattern for running multiple operations concurrently
	// Variables: .MaxConcurrency, .Operations[] (each has .Code)
	ConcurrentExecutionPattern = `{
	semaphore := make(chan struct{}, {{.MaxConcurrency}})
	var wg sync.WaitGroup
	errChan := make(chan error, {{len .Operations}})

	{{range $i, $op := .Operations}}
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		// Acquire semaphore
		semaphore <- struct{}{}
		defer func() { <-semaphore }()

		// Execute operation with panic recovery
		if err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic in operation {{$i}}: %v", r)
				}
			}()
			{{.Code}}
			return nil
		}(); err != nil {
			errChan <- err
			return
		}
		errChan <- nil
	}()
	{{end}}

	// Wait and collect errors
	go func() { wg.Wait(); close(errChan) }()
	var errors []string
	for err := range errChan {
		if err != nil {
			errors = append(errors, err.Error())
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("concurrent execution failed: %s", strings.Join(errors, "; "))
	}
}`

	// TimeoutPattern - Generic timeout wrapper for any operation
	// Variables: .Duration (string), .Operation.Code
	TimeoutPattern = `{
	duration, err := time.ParseDuration({{printf "%q" .Duration}})
	if err != nil {
		return fmt.Errorf("invalid duration '%s': %w", {{printf "%q" .Duration}}, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	done := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic during execution: %v", r)
			}
		}()
		
		// Check for cancellation
		select {
		case <-ctx.Done():
			done <- ctx.Err()
			return
		default:
		}

		if err := func() error {
			{{.Operation.Code}}
			return nil
		}(); err != nil {
			done <- err
			return
		}
		done <- nil
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("operation timed out after %s", duration)
	}
}`

	// RetryPattern - Generic retry wrapper for any operation
	// Variables: .MaxAttempts, .DelayDuration (string), .Operation.Code
	RetryPattern = `{
	maxAttempts := {{.MaxAttempts}}
	delay, err := time.ParseDuration({{printf "%q" .DelayDuration}})
	if err != nil {
		return fmt.Errorf("invalid delay duration '%s': %w", {{printf "%q" .DelayDuration}}, err)
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic in attempt %d: %v", attempt, r)
				}
			}()
			{{.Operation.Code}}
			return nil
		}(); err == nil {
			break
		} else {
			lastErr = err
			if attempt < maxAttempts {
				time.Sleep(delay)
			}
		}
	}
	if lastErr != nil {
		return fmt.Errorf("all %d attempts failed, last error: %w", maxAttempts, lastErr)
	}
}`

	// CancellableOperationPattern - Generic cancellable operation
	// Variables: .Operation.Code
	CancellableOperationPattern = `{
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic during execution: %v", r)
			}
		}()

		select {
		case <-ctx.Done():
			done <- ctx.Err()
			return
		default:
		}

		if err := func() error {
			{{.Operation.Code}}
			return nil
		}(); err != nil {
			done <- err
			return
		}
		done <- nil
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}`

	// SequentialExecutionPattern - Generic sequential execution with early termination
	// Variables: .Operations[] (each has .Code), .StopOnError (bool)
	SequentialExecutionPattern = `{
	{{range $i, $op := .Operations}}
	// Execute operation {{$i}}
	if err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic in operation {{$i}}: %v", r)
			}
		}()
		{{.Code}}
		return nil
	}(); err != nil {
		{{if $.StopOnError}}return fmt.Errorf("operation {{$i}} failed: %w", err){{else}}// Continue on error{{end}}
	}
	{{end}}
}`

	// ConditionalExecutionPattern - Generic conditional execution
	// Variables: .Condition.Code, .ThenOperation.Code, .ElseOperation.Code (optional)
	ConditionalExecutionPattern = `{
	shouldExecute := func() bool {
		{{.Condition.Code}}
	}()

	if shouldExecute {
		if err := func() error {
			{{.ThenOperation.Code}}
			return nil
		}(); err != nil {
			return err
		}
	}{{if .ElseOperation}} else {
		if err := func() error {
			{{.ElseOperation.Code}}
			return nil
		}(); err != nil {
			return err
		}
	}{{end}}
}`

	// ResourceCleanupPattern - Generic resource cleanup with defer
	// Variables: .SetupCode, .Operation.Code, .CleanupCode
	ResourceCleanupPattern = `{
	// Setup resources
	{{.SetupCode}}
	
	// Ensure cleanup
	defer func() {
		{{.CleanupCode}}
	}()

	// Execute operation
	if err := func() error {
		{{.Operation.Code}}
		return nil
	}(); err != nil {
		return err
	}
}`

	// ErrorCollectionPattern - Generic error collection from multiple operations
	// Variables: .Operations[] (each has .Code), .ContinueOnError (bool)
	ErrorCollectionPattern = `{
	var errors []error
	
	{{range $i, $op := .Operations}}
	// Execute operation {{$i}}
	if err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic in operation {{$i}}: %v", r)
			}
		}()
		{{.Code}}
		return nil
	}(); err != nil {
		errors = append(errors, err)
		{{if not $.ContinueOnError}}return err{{end}}
	}
	{{end}}

	if len(errors) > 0 {
		if len(errors) == 1 {
			return errors[0]
		}
		return fmt.Errorf("multiple errors occurred: %v", errors)
	}
}`

	// TryCatchFinallyPattern - Generic try-catch-finally pattern
	// Variables: .MainOperation.Code, .CatchOperation.Code (optional), .FinallyOperation.Code (optional), .HasCatch (bool), .HasFinally (bool)
	TryCatchFinallyPattern = `{
	var tryMainErr error
	var tryCatchErr error
	var tryFinallyErr error

	// Execute main block
	tryMainErr = func() error {
		{{.MainOperation.Code}}
		return nil
	}()

	{{if .HasCatch}}
	// Execute catch block if main failed
	if tryMainErr != nil {
		tryCatchErr = func() error {
			{{.CatchOperation.Code}}
			return nil
		}()
		if tryCatchErr != nil {
			fmt.Fprintf(os.Stderr, "Catch block failed: %v\n", tryCatchErr)
		}
	}
	{{end}}

	{{if .HasFinally}}
	// Always execute finally block regardless of main/catch success
	tryFinallyErr = func() error {
		{{.FinallyOperation.Code}}
		return nil
	}()
	if tryFinallyErr != nil {
		fmt.Fprintf(os.Stderr, "Finally block failed: %v\n", tryFinallyErr)
	}
	{{end}}

	// Return the most significant error: main error takes precedence
	if tryMainErr != nil {
		return fmt.Errorf("main block failed: %w", tryMainErr)
	}
	if tryCatchErr != nil {
		return fmt.Errorf("catch block failed: %w", tryCatchErr)
	}
	if tryFinallyErr != nil {
		return fmt.Errorf("finally block failed: %w", tryFinallyErr)
	}
}`
)

// Common import groups that patterns can reference
var (
	CoreImports       = []string{"fmt"}
	ConcurrencyImports = []string{"sync"}
	TimeImports       = []string{"time"}
	ContextImports    = []string{"context"}
	FileSystemImports = []string{"os"}
	StringImports     = []string{"strings"}
)

// PatternImports maps each pattern to its required standard library imports
var PatternImports = map[string][]string{
	"ConcurrentExecutionPattern":  CombineImports(CoreImports, StringImports, ConcurrencyImports),
	"TimeoutPattern":              CombineImports(ContextImports, CoreImports, TimeImports),
	"RetryPattern":                CombineImports(CoreImports, TimeImports),
	"CancellableOperationPattern": CombineImports(ContextImports, CoreImports),
	"SequentialExecutionPattern":  CombineImports(CoreImports),
	"ConditionalExecutionPattern": CombineImports(CoreImports),
	"ResourceCleanupPattern":      CombineImports(CoreImports),
	"ErrorCollectionPattern":      CombineImports(CoreImports),
	"TryCatchFinallyPattern":      CombineImports(CoreImports, FileSystemImports),
}

// CombineImports merges multiple import slices and deduplicates them
func CombineImports(importGroups ...[]string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, group := range importGroups {
		for _, imp := range group {
			if !seen[imp] {
				seen[imp] = true
				result = append(result, imp)
			}
		}
	}
	
	return result
}

// StandardImportRequirement creates an ImportRequirement with standard library imports only
func StandardImportRequirement(importGroups ...[]string) ImportRequirement {
	return ImportRequirement{
		StandardLibrary: CombineImports(importGroups...),
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// RequiresCore is a helper for decorators that need basic fmt functionality
func RequiresCore() ImportRequirement {
	return StandardImportRequirement(CoreImports)
}

// RequiresConcurrency is a helper for decorators that need concurrency primitives
func RequiresConcurrency() ImportRequirement {
	return StandardImportRequirement(CoreImports, ConcurrencyImports)
}

// RequiresTime is a helper for decorators that need time operations
func RequiresTime() ImportRequirement {
	return StandardImportRequirement(CoreImports, TimeImports)
}

// RequiresContext is a helper for decorators that need context operations
func RequiresContext() ImportRequirement {
	return StandardImportRequirement(CoreImports, ContextImports)
}

// RequiresFileSystem is a helper for decorators that need file system operations
func RequiresFileSystem() ImportRequirement {
	return StandardImportRequirement(CoreImports, FileSystemImports)
}

// RequiresResourceCleanup is a helper for decorators that use ResourceCleanupPattern
func RequiresResourceCleanup() ImportRequirement {
	return StandardImportRequirement(CoreImports)
}

// RequiresTryCatchFinally is a helper for decorators that use TryCatchFinallyPattern
func RequiresTryCatchFinally() ImportRequirement {
	return StandardImportRequirement(CoreImports, FileSystemImports)
}