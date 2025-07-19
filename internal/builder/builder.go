package builder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/generator"
)

// Builder handles compilation of CLI programs to binaries
type Builder struct {
	debug bool
}

// New creates a new builder
func New(debug bool) *Builder {
	return &Builder{
		debug: debug,
	}
}

// BuildBinary compiles a CLI program to a binary
func (b *Builder) BuildBinary(program *ast.Program, templateFile, binaryName, output string) error {
	// Generate Go source code
	goSource, err := b.generateGo(program, templateFile, binaryName)
	if err != nil {
		return fmt.Errorf("error generating Go source: %w", err)
	}

	// Determine output path
	outputPath := output
	if outputPath == "" {
		outputPath = "./" + binaryName
	}
	
	// Make output path absolute
	if !filepath.IsAbs(outputPath) {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting working directory: %w", err)
		}
		outputPath = filepath.Join(wd, outputPath)
	}

	// Create temporary directory for build
	tempDir, err := os.MkdirTemp("", "devcmd-build-*")
	if err != nil {
		return fmt.Errorf("error creating temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write Go source to temp directory
	mainGoPath := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(goSource), 0644); err != nil {
		return fmt.Errorf("error writing Go source: %w", err)
	}

	// Create go.mod file
	moduleName := strings.ReplaceAll(binaryName, "-", "_")
	goModContent := fmt.Sprintf("module %s\n\ngo 1.21\n", moduleName)
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		return fmt.Errorf("error writing go.mod: %w", err)
	}

	// Build the binary
	buildCmd := exec.Command("go", "build", "-o", outputPath, ".")
	buildCmd.Dir = tempDir
	buildCmd.Stderr = os.Stderr

	if b.debug {
		fmt.Fprintf(os.Stderr, "Building binary: %s\n", outputPath)
		buildCmd.Stdout = os.Stderr
	}

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("error building binary: %w", err)
	}

	if b.debug {
		fmt.Fprintf(os.Stderr, "✅ Successfully built: %s\n", outputPath)
	}

	return nil
}

// generateGo generates Go CLI output
func (b *Builder) generateGo(program *ast.Program, templateFile string, binaryName string) (string, error) {
	if templateFile != "" {
		templateContent, err := os.ReadFile(templateFile)
		if err != nil {
			return "", fmt.Errorf("error reading template file: %w", err)
		}
		return generator.GenerateGoWithTemplate(program, string(templateContent))
	}
	return generator.GenerateGoWithBinaryName(program, binaryName)
}