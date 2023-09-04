// Package pitays only exists to register the pitaya extension
package pitaya

import (
	"go.k6.io/k6/js/modules"
)

// Register the extension on module initialization, available to
// import from JS as "k6/x/pitaya".
func init() {
	modules.Register("k6/x/pitaya", new(RootModule))
}
