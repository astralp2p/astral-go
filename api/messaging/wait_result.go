package messaging

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// WaitResult is what one MethodWait park came back with.
//
// Messages are the inbox rows the park found, without their bodies: a
// participant woken by a stranger reads the sender off the envelope rather
// than stamping the stranger's body read. NextSince is the Since to pass on the
// next wait, as NextSince computes it. TimedOut says the granted window closed
// with nothing new.
//
// Granted is the window the park was given: the caller's ask or the module's
// default, never over its ceiling. Waited is how long the node held the park;
// near zero means the answer was already waiting.
//
// why the grant is answered beside the wait: the ceiling is the deployment's
// and the ask the caller's, so a clamp reads as two numbers rather than as
// silence the caller misreads.
type WaitResult struct {
	Messages  []*Envelope
	NextSince astral.Uint64
	TimedOut  astral.Bool
	Granted   astral.Duration
	Waited    astral.Duration
}

// astral

var _ astral.Object = &WaitResult{}

func (r WaitResult) ObjectType() string { return "messaging.wait_result" }

func (r WaitResult) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *WaitResult) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

// json

func (r WaitResult) MarshalJSON() ([]byte, error) {
	type alias WaitResult
	return json.Marshal(alias(r))
}

func (r *WaitResult) UnmarshalJSON(bytes []byte) error {
	type alias WaitResult
	var v alias

	err := json.Unmarshal(bytes, &v)
	if err != nil {
		return err
	}

	*r = WaitResult(v)
	return nil
}

func init() {
	astral.MustAdd(&WaitResult{})
}
