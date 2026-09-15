package nat

import (
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/ip"
	"github.com/astralp2p/astral-go/astral"
)

// newTestProtocol builds a protocol between two fresh identities, the state both
// punch paths start from.
func newTestProtocol(t *testing.T) *PunchProtocol {
	t.Helper()

	localIP, err := ip.ParseIP("198.51.100.7")
	if err != nil {
		t.Fatalf("parse local ip: %v", err)
	}

	return NewPunchProtocol(astral.GenerateIdentity(), astral.GenerateIdentity(), localIP)
}

// samplePunchResult is what a puncher reports for the peer it reached.
func samplePunchResult(t *testing.T) *PunchResult {
	t.Helper()

	remoteIP, err := ip.ParseIP("203.0.113.10")
	if err != nil {
		t.Fatalf("parse remote ip: %v", err)
	}

	return &PunchResult{LocalPort: 41000, RemoteIP: remoteIP, RemotePort: 52000}
}

// sampleResultSignal is the result signal the peer echoes back, carrying the
// hole nonce and the active endpoint.
func sampleResultSignal(t *testing.T) *PunchSignal {
	t.Helper()

	activeIP, err := ip.ParseIP("198.51.100.7")
	if err != nil {
		t.Fatalf("parse active ip: %v", err)
	}

	return &PunchSignal{
		Signal:    PunchSignalTypeResult,
		IP:        activeIP,
		Port:      41000,
		PairNonce: astral.NewNonce(),
	}
}

// assertCreatedAtSurvives encodes the hole and requires the decoded timestamp to
// fall inside the interval the hole was created in, where before and after
// bracket the SetPunchResult call.
//
// why: the interval is the load-bearing assertion, not nanosecond equality. The
// Go zero time's UnixNano overflows int64, and the overflowed integer is the one
// the codec writes, so an unstamped hole decodes as 1754-08-30 while comparing
// equal to itself in nanoseconds. Only a wall-clock bound rejects that.
func assertCreatedAtSurvives(t *testing.T, h *Hole, before, after time.Time) {
	t.Helper()

	data, err := astral.EncodeBytes(h)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, err := astral.DecodeAs[*Hole](data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	decoded := got.CreatedAt.Time()
	if decoded.Before(before) || decoded.After(after) {
		t.Fatalf("decoded CreatedAt %v outside the creation interval [%v, %v]", decoded, before, after)
	}

	if decoded.UnixNano() != h.CreatedAt.Time().UnixNano() {
		t.Fatalf("CreatedAt changed through the round trip: in=%v out=%v", h.CreatedAt, got.CreatedAt)
	}
}

func TestSetPunchResult_StampsCreatedAt(t *testing.T) {
	proto := newTestProtocol(t)

	before := time.Now()
	proto.SetPunchResult(samplePunchResult(t))
	after := time.Now()

	got := proto.Hole.CreatedAt.Time()
	if got.IsZero() {
		t.Fatal("CreatedAt is the zero time")
	}
	if got.Before(before) || got.After(after) {
		t.Fatalf("CreatedAt %v outside the creation interval [%v, %v]", got, before, after)
	}
}

// TestHole_ActivePath_CreatedAtSurvives replays the initiator's ordering:
// SetPunchResult, a locally generated nonce, then the peer's result signal.
func TestHole_ActivePath_CreatedAtSurvives(t *testing.T) {
	proto := newTestProtocol(t)

	before := time.Now()
	proto.SetPunchResult(samplePunchResult(t))
	after := time.Now()

	proto.Hole.Nonce = astral.NewNonce()
	proto.OnResult(sampleResultSignal(t))

	assertCreatedAtSurvives(t, &proto.Hole, before, after)
}

// TestHole_PassivePath_CreatedAtSurvives replays the responder's ordering:
// SetPunchResult, then the initiator's result signal.
func TestHole_PassivePath_CreatedAtSurvives(t *testing.T) {
	proto := newTestProtocol(t)

	before := time.Now()
	proto.SetPunchResult(samplePunchResult(t))
	after := time.Now()

	proto.OnResult(sampleResultSignal(t))

	assertCreatedAtSurvives(t, &proto.Hole, before, after)
}
