package routing

import (
	"time"

	"github.com/astralp2p/astral-go/astral"
)

// Report holds information about a finished op call
type Report struct {
	Query *astral.Query
	Time  time.Duration
	Err   error
}
