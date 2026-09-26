package messaging

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/messaging"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/query"
)

// fakeRouter records the query it routes and what the client writes on it,
// and answers with pre-encoded objects.
type fakeRouter struct {
	queryString string
	sent        bytes.Buffer
	answer      io.Reader
	onClose     func() // nil = closing the query does nothing
}

func (r *fakeRouter) RouteQuery(_ *astral.Context, q *astral.InFlightQuery) (astral.Conn, error) {
	r.queryString = q.QueryString.String()
	return query.NewConn(nil, nil, closeHook{&r.sent, r.onClose}, r.answer, true), nil
}

func (r *fakeRouter) GuestID() *astral.Identity { return nil }
func (r *fakeRouter) HostID() *astral.Identity  { return nil }

type closeHook struct {
	io.Writer
	fn func()
}

func (c closeHook) Close() error {
	if c.fn != nil {
		c.fn()
	}
	return nil
}

func answeringClient(t *testing.T, objects ...astral.Object) (*Client, *fakeRouter) {
	t.Helper()

	var buf bytes.Buffer
	s := channel.NewSender(&buf)
	for _, o := range objects {
		if err := s.Send(o); err != nil {
			t.Fatalf("encode %s: %v", o.ObjectType(), err)
		}
	}

	router := &fakeRouter{answer: &buf}
	return New(nil, astrald.New(router)), router
}

// assertQuery fails unless the client queried path with exactly params.
func assertQuery(t *testing.T, router *fakeRouter, path string, params map[string]string) {
	t.Helper()

	gotPath, gotParams := query.Parse(router.queryString)
	if gotPath != path {
		t.Fatalf("path: want %v, got %q", path, gotPath)
	}
	if len(gotParams) != len(params) {
		t.Fatalf("params: want %v, got %v", params, gotParams)
	}
	for k, v := range params {
		if gotParams[k] != v {
			t.Fatalf("%v: want %q, got %q in %v", k, v, gotParams[k], gotParams)
		}
	}
}

// sentObject decodes the one object the client wrote after the query opened.
func sentObject(t *testing.T, router *fakeRouter) astral.Object {
	t.Helper()

	obj, err := channel.NewReceiver(&router.sent).Receive()
	if err != nil {
		t.Fatalf("decode the request body: %v", err)
	}
	if router.sent.Len() != 0 {
		t.Fatalf("%v bytes follow the request body", router.sent.Len())
	}
	return obj
}

func ctx() *astral.Context { return astral.NewContext(nil) }

// The literals are spelled out rather than taken from the Method constants,
// because a test that reads the same constant as the code cannot catch the
// constant being wrong.

func TestCreateIdentity_SendsOnlyWhatIsSet(t *testing.T) {
	want := &messaging.IdentityCredential{Identity: astral.GenerateIdentity(), Token: "t0k3n"}

	c, router := answeringClient(t, want)
	got, err := c.CreateIdentity(ctx(), "", 0)
	if err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}
	assertQuery(t, router, "messaging.create_identity", nil)
	if got.Token != want.Token || !got.Identity.IsEqual(want.Identity) {
		t.Fatalf("credential: want %+v, got %+v", want, got)
	}

	c, router = answeringClient(t, want)
	if _, err = c.CreateIdentity(ctx(), "scout", astral.Duration(time.Hour)); err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}
	assertQuery(t, router, "messaging.create_identity", map[string]string{"alias": "scout", "duration": "1h0m0s"})
}

func TestIdentity_NamesTheIdentity(t *testing.T) {
	want := &messaging.IdentityInfo{Identity: astral.GenerateIdentity(), Alias: "scout"}

	c, router := answeringClient(t, want)
	got, err := c.Identity(ctx(), "scout")
	if err != nil {
		t.Fatalf("Identity: %v", err)
	}
	assertQuery(t, router, "messaging.identity", map[string]string{"identity": "scout"})
	if got.Alias != want.Alias {
		t.Fatalf("alias: want %v, got %v", want.Alias, got.Alias)
	}
}

func TestDeleteIdentity_NamesTheIdentityAndExpectsAnAck(t *testing.T) {
	c, router := answeringClient(t, &astral.Ack{})
	if err := c.DeleteIdentity(ctx(), "scout"); err != nil {
		t.Fatalf("DeleteIdentity: %v", err)
	}
	assertQuery(t, router, "messaging.delete_identity", map[string]string{"identity": "scout"})
}

// An unknown participant arrives as an error object, not as an ack.
func TestDeleteIdentity_NodeErrorIsReturned(t *testing.T) {
	c, _ := answeringClient(t, astral.NewError("identity not found"))

	err := c.DeleteIdentity(ctx(), "scout")
	if err == nil || err.Error() != "identity not found" {
		t.Fatalf("want the node's error, got %v", err)
	}
}

func TestSendMessage_SendsTheRequestAsTheBody(t *testing.T) {
	minted := messaging.NewMessageID()
	req := &messaging.SendMessageRequest{To: "scout", Content: "hello", ParentID: messaging.NewMessageID()}

	c, router := answeringClient(t, &minted)
	id, err := c.SendMessage(ctx(), req)
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if id != minted {
		t.Fatalf("id: want %v, got %v", minted, id)
	}
	assertQuery(t, router, "messaging.send_message", nil)

	body, ok := sentObject(t, router).(*messaging.SendMessageRequest)
	if !ok {
		t.Fatal("the body is not a send_message_request")
	}
	if *body != *req {
		t.Fatalf("body: want %+v, got %+v", *req, *body)
	}
}

func TestListMessages_SendsOnlyWhatIsSet(t *testing.T) {
	rows := []astral.Object{&messaging.Envelope{Cursor: 3}, &messaging.Envelope{Cursor: 5}, &astral.EOS{}}

	c, router := answeringClient(t, rows...)
	list, err := c.ListMessages(ctx(), messaging.ListMessagesRequest{})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	assertQuery(t, router, "messaging.list_messages", nil)
	if len(list) != 2 || list[1].Cursor != 5 {
		t.Fatalf("list: want cursors 3 and 5, got %+v", list)
	}

	c, router = answeringClient(t, &astral.EOS{})
	_, err = c.ListMessages(ctx(), messaging.ListMessagesRequest{
		List: "inbox", From: "scout", To: "ranger", Since: 42, UnreadOnly: true, AwaitingPickup: true,
		Mailbox: "ranger",
	})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	assertQuery(t, router, "messaging.list_messages", map[string]string{
		"list": "inbox", "from": "scout", "to": "ranger", "since": "42",
		"unread_only": "true", "awaiting_pickup": "true", "mailbox": "ranger",
	})
}

// A delegated listing names the mailbox beside the list, and names nothing else
// the request left unset.
func TestListMessages_NamesAnotherMailbox(t *testing.T) {
	mailbox := astral.GenerateIdentity().String()

	c, router := answeringClient(t, &messaging.Envelope{Cursor: 8}, &astral.EOS{})
	list, err := c.ListMessages(ctx(), messaging.ListMessagesRequest{List: "outbox", Mailbox: mailbox})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	assertQuery(t, router, "messaging.list_messages", map[string]string{"list": "outbox", "mailbox": mailbox})
	if len(list) != 1 || list[0].Cursor != 8 {
		t.Fatalf("list: want cursor 8, got %+v", list)
	}
}

// A listing the node ends before its eos is an error with no envelopes: the eos
// is what says the list is whole, and a stream cut short reads like one that
// ended.
func TestListMessages_AStreamCutShortIsAnError(t *testing.T) {
	c, _ := answeringClient(t, &messaging.Envelope{Cursor: 3}, &messaging.Envelope{Cursor: 4})
	list, err := c.ListMessages(ctx(), messaging.ListMessagesRequest{List: "inbox"})
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("want io.ErrUnexpectedEOF, got %v", err)
	}
	if list != nil {
		t.Fatalf("list: want none, got %v envelopes", len(list))
	}

	c, _ = answeringClient(t)
	if _, err = c.ListMessages(ctx(), messaging.ListMessagesRequest{}); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("an empty stream: want io.ErrUnexpectedEOF, got %v", err)
	}
}

// A listing the node ends with an error object returns that error and no
// envelopes.
func TestListMessages_AnErrorObjectIsTheError(t *testing.T) {
	c, _ := answeringClient(t, &messaging.Envelope{Cursor: 3}, astral.NewError("unknown correspondent: scout"))
	list, err := c.ListMessages(ctx(), messaging.ListMessagesRequest{From: "scout"})
	var said *astral.ErrorMessage
	if !errors.As(err, &said) || said.Error() != "unknown correspondent: scout" {
		t.Fatalf("want the node's error, got %v", err)
	}
	if list != nil {
		t.Fatalf("list: want none, got %v envelopes", len(list))
	}
}

func TestReadMessages_SendsTheRequestAsTheBody(t *testing.T) {
	ref := &messaging.MessageRef{Box: "inbox", ID: messaging.NewMessageID()}
	req := &messaging.ReadMessagesRequest{Refs: []*messaging.MessageRef{ref}, Children: "full", MaxChildren: 4}
	want := &messaging.ReadMessagesResult{NotFound: []*messaging.MessageRef{ref}}

	c, router := answeringClient(t, want)
	got, err := c.ReadMessages(ctx(), req)
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	assertQuery(t, router, "messaging.read_messages", nil)
	if len(got.NotFound) != 1 || *got.NotFound[0] != *ref {
		t.Fatalf("result: want %+v, got %+v", want, got)
	}

	body, ok := sentObject(t, router).(*messaging.ReadMessagesRequest)
	if !ok {
		t.Fatal("the body is not a read_messages_request")
	}
	if len(body.Refs) != 1 || *body.Refs[0] != *ref || body.Children != "full" || body.MaxChildren != 4 {
		t.Fatalf("body: want %+v, got %+v", req, body)
	}
	if body.Mailbox != nil {
		t.Fatalf("mailbox: want none for the caller's own, got %v", body.Mailbox)
	}
}

// A delegated read names the mailbox in the body, never as a query argument.
func TestReadMessages_CarriesTheNamedMailboxInTheBody(t *testing.T) {
	mailbox := astral.GenerateIdentity()
	ref := &messaging.MessageRef{Box: "outbox", ID: messaging.NewMessageID()}
	req := &messaging.ReadMessagesRequest{Refs: []*messaging.MessageRef{ref}, Mailbox: mailbox}

	c, router := answeringClient(t, &messaging.ReadMessagesResult{})
	if _, err := c.ReadMessages(ctx(), req); err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	assertQuery(t, router, "messaging.read_messages", nil)

	body, ok := sentObject(t, router).(*messaging.ReadMessagesRequest)
	if !ok {
		t.Fatal("the body is not a read_messages_request")
	}
	if body.Mailbox == nil || !body.Mailbox.IsEqual(mailbox) {
		t.Fatalf("mailbox: want %v, got %v", mailbox, body.Mailbox)
	}
	if len(body.Refs) != 1 || *body.Refs[0] != *ref {
		t.Fatalf("refs: want %+v, got %+v", req.Refs, body.Refs)
	}
}

func TestWait_SendsOnlyWhatIsSet(t *testing.T) {
	want := &messaging.WaitResult{NextSince: 9, TimedOut: true}

	c, router := answeringClient(t, want)
	got, err := c.Wait(ctx(), messaging.WaitRequest{})
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	assertQuery(t, router, "messaging.wait", nil)
	if got.NextSince != 9 || !got.TimedOut {
		t.Fatalf("result: want %+v, got %+v", want, got)
	}

	c, router = answeringClient(t, want)
	if _, err = c.Wait(ctx(), messaging.WaitRequest{From: "scout", Since: 7, Timeout: 90 * time.Second}); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	assertQuery(t, router, "messaging.wait", map[string]string{"from": "scout", "since": "7", "timeout": "1m30s"})
}

// A park the caller cancelled answers the cancellation, not an empty result.
func TestWait_CancelledParkAnswersTheContext(t *testing.T) {
	c, _ := answeringClient(t)

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	res, err := c.Wait(astral.NewContext(cancelled), messaging.WaitRequest{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v (result %+v)", err, res)
	}
}

// A park cancelled while the node holds it closes the query, and the read that
// fails on the closed query answers the cancellation rather than its own error.
func TestWait_CancellingAHeldParkAnswersTheContext(t *testing.T) {
	answer, never := io.Pipe()
	defer never.Close()

	router := &fakeRouter{answer: answer}
	router.onClose = func() { answer.CloseWithError(io.ErrClosedPipe) }
	c := New(nil, astrald.New(router))

	parent, cancel := context.WithCancel(context.Background())
	time.AfterFunc(20*time.Millisecond, cancel)

	res, err := c.Wait(astral.NewContext(parent), messaging.WaitRequest{Timeout: time.Minute})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v (result %+v)", err, res)
	}
}

func TestArchive_NamesTheRow(t *testing.T) {
	ref := messaging.MessageRef{Box: "outbox", ID: messaging.NewMessageID()}

	c, router := answeringClient(t, &messaging.ArchiveResult{Changed: true})
	changed, err := c.Archive(ctx(), ref, false)
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if !changed {
		t.Fatal("changed: want true")
	}
	assertQuery(t, router, "messaging.archive", map[string]string{"box": "outbox", "id": ref.ID.String()})

	c, router = answeringClient(t, &messaging.ArchiveResult{})
	if changed, err = c.Archive(ctx(), ref, true); err != nil || changed {
		t.Fatalf("Archive undo: changed %v, err %v", changed, err)
	}
	assertQuery(t, router, "messaging.archive", map[string]string{"box": "outbox", "id": ref.ID.String(), "undo": "true"})
}

// errOf drops an operation's result and keeps its error.
func errOf[T any](_ T, err error) error { return err }

// A query the node closes without the answer is an error, not a nil result, a
// zero id or a deletion that did not happen.
func TestSingleAnswerOps_NoAnswerIsAnError(t *testing.T) {
	ref := messaging.MessageRef{Box: "inbox", ID: messaging.NewMessageID()}
	calls := map[string]func(c *Client) error{
		"CreateIdentity": func(c *Client) error { return errOf(c.CreateIdentity(ctx(), "", 0)) },
		"Identity":       func(c *Client) error { return errOf(c.Identity(ctx(), "scout")) },
		"DeleteIdentity": func(c *Client) error { return c.DeleteIdentity(ctx(), "scout") },
		"SendMessage": func(c *Client) error {
			return errOf(c.SendMessage(ctx(), &messaging.SendMessageRequest{To: "scout"}))
		},
		"ReadMessages": func(c *Client) error {
			return errOf(c.ReadMessages(ctx(), &messaging.ReadMessagesRequest{Refs: []*messaging.MessageRef{&ref}}))
		},
		"Wait":    func(c *Client) error { return errOf(c.Wait(ctx(), messaging.WaitRequest{})) },
		"Archive": func(c *Client) error { return errOf(c.Archive(ctx(), ref, false)) },
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			c, _ := answeringClient(t)
			if err := call(c); !errors.Is(err, ErrNoAnswer) {
				t.Fatalf("want ErrNoAnswer, got %v", err)
			}
		})
	}
}
