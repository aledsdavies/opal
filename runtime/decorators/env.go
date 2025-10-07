package decorators

import (
	"fmt"

	"github.com/aledsdavies/opal/core/types"
)

func init() {
	// Register the @env decorator as a value decorator
	// Usage: @env.HOME accesses environment variable "HOME"
	// Returns data with no side effects, can be interpolated in strings
	types.Global().RegisterValue("env", envHandler)
}

// envHandler implements the @env decorator
// Accesses environment variables from the context
func envHandler(ctx types.Context, args types.Args) (types.Value, error) {
	// @env requires a primary property (the env var name)
	if args.Primary == nil {
		return nil, fmt.Errorf("@env requires an environment variable name")
	}

	envVar := (*args.Primary).(string)

	// Look up the environment variable
	value, exists := ctx.Env[envVar]
	if !exists {
		// Check for default parameter
		if args.Params != nil {
			if defaultVal, hasDefault := args.Params["default"]; hasDefault {
				return defaultVal, nil
			}
		}
		return nil, fmt.Errorf("environment variable %q not found", envVar)
	}

	return value, nil
}
