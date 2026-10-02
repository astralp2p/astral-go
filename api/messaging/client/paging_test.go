package messaging

import (
	"errors"
	"io"
	"testing"

	"github.com/astralp2p/astral-go/api/messaging"
	"github.com/astralp2p/astral-go/astral"
)

func TestPageMessages_SendsOnlyWhatIsSet(t *testing.T) {
	client, router := answeringClient(t, &messaging.MessagePage{NextBefore: 7, Rev: 9}, &astral.EOS{})

	page, err := client.PageMessages(astral.NewContext(nil), messaging.PageMessagesRequest{Peer: "scout"})
	if err != nil {
		t.Fatal(err)
	}
	if page.NextBefore != 7 || page.Rev != 9 {
		t.Fatalf("page %+v", page)
	}
	assertQuery(t, router, messaging.MethodPageMessages, map[string]string{"peer": "scout"})
}

// A position travels with its generation, and a generation of zero is still
// sent: it is a value, not an absence.
func TestPageMessages_CarriesThePositionAndItsGeneration(t *testing.T) {
	client, router := answeringClient(t, &messaging.MessagePage{}, &astral.EOS{})
	gen := uint64(0)

	_, err := client.PageMessages(astral.NewContext(nil), messaging.PageMessagesRequest{
		List: messaging.ListArchive, Before: 40, Limit: 10, Generation: &gen, Mailbox: "scout",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertQuery(t, router, messaging.MethodPageMessages, map[string]string{
		"list": "archive", "before": "40", "limit": "10", "generation": "0", "mailbox": "scout",
	})
}

func TestListMessageChanges_AlwaysSendsSinceAndGeneration(t *testing.T) {
	client, router := answeringClient(t, &messaging.MessageChanges{NextRev: 3}, &astral.EOS{})

	if _, err := client.ListMessageChanges(astral.NewContext(nil), messaging.ListMessageChangesRequest{}); err != nil {
		t.Fatal(err)
	}
	assertQuery(t, router, messaging.MethodListMessageChanges, map[string]string{"since": "0", "generation": "0"})
}

func TestPageConversations_NamesOnePeer(t *testing.T) {
	client, router := answeringClient(t, &messaging.ConversationPage{}, &astral.EOS{})

	if _, err := client.PageConversations(astral.NewContext(nil), messaging.PageConversationsRequest{Peer: "scout"}); err != nil {
		t.Fatal(err)
	}
	assertQuery(t, router, messaging.MethodPageConversations, map[string]string{"peer": "scout"})
}

func TestListConversationChanges_SendsSince(t *testing.T) {
	client, router := answeringClient(t, &messaging.ConversationChanges{}, &astral.EOS{})

	if _, err := client.ListConversationChanges(astral.NewContext(nil), messaging.ListConversationChangesRequest{Since: 5, Generation: 2}); err != nil {
		t.Fatal(err)
	}
	assertQuery(t, router, messaging.MethodListConversationChanges, map[string]string{"since": "5", "generation": "2"})
}

// An answer is complete only at its eos: an object without one, two objects,
// or an eos alone is no page.
func TestPaging_OnlyOneObjectAndItsEOSIsAnAnswer(t *testing.T) {
	for name, tc := range map[string]struct {
		objects []astral.Object
		want    error
	}{
		"no eos":      {[]astral.Object{&messaging.MessagePage{}}, io.ErrUnexpectedEOF},
		"two answers": {[]astral.Object{&messaging.MessagePage{}, &messaging.MessagePage{}, &astral.EOS{}}, ErrNoAnswer},
		"eos alone":   {[]astral.Object{&astral.EOS{}}, ErrNoAnswer},
	} {
		t.Run(name, func(t *testing.T) {
			client, _ := answeringClient(t, tc.objects...)
			page, err := client.PageMessages(astral.NewContext(nil), messaging.PageMessagesRequest{})
			if !errors.Is(err, tc.want) || page != nil {
				t.Fatalf("answered %v, %v; want nil, %v", page, err, tc.want)
			}
		})
	}
}

func TestPaging_AnErrorObjectIsTheError(t *testing.T) {
	client, _ := answeringClient(t, astral.Err(errors.New(messaging.ErrGeneration)))
	_, err := client.ListMessageChanges(astral.NewContext(nil), messaging.ListMessageChangesRequest{})
	if err == nil || err.Error() != messaging.ErrGeneration {
		t.Fatalf("err %v, want the generation error", err)
	}
}
