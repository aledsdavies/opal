package lexer

import (
	"io"
	"os"

	"github.com/aledsdavies/devcmd/core/types"
)

// TokenProcessor represents the interface for processing input into tokens
// Optimized for sub-5ms parsing performance with minimal allocations
type TokenProcessor interface {
	// NextToken returns the next token from the input (fastest for streaming parsing)
	NextToken() types.Token

	// AllTokens returns all tokens from the input (for batch processing, pre-allocates slice)
	AllTokens() []types.Token

	// Reset resets the processor to start from the beginning
	Reset()

	// GetInput returns the raw input string for error reporting
	GetInput() string
}

// ProcessorFactory creates token processors from input readers
type ProcessorFactory interface {
	Create(reader io.Reader) TokenProcessor
}

// ProcessorManager manages the selection between different processor implementations
type ProcessorManager struct {
	useSimplified     bool
	legacyFactory     ProcessorFactory
	simplifiedFactory ProcessorFactory
}

// NewProcessorManager creates a new processor manager
func NewProcessorManager() *ProcessorManager {
	return &ProcessorManager{
		useSimplified:     shouldUseSimplifiedLexer(),
		legacyFactory:     &LegacyProcessorFactory{},
		simplifiedFactory: &SimplifiedProcessorFactory{},
	}
}

// Create returns the appropriate processor based on configuration
func (m *ProcessorManager) Create(reader io.Reader) TokenProcessor {
	if m.useSimplified {
		return m.simplifiedFactory.Create(reader)
	}
	return m.legacyFactory.Create(reader)
}

// shouldUseSimplifiedLexer checks environment variables and feature flags
func shouldUseSimplifiedLexer() bool {
	// Check environment variable for feature flag
	if os.Getenv("DEVCMD_USE_SIMPLIFIED_LEXER") == "true" {
		return true
	}

	// Check environment variable for testing
	if os.Getenv("DEVCMD_TEST_SIMPLIFIED_LEXER") == "true" {
		return true
	}

	// Default to legacy for now
	return false
}

// LegacyProcessorFactory creates the current complex lexer
type LegacyProcessorFactory struct{}

func (f *LegacyProcessorFactory) Create(reader io.Reader) TokenProcessor {
	return &LegacyTokenProcessor{lexer: NewLegacy(reader)}
}

// SimplifiedProcessorFactory creates the new simplified lexer (to be implemented)
type SimplifiedProcessorFactory struct{}

func (f *SimplifiedProcessorFactory) Create(reader io.Reader) TokenProcessor {
	// TODO: Implement SimplifiedLexer
	// For now, wrap legacy processor
	return &SimplifiedTokenProcessor{
		legacy: &LegacyTokenProcessor{lexer: NewLegacy(reader)},
	}
}

// LegacyTokenProcessor wraps the current Lexer to implement TokenProcessor
type LegacyTokenProcessor struct {
	lexer *Lexer
}

func (p *LegacyTokenProcessor) NextToken() types.Token {
	return p.lexer.NextToken()
}

func (p *LegacyTokenProcessor) AllTokens() []types.Token {
	// Use the existing optimized TokenizeToSlice method
	return p.lexer.TokenizeToSlice()
}

func (p *LegacyTokenProcessor) Reset() {
	// Reset the lexer to start position
	p.lexer.position = 0
	p.lexer.readPos = 0
	p.lexer.line = 1
	p.lexer.column = 0
	p.lexer.mode = LanguageMode
	p.lexer.braceLevel = 0
	p.lexer.patternBraceLevel = 0
	p.lexer.commandBlockLevel = 0
	p.lexer.inFunctionDecorator = false
	p.lexer.shellBraceLevel = 0
	p.lexer.shellParenLevel = 0
	p.lexer.shellAnyBraceLevel = 0
	p.lexer.needsShellEnd = false
	p.lexer.shellInSingleQuote = false
	p.lexer.shellInDoubleQuote = false
	p.lexer.shellInBacktick = false
	p.lexer.inInterpolatedString = false
	p.lexer.inStringDecorator = false
	p.lexer.inLiteralString = false
	p.lexer.readChar()
}

func (p *LegacyTokenProcessor) GetInput() string {
	return p.lexer.input
}

// SimplifiedTokenProcessor implements the new simplified lexer (placeholder)
type SimplifiedTokenProcessor struct {
	// TODO: Implement when SimplifiedLexer is created
	// For now, delegate to legacy processor
	legacy *LegacyTokenProcessor
}

func (p *SimplifiedTokenProcessor) NextToken() types.Token {
	return p.legacy.NextToken()
}

func (p *SimplifiedTokenProcessor) AllTokens() []types.Token {
	return p.legacy.AllTokens()
}

func (p *SimplifiedTokenProcessor) Reset() {
	p.legacy.Reset()
}

func (p *SimplifiedTokenProcessor) GetInput() string {
	return p.legacy.GetInput()
}
