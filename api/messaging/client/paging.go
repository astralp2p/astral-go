package messaging

import (
	"io"

	"github.com/astralp2p/astral-go/api/messaging"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// PageMessages reads one page of a mailbox, newest first.
func (client *Client) PageMessages(ctx *astral.Context, req messaging.PageMessagesRequest) (*messaging.MessagePage, error) {
	args := mailboxArgs(req.Mailbox, req.Limit, req.Generation)
	set(args, "list", req.List)
	set(args, "peer", req.Peer)
	if req.Before != 0 {
		args["before"] = req.Before
	}
	var page *messaging.MessagePage
	err := expectOne(ctx, client, messaging.MethodPageMessages, args, &page)
	return page, err
}

// ListMessageChanges reads what changed in a mailbox after req.Since.
func (client *Client) ListMessageChanges(ctx *astral.Context, req messaging.ListMessageChangesRequest) (*messaging.MessageChanges, error) {
	args := mailboxArgs(req.Mailbox, req.Limit, &req.Generation)
	set(args, "peer", req.Peer)
	args["since"] = req.Since
	var changes *messaging.MessageChanges
	err := expectOne(ctx, client, messaging.MethodListMessageChanges, args, &changes)
	return changes, err
}

// PageConversations reads one page of a mailbox's conversations, or the one
// with req.Peer.
func (client *Client) PageConversations(ctx *astral.Context, req messaging.PageConversationsRequest) (*messaging.ConversationPage, error) {
	args := mailboxArgs(req.Mailbox, req.Limit, req.Generation)
	set(args, "peer", req.Peer)
	if req.Before != 0 {
		args["before"] = req.Before
	}
	var page *messaging.ConversationPage
	err := expectOne(ctx, client, messaging.MethodPageConversations, args, &page)
	return page, err
}

// ListConversationChanges reads the conversations that changed after req.Since.
func (client *Client) ListConversationChanges(ctx *astral.Context, req messaging.ListConversationChangesRequest) (*messaging.ConversationChanges, error) {
	args := mailboxArgs(req.Mailbox, req.Limit, &req.Generation)
	args["since"] = req.Since
	var changes *messaging.ConversationChanges
	err := expectOne(ctx, client, messaging.MethodListConversationChanges, args, &changes)
	return changes, err
}

// mailboxArgs carries what every paging request may set.
func mailboxArgs(mailbox string, limit uint64, generation *uint64) query.Args {
	args := query.Args{}
	set(args, "mailbox", mailbox)
	if limit != 0 {
		args["limit"] = limit
	}
	if generation != nil {
		args["generation"] = *generation
	}
	return args
}

// set carries a string argument only when it is set: the node refuses an
// argument a request cannot combine, so an empty one sent could be refused.
func set(args query.Args, name, value string) {
	if value != "" {
		args[name] = value
	}
}

// expectOne reads the one object an operation answers, then its eos.
//
// why the eos is required: the answer is complete only when the node says the
// stream ended. An object alone may be followed by an error, and a stream cut
// after it reads exactly like one that ended.
func expectOne[T astral.Object](ctx *astral.Context, client *Client, method string, args query.Args, out *T) error {
	ch, err := client.queryCh(ctx, method, args)
	if err != nil {
		return err
	}
	defer ch.Close()

	var got []T
	var sawEOS bool
	err = ch.Switch(
		channel.Collect[T](&got),
		channel.PassErrors,
		channel.MarkEOS(&sawEOS),
		channel.WithContext(ctx),
	)
	switch {
	case ctx.Err() != nil:
		return ctx.Err()
	case err != nil:
		return err
	case !sawEOS:
		return io.ErrUnexpectedEOF
	case len(got) != 1:
		return ErrNoAnswer
	}
	*out = got[0]
	return nil
}
