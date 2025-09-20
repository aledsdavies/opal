package v2

// TokenType represents lexical tokens for the v2 language design
type TokenType int

const (
	// Special tokens
	EOF TokenType = iota
	ILLEGAL

	// Chaining operations (execution flow)
	NEWLINE   // \n - sequential execution
	SEMICOLON // ; - sequential execution

	// Meta-programming keywords
	FOR     // for
	IN      // in
	IF      // if
	ELSE    // else
	WHEN    // when - pattern matching
	TRY     // try - error handling decorator
	CATCH   // catch - error handling decorator
	FINALLY // finally - error handling decorator

	// Language structure
	VAR    // var
	AT     // @
	COLON  // :
	EQUALS // =
	COMMA  // ,
	ARROW  // -> (for when patterns)

	// Brackets and braces
	LPAREN  // (
	RPAREN  // )
	LBRACE  // {
	RBRACE  // }
	LSQUARE // [
	RSQUARE // ]

	// Comparison operators (for if statements)
	EQ_EQ  // ==
	NOT_EQ // !=
	LT     // <
	LT_EQ  // <=
	GT     // >
	GT_EQ  // >=

	// Logical operators
	AND_AND // && (logical and)
	OR_OR   // || (logical or)
	NOT     // !

	// Shell chain operators
	AND    // && (chain success)
	OR     // || (chain failure)
	PIPE   // |
	APPEND // >>

	// Literals and content
	IDENTIFIER // command names, variable names, decorator names
	SHELL_TEXT // shell command text
	NUMBER     // 8080, 3.14, -100
	STRING     // "string" or 'string' content
	DURATION   // 30s, 5m, 1h
	BOOLEAN    // true, false

	// Comments
	COMMENT // # single line comment
)

// Token represents a lexical token
type Token struct {
	Type     TokenType
	Text     string
	Position Position
}

// Position represents a position in the source code
type Position struct {
	Line   int // 1-based line number
	Column int // 1-based column number
	Offset int // 0-based byte offset
}

// String returns a string representation of the token type
func (t TokenType) String() string {
	switch t {
	case EOF:
		return "EOF"
	case ILLEGAL:
		return "ILLEGAL"
	case NEWLINE:
		return "NEWLINE"
	case SEMICOLON:
		return "SEMICOLON"
	case FOR:
		return "FOR"
	case IN:
		return "IN"
	case IF:
		return "IF"
	case ELSE:
		return "ELSE"
	case WHEN:
		return "WHEN"
	case TRY:
		return "TRY"
	case CATCH:
		return "CATCH"
	case FINALLY:
		return "FINALLY"
	case VAR:
		return "VAR"
	case AT:
		return "AT"
	case COLON:
		return "COLON"
	case EQUALS:
		return "EQUALS"
	case COMMA:
		return "COMMA"
	case ARROW:
		return "ARROW"
	case LPAREN:
		return "LPAREN"
	case RPAREN:
		return "RPAREN"
	case LBRACE:
		return "LBRACE"
	case RBRACE:
		return "RBRACE"
	case LSQUARE:
		return "LSQUARE"
	case RSQUARE:
		return "RSQUARE"
	case EQ_EQ:
		return "EQ_EQ"
	case NOT_EQ:
		return "NOT_EQ"
	case LT:
		return "LT"
	case LT_EQ:
		return "LT_EQ"
	case GT:
		return "GT"
	case GT_EQ:
		return "GT_EQ"
	case AND_AND:
		return "AND_AND"
	case OR_OR:
		return "OR_OR"
	case NOT:
		return "NOT"
	case AND:
		return "AND"
	case OR:
		return "OR"
	case PIPE:
		return "PIPE"
	case APPEND:
		return "APPEND"
	case IDENTIFIER:
		return "IDENTIFIER"
	case SHELL_TEXT:
		return "SHELL_TEXT"
	case NUMBER:
		return "NUMBER"
	case STRING:
		return "STRING"
	case DURATION:
		return "DURATION"
	case BOOLEAN:
		return "BOOLEAN"
	case COMMENT:
		return "COMMENT"
	default:
		return "UNKNOWN"
	}
}

// Keywords maps string literals to their corresponding token types
var Keywords = map[string]TokenType{
	"for":     FOR,
	"in":      IN,
	"if":      IF,
	"else":    ELSE,
	"when":    WHEN,
	"try":     TRY,
	"catch":   CATCH,
	"finally": FINALLY,
	"var":     VAR,
	"true":    BOOLEAN,
	"false":   BOOLEAN,
}

// SingleCharTokens maps single characters to their token types
var SingleCharTokens = map[byte]TokenType{
	'@':  AT,
	':':  COLON,
	'=':  EQUALS,
	',':  COMMA,
	'(':  LPAREN,
	')':  RPAREN,
	'{':  LBRACE,
	'}':  RBRACE,
	'[':  LSQUARE,
	']':  RSQUARE,
	'|':  PIPE,
	'<':  LT,
	'>':  GT,
	'!':  NOT,
	'\n': NEWLINE,
	';':  SEMICOLON,
}

// TwoCharTokens maps two-character sequences to their token types
var TwoCharTokens = map[string]TokenType{
	"->": ARROW,
	"==": EQ_EQ,
	"!=": NOT_EQ,
	"<=": LT_EQ,
	">=": GT_EQ,
	"&&": AND_AND,
	"||": OR_OR,
	">>": APPEND,
}
