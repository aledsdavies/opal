package ast

import (
	"encoding/json"
)

// Tree-sitter Integration Functions
// This file contains all Tree-sitter specific functionality for syntax highlighting and parsing

// GetTreeSitterNode converts AST to Tree-sitter compatible structure
func (p *Program) ToTreeSitterJSON() map[string]interface{} {
	return map[string]interface{}{
		"type": "program",
		"start_position": map[string]int{
			"row":    p.Pos.Line - 1,
			"column": p.Pos.Column - 1,
		},
		"end_position": map[string]int{
			"row":    p.Tokens.End.Line - 1,
			"column": p.Tokens.End.Column - 1,
		},
		"children": []interface{}{
			p.variablesToTreeSitter(),
			p.commandsToTreeSitter(),
		},
	}
}

// variablesToTreeSitter converts variable declarations to Tree-sitter format
func (p *Program) variablesToTreeSitter() []interface{} {
	var vars []interface{}

	// Individual variables
	for _, varDecl := range p.Variables {
		vars = append(vars, map[string]interface{}{
			"type": "variable_declaration",
			"name": varDecl.Name,
			"value": varDecl.Value.String(),
			"grouped": false,
			"start_position": map[string]int{
				"row":    varDecl.Pos.Line - 1,
				"column": varDecl.Pos.Column - 1,
			},
			"end_position": map[string]int{
				"row":    varDecl.Tokens.End.Line - 1,
				"column": varDecl.Tokens.End.Column - 1,
			},
		})
	}

	// Grouped variables
	for _, varGroup := range p.VarGroups {
		groupVars := make([]interface{}, len(varGroup.Variables))
		for i, varDecl := range varGroup.Variables {
			groupVars[i] = map[string]interface{}{
				"type": "variable_declaration",
				"name": varDecl.Name,
				"value": varDecl.Value.String(),
				"grouped": true,
				"start_position": map[string]int{
					"row":    varDecl.Pos.Line - 1,
					"column": varDecl.Pos.Column - 1,
				},
				"end_position": map[string]int{
					"row":    varDecl.Tokens.End.Line - 1,
					"column": varDecl.Tokens.End.Column - 1,
				},
			}
		}

		vars = append(vars, map[string]interface{}{
			"type": "variable_group",
			"variables": groupVars,
			"start_position": map[string]int{
				"row":    varGroup.Pos.Line - 1,
				"column": varGroup.Pos.Column - 1,
			},
			"end_position": map[string]int{
				"row":    varGroup.Tokens.End.Line - 1,
				"column": varGroup.Tokens.End.Column - 1,
			},
		})
	}

	return vars
}

// commandsToTreeSitter converts command declarations to Tree-sitter format
func (p *Program) commandsToTreeSitter() []interface{} {
	var cmds []interface{}
	for _, cmdDecl := range p.Commands {
		cmd := map[string]interface{}{
			"type": "command_declaration",
			"name": cmdDecl.Name,
			"command_type": cmdDecl.Type.String(),
			"start_position": map[string]int{
				"row":    cmdDecl.Pos.Line - 1,
				"column": cmdDecl.Pos.Column - 1,
			},
			"end_position": map[string]int{
				"row":    cmdDecl.Tokens.End.Line - 1,
				"column": cmdDecl.Tokens.End.Column - 1,
			},
		}

		// Add command body information
		cmd["body"] = cmdDecl.Body.toTreeSitter()

		// Add decorators if present
		decorators := findCommandDecorators(&cmdDecl)
		if len(decorators) > 0 {
			var decoratorNodes []interface{}
			for _, decorator := range decorators {
				decoratorNodes = append(decoratorNodes, decorator.toTreeSitter())
			}
			cmd["decorators"] = decoratorNodes
		}

		cmds = append(cmds, cmd)
	}
	return cmds
}

// toTreeSitter converts CommandBody to Tree-sitter format
func (b *CommandBody) toTreeSitter() map[string]interface{} {
	return map[string]interface{}{
		"type": "command_body",
		"is_block": b.IsBlock,
		"content": commandContentToTreeSitter(b.Content),
		"start_position": map[string]int{
			"row":    b.Pos.Line - 1,
			"column": b.Pos.Column - 1,
		},
		"end_position": map[string]int{
			"row":    b.Tokens.End.Line - 1,
			"column": b.Tokens.End.Column - 1,
		},
	}
}

// commandContentToTreeSitter converts CommandContent to Tree-sitter format using type switches
func commandContentToTreeSitter(content CommandContent) map[string]interface{} {
	switch c := content.(type) {
	case *ShellContent:
		return c.toTreeSitter()
	case *DecoratedContent:
		return c.toTreeSitter()
	default:
		return map[string]interface{}{
			"type": "unknown_content",
		}
	}
}

// toTreeSitter converts ShellContent to Tree-sitter format
func (s *ShellContent) toTreeSitter() map[string]interface{} {
	parts := make([]interface{}, len(s.Parts))
	for i, part := range s.Parts {
		parts[i] = shellPartToTreeSitter(part)
	}

	return map[string]interface{}{
		"type": "shell_content",
		"parts": parts,
		"start_position": map[string]int{
			"row":    s.Pos.Line - 1,
			"column": s.Pos.Column - 1,
		},
		"end_position": map[string]int{
			"row":    s.Tokens.End.Line - 1,
			"column": s.Tokens.End.Column - 1,
		},
	}
}

// shellPartToTreeSitter converts ShellPart to Tree-sitter format using type switches
func shellPartToTreeSitter(part ShellPart) map[string]interface{} {
	switch p := part.(type) {
	case *TextPart:
		return p.toTreeSitter()
	case *FunctionDecorator:
		return p.toTreeSitter()
	default:
		return map[string]interface{}{
			"type": "unknown_shell_part",
		}
	}
}

// toTreeSitter converts TextPart to Tree-sitter format
func (t *TextPart) toTreeSitter() map[string]interface{} {
	return map[string]interface{}{
		"type": "text_part",
		"text": t.Text,
		"start_position": map[string]int{
			"row":    t.Pos.Line - 1,
			"column": t.Pos.Column - 1,
		},
		"end_position": map[string]int{
			"row":    t.Tokens.End.Line - 1,
			"column": t.Tokens.End.Column - 1,
		},
	}
}

// toTreeSitter converts DecoratedContent to Tree-sitter format
func (d *DecoratedContent) toTreeSitter() map[string]interface{} {
	decoratorNodes := make([]interface{}, len(d.Decorators))
	for i, decorator := range d.Decorators {
		decoratorNodes[i] = decorator.toTreeSitter()
	}

	return map[string]interface{}{
		"type": "decorated_content",
		"decorators": decoratorNodes,
		"content": commandContentToTreeSitter(d.Content),
		"start_position": map[string]int{
			"row":    d.Pos.Line - 1,
			"column": d.Pos.Column - 1,
		},
		"end_position": map[string]int{
			"row":    d.Tokens.End.Line - 1,
			"column": d.Tokens.End.Column - 1,
		},
	}
}

// GetTreeSitterGrammar returns the Tree-sitter grammar definition for Devcmd
func GetTreeSitterGrammar() map[string]interface{} {
	return map[string]interface{}{
		"name": "devcmd",
		"rules": map[string]interface{}{
			"program": map[string]interface{}{
				"type": "REPEAT",
				"content": map[string]interface{}{
					"type": "CHOICE",
					"members": []interface{}{
						map[string]string{"type": "SYMBOL", "name": "variable_declaration"},
						map[string]string{"type": "SYMBOL", "name": "command_declaration"},
						map[string]string{"type": "SYMBOL", "name": "newline"},
					},
				},
			},
			"variable_declaration": map[string]interface{}{
				"type": "SEQ",
				"members": []interface{}{
					map[string]string{"type": "STRING", "value": "var"},
					map[string]string{"type": "SYMBOL", "name": "identifier"},
					map[string]string{"type": "STRING", "value": "="},
					map[string]string{"type": "SYMBOL", "name": "expression"},
				},
			},
			"command_declaration": map[string]interface{}{
				"type": "SEQ",
				"members": []interface{}{
					map[string]interface{}{
						"type": "CHOICE",
						"members": []interface{}{
							map[string]string{"type": "STRING", "value": "watch"},
							map[string]string{"type": "STRING", "value": "stop"},
							map[string]interface{}{
								"type": "BLANK",
							},
						},
					},
					map[string]string{"type": "SYMBOL", "name": "identifier"},
					map[string]string{"type": "STRING", "value": ":"},
					map[string]string{"type": "SYMBOL", "name": "command_body"},
				},
			},
			"command_body": map[string]interface{}{
				"type": "CHOICE",
				"members": []interface{}{
					map[string]string{"type": "SYMBOL", "name": "shell_content"},
					map[string]string{"type": "SYMBOL", "name": "decorated_content"},
					map[string]string{"type": "SYMBOL", "name": "block_content"},
				},
			},
			"shell_content": map[string]interface{}{
				"type": "PATTERN",
				"value": "[^@{}\\n]+",
			},
			"decorated_content": map[string]interface{}{
				"type": "SEQ",
				"members": []interface{}{
					map[string]interface{}{
						"type": "REPEAT1",
						"content": map[string]string{"type": "SYMBOL", "name": "decorator"},
					},
					map[string]string{"type": "SYMBOL", "name": "command_content"},
				},
			},
			"block_content": map[string]interface{}{
				"type": "SEQ",
				"members": []interface{}{
					map[string]string{"type": "STRING", "value": "{"},
					map[string]string{"type": "SYMBOL", "name": "command_content"},
					map[string]string{"type": "STRING", "value": "}"},
				},
			},
			"decorator": map[string]interface{}{
				"type": "SEQ",
				"members": []interface{}{
					map[string]string{"type": "STRING", "value": "@"},
					map[string]string{"type": "SYMBOL", "name": "identifier"},
					map[string]interface{}{
						"type": "CHOICE",
						"members": []interface{}{
							map[string]string{"type": "SYMBOL", "name": "argument_list"},
							map[string]interface{}{
								"type": "BLANK",
							},
						},
					},
				},
			},
			"argument_list": map[string]interface{}{
				"type": "SEQ",
				"members": []interface{}{
					map[string]string{"type": "STRING", "value": "("},
					map[string]interface{}{
						"type": "CHOICE",
						"members": []interface{}{
							map[string]interface{}{
								"type": "SEQ",
								"members": []interface{}{
									map[string]string{"type": "SYMBOL", "name": "expression"},
									map[string]interface{}{
										"type": "REPEAT",
										"content": map[string]interface{}{
											"type": "SEQ",
											"members": []interface{}{
												map[string]string{"type": "STRING", "value": ","},
												map[string]string{"type": "SYMBOL", "name": "expression"},
											},
										},
									},
								},
							},
							map[string]interface{}{
								"type": "BLANK",
							},
						},
					},
					map[string]string{"type": "STRING", "value": ")"},
				},
			},
			"expression": map[string]interface{}{
				"type": "CHOICE",
				"members": []interface{}{
					map[string]string{"type": "SYMBOL", "name": "string_literal"},
					map[string]string{"type": "SYMBOL", "name": "number_literal"},
					map[string]string{"type": "SYMBOL", "name": "duration_literal"},
					map[string]string{"type": "SYMBOL", "name": "identifier"},
				},
			},
			"string_literal": map[string]interface{}{
				"type": "CHOICE",
				"members": []interface{}{
					map[string]interface{}{
						"type": "SEQ",
						"members": []interface{}{
							map[string]string{"type": "STRING", "value": "\""},
							map[string]interface{}{
								"type": "REPEAT",
								"content": map[string]interface{}{
									"type": "CHOICE",
									"members": []interface{}{
										map[string]string{"type": "PATTERN", "value": "[^\"\\\\]"},
										map[string]string{"type": "SYMBOL", "name": "escape_sequence"},
									},
								},
							},
							map[string]string{"type": "STRING", "value": "\""},
						},
					},
					map[string]interface{}{
						"type": "SEQ",
						"members": []interface{}{
							map[string]string{"type": "STRING", "value": "'"},
							map[string]interface{}{
								"type": "REPEAT",
								"content": map[string]interface{}{
									"type": "CHOICE",
									"members": []interface{}{
										map[string]string{"type": "PATTERN", "value": "[^'\\\\]"},
										map[string]string{"type": "SYMBOL", "name": "escape_sequence"},
									},
								},
							},
							map[string]string{"type": "STRING", "value": "'"},
						},
					},
				},
			},
			"number_literal": map[string]interface{}{
				"type": "PATTERN",
				"value": "-?[0-9]+(\\.[0-9]+)?",
			},
			"duration_literal": map[string]interface{}{
				"type": "PATTERN",
				"value": "[0-9]+(\\.[0-9]+)?(ns|us|ms|s|m|h)",
			},
			"identifier": map[string]interface{}{
				"type": "PATTERN",
				"value": "[a-zA-Z_][a-zA-Z0-9_-]*",
			},
			"escape_sequence": map[string]interface{}{
				"type": "PATTERN",
				"value": "\\\\.",
			},
			"newline": map[string]interface{}{
				"type": "STRING",
				"value": "\n",
			},
		},
	}
}

// SerializeToTreeSitter returns the AST as a JSON string for Tree-sitter
func (p *Program) SerializeToTreeSitter() (string, error) {
	treeSitterAST := p.ToTreeSitterJSON()
	jsonBytes, err := json.MarshalIndent(treeSitterAST, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// GetHighlightQuery returns the Tree-sitter highlight query for syntax highlighting
func GetHighlightQuery() string {
	return `
; Keywords
"var" @keyword
"watch" @keyword
"stop" @keyword

; Operators
":" @operator
"=" @operator
"@" @operator

; Punctuation
"(" @punctuation.bracket
")" @punctuation.bracket
"{" @punctuation.bracket
"}" @punctuation.bracket
"," @punctuation.delimiter

; Literals
(string_literal) @string
(number_literal) @number
(duration_literal) @number.time

; Identifiers
(identifier) @variable

; Commands
(command_declaration
  name: (identifier) @function)

; Variables
(variable_declaration
  name: (identifier) @variable.definition)

; Decorators
(decorator
  name: (identifier) @function.decorator)

; Shell content
(shell_content) @string.special

; Comments
(comment) @comment
`
}

// GetTagsQuery returns the Tree-sitter tags query for code navigation
func GetTagsQuery() string {
	return `
(variable_declaration
  name: (identifier) @name) @definition.variable

(command_declaration
  name: (identifier) @name) @definition.function

(decorator
  name: (identifier) @name) @reference.decorator
`
}

// GetInjectionQuery returns the Tree-sitter injection query for embedded languages
func GetInjectionQuery() string {
	return `
(shell_content) @injection.content
(#set! injection.language "bash")
`
}
