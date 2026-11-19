package nat

import (
	"bytes"
	"fmt"

	"github.com/astralp2p/astral-go/api/ip"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// PunchProtocol is a pure state machine for NAT traversal protocol.
type PunchProtocol struct {
	Session []byte

	LocalIdentity *astral.Identity
	PeerIdentity  *astral.Identity

	// LocalIP and LocalPort are the endpoint this side advertises to the peer.
	LocalIP   ip.IP
	LocalPort astral.Uint16
	// LocalBindPort is the local UDP port of this side's punch socket.
	LocalBindPort astral.Uint16

	PeerIP       ip.IP
	PeerPort     astral.Uint16
	PeerBindPort astral.Uint16

	Hole Hole
}

func NewPunchProtocol(localID, peerID *astral.Identity, localIP ip.IP) *PunchProtocol {
	return &PunchProtocol{
		LocalIdentity: localID,
		PeerIdentity:  peerID,
		LocalIP:       localIP,
	}
}

// Announce sets the endpoint this side advertises in its offer or answer.
//
// The punch socket's local port bindPort is always the bind port. The advertised
// endpoint is the IPv4 endpoint reflector observes for the socket when reflector
// reports one within ReflectTimeout, and LocalIP with bindPort otherwise. The
// returned error is the reflection's failure, for the caller to log; the
// protocol stays usable either way.
func (t *PunchProtocol) Announce(ctx *astral.Context, bindPort int, reflector Reflector) error {
	t.LocalBindPort = astral.Uint16(bindPort)
	t.LocalPort = astral.Uint16(bindPort)
	if reflector == nil {
		return nil
	}

	rctx, cancel := ctx.WithTimeout(ReflectTimeout)
	defer cancel()

	ext, err := reflector.Reflect(rctx)
	switch {
	case err != nil:
		return err
	case ext == nil || !ext.IP.IsIPv4():
		return fmt.Errorf("reflected endpoint %v is not IPv4", ext)
	}

	// why: under endpoint-independent mapping the reflected port is the one
	// the peer reaches; the local port is never seen behind a translating NAT.
	t.LocalIP = ext.IP
	t.LocalPort = ext.Port
	return nil
}

// OnOffer records the peer's advertised endpoint, bind port, and session from an incoming offer signal.
func (t *PunchProtocol) OnOffer(sig *PunchSignal) {
	t.PeerIP = sig.IP
	t.PeerPort = sig.Port
	t.PeerBindPort = sig.BindPort
	t.Session = sig.Session
}

// OnAnswer records the peer's advertised endpoint and bind port from an answer signal.
func (t *PunchProtocol) OnAnswer(sig *PunchSignal) {
	t.PeerIP = sig.IP
	t.PeerPort = sig.Port
	t.PeerBindPort = sig.BindPort
}

// OnResult records the hole nonce, the active endpoint the peer observed, and
// the peer's bind port from a result signal.
func (t *PunchProtocol) OnResult(sig *PunchSignal) {
	t.Hole.Nonce = sig.PairNonce
	t.Hole.ActiveEndpoint = Endpoint{IP: sig.IP, Port: sig.Port}
	t.Hole.PassiveBindPort = sig.BindPort
}

// ExpectSignal returns a handler that accepts only the given signalType and, for all types
// except Offer, validates that the incoming session matches the established session.
func (t *PunchProtocol) ExpectSignal(signalType astral.String8, on func(*PunchSignal)) func(*PunchSignal) error {
	return func(sig *PunchSignal) error {
		if sig.Signal != signalType {
			return fmt.Errorf("expected %s, got %s", signalType, sig.Signal)
		}
		// Validate session for all signals except Offer (which establishes the session)
		if signalType != PunchSignalTypeOffer && !bytes.Equal(sig.Session, t.Session) {
			return fmt.Errorf("session mismatch")
		}
		if on != nil {
			on(sig)
		}
		return channel.ErrBreak
	}
}

// SetPunchResult populates the Hole with active/passive role assignments, the
// remote endpoint observed during punching, both bind ports, and the creation
// timestamp; must be called before ResultSignal.
func (t *PunchProtocol) SetPunchResult(result *PunchResult) {
	t.Hole = Hole{
		ActiveIdentity: t.LocalIdentity,
		// why: behind a NAT that translates ports the socket's local port is not
		// the observed one; the hole rebinds the local port to keep the mapping.
		ActiveBindPort:  result.LocalPort,
		PassiveIdentity: t.PeerIdentity,
		PassiveEndpoint: Endpoint{IP: result.RemoteIP, Port: result.RemotePort},
		PassiveBindPort: t.PeerBindPort,
		// why: the punch result is the moment the hole exists, and it is the one
		// construction point both the active and the passive path share. A zero
		// astral.Time encodes as UnixNano out of range and decodes as 1754.
		CreatedAt: astral.Now(),
	}
}

func (t *PunchProtocol) OfferSignal() *PunchSignal {
	return &PunchSignal{
		Signal:   PunchSignalTypeOffer,
		Session:  t.Session,
		IP:       t.LocalIP,
		Port:     t.LocalPort,
		BindPort: t.LocalBindPort,
	}
}

func (t *PunchProtocol) AnswerSignal() *PunchSignal {
	return &PunchSignal{
		Signal:   PunchSignalTypeAnswer,
		Session:  t.Session,
		IP:       t.LocalIP,
		Port:     t.LocalPort,
		BindPort: t.LocalBindPort,
	}
}

func (t *PunchProtocol) ReadySignal() *PunchSignal {
	return &PunchSignal{
		Signal:  PunchSignalTypeReady,
		Session: t.Session,
	}
}

func (t *PunchProtocol) GoSignal() *PunchSignal {
	return &PunchSignal{
		Signal:  PunchSignalTypeGo,
		Session: t.Session,
	}
}

func (t *PunchProtocol) ResultSignal() *PunchSignal {
	return &PunchSignal{
		Signal:    PunchSignalTypeResult,
		Session:   t.Session,
		IP:        t.Hole.PassiveEndpoint.IP,
		Port:      t.Hole.PassiveEndpoint.Port,
		BindPort:  t.Hole.ActiveBindPort,
		PairNonce: t.Hole.Nonce,
	}
}
