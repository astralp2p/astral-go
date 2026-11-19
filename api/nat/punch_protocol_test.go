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

func mustIP(t *testing.T, s string) ip.IP {
	t.Helper()

	addr, err := ip.ParseIP(s)
	if err != nil {
		t.Fatalf("parse ip %q: %v", s, err)
	}
	return addr
}

// TestHole_BindPortsSurviveTheExchange replays a punch where the initiator sits
// behind a NAT that maps its socket 38826 to 6635 and the participant is public
// on 55001. Each side's hole binds its own local port and addresses the other
// side's external endpoint.
func TestHole_BindPortsSurviveTheExchange(t *testing.T) {
	initiatorID, participantID := astral.GenerateIdentity(), astral.GenerateIdentity()
	natIP, publicIP := mustIP(t, "212.105.131.237"), mustIP(t, "45.59.119.13")

	initiator := NewPunchProtocol(initiatorID, participantID, natIP)
	initiator.LocalPort, initiator.LocalBindPort = 38826, 38826
	initiator.Session = make([]byte, 16)

	participant := NewPunchProtocol(participantID, initiatorID, publicIP)
	participant.LocalPort, participant.LocalBindPort = 55001, 55001

	participant.OnOffer(initiator.OfferSignal())
	initiator.OnAnswer(participant.AnswerSignal())

	// the participant sees the initiator's probes from the mapped port 6635
	participant.SetPunchResult(&PunchResult{LocalPort: 55001, RemoteIP: natIP, RemotePort: 6635})
	initiator.SetPunchResult(&PunchResult{LocalPort: 38826, RemoteIP: publicIP, RemotePort: 55001})
	initiator.Hole.Nonce = astral.NewNonce()

	participant.OnResult(initiator.ResultSignal())
	initiator.OnResult(participant.ResultSignal())

	sides := []struct {
		name       string
		hole       *Hole
		self       *astral.Identity
		local      Endpoint
		localBind  int
		remote     Endpoint
		remoteBind int
	}{
		{"initiator", &initiator.Hole, initiatorID, Endpoint{natIP, 6635}, 38826, Endpoint{publicIP, 55001}, 55001},
		{"participant", &participant.Hole, participantID, Endpoint{publicIP, 55001}, 55001, Endpoint{natIP, 6635}, 38826},
	}
	for _, s := range sides {
		t.Run(s.name, func(t *testing.T) {
			if got := s.hole.GetLocalBindPort(s.self); got != s.localBind {
				t.Errorf("local bind port = %d; want %d", got, s.localBind)
			}
			if got := s.hole.GetRemoteBindPort(s.self); got != s.remoteBind {
				t.Errorf("remote bind port = %d; want %d", got, s.remoteBind)
			}
			if got, want := s.hole.GetLocalAddr(s.self).String(), s.local.UDPAddr().String(); got != want {
				t.Errorf("local external endpoint = %v; want %v", got, want)
			}
			if got, want := s.hole.GetRemoteAddr(s.self).String(), s.remote.UDPAddr().String(); got != want {
				t.Errorf("remote external endpoint = %v; want %v", got, want)
			}
		})
	}

	if initiator.Hole.Nonce != participant.Hole.Nonce {
		t.Errorf("nonce differs: initiator %v, participant %v", initiator.Hole.Nonce, participant.Hole.Nonce)
	}
}

// TestHole_BindPortsSurviveEncoding requires the bind ports to cross the wire.
func TestHole_BindPortsSurviveEncoding(t *testing.T) {
	h := &Hole{ActiveBindPort: 38826, PassiveBindPort: 55001, CreatedAt: astral.Now()}

	data, err := astral.EncodeBytes(h)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	got, err := astral.DecodeAs[*Hole](data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ActiveBindPort != 38826 || got.PassiveBindPort != 55001 {
		t.Fatalf("bind ports = %d/%d; want 38826/55001", got.ActiveBindPort, got.PassiveBindPort)
	}

	sig, err := astral.EncodeBytes(&PunchSignal{Signal: PunchSignalTypeOffer, Port: 6635, BindPort: 38826})
	if err != nil {
		t.Fatalf("encode signal: %v", err)
	}
	gotSig, err := astral.DecodeAs[*PunchSignal](sig)
	if err != nil {
		t.Fatalf("decode signal: %v", err)
	}
	if gotSig.Port != 6635 || gotSig.BindPort != 38826 {
		t.Fatalf("signal ports = %d/%d; want 6635/38826", gotSig.Port, gotSig.BindPort)
	}
}
