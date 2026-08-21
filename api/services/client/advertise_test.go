package services

import (
	"bytes"
	"errors"
	"io"
	"testing"

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

func sentTypes(t *testing.T, buf *bytes.Buffer) []string {
	t.Helper()
	r := channel.NewReceiver(bytes.NewReader(buf.Bytes()))
	var types []string
	for {
		o, err := r.Receive()
		if errors.Is(err, io.EOF) {
			return types
		}
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		types = append(types, o.ObjectType())
	}
}

func TestAdvertisementStandsOnAck(t *testing.T) {
	in := encode(t, &astral.Ack{})
	var out bytes.Buffer

	ad, err := newAdvertisement(channel.New(channel.Join(in, &out)), nil)
	if err != nil {
		t.Fatalf("newAdvertisement: %v", err)
	}
	if ad == nil {
		t.Fatal("expected an advertisement")
	}

	// nothing is sent for an advertisement carrying no info: the name travelled
	// in the query
	if got := sentTypes(t, &out); len(got) != 0 {
		t.Fatalf("sent %v, expected nothing", got)
	}
}

func TestInitialInfoIsSentOnce(t *testing.T) {
	in := encode(t, &astral.Ack{})
	var out bytes.Buffer

	_, err := newAdvertisement(channel.New(channel.Join(in, &out)), &astral.Bundle{})
	if err != nil {
		t.Fatalf("newAdvertisement: %v", err)
	}

	got := sentTypes(t, &out)
	if len(got) != 1 || got[0] != (&astral.Bundle{}).ObjectType() {
		t.Fatalf("sent %v, expected one bundle", got)
	}
}

func TestSetInfoSendsAgainWithoutWithdrawing(t *testing.T) {
	in := encode(t, &astral.Ack{})
	var out bytes.Buffer

	ad, err := newAdvertisement(channel.New(channel.Join(in, &out)), &astral.Bundle{})
	if err != nil {
		t.Fatalf("newAdvertisement: %v", err)
	}

	if err = ad.SetInfo(&astral.Bundle{}); err != nil {
		t.Fatalf("SetInfo: %v", err)
	}

	// two bundles and nothing else — a changed info is not a withdrawal
	got := sentTypes(t, &out)
	if len(got) != 2 {
		t.Fatalf("sent %v, expected two bundles", got)
	}
	for _, typ := range got {
		if typ != (&astral.Bundle{}).ObjectType() {
			t.Fatalf("sent %v, expected bundles only", got)
		}
	}
}

func TestNodeRefusalIsReturnedNotSwallowed(t *testing.T) {
	in := encode(t, astral.NewError("name is required"))
	var out bytes.Buffer

	ad, err := newAdvertisement(channel.New(channel.Join(in, &out)), nil)
	if err == nil {
		t.Fatal("expected the node's refusal to surface")
	}
	if ad != nil {
		t.Fatal("expected no advertisement after a refusal")
	}
}
