package decorators

import (
	"fmt"

	"github.com/aledsdavies/opal/core/types"
)

func init() {
	// Register the @var decorator as a value decorator
	// Usage: @var.name accesses variable "name"
	// Returns data with no side effects, can be interpolated in strings
	types.Global().RegisterValue("var", varHandler)
}

// varHandler implements the @var decorator
// Accesses variables from the context
func varHandler(ctx types.Context, args types.Args) (types.Value, error) {
	// @var requires a primary property (the variable name)
	if args.Primary == nil {
		return nil, fmt.Errorf("@var requires a variable name")
	}

	varName := (*args.Primary).(string)

	// Look up the variable in the context
	value, exists := ctx.Variables[varName]
	if !exists {
		return nil, fmt.Errorf("variable %q not found", varName)
	}

	return value, nil
}
