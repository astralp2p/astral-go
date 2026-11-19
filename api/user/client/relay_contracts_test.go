package user

import (
	"errors"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/routing"
)

// relayContractsArgs mirrors astrald's declaration of user.relay_contracts, so
// a client that names the argument anything else leaves the required field
// unbound and the op never runs.
type relayContractsArgs struct {
	Issuer *astral.Identity `query:"required"`
	Out    string
}

// relayContractsNode records the bound issuer and answers replies, standing in
// for the node.
type relayContractsNode struct {
	bound   chan *astral.Identity
	replies []astral.Object
}

func (n *relayContractsNode) RelayContracts(_ *astral.Context, q *routing.IncomingQuery, args relayContractsArgs) error {
	n.bound <- args.Issuer

	ch := q.Accept()
	defer ch.Close()

	for _, o := range n.replies {
		if err := ch.Send(o); err != nil {
			return err
		}
	}
	return nil
}

// clientForRelayContracts routes the op under the literal name astrald serves,
// so the client's constant must match it.
func clientForRelayContracts(t *testing.T, node *relayContractsNode) *Client {
	t.Helper()

	ops := routing.NewOpRouter()
	op, err := routing.NewOp(node.RelayContracts)
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}
	if err = ops.AddOp("user.relay_contracts", op); err != nil {
		t.Fatalf("AddOp: %v", err)
	}

	id := astral.GenerateIdentity()
	return New(id, astrald.New(&localRouter{inner: ops, id: id}))
}

func relayContract(issuer *astral.Identity, n byte) *auth.SignedContract {
	return &auth.SignedContract{
		Contract: &auth.Contract{
			Issuer:    issuer,
			Subject:   astral.GenerateIdentity(),
			Permits:   []*auth.Permit{{Action: "mod.nodes.relay_for_action"}},
			ExpiresAt: astral.Time(time.Unix(1800000000, 0)),
		},
		IssuerSig:  &crypto.Signature{Scheme: "asn1", Data: []byte{n}},
		SubjectSig: &crypto.Signature{Scheme: "asn1", Data: []byte{n}},
	}
}

func objectID(t *testing.T, o astral.Object) *astral.ObjectID {
	t.Helper()
	id, err := astral.ResolveObjectID(o)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// The client sends issuer under the op's required argument and collects every
// contract up to eos, in order.
func TestRelayContractsCollectsUntilEOS(t *testing.T) {
	issuer := astral.GenerateIdentity()
	want := []*auth.SignedContract{relayContract(issuer, 1), relayContract(issuer, 2)}
	node := &relayContractsNode{
		bound:   make(chan *astral.Identity, 1),
		replies: []astral.Object{want[0], want[1], &astral.EOS{}},
	}

	got, err := clientForRelayContracts(t, node).RelayContracts(astral.NewContext(nil), issuer)
	if err != nil {
		t.Fatalf("RelayContracts: %v", err)
	}

	select {
	case bound := <-node.bound:
		if bound == nil || !bound.IsEqual(issuer) {
			t.Fatalf("op bound issuer %v; want %v", bound, issuer)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the op was never called")
	}

	if len(got) != len(want) {
		t.Fatalf("got %d contracts; want %d", len(got), len(want))
	}
	for i := range want {
		if !objectID(t, got[i]).IsEqual(objectID(t, want[i])) {
			t.Errorf("contract %d differs from the one sent", i)
		}
	}
}

// eos alone is an empty list and no error.
func TestRelayContractsEmpty(t *testing.T) {
	node := &relayContractsNode{bound: make(chan *astral.Identity, 1), replies: []astral.Object{&astral.EOS{}}}

	got, err := clientForRelayContracts(t, node).RelayContracts(astral.NewContext(nil), astral.GenerateIdentity())
	if err != nil || len(got) != 0 {
		t.Fatalf("got %d contracts, err %v; want none and no error", len(got), err)
	}
}

// A refusal the op sends reaches the caller as an error carrying its message.
func TestRelayContractsPassesTheRefusal(t *testing.T) {
	node := &relayContractsNode{bound: make(chan *astral.Identity, 1), replies: []astral.Object{auth.ErrNotPermitted}}

	_, err := clientForRelayContracts(t, node).RelayContracts(astral.NewContext(nil), astral.GenerateIdentity())
	var msg *astral.ErrorMessage
	if !errors.As(err, &msg) || err.Error() != auth.ErrNotPermitted.Error() {
		t.Fatalf("got %v; want %q", err, auth.ErrNotPermitted)
	}
}
