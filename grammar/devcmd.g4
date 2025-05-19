/**
 * Devcmd Grammar Definition
 *
 * This ANTLR4 grammar defines the syntax for 'devcmd' (**D**eclarative **E**xecution **V**ocabulary **Cmd**),
 * a domain-specific language for orchestrating build tasks, development environments, and service management.
 *
 * Core features:
 * 1. Named commands with optional modifiers: 'build: npm run build', 'watch server: node start'
 * 2. Variables for reuse: 'def SRC = ./src'
 * 3. Variable references in commands: 'build: cd $(SRC) && make'
 * 4. Service management with 'watch' and 'stop' commands
 * 5. Multi-statement blocks: 'setup: { npm install; go mod tidy; }'
 * 6. Background processes with ampersand: 'run-all: { server & client & db & }'
 *
 * This grammar handles lexical structure and syntax only. Semantic rules
 * (variable definition before use, watch/stop pairing, unique command names)
 * are enforced during analysis phases after parsing.
 */
grammar devcmd;

/**
 * Parser Rules
 * These define the structural hierarchy of the devcmd language
 */

// A devcmd program consists of multiple lines followed by EOF
program : line* EOF ;

// Each line represents a discrete unit in the program
line
    : variableDefinition   // A variable assignment with 'def'
    | commandDefinition    // A named command pattern
    | NEWLINE              // Empty lines for formatting
    ;

// Variables store reusable text values for reference in commands
variableDefinition : DEF NAME (EQUALS commandText?)? (NEWLINE | EOF) ;

// Commands define executable operations, optionally with service lifecycle modifiers
commandDefinition
    : (WATCH | STOP)? NAME COLON (simpleCommand | blockCommand)
    ;

// A simple command contains a single instruction, potentially with line continuations
simpleCommand : (commandText continuationLine*)? (NEWLINE | EOF) ;

// Block commands group multiple statements within braces
blockCommand : LBRACE NEWLINE? blockStatements RBRACE (NEWLINE | EOF)? ;

// Block statements can be empty or contain one or more commands
blockStatements
    : /* empty */               // Allow empty blocks
    | nonEmptyBlockStatements   // One or more statements
    ;

// Multiple statements are separated by semicolons
// The optional final semicolon allows for trailing semicolon in blocks like: { cmd1; cmd2; }
nonEmptyBlockStatements
    : blockStatement (SEMICOLON NEWLINE* blockStatement)* SEMICOLON? NEWLINE*
    ;

// Each block statement can be backgrounded with ampersand
// Space before ampersand is implicit since whitespace is skipped by the lexer
blockStatement : commandText AMPERSAND? ;

// Line continuations let commands span multiple lines
continuationLine : BACKSLASH NEWLINE commandText ;

// Command text is the actual instruction content
// Must match at least one element to avoid ambiguity
commandText
    : (ESCAPED_CHAR
      | variableReference
      | INCOMPLETE_VARIABLE_REFERENCE
      | COLON        // NEW
      | EQUALS       // NEW
      | COMMAND_TEXT
      | NAME
      )+
    ;

// Variables are referenced using $(name) syntax
variableReference : VAR_START NAME VAR_END ;

/**
 * Lexer Rules
 * These define the atomic elements and token patterns of the language
 */

// Keywords that have special meaning in devcmd
DEF : 'def' ;     // Variable definition marker
EQUALS : '=' ;    // Assignment operator
COLON : ':' ;     // Command separator
WATCH : 'watch' ; // Service startup modifier
STOP : 'stop' ;   // Service shutdown modifier

// Structural delimiters
LBRACE : '{' ;    // Block start
RBRACE : '}' ;    // Block end
SEMICOLON : ';' ; // Statement separator
AMPERSAND : '&' ; // Background process indicator
BACKSLASH : '\\' ; // Line continuation marker

// Variable reference delimiters
VAR_START : '$(' ; // Start of variable reference
VAR_END : ')' ;    // End of variable reference

// Error recovery for incomplete variable references
INCOMPLETE_VARIABLE_REFERENCE : '$(' ~[)\r\n]* ;

// Handle escape sequences for special characters and common escape patterns
// Supports standard escapes (\n, \r, \t), shell-relevant chars (\$, \;, \{, \}, \(, \), \"), and Unicode
ESCAPED_CHAR : '\\' ( [\\nrt$;{}()"]
                     | 'x' [0-9a-fA-F][0-9a-fA-F]
                     | 'u' [0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F]
                     ) ;

// Identifiers for variables and command names
NAME : [A-Za-z][A-Za-z0-9_-]* ;

// General command text content - Modified to include = and : characters
COMMAND_TEXT : ~[\r\n \t:=;{}()&]+ ;

// Comments and formatting elements
COMMENT : '#' ~[\r\n]* -> channel(HIDDEN) ;  // Comments don't affect execution, so hide from parser
NEWLINE : '\r'? '\n' ;
WS    : [ \t]+ -> channel(HIDDEN) ;

/**
 * Implementation Guidelines
 *
 * A compliant devcmd compiler should implement these features:
 *
 * 1. Runtime Environment
 *    • Commands execute in a POSIX-compatible shell environment
 *    • Environment variables from parent process are preserved
 *    • Working directory is maintained across commands within a block
 *
 * 2. Variable Handling
 *    • $(VAR) references expand to their defined value before execution
 *    • Shell variables like $HOME and ${PATH} pass through to the shell
 *    • All devcmd variables must be defined before use
 *
 * 3. Process Management
 *    • 'watch' commands create persistent process groups
 *    • Process groups register with a process registry for cleanup
 *    • 'stop' commands gracefully terminate matching process groups
 *    • Background processes ('&') run concurrently within their block
 *    • Foreground commands block until completion
 *
 * 4. Error Handling
 *    • Syntax errors report line and column of failure
 *    • Command failures propagate exit codes
 *    • Process termination ensures cleanup of all child processes
 *
 * 5. Performance Requirements
 *    • Parsing: O(n) time complexity for n lines of input
 *    • Memory: Peak usage below 5x input file size
 *    • Startup: Command execution begins within 100ms
 */
