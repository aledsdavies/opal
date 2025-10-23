package planner

import (
	"github.com/aledsdavies/opal/core/invariant"
	"github.com/aledsdavies/opal/core/planfmt"
)

// buildStepTree converts flat command list to operator precedence tree.
// Precedence (high to low): | > redirect > && > || > ;
//
// This implements the same logic as executor/execution_tree.go but at plan time.
// The tree structure captures operator precedence and enables:
// - Deterministic execution order
// - Parallel variable resolution
// - Plan serialization with operator structure
// - Beautiful dry-run visualization
func buildStepTree(commands []Command) planfmt.ExecutionNode {
	invariant.Precondition(len(commands) > 0, "commands cannot be empty")

	// Single command - check if it has a redirect operator
	if len(commands) == 1 {
		cmd := commands[0]

		// Check if this command has a redirect
		if (cmd.Operator == ">" || cmd.Operator == ">>") && cmd.RedirectTarget != nil {
			source := &planfmt.CommandNode{
				Decorator: cmd.Decorator,
				Args:      cmd.Args,
				Block:     cmd.Block,
			}

			target := commandToNode(*cmd.RedirectTarget)

			mode := planfmt.RedirectOverwrite
			if cmd.Operator == ">>" {
				mode = planfmt.RedirectAppend
			}

			return &planfmt.RedirectNode{
				Source: source,
				Target: *target,
				Mode:   mode,
			}
		}

		// No redirect - just return the command
		return commandToNode(cmd)
	}

	// Parse operators by precedence (lowest to highest)
	// This ensures higher precedence operators bind tighter

	// 1. Semicolon (lowest precedence) - splits into sequence
	if node := parseSemicolon(commands); node != nil {
		return node
	}

	// 2. OR operator
	if node := parseOr(commands); node != nil {
		return node
	}

	// 3. AND operator
	if node := parseAnd(commands); node != nil {
		return node
	}

	// 4. Redirect operator (> and >>)
	if node := parseRedirect(commands); node != nil {
		return node
	}

	// 5. Pipe operator (highest precedence) - must be contiguous
	if node := parsePipe(commands); node != nil {
		return node
	}

	// No operators found - single command
	return commandToNode(commands[0])
}

// commandToNode converts planfmt.Command to CommandNode
func commandToNode(cmd Command) *planfmt.CommandNode {
	return &planfmt.CommandNode{
		Decorator: cmd.Decorator,
		Args:      cmd.Args,
		Block:     cmd.Block,
	}
}

// parseSemicolon splits on semicolon operators (lowest precedence)
func parseSemicolon(commands []Command) planfmt.ExecutionNode {
	var segments [][]Command
	start := 0

	for i, cmd := range commands {
		if cmd.Operator == ";" {
			// Clone segment and clear operator on last command
			// (prevents infinite recursion when segment contains other operators)
			segment := make([]Command, i+1-start)
			copy(segment, commands[start:i+1])

			// CRITICAL: Clear the operator to prevent infinite recursion
			// Without this, buildStepTree(segment) would see the same ; operator
			// and call parseSemicolon again with identical input, looping forever
			invariant.Postcondition(segment[len(segment)-1].Operator == ";", "last command must have ; operator before clearing")
			segment[len(segment)-1].Operator = "" // Clear the ; operator
			invariant.Postcondition(segment[len(segment)-1].Operator == "", "operator must be cleared to prevent infinite recursion")

			segments = append(segments, segment)
			start = i + 1
		}
	}

	// No semicolons found
	if len(segments) == 0 {
		return nil
	}

	// Add remaining commands
	if start < len(commands) {
		segments = append(segments, commands[start:])
	}

	// Build sequence node
	var nodes []planfmt.ExecutionNode
	for _, seg := range segments {
		// Verify segment won't cause infinite recursion
		// Each segment must either:
		// 1. Have no ; operators (we cleared them above), OR
		// 2. Be a different slice (remaining commands after last ;)
		nodes = append(nodes, buildStepTree(seg))
	}

	return &planfmt.SequenceNode{Nodes: nodes}
}

// parseOr splits on OR operators (|| has lower precedence than &&)
func parseOr(commands []Command) planfmt.ExecutionNode {
	// Find rightmost || (left-to-right associativity)
	// Operator is on the command BEFORE the split point
	for i := len(commands) - 1; i >= 0; i-- {
		if commands[i].Operator == "||" {
			// Split: commands[0..i] (without operator) || commands[i+1..end]
			// Need to copy left side and clear the operator on last command
			leftCmds := make([]Command, i+1)
			copy(leftCmds, commands[:i+1])
			leftCmds[i].Operator = "" // Clear the || operator

			left := buildStepTree(leftCmds)
			right := buildStepTree(commands[i+1:])
			return &planfmt.OrNode{Left: left, Right: right}
		}
	}
	return nil
}

// parseAnd splits on AND operators (&& has lower precedence than |)
func parseAnd(commands []Command) planfmt.ExecutionNode {
	// Find rightmost && (left-to-right associativity)
	// Operator is on the command BEFORE the split point
	for i := len(commands) - 1; i >= 0; i-- {
		if commands[i].Operator == "&&" {
			// Split: commands[0..i] (without operator) && commands[i+1..end]
			// Need to copy left side and clear the operator on last command
			leftCmds := make([]Command, i+1)
			copy(leftCmds, commands[:i+1])
			leftCmds[i].Operator = "" // Clear the && operator

			left := buildStepTree(leftCmds)
			right := buildStepTree(commands[i+1:])
			return &planfmt.AndNode{Left: left, Right: right}
		}
	}
	return nil
}

// parseRedirect splits on redirect operators (> and >> have lower precedence than |, higher than &&)
func parseRedirect(commands []Command) planfmt.ExecutionNode {
	// Find rightmost redirect operator (left-to-right associativity)
	// Operator is on the command BEFORE the split point
	for i := len(commands) - 1; i >= 0; i-- {
		if commands[i].Operator == ">" || commands[i].Operator == ">>" {
			// The command at position i has the redirect operator and target
			// Build the source (everything up to and including command i, without the redirect)
			leftCmds := make([]Command, i+1)
			copy(leftCmds, commands[:i+1])
			leftCmds[i].Operator = ""        // Clear the redirect operator
			leftCmds[i].RedirectTarget = nil // Clear the target

			source := buildStepTree(leftCmds)

			// The redirect target is stored in commands[i].RedirectTarget
			if commands[i].RedirectTarget == nil {
				// No target - this shouldn't happen if parser is correct, but handle gracefully
				return source
			}

			target := commandToNode(*commands[i].RedirectTarget)

			// Determine redirect mode
			mode := planfmt.RedirectOverwrite
			if commands[i].Operator == ">>" {
				mode = planfmt.RedirectAppend
			}

			return &planfmt.RedirectNode{
				Source: source,
				Target: *target,
				Mode:   mode,
			}
		}
	}
	return nil
}

// parsePipe scans for contiguous pipe operators (highest precedence)
// All commands with | operators form a single pipeline
func parsePipe(commands []Command) planfmt.ExecutionNode {
	// Check if all operators are pipes (contiguous pipeline)
	allPipes := true
	for i := 0; i < len(commands)-1; i++ {
		if commands[i].Operator != "|" {
			allPipes = false
			break
		}
	}

	if allPipes {
		// Convert all commands to CommandNodes
		nodes := make([]planfmt.CommandNode, len(commands))
		for i, cmd := range commands {
			nodes[i] = planfmt.CommandNode{
				Decorator: cmd.Decorator,
				Args:      cmd.Args,
				Block:     cmd.Block,
			}
		}
		return &planfmt.PipelineNode{Commands: nodes}
	}

	return nil
}
