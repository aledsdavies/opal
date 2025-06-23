  /**
 * Devcmd Lexer Grammar
 *
 * Lexer for the devcmd language - a declarative syntax for defining
 * CLI tools from simple command definitions. Devcmd transforms command
 * definitions into standalone CLI binaries with process management,
 * variable substitution, and workflow automation.
 *
 * Language features:
 * - Variable definitions: def NAME = value;
 * - Simple commands: build: go build ./cmd;
 * - Block commands: deploy: { build; test; kubectl apply -f k8s/ }
 * - Process management: watch/stop command pairs
 * - Decorators: @name(...) for command metadata, processing, and variables
 * - Shell command syntax: pipes, redirections, background processes
 */
lexer grammar DevcmdLexer;

// Keywords - must come first for precedence
DEF : 'def' ;
WATCH : 'watch' ;
STOP : 'stop' ;

// Regular decorator start - for @name: syntax and single @
AT : '@' ;

// Structural operators and delimiters
EQUALS : '=' ;
COLON : ':' ;
SEMICOLON : ';' ;
LBRACE : '{' ;
RBRACE : '}' ;
LPAREN : '(' ;
RPAREN : ')' ;
BACKSLASH : '\\' ;

// String literals - must come before other character tokens
STRING : '"' (~["\\\r\n] | '\\' .)* '"' ;
SINGLE_STRING : '\'' (~['\\\r\n] | '\\' .)* '\'' ;

// Identifiers and literals - NAME must be early to capture command names correctly
// Updated to explicitly support hyphens and underscores in command/decorator names
NAME : [A-Za-z] [A-Za-z0-9_-]* ;
NUMBER : '-'? [0-9]+ ('.' [0-9]+)? ;

// Path-like content (handles things like ./src, *.tmp, etc.)
// More specific pattern to avoid conflicts with NAME
PATH_CONTENT : [./~] [A-Za-z0-9._/*-]+ ;

// Shell operators and special characters as individual tokens
// Reordered to put more specific tokens first
AMPERSAND : '&' ;
PIPE : '|' ;
LT : '<' ;
GT : '>' ;
DOT : '.' ;
COMMA : ',' ;
SLASH : '/' ;
DASH : '-' ;
STAR : '*' ;
PLUS : '+' ;
QUESTION : '?' ;
EXCLAIM : '!' ;
PERCENT : '%' ;
CARET : '^' ;
TILDE : '~' ;
UNDERSCORE : '_' ;
LBRACKET : '[' ;
RBRACKET : ']' ;
DOLLAR : '$' ;
HASH : '#' ;
DOUBLEQUOTE : '"' ;

// Whitespace and comments - must be at the end
COMMENT : '#' ~[\r\n]* -> channel(HIDDEN) ;
NEWLINE : '\r'? '\n' ;
WS : [ \t]+ -> channel(HIDDEN) ;
