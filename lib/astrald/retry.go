package astrald

import (
	libapphost "github.com/astralp2p/astral-go/lib/apphost"
	"github.com/astralp2p/astral-go/sig"
)

// SetRetry wraps the default client's router in a RetryRouter with the given retry policy.
// This affects all outbound queries made via Default(), including all mod/*/client packages.
func SetRetry(r *sig.Retry) {
	SetDefault(New(NewRetryRouter(libapphost.DefaultRouter(), r)))
}
