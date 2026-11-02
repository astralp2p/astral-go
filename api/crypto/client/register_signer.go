package crypto

import (
	"errors"
	"sync"
	"time"

	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/sig"
)

// Claim retry: the node can still hold the previous binding right after a
// reconnect, until it notices that binding closed.
const (
	claimRetryFirst = 250 * time.Millisecond
	claimRetryMax   = 4 * time.Second
	claimRetryFor   = 30 * time.Second
)

// maxSignerError bounds the error text of one answer, well inside string16.
const maxSignerError = 1024

// SignerHandler signs on behalf of a binding's keys. Each request runs on its
// own goroutine, and its context ends when the node cancels the request or the
// binding closes.
type SignerHandler interface {
	SignText(ctx *astral.Context, req *crypto.SignTextRequest) (*crypto.Signature, error)
	SignHash(ctx *astral.Context, req *crypto.SignHashRequest) (*crypto.Signature, error)
}

// SignerBinding is an open crypto.register_signer binding: the node asks, the
// handler answers. The key set is fixed for its lifetime.
type SignerBinding struct {
	ch      *channel.Channel
	handler SignerHandler
	cancels sig.Map[astral.Nonce, func()]
	done    chan struct{}
	once    sync.Once
}

// RegisterSigner opens a binding for keys on the target node. It returns once
// the node acknowledges the binding, or with the node's refusal. It retries a
// refusal naming an already claimed key for up to 30 seconds and returns any
// other error at once.
func (client *Client) RegisterSigner(ctx *astral.Context, keys []*crypto.SignerKey, h SignerHandler) (*SignerBinding, error) {
	delay := claimRetryFirst
	deadline := time.Now().Add(claimRetryFor)
	for {
		b, err := client.openSigner(ctx, keys, h)
		if err == nil {
			return b, nil
		}
		if !isKeyClaimed(err) || time.Now().Add(delay).After(deadline) {
			return nil, err
		}
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		delay = min(delay*2, claimRetryMax)
	}
}

// RegisterSigner opens a binding on the default node.
func RegisterSigner(ctx *astral.Context, keys []*crypto.SignerKey, h SignerHandler) (*SignerBinding, error) {
	return Default().RegisterSigner(ctx, keys, h)
}

func (client *Client) openSigner(ctx *astral.Context, keys []*crypto.SignerKey, h SignerHandler) (*SignerBinding, error) {
	// why: answers run on per-request goroutines, so writes must not interleave.
	ch, err := client.queryCh(ctx, crypto.MethodRegisterSigner, nil, channel.WithLockedWrites())
	if err != nil {
		return nil, err
	}
	return startSigner(ctx, ch, keys, h)
}

func startSigner(ctx *astral.Context, ch *channel.Channel, keys []*crypto.SignerKey, h SignerHandler) (*SignerBinding, error) {
	if err := ch.Send(&crypto.SignerKeys{Keys: keys}); err != nil {
		ch.Close()
		return nil, err
	}
	if err := ch.Switch(channel.ExpectAck, channel.PassErrors, channel.FailOnEOF()); err != nil {
		ch.Close()
		return nil, err
	}

	b := &SignerBinding{ch: ch, handler: h, done: make(chan struct{})}
	go b.serve(ctx)
	return b, nil
}

// why compared by text too: a refusal arrives as an error_message, which carries
// only the text of the node's error.
func isKeyClaimed(err error) bool {
	return errors.Is(err, crypto.ErrKeyClaimed) || err.Error() == crypto.ErrKeyClaimed.Error()
}

func (b *SignerBinding) serve(ctx *astral.Context) {
	defer close(b.done)
	defer b.closeChannel()

	// why: a request outlives no binding, because its answer has nowhere to go.
	ctx, cancel := ctx.WithCancel()
	defer cancel()

	_ = b.ch.Handle(ctx, func(o astral.Object) {
		switch req := o.(type) {
		case *crypto.SignTextRequest:
			b.answer(ctx, req.RequestID, func(ctx *astral.Context) (*crypto.Signature, error) {
				return b.handler.SignText(ctx, req)
			})
		case *crypto.SignHashRequest:
			b.answer(ctx, req.RequestID, func(ctx *astral.Context) (*crypto.Signature, error) {
				return b.handler.SignHash(ctx, req)
			})
		case *crypto.SignCancel:
			if cancelRequest, found := b.cancels.Get(req.RequestID); found {
				cancelRequest()
			}
		default:
			b.closeChannel()
		}
	})
}

// answer runs one request on its own goroutine, so a person approving one
// signature never delays another.
func (b *SignerBinding) answer(ctx *astral.Context, id astral.Nonce, sign func(*astral.Context) (*crypto.Signature, error)) {
	ctx, cancel := ctx.WithCancel()
	b.cancels.Set(id, cancel)

	go func() {
		defer cancel()
		defer b.cancels.Delete(id)

		signature, err := sign(ctx)
		_ = b.ch.Send(newSignAnswer(id, signature, err))
	}()
}

// newSignAnswer holds to the rule that an answer carries exactly one of a
// signature and an error, because the node ends a binding that breaks it.
func newSignAnswer(id astral.Nonce, signature *crypto.Signature, err error) *crypto.SignAnswer {
	if err == nil && signature != nil {
		return &crypto.SignAnswer{RequestID: id, Signature: signature}
	}

	text := "signer produced no signature"
	if err != nil && err.Error() != "" {
		text = err.Error()
	}
	if len(text) > maxSignerError {
		text = text[:maxSignerError]
	}
	return &crypto.SignAnswer{RequestID: id, Error: astral.String16(text)}
}

// Close ends the binding; the node releases its keys and fails its outstanding
// requests.
func (b *SignerBinding) Close() error {
	b.closeChannel()
	<-b.done
	return nil
}

// Done is closed when the binding ends, by Close or by the node.
func (b *SignerBinding) Done() <-chan struct{} { return b.done }

func (b *SignerBinding) closeChannel() {
	b.once.Do(func() { b.ch.Close() })
}
