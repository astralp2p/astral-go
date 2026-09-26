package messaging

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// A read of the caller's own mailbox and a delegated read are two requests. The
// pointer is what tells them apart, so an unset mailbox and a named one must
// cross either wire as they left.
func TestReadMessagesRequest_MailboxSurvives(t *testing.T) {
	mailbox := astral.GenerateIdentity()
	refs := []*MessageRef{{Box: BoxInbox, ID: NewMessageID()}}

	for name, trip := range map[string]func(*testing.T, astral.Object) astral.Object{
		"binary": binaryRoundTrip,
		"json":   jsonRoundTrip,
	} {
		t.Run(name, func(t *testing.T) {
			own := trip(t, &ReadMessagesRequest{Refs: refs}).(*ReadMessagesRequest)
			if own.Mailbox != nil {
				t.Fatalf("an unset mailbox decoded as %v", own.Mailbox)
			}

			named := trip(t, &ReadMessagesRequest{Refs: refs, Mailbox: mailbox}).(*ReadMessagesRequest)
			if named.Mailbox == nil || !named.Mailbox.IsEqual(mailbox) {
				t.Fatalf("mailbox: want %v, got %v", mailbox, named.Mailbox)
			}
			if len(named.Refs) != 1 || *named.Refs[0] != *refs[0] {
				t.Fatalf("refs: want %v, got %v", refs, named.Refs)
			}
		})
	}
}

// The frame is positional, so Mailbox is appended after the fields the request
// carried before it: a frame is exactly those fields as they were written
// before, followed by the mailbox.
func TestReadMessagesRequest_MailboxIsTheLastField(t *testing.T) {
	before := struct {
		Refs        []*MessageRef
		Children    astral.String8
		MaxChildren astral.Uint16
	}{[]*MessageRef{{Box: BoxOutbox, ID: NewMessageID()}}, ChildrenFull, 3}

	for name, mailbox := range map[string]*astral.Identity{
		"unset": nil,
		"named": astral.GenerateIdentity(),
	} {
		t.Run(name, func(t *testing.T) {
			after := struct{ Mailbox *astral.Identity }{mailbox}

			var want bytes.Buffer
			if _, err := astral.Objectify(&before).WriteTo(&want); err != nil {
				t.Fatalf("write the earlier fields: %v", err)
			}
			if _, err := astral.Objectify(&after).WriteTo(&want); err != nil {
				t.Fatalf("write the mailbox: %v", err)
			}

			req := &ReadMessagesRequest{
				Refs:        before.Refs,
				Children:    before.Children,
				MaxChildren: before.MaxChildren,
				Mailbox:     mailbox,
			}
			var got bytes.Buffer
			if _, err := req.WriteTo(&got); err != nil {
				t.Fatalf("write: %v", err)
			}

			if !bytes.Equal(got.Bytes(), want.Bytes()) {
				t.Fatalf("the frame moved:\nwant %x\n got %x", want.Bytes(), got.Bytes())
			}
		})
	}
}
