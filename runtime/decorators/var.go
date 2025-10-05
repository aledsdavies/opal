package decorators

import "github.com/aledsdavies/opal/core/types"

func init() {
	// Register the @var decorator
	// Usage: @var.name accesses variable "name"
	types.Global().Register("var")
}
