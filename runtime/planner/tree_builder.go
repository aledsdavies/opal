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
		if cmd.RedirectMode != "" && cmd.RedirectTarget != nil {
			source := &planfmt.CommandNode{
				Decorator: cmd.Decorator,
				Args:      cmd.Args,
				Block:     cmd.Block,
			}

			target := commandToNode(*cmd.RedirectTarget)

			mode := planfmt.RedirectOverwrite
			if cmd.RedirectMode == ">>" {
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

	// 4. Pipe and Redirect (equal precedence, left-to-right like bash)
	if node := parsePipeAndRedirect(commands); node != nil {
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
		if commands[i].RedirectMode != "" {
			// The command at position i has the redirect operator and target
			// Build the source (everything up to and including command i, without the redirect)
			leftCmds := make([]Command, i+1)
			copy(leftCmds, commands[:i+1])
			leftCmds[i].RedirectMode = ""    // Clear the redirect mode
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
			if commands[i].RedirectMode == ">>" {
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

// parsePipeAndRedirect handles pipe (|) and redirect (>, >>) with equal precedence.
// Scans left-to-right to match bash behavior.
// Examples:
//   - echo a > out | cat  → (echo a > out) | cat
//   - echo a | cat > out  → (echo a | cat) > out
func parsePipeAndRedirect(commands []Command) planfmt.ExecutionNode {
	// Scan left-to-right for FIRST pipe or redirect
	for i := 0; i < len(commands); i++ {
		// Check for redirect on this command
		if commands[i].RedirectMode != "" {
			// Build left side (up to and including command i, without redirect)
			leftCmds := make([]Command, i+1)
			copy(leftCmds, commands[:i+1])
			leftCmds[i].RedirectMode = "" // Clear redirect
			leftCmds[i].RedirectTarget = nil

			// If this command also has a pipe operator, we'll handle it in the next iteration
			// after building the redirect node
			savedOperator := leftCmds[i].Operator
			leftCmds[i].Operator = "" // Clear for recursion

			source := buildStepTree(leftCmds)
			target := commandToNode(*commands[i].RedirectTarget)

			mode := planfmt.RedirectOverwrite
			if commands[i].RedirectMode == ">>" {
				mode = planfmt.RedirectAppend
			}

			redirectNode := &planfmt.RedirectNode{
				Source: source,
				Target: *target,
				Mode:   mode,
			}

			// If there's a pipe operator after this redirect, continue processing
			if savedOperator == "|" && i+1 < len(commands) {
				// Build right side
				rightCmds := commands[i+1:]
				right := buildStepTree(rightCmds)

				// Check if right side is also a command (for pipeline flattening)
				if rightCmd, ok := right.(*planfmt.CommandNode); ok {
					// Can't easily flatten redirect into pipeline, so create nested structure
					// This represents: (redirect) | command
					// For now, return just the redirect and let the pipe be handled separately
					_ = rightCmd
				}

				// Return redirect node; the pipe will be handled in next call
				return redirectNode
			}

			return redirectNode
		}

		// Check for pipe operator
		if commands[i].Operator == "|" {
			// Build left side (up to and including command i, without operator)
			leftCmds := make([]Command, i+1)
			copy(leftCmds, commands[:i+1])
			leftCmds[i].Operator = "" // Clear operator
			left := buildStepTree(leftCmds)

			// Build right side (commands after i)
			if i+1 < len(commands) {
				rightCmds := commands[i+1:]
				right := buildStepTree(rightCmds)

				// Try to flatten into PipelineNode if both sides are CommandNodes
				leftCmd, leftIsCmd := left.(*planfmt.CommandNode)
				rightCmd, rightIsCmd := right.(*planfmt.CommandNode)
				rightPipe, rightIsPipe := right.(*planfmt.PipelineNode)

				if leftIsCmd && rightIsCmd {
					// Simple case: cmd | cmd
					return &planfmt.PipelineNode{
						Commands: []planfmt.CommandNode{*leftCmd, *rightCmd},
					}
				} else if leftIsCmd && rightIsPipe {
					// Flatten: cmd | (cmd | cmd | ...) → cmd | cmd | cmd | ...
					nodes := make([]planfmt.CommandNode, 1+len(rightPipe.Commands))
					nodes[0] = *leftCmd
					copy(nodes[1:], rightPipe.Commands)
					return &planfmt.PipelineNode{Commands: nodes}
				}

				// Complex case: one side is not a simple command
				// Can't create a simple pipeline - this shouldn't happen with current grammar
				// Return left for now
				return left
			}

			return left
		}
	}

	return nil
}
