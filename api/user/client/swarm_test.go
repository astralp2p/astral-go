package user

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

func encode(t *testing.T, objects ...astral.Object) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	s := channel.NewSender(&buf)
	for _, o := range objects {
		if err := s.Send(o); err != nil {
			t.Fatalf("encode %s: %v", o.ObjectType(), err)
		}
	}
	return &buf
}

func identity(t *testing.T, hex string) *astral.Identity {
	t.Helper()
	id, err := astral.ParseIdentity(hex)
	if err != nil {
		t.Fatalf("parse %s: %v", hex, err)
	}
	return id
}

const (
	phone  = "026923d06a51098170093fe989d30a432283f56d89d307176fd6f947c3a9d285ff"
	laptop = "02bef8840eb35ef2ae3c83c07cb5779278904f99cb4103f71e37cc69931ae5e15f"
)

// The stream is terminated by eos, and every member before it belongs to the
// answer. A reader that stopped early would under-report the swarm, which for a
// caller enumerating the user's nodes means silently missing one.
func TestReadSwarmMembersCollectsUntilEOS(t *testing.T) {
	in := encode(t,
		&user.SwarmMember{Identity: identity(t, phone), Alias: "phone", Linked: true},
		&user.SwarmMember{Identity: identity(t, laptop), Alias: "laptop", Linked: false},
		&astral.EOS{},
	)
	var out bytes.Buffer

	members, err := readSwarmMembers(channel.New(channel.Join(in, &out)))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("members: got %d, want 2", len(members))
	}
	if members[0].Alias != "phone" || !bool(members[0].Linked) {
		t.Errorf("first member: got alias=%q linked=%v", members[0].Alias, members[0].Linked)
	}
	if members[1].Alias != "laptop" || bool(members[1].Linked) {
		t.Errorf("second member: got alias=%q linked=%v", members[1].Alias, members[1].Linked)
	}
}

// An empty swarm is an empty answer rather than an error: a node that is the
// only one its user holds reports nothing and is not broken.
func TestReadSwarmMembersEmpty(t *testing.T) {
	var out bytes.Buffer

	members, err := readSwarmMembers(channel.New(channel.Join(encode(t, &astral.EOS{}), &out)))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(members) != 0 {
		t.Errorf("members: got %d, want 0", len(members))
	}
}

// The op answers an error_message where it cannot report, and that has to reach
// the caller as an error rather than as an empty swarm — the two mean opposite
// things to anything enumerating the user's nodes.
func TestReadSwarmMembersSurfacesAnError(t *testing.T) {
	var out bytes.Buffer
	msg := astral.NewError("no active contract")

	members, err := readSwarmMembers(channel.New(channel.Join(encode(t, msg), &out)))
	if err == nil {
		t.Fatal("read: want an error, got nil")
	}
	if len(members) != 0 {
		t.Errorf("members: got %d, want none alongside an error", len(members))
	}
}

func TestReadSiblingsCollectsUntilEOS(t *testing.T) {
	in := encode(t, identity(t, phone), identity(t, laptop), &astral.EOS{})
	var out bytes.Buffer

	siblings, err := readSiblings(channel.New(channel.Join(in, &out)))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(siblings) != 2 {
		t.Fatalf("siblings: got %d, want 2", len(siblings))
	}
	if !siblings[0].IsEqual(identity(t, phone)) || !siblings[1].IsEqual(identity(t, laptop)) {
		t.Errorf("siblings: got %v", siblings)
	}
}
