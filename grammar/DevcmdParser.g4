/**
 * Devcmd Parser Grammar - Clean Design with Backward Compatibility
 *
 * This parser maintains rule names expected by existing Go code while
 * implementing a cleaner design that properly handles annotation syntax.
 *
 * Key design principles:
 * 1. Annotation syntax is handled cleanly with proper precedence
 * 2. Shell syntax (parentheses, braces) works normally in commands
 * 3. Variable expansion and escaping work as expected
 * 4. Rule names are compatible with existing visitor code
 * 5. Proper newline handling in block statements
 * 6. Simple annotations work without requiring semicolons in blocks
 */
parser grammar DevcmdParser;

options {
    tokenVocab = DevcmdLexer;
}

/**
 * TOP LEVEL STRUCTURE
 * Program consists of variable definitions and command definitions
 */

// Entry point - sequence of lines ending with EOF
program : line* EOF ;

// Each line can be a definition, command, or empty line
line
    : variableDefinition   // def NAME = value;
    | commandDefinition    // [watch|stop] NAME: body
    | NEWLINE              // Empty lines for formatting
    ;

/**
 * VARIABLE DEFINITIONS
 * Format: def NAME = value;
 */

// Variable definition with optional value
variableDefinition : DEF NAME EQUALS variableValue SEMICOLON ;

// Variable value - can be empty or contain command text
variableValue
    : commandText     // Variable has content
    | /* empty */     // Variable is empty (def VAR = ;)
    ;

/**
 * COMMAND DEFINITIONS
 * Format: [watch|stop] NAME: body
 * Body can be simple command, block, or annotation
 */

// Command with optional watch/stop modifier
commandDefinition : (WATCH | STOP)? NAME COLON commandBody ;

// Command body - multiple alternatives for different command types
commandBody
    : annotatedCommand     // @name(...) or @name: ...
    | blockCommand         // { ... }
    | simpleCommand        // command;
    ;

/**
 * ANNOTATION SYNTAX
 * Three forms:
 * 1. Function: @name(raw shell command) - with optional semicolon for top-level
 * 2. Block: @name: { ... } - MOVED TO HIGHER PRECEDENCE
 * 3. Simple: @name: processed command - no semicolon needed in blocks
 *
 * CRITICAL FIX: blockAnnot must come before simpleAnnot to get precedence
 * when parsing @name: { ... } syntax
 */

// Annotation command with labels for visitor compatibility
// REORDERED: blockAnnot now comes before simpleAnnot for correct precedence
annotatedCommand
    : AT_NAME_LPAREN RAW_TEXT* RAW_RPAREN SEMICOLON?    #functionAnnot
    | AT annotation COLON blockCommand                  #blockAnnot
    | AT annotation COLON annotationCommand             #simpleAnnot
    ;

// Annotation name (kept for compatibility)
annotation : NAME ;

/**
 * REGULAR COMMANDS
 * Simple and block commands with support for continuations and newlines
 */

// Simple command with optional line continuations and required semicolon
simpleCommand : commandText continuationLine* SEMICOLON ;

// Command text without semicolon requirement (for use in simple annotations)
annotationCommand : commandText continuationLine* ;

// Block command containing multiple statements with proper newline handling
blockCommand : LBRACE NEWLINE? blockStatements RBRACE ;

// Block content structure (compatible with existing code)
blockStatements
    : /* empty */               // Allow empty blocks
    | nonEmptyBlockStatements   // One or more statements
    ;

// Non-empty block statements separated by semicolons with optional newlines
// Filter out empty statements to fix block counting issues
nonEmptyBlockStatements
    : blockStatement (SEMICOLON NEWLINE* blockStatement)* SEMICOLON? NEWLINE*
    ;

// Individual statement within a block
blockStatement
    : annotatedCommand                    // Annotations work in blocks
    | commandText continuationLine*       // Regular commands (no semicolon in blocks)
    ;

/**
 * LINE CONTINUATIONS
 * Support for multi-line commands using backslash
 */

// Line continuation: backslash + newline + more command text
continuationLine : BACKSLASH NEWLINE commandText ;

/**
 * COMMAND TEXT PARSING
 * Flexible parsing of shell-like command content
 */

// Command text - sequence of content elements
commandText : commandTextElement* ;

// Individual elements that can appear in command text
commandTextElement
    : VAR_REF           // $(VAR) - devcmd variable
    | SHELL_VAR         // $VAR - shell variable
    | ESCAPED_DOLLAR    // \$ - literal dollar
    | NAME              // Identifiers
    | NUMBER            // Numeric literals
    | STRING            // Quoted strings
    | LPAREN            // ( - shell subshells, grouping
    | RPAREN            // ) - shell subshells, grouping
    | LBRACE            // { - shell brace expansion
    | RBRACE            // } - shell brace expansion
    | AMPERSAND         // & - shell background processes
    | COLON             // : - allowed in commands
    | EQUALS            // = - allowed in commands
    | BACKSLASH         // \ - shell escaping
    | WATCH             // Allow keywords in command text
    | STOP              // Allow keywords in command text
    | CONTENT           // General content token
    ;
