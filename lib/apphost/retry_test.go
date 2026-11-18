package apphost

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
)

// IsRetryable answers true only for the causes a repeat can cure: the next hop
// was unreachable (7) or the query was not answered in time.
func TestIsRetryable(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"rejected 7", astral.NewErrRejected(astral.CodeUnavailable), true},
		{"rejected 6", astral.NewErrRejected(astral.CodeTimeout), true},
		{"wrapped rejected 7", fmt.Errorf("query: %w", astral.NewErrRejected(astral.CodeUnavailable)), true},
		{"timeout error_msg", &apphost.ErrorMsg{Code: apphost.ErrCodeTimeout}, true},
		{"astral timeout", astral.ErrTimeout, true},
		{"deadline", context.DeadlineExceeded, true},
		{"rejected 1", astral.NewErrRejected(astral.CodeRejected), false},
		{"rejected 9", astral.NewErrRejected(9), false},
		{"route_not_found error_msg", &apphost.ErrorMsg{Code: apphost.ErrCodeRouteNotFound}, false},
		{"route not found", astral.NewErrRouteNotFound(), false},
		{"target_not_allowed error_msg", &apphost.ErrorMsg{Code: apphost.ErrCodeTargetNotAllowed}, false},
		{"target not allowed", astral.ErrTargetNotAllowed, false},
		{"canceled", context.Canceled, false},
		{"other", errors.New("boom"), false},
		{"nil", nil, false},
	}
	for _, c := range cases {
		if got := IsRetryable(c.err); got != c.want {
			t.Errorf("%s: IsRetryable(%v) = %v, want %v", c.name, c.err, got, c.want)
		}
	}
}
