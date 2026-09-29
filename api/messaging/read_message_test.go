package messaging

import (
	"encoding/json"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// A withheld body and an empty one are two answers. The pointer is what tells
// them apart, so both must cross either wire as they left.
func TestReadMessage_AWithheldBodyIsNotAnEmptyOne(t *testing.T) {
	for name, trip := range map[string]func(*testing.T, astral.Object) astral.Object{
		"binary": binaryRoundTrip,
		"json":   jsonRoundTrip,
	} {
		t.Run(name, func(t *testing.T) {
			withheld := trip(t, &ReadMessage{Envelope: sampleStoredMessage().Envelope(), Truncated: true}).(*ReadMessage)
			if withheld.Content != nil {
				t.Fatalf("a withheld body decoded as %q", *withheld.Content)
			}
			if !withheld.Truncated {
				t.Fatal("truncated did not survive")
			}

			empty := trip(t, &ReadMessage{Envelope: sampleStoredMessage().Envelope(), Content: ptrString32("")}).(*ReadMessage)
			if empty.Content == nil {
				t.Fatal("an empty body decoded as a withheld one")
			}
		})
	}
}

// A read names every reply id whatever it carried of the replies, so the ids
// survive beside a body left out.
func TestReadMessage_ChildIDsSurvive(t *testing.T) {
	src := &ReadMessage{
		Envelope: sampleStoredMessage().Envelope(),
		ChildIDs: []MessageID{NewMessageID(), NewMessageID(), NewMessageID()},
	}

	dst := binaryRoundTrip(t, src).(*ReadMessage)
	if len(dst.ChildIDs) != len(src.ChildIDs) {
		t.Fatalf("child ids: want %v, got %v", src.ChildIDs, dst.ChildIDs)
	}
	for i := range src.ChildIDs {
		if dst.ChildIDs[i] != src.ChildIDs[i] {
			t.Fatalf("child %v: want %v, got %v", i, src.ChildIDs[i], dst.ChildIDs[i])
		}
	}
}

// An answer that found nothing is still an answer: its lists decode empty,
// whether they left nil or empty.
func TestReadMessagesResult_EmptyListsDecodeEmpty(t *testing.T) {
	for name, src := range map[string]*ReadMessagesResult{
		"nil":   {},
		"empty": {Messages: []*ReadMessage{}, Replies: []*ReadMessage{}, NotFound: []*MessageRef{}},
	} {
		t.Run(name, func(t *testing.T) {
			for _, got := range []astral.Object{binaryRoundTrip(t, src), jsonRoundTrip(t, src)} {
				dst := got.(*ReadMessagesResult)
				if len(dst.Messages)+len(dst.Replies)+len(dst.NotFound) != 0 {
					t.Fatalf("want nothing, got %+v", dst)
				}
			}
		})
	}
}

func TestReadMessagesResult_NotFoundNamesTheRow(t *testing.T) {
	ref := &MessageRef{Box: BoxOutbox, ID: NewMessageID()}

	dst := jsonRoundTrip(t, &ReadMessagesResult{NotFound: []*MessageRef{ref}}).(*ReadMessagesResult)
	if len(dst.NotFound) != 1 || *dst.NotFound[0] != *ref {
		t.Fatalf("not found: want [%+v], got %+v", *ref, dst.NotFound)
	}
}

// astral's slice codec writes an empty slice as [], and a reader in another
// language may reject null where it expects a list.
func TestMessaging_AnEmptySliceMarshalsAsAList(t *testing.T) {
	for name, tc := range map[string]struct {
		obj    json.Marshaler
		fields []string
	}{
		"read_message":          {ReadMessage{}, []string{"ChildIDs"}},
		"read_messages_request": {ReadMessagesRequest{}, []string{"Refs"}},
		"read_messages_result":  {ReadMessagesResult{}, []string{"Messages", "Replies", "NotFound"}},
		"wait_result":           {WaitResult{}, []string{"Messages"}},
	} {
		t.Run(name, func(t *testing.T) {
			raw, err := tc.obj.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(raw, &fields); err != nil {
				t.Fatal(err)
			}
			for _, k := range tc.fields {
				if got := string(fields[k]); got != "[]" {
					t.Errorf("%s marshals as %s, want []", k, got)
				}
			}
		})
	}
}
