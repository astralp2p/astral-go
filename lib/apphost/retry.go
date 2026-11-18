package apphost

import (
	"context"
	"errors"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
)

// IsRetryable reports whether a failed query may succeed if repeated: true for
// a rejection with CodeUnavailable or CodeTimeout and for a timeout, false for
// every other error. A caller retries with backoff and bounds its retries.
//
// note: route_not_found is not retryable: the caller discovers the target
// again or stops. A refusal and target_not_allowed are final.
// See astral-docs core-definitions/query.md, Routing outcomes.
func IsRetryable(err error) bool {
	var rejected *astral.ErrRejected
	var msg *apphost.ErrorMsg
	switch {
	case errors.As(err, &rejected):
		return rejected.Code == astral.CodeUnavailable || rejected.Code == astral.CodeTimeout
	case errors.As(err, &msg):
		return msg.Code == apphost.ErrCodeTimeout
	default:
		return errors.Is(err, astral.ErrTimeout) || errors.Is(err, context.DeadlineExceeded)
	}
}
