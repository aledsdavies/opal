 /**
 * Devcmd Lexer Grammar - Clean Design with Strategic Mode Usage
 *
 * This lexer uses a simple two-mode approach:
 * 1. DEFAULT mode: handles all basic devcmd structure and most content
 * 2. RAW_CONTENT mode: captures raw shell commands inside @name(...) annotations
 *
 * Key design principles:
 * - Use lexer modes sparingly - only where they provide clear value
 * - For complex shell commands (with ;, {}, etc.), use @sh() annotations
 * - Keep escape sequences minimal - only \$ for literal dollar signs
 * - Escaped semicolons (\;) should be in @sh() as raw content
 */
lexer grammar DevcmdLexer;

/**
 * DEFAULT MODE - Primary devcmd tokenization
 *
 * Handles:
 * - Keywords (def, watch, stop)
 * - Operators (:, =, ;, etc.)
 * - Variable references ($(VAR), $VAR)
 * - Annotation detection (@name( triggers mode switch)
 * - All other shell syntax as flexible content tokens
 */

// Keywords - must come first for precedence
DEF : 'def' ;
WATCH : 'watch' ;
STOP : 'stop' ;

// Special annotation pattern - triggers raw content capture
// This handles @sh(, @parallel(, etc. and switches to RAW_CONTENT mode
AT_NAME_LPAREN : '@' [A-Za-z][A-Za-z0-9_-]* '(' -> pushMode(RAW_CONTENT) ;

// Regular annotation start - for @name: syntax
AT : '@' ;

// Structural operators and delimiters
EQUALS : '=' ;              // Variable assignment
COLON : ':' ;               // Command separator
SEMICOLON : ';' ;           // Statement terminator
LBRACE : '{' ;              // Block start
RBRACE : '}' ;              // Block end
LPAREN : '(' ;              // Shell parentheses (subshells, grouping)
RPAREN : ')' ;              // Shell parentheses
BACKSLASH : '\\' ;          // Line continuation and shell escaping
AMPERSAND : '&' ;           // Shell background processes

// Variable references - highest priority after keywords
// $(VAR) - devcmd variable expansion
VAR_REF : '$(' [A-Za-z][A-Za-z0-9_-]* ')' ;
// $VAR - shell variable pass-through
SHELL_VAR : '$' [A-Za-z][A-Za-z0-9_]* ;

// Escape sequences for special devcmd syntax
ESCAPED_DOLLAR : '\\$' ;

// Basic content tokens
NAME : [A-Za-z][A-Za-z0-9_-]* ;        // Identifiers
NUMBER : [0-9]+ ('.' [0-9]+)? | '.' [0-9]+ ; // Numeric literals
STRING : '"' (~["\r\n] | '\\"')* '"' ;  // Quoted strings

// Flexible content token - captures most shell syntax
// Excludes only devcmd structural characters to avoid conflicts
CONTENT : ~[\r\n \t@:=;{}()\\$]+ ;

// Whitespace and comments
COMMENT : '#' ~[\r\n]* -> channel(HIDDEN) ; // Hide comments from parser
NEWLINE : '\r'? '\n' ;      // Line boundaries
WS : [ \t]+ -> channel(HIDDEN) ;       // Hide whitespace from parser

/**
 * RAW_CONTENT MODE - Literal shell command capture
 *
 * Entered when lexer sees @name( pattern.
 * Captures everything until closing ) as raw text without interpretation.
 * This is where complex shell commands with semicolons, braces, etc. should go:
 *
 * @sh(find . -name "*.tmp" -exec rm {} \;)
 *     ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
 *     All captured as RAW_TEXT tokens - no escaping needed
 */
mode RAW_CONTENT;

// Capture all content until closing parenthesis
RAW_TEXT : ~[)]+ ;

// Closing paren exits raw mode and returns to default tokenization
RAW_RPAREN : ')' -> popMode ;

// Whitespace handling in raw mode
RAW_WS : [ \t\r\n]+ -> channel(HIDDEN) ;
