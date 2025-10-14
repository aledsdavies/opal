package planner

import (
	"fmt"
	"time"

	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/aledsdavies/opal/runtime/lexer"
	"github.com/aledsdavies/opal/runtime/parser"
)

// Config configures the planner
type Config struct {
	Target    string         // Command name (e.g., "hello") or "" for script mode
	Telemetry TelemetryLevel // Telemetry level (production-safe)
	Debug     DebugLevel     // Debug level (development only)
}

// TelemetryLevel controls telemetry collection (production-safe)
type TelemetryLevel int

const (
	TelemetryOff    TelemetryLevel = iota // Zero overhead (default)
	TelemetryBasic                        // Step counts only
	TelemetryTiming                       // Counts + timing per phase
)

// DebugLevel controls debug tracing (development only)
type DebugLevel int

const (
	DebugOff      DebugLevel = iota // No debug info (default)
	DebugPaths                      // Method call tracing (enter/exit)
	DebugDetailed                   // Event-level tracing (every step)
)

// PlanResult holds the plan and observability data
type PlanResult struct {
	Plan        *planfmt.Plan  // The execution plan
	PlanTime    time.Duration  // Planning time (always collected)
	Telemetry   *PlanTelemetry // Additional metrics (nil if TelemetryOff)
	DebugEvents []DebugEvent   // Debug events (nil if DebugOff)
}

// PlanTelemetry holds additional planner metrics (optional, production-safe)
type PlanTelemetry struct {
	EventCount int // Number of events processed
	StepCount  int // Number of steps created
}

// DebugEvent holds debug tracing information (development only)
type DebugEvent struct {
	Timestamp time.Time
	Event     string // "enter_plan", "function_found", "step_created", etc.
	EventPos  int    // Current position in event stream
	Context   string // Additional context
}

// Plan consumes parser events and generates an execution plan
func Plan(events []parser.Event, tokens []lexer.Token, config Config) (*planfmt.Plan, error) {
	result, err := PlanWithObservability(events, tokens, config)
	if err != nil {
		return nil, err
	}
	return result.Plan, nil
}

// PlanWithObservability returns plan with telemetry and debug events
func PlanWithObservability(events []parser.Event, tokens []lexer.Token, config Config) (*PlanResult, error) {
	var telemetry *PlanTelemetry
	var debugEvents []DebugEvent

	// Always track planning time
	startTime := time.Now()

	// Initialize telemetry if enabled
	if config.Telemetry >= TelemetryBasic {
		telemetry = &PlanTelemetry{}
	}

	// Initialize debug events if enabled
	if config.Debug >= DebugPaths {
		debugEvents = make([]DebugEvent, 0, 100)
	}

	p := &planner{
		events:      events,
		tokens:      tokens,
		config:      config,
		pos:         0,
		stepID:      1,
		telemetry:   telemetry,
		debugEvents: debugEvents,
	}

	plan, err := p.plan()
	if err != nil {
		return nil, err
	}

	// Finalize telemetry
	planTime := time.Since(startTime)
	if telemetry != nil {
		telemetry.EventCount = len(events)
		telemetry.StepCount = countSteps(plan.Root)
	}

	return &PlanResult{
		Plan:        plan,
		PlanTime:    planTime,
		Telemetry:   telemetry,
		DebugEvents: p.debugEvents,
	}, nil
}

// planner holds state during planning
type planner struct {
	events []parser.Event
	tokens []lexer.Token
	config Config

	pos    int    // Current position in event stream
	stepID uint64 // Next step ID to assign

	// Observability
	telemetry   *PlanTelemetry
	debugEvents []DebugEvent
}

// recordDebugEvent records debug events when debug tracing is enabled
func (p *planner) recordDebugEvent(event, context string) {
	if p.config.Debug == DebugOff || p.debugEvents == nil {
		return
	}
	p.debugEvents = append(p.debugEvents, DebugEvent{
		Timestamp: time.Now(),
		Event:     event,
		EventPos:  p.pos,
		Context:   context,
	})
}

// plan is the main planning entry point
func (p *planner) plan() (*planfmt.Plan, error) {
	if p.config.Debug >= DebugPaths {
		p.recordDebugEvent("enter_plan", "target="+p.config.Target)
	}

	plan := &planfmt.Plan{
		Target: p.config.Target,
		Header: planfmt.PlanHeader{
			PlanKind: 0, // View plan
		},
	}

	// Command mode: find target function
	if p.config.Target != "" {
		step, err := p.planTargetFunction()
		if err != nil {
			return nil, err
		}
		plan.Root = step
	} else {
		// Script mode: plan all top-level commands
		step, err := p.planSource()
		if err != nil {
			return nil, err
		}
		plan.Root = step
	}

	// Canonicalize plan for deterministic output
	plan.Canonicalize()

	if p.config.Debug >= DebugPaths {
		stepCount := countSteps(plan.Root)
		p.recordDebugEvent("exit_plan", fmt.Sprintf("steps=%d", stepCount))
	}

	return plan, nil
}

// planTargetFunction finds and plans a specific function
func (p *planner) planTargetFunction() (*planfmt.Step, error) {
	if p.config.Debug >= DebugPaths {
		p.recordDebugEvent("enter_planTargetFunction", "target="+p.config.Target)
	}

	// Walk events to find the target function
	for p.pos < len(p.events) {
		evt := p.events[p.pos]

		if evt.Kind == parser.EventOpen && parser.NodeKind(evt.Data) == parser.NodeFunction {
			// Found a function, check if it's our target
			// Event structure: OPEN Function, TOKEN(fun), TOKEN(name), TOKEN(=), ...
			funcNamePos := p.pos + 2 // Skip OPEN Function and TOKEN(fun)
			if funcNamePos < len(p.events) && p.events[funcNamePos].Kind == parser.EventToken {
				funcNameTokenIdx := p.events[funcNamePos].Data
				funcName := string(p.tokens[funcNameTokenIdx].Text)

				if funcName == p.config.Target {
					if p.config.Debug >= DebugDetailed {
						p.recordDebugEvent("function_found", fmt.Sprintf("name=%s pos=%d", funcName, p.pos))
					}

					// Plan the function body
					return p.planFunctionBody()
				}
			}
		}

		p.pos++
	}

	return nil, fmt.Errorf("function not found: %s", p.config.Target)
}

// planFunctionBody plans the body of a function (assumes p.pos is at OPEN Function)
func (p *planner) planFunctionBody() (*planfmt.Step, error) {
	// Skip to function body (past OPEN Function, name token, '=' token)
	depth := 1
	p.pos++ // Move past OPEN Function

	for p.pos < len(p.events) && depth > 0 {
		evt := p.events[p.pos]

		if evt.Kind == parser.EventOpen {
			if parser.NodeKind(evt.Data) == parser.NodeShellCommand {
				// Found shell command in function body
				return p.planShellCommand()
			}
			depth++
		} else if evt.Kind == parser.EventClose {
			depth--
		}

		p.pos++
	}

	return nil, fmt.Errorf("no commands found in function body")
}

// planSource plans all top-level commands in script mode
func (p *planner) planSource() (*planfmt.Step, error) {
	if p.config.Debug >= DebugPaths {
		p.recordDebugEvent("enter_planSource", "script mode")
	}

	var steps []*planfmt.Step

	// Walk events looking for top-level shell commands
	depth := 0
	for p.pos < len(p.events) {
		prevPos := p.pos
		evt := p.events[p.pos]

		if evt.Kind == parser.EventOpen {
			nodeKind := parser.NodeKind(evt.Data)

			// Only plan shell commands at depth 1 (inside Source, not inside Function)
			if nodeKind == parser.NodeShellCommand && depth == 1 {
				step, err := p.planShellCommand()
				if err != nil {
					return nil, err
				}
				steps = append(steps, step)
				// planShellCommand already advanced p.pos, so continue without incrementing
				continue
			}

			depth++
		} else if evt.Kind == parser.EventClose {
			depth--
		}

		p.pos++

		// Assert progress
		if p.pos == prevPos {
			panic(fmt.Sprintf("planSource stuck at pos %d", p.pos))
		}
	}

	// If no steps, return nil (empty plan)
	if len(steps) == 0 {
		return nil, nil
	}

	// If single step, return it directly
	if len(steps) == 1 {
		return steps[0], nil
	}

	// Multiple steps: wrap in a sequence container
	sequence := &planfmt.Step{
		ID:       p.nextStepID(),
		Kind:     planfmt.KindDecorator,
		Op:       "sequence",
		Children: steps,
	}

	return sequence, nil
}

// planShellCommand plans a shell command (assumes p.pos is at OPEN ShellCommand)
func (p *planner) planShellCommand() (*planfmt.Step, error) {
	if p.config.Debug >= DebugDetailed {
		p.recordDebugEvent("enter_planShellCommand", fmt.Sprintf("pos=%d", p.pos))
	}

	startPos := p.pos
	p.pos++ // Move past OPEN ShellCommand

	// Collect all tokens in the shell command
	var commandTokens []string
	depth := 1

	for p.pos < len(p.events) && depth > 0 {
		evt := p.events[p.pos]

		if evt.Kind == parser.EventOpen {
			depth++
		} else if evt.Kind == parser.EventClose {
			depth--
			if depth == 0 {
				// Move past the CLOSE ShellCommand event
				p.pos++
				break
			}
		} else if evt.Kind == parser.EventToken {
			tokenIdx := evt.Data
			tokenText := string(p.tokens[tokenIdx].Text)
			commandTokens = append(commandTokens, tokenText)
		}

		p.pos++
	}

	// Build command string
	command := ""
	for i, tok := range commandTokens {
		if i > 0 {
			command += " "
		}
		command += tok
	}

	step := &planfmt.Step{
		ID:   p.nextStepID(),
		Kind: planfmt.KindDecorator,
		Op:   "shell",
		Args: []planfmt.Arg{
			{
				Key: "command",
				Val: planfmt.Value{
					Kind: planfmt.ValueString,
					Str:  command,
				},
			},
		},
	}

	if p.config.Debug >= DebugDetailed {
		p.recordDebugEvent("step_created", fmt.Sprintf("id=%d op=shell command=%q", step.ID, command))
	}

	// Assert position advanced
	if p.pos == startPos {
		panic(fmt.Sprintf("planner stuck at pos %d", p.pos))
	}

	return step, nil
}

// nextStepID returns the next step ID and increments the counter
func (p *planner) nextStepID() uint64 {
	id := p.stepID
	p.stepID++
	return id
}

// Helper to count steps in a plan
func countSteps(step *planfmt.Step) int {
	if step == nil {
		return 0
	}
	count := 1
	for _, child := range step.Children {
		count += countSteps(child)
	}
	return count
}
