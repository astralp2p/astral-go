package user

import (
	"errors"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/routing"
)

// contractNode answers user.contract with reply, standing in for the node.
type contractNode struct {
	reply astral.Object
}

func (n *contractNode) Contract(_ *astral.Context, q *routing.IncomingQuery) error {
	ch := q.Accept()
	defer ch.Close()

	return ch.Send(n.reply)
}

// clientForContract routes the op under the literal name astrald serves, so the
// client's constant must match it.
func clientForContract(t *testing.T, node *contractNode) *Client {
	t.Helper()

	ops := routing.NewOpRouter()
	op, err := routing.NewOp(node.Contract)
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}
	if err = ops.AddOp("user.contract", op); err != nil {
		t.Fatalf("AddOp: %v", err)
	}

	id := astral.GenerateIdentity()
	return New(id, astrald.New(&localRouter{inner: ops, id: id}))
}

func TestContractReturnsTheSignedContract(t *testing.T) {
	issuer, subject := astral.GenerateIdentity(), astral.GenerateIdentity()
	want := &auth.SignedContract{
		Contract: &auth.Contract{
			Issuer:    issuer,
			Subject:   subject,
			Permits:   []*auth.Permit{{Action: "mod.user.see_swarm_action"}},
			ExpiresAt: astral.Time(time.Unix(1800000000, 0)),
		},
		IssuerSig:  &crypto.Signature{Scheme: "asn1", Data: []byte{1, 2, 3}},
		SubjectSig: &crypto.Signature{Scheme: "asn1", Data: []byte{4, 5, 6}},
	}

	got, err := clientForContract(t, &contractNode{reply: want}).Contract(astral.NewContext(nil))
	if err != nil {
		t.Fatalf("Contract: %v", err)
	}
	if got.IsNil() {
		t.Fatal("want the contract, got none")
	}

	gotID, err := astral.ResolveObjectID(got)
	if err != nil {
		t.Fatal(err)
	}
	wantID, err := astral.ResolveObjectID(want)
	if err != nil {
		t.Fatal(err)
	}
	if !gotID.IsEqual(wantID) {
		t.Errorf("contract %v; want %v", gotID, wantID)
	}
}

// A refusal the op sends reaches the caller as an error carrying its message.
func TestContractPassesTheRefusals(t *testing.T) {
	for _, refusal := range []astral.Error{user.ErrNoActiveContract, auth.ErrNotPermitted} {
		t.Run(refusal.Error(), func(t *testing.T) {
			got, err := clientForContract(t, &contractNode{reply: refusal}).Contract(astral.NewContext(nil))
			if err == nil {
				t.Fatalf("want the refusal, got %v", got)
			}
			if err.Error() != refusal.Error() {
				t.Fatalf("want %q, got %q", refusal, err)
			}
			var msg *astral.ErrorMessage
			if !errors.As(err, &msg) {
				t.Fatalf("want an *astral.ErrorMessage, got %T", err)
			}
		})
	}
}
