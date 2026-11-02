package crypto

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

const testWait = 5 * time.Second

type duplex struct {
	io.Reader
	io.Writer
	closers []io.Closer
}

func (d duplex) Close() error {
	for _, c := range d.closers {
		c.Close()
	}
	return nil
}

// pipePair returns the app's and the node's ends of one binding.
func pipePair() (app, node *channel.Channel) {
	appIn, nodeOut := io.Pipe()
	nodeIn, appOut := io.Pipe()
	app = channel.New(duplex{appIn, appOut, []io.Closer{appIn, appOut}}, channel.WithLockedWrites())
	node = channel.New(duplex{nodeIn, nodeOut, []io.Closer{nodeIn, nodeOut}}, channel.WithLockedWrites())
	return app, node
}

// signerFunc answers text requests and refuses hash requests.
type signerFunc func(ctx *astral.Context, req *crypto.SignTextRequest) (*crypto.Signature, error)

func (f signerFunc) SignText(ctx *astral.Context, req *crypto.SignTextRequest) (*crypto.Signature, error) {
	return f(ctx, req)
}

func (f signerFunc) SignHash(*astral.Context, *crypto.SignHashRequest) (*crypto.Signature, error) {
	return nil, errors.New("this signer signs text only")
}

var testKeys = []*crypto.SignerKey{{Key: &crypto.PublicKey{Type: "secp256k1", Key: make([]byte, 33)}, Scheme: crypto.SchemeBIP137}}

// admit plays the node through admission: it reads the key set and acks.
func admit(t *testing.T, node *channel.Channel) {
	t.Helper()
	obj := receive(t, node)
	if _, ok := obj.(*crypto.SignerKeys); !ok {
		t.Fatalf("first object = %T; want *crypto.SignerKeys", obj)
	}
	if err := node.Send(&astral.Ack{}); err != nil {
		t.Fatal(err)
	}
}

func receive(t *testing.T, ch *channel.Channel) astral.Object {
	t.Helper()
	type result struct {
		obj astral.Object
		err error
	}
	out := make(chan result, 1)
	go func() {
		obj, err := ch.Receive()
		out <- result{obj, err}
	}()
	select {
	case r := <-out:
		if r.err != nil {
			t.Fatalf("receive: %v", r.err)
		}
		return r.obj
	case <-time.After(testWait):
		t.Fatal("nothing received")
		return nil
	}
}

// startBound opens a binding over a pipe with h behind it.
func startBound(t *testing.T, h SignerHandler) (*SignerBinding, *channel.Channel) {
	t.Helper()
	app, node := pipePair()
	ctx, cancel := astral.NewContext(nil).WithCancel()
	t.Cleanup(cancel)

	type result struct {
		b   *SignerBinding
		err error
	}
	out := make(chan result, 1)
	go func() {
		b, err := startSigner(ctx, app, testKeys, h)
		out <- result{b, err}
	}()
	admit(t, node)

	r := <-out
	if r.err != nil {
		t.Fatalf("startSigner: %v", r.err)
	}
	t.Cleanup(func() { r.b.Close() })
	return r.b, node
}

func answerOf(t *testing.T, node *channel.Channel) *crypto.SignAnswer {
	t.Helper()
	obj := receive(t, node)
	answer, ok := obj.(*crypto.SignAnswer)
	if !ok {
		t.Fatalf("object = %T (%v); want *crypto.SignAnswer", obj, obj)
	}
	return answer
}

func TestStartSignerReturnsTheRefusal(t *testing.T) {
	app, node := pipePair()
	go func() {
		_, _ = node.Receive()
		_ = node.Send(astral.Err(crypto.ErrKeyClaimed))
	}()

	_, err := startSigner(astral.NewContext(nil), app, testKeys, nil)
	if err == nil || !isKeyClaimed(err) {
		t.Fatalf("startSigner = %v; want a claim refusal", err)
	}
}

func TestStartSignerFailsOnAClosedChannel(t *testing.T) {
	app, node := pipePair()
	go func() {
		_, _ = node.Receive()
		node.Close()
	}()

	if _, err := startSigner(astral.NewContext(nil), app, testKeys, nil); err == nil {
		t.Fatal("startSigner returned a binding the node never acknowledged")
	}
}

// TestSignerAnswersEchoTheRequest pins the two answers a handler can produce.
func TestSignerAnswersEchoTheRequest(t *testing.T) {
	want := &crypto.Signature{Scheme: crypto.SchemeBIP137, Data: []byte{1, 2, 3}}
	_, node := startBound(t, signerFunc(func(_ *astral.Context, req *crypto.SignTextRequest) (*crypto.Signature, error) {
		if req.Text == "decline" {
			return nil, errors.New("declined on the device")
		}
		return want, nil
	}))

	signed, declined := astral.NewNonce(), astral.NewNonce()
	if err := node.Send(&crypto.SignTextRequest{RequestID: signed, Text: "sign"}); err != nil {
		t.Fatal(err)
	}
	if answer := answerOf(t, node); answer.RequestID != signed || answer.Signature == nil || answer.Error != "" {
		t.Fatalf("answer = %+v; want a signature for %v", answer, signed)
	}

	if err := node.Send(&crypto.SignTextRequest{RequestID: declined, Text: "decline"}); err != nil {
		t.Fatal(err)
	}
	if answer := answerOf(t, node); answer.RequestID != declined || answer.Signature != nil || answer.Error != "declined on the device" {
		t.Fatalf("answer = %+v; want the refusal for %v", answer, declined)
	}

	if err := node.Send(&crypto.SignHashRequest{RequestID: declined}); err != nil {
		t.Fatal(err)
	}
	if answer := answerOf(t, node); answer.Error == "" {
		t.Fatalf("answer = %+v; want the hash refusal", answer)
	}
}

// TestSignAnswerCarriesExactlyOne: the node ends a binding on any other answer,
// so a handler's mistake must not reach the wire as one.
func TestSignAnswerCarriesExactlyOne(t *testing.T) {
	id := astral.NewNonce()
	signature := &crypto.Signature{Scheme: crypto.SchemeBIP137, Data: []byte{1}}

	for _, tc := range []struct {
		name      string
		signature *crypto.Signature
		err       error
		wantSig   bool
	}{
		{"a signature", signature, nil, true},
		{"an error", nil, errors.New("no"), false},
		{"both", signature, errors.New("no"), false},
		{"neither", nil, nil, false},
		{"an error without text", nil, errors.New(""), false},
		{"a long error", nil, errors.New(strings.Repeat("x", 70000)), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			answer := newSignAnswer(id, tc.signature, tc.err)
			if (answer.Signature != nil) == (answer.Error != "") {
				t.Fatalf("answer = %+v; want exactly one of signature and error", answer)
			}
			if tc.wantSig != (answer.Signature != nil) {
				t.Fatalf("answer = %+v; want signature: %v", answer, tc.wantSig)
			}
			if len(answer.Error) > maxSignerError {
				t.Fatalf("error text is %d bytes; want at most %d", len(answer.Error), maxSignerError)
			}
		})
	}
}

// TestSignCancelEndsTheRequestContext: a person approving on a device is the
// wait the cancel has to reach.
func TestSignCancelEndsTheRequestContext(t *testing.T) {
	started := make(chan struct{})
	_, node := startBound(t, signerFunc(func(ctx *astral.Context, _ *crypto.SignTextRequest) (*crypto.Signature, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}))

	id := astral.NewNonce()
	if err := node.Send(&crypto.SignTextRequest{RequestID: id}); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := node.Send(&crypto.SignCancel{RequestID: id}); err != nil {
		t.Fatal(err)
	}
	if answer := answerOf(t, node); answer.RequestID != id || answer.Error == "" {
		t.Fatalf("answer = %+v; want the cancellation of %v", answer, id)
	}
}

// TestSlowRequestDelaysNoOther: requests run concurrently, so one waiting on a
// person does not hold up another.
func TestSlowRequestDelaysNoOther(t *testing.T) {
	release := make(chan struct{})
	_, node := startBound(t, signerFunc(func(_ *astral.Context, req *crypto.SignTextRequest) (*crypto.Signature, error) {
		if req.Text == "slow" {
			<-release
		}
		return &crypto.Signature{Scheme: crypto.SchemeBIP137, Data: []byte{1}}, nil
	}))

	slow, fast := astral.NewNonce(), astral.NewNonce()
	if err := node.Send(&crypto.SignTextRequest{RequestID: slow, Text: "slow"}); err != nil {
		t.Fatal(err)
	}
	if err := node.Send(&crypto.SignTextRequest{RequestID: fast, Text: "fast"}); err != nil {
		t.Fatal(err)
	}
	if answer := answerOf(t, node); answer.RequestID != fast {
		t.Fatalf("first answer is for %v; want %v", answer.RequestID, fast)
	}
	close(release)
	if answer := answerOf(t, node); answer.RequestID != slow {
		t.Fatalf("second answer is for %v; want %v", answer.RequestID, slow)
	}
}

// TestBindingEndsOnAnUnexpectedObject and on the node closing the channel.
func TestBindingEnds(t *testing.T) {
	t.Run("an unexpected object", func(t *testing.T) {
		b, node := startBound(t, signerFunc(nil))
		if err := node.Send(&astral.Ack{}); err != nil {
			t.Fatal(err)
		}
		awaitDone(t, b)
	})
	t.Run("the node closing", func(t *testing.T) {
		b, node := startBound(t, signerFunc(nil))
		node.Close()
		awaitDone(t, b)
	})
	t.Run("Close", func(t *testing.T) {
		b, _ := startBound(t, signerFunc(nil))
		if err := b.Close(); err != nil {
			t.Fatal(err)
		}
		awaitDone(t, b)
	})
}

func awaitDone(t *testing.T, b *SignerBinding) {
	t.Helper()
	select {
	case <-b.Done():
	case <-time.After(testWait):
		t.Fatal("the binding did not end")
	}
}

// TestBindingEndCancelsItsRequests: a request whose binding is gone has nowhere
// to answer, so its wait on a person must not go on.
func TestBindingEndCancelsItsRequests(t *testing.T) {
	started, ended := make(chan struct{}), make(chan struct{})
	_, node := startBound(t, signerFunc(func(ctx *astral.Context, _ *crypto.SignTextRequest) (*crypto.Signature, error) {
		close(started)
		<-ctx.Done()
		close(ended)
		return nil, ctx.Err()
	}))

	if err := node.Send(&crypto.SignTextRequest{RequestID: astral.NewNonce()}); err != nil {
		t.Fatal(err)
	}
	<-started
	node.Close()

	select {
	case <-ended:
	case <-time.After(testWait):
		t.Fatal("the request outlived its binding")
	}
}
