package decorators

import "github.com/aledsdavies/opal/core/types"

func init() {
	// Register the @env decorator
	// Usage: @env.HOME accesses environment variable "HOME"
	types.Global().Register("env")
}
