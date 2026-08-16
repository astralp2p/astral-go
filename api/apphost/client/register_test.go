package apphost

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

func sampleToken() *apphost.AccessToken {
	return &apphost.AccessToken{
		Identity:  astral.GenerateIdentity(),
		Token:     astral.String8("s3cr3t"),
		ExpiresAt: astral.Time(time.Date(2036, time.May, 25, 12, 0, 0, 0, time.UTC)),
	}
}

// answer builds a channel over the objects the node sends back.
func answer(t *testing.T, objects ...astral.Object) *channel.Channel {
	t.Helper()

	var in, out bytes.Buffer
	s := channel.NewSender(&in)
	for _, o := range objects {
		if err := s.Send(o); err != nil {
			t.Fatalf("encode %s: %v", o.ObjectType(), err)
		}
	}

	return channel.New(channel.Join(&in, &out))
}

// A caller asking for nothing sends no permits argument: the query is the one
// it always was.
func TestRegisterArgs_NoPermitsSendsNoArgument(t *testing.T) {
	args, err := registerArgs(nil)
	if err != nil {
		t.Fatalf("registerArgs: %v", err)
	}

	if _, ok := args["permits"]; ok {
		t.Fatalf("want no permits argument, got %v", args)
	}
}

func TestRegisterArgs_PermitsAreCommaJoined(t *testing.T) {
	args, err := registerArgs([]string{"mod.user.info_action", "mod.auth.see_objects_action"})
	if err != nil {
		t.Fatalf("registerArgs: %v", err)
	}

	if got := args["permits"]; got != "mod.user.info_action,mod.auth.see_objects_action" {
		t.Fatalf("permits: got %q", got)
	}
}

// A permit carrying the separator would reach the node as two permits, so it
// is refused here instead of silently becoming an action nobody asked for.
func TestRegisterArgs_PermitWithACommaIsRefused(t *testing.T) {
	_, err := registerArgs([]string{"mod.user.info_action,mod.user.expel_action"})
	if err == nil {
		t.Fatal("expected a comma in a permit name to be refused")
	}
	if !errors.Is(err, apphost.ErrProtocolError) {
		t.Fatalf("want ErrProtocolError, got %v", err)
	}
}

func TestReadAccessToken_ReturnsTheMintedToken(t *testing.T) {
	src := sampleToken()

	token, err := readAccessToken(answer(t, src))
	if err != nil {
		t.Fatalf("readAccessToken: %v", err)
	}

	if !token.Identity.IsEqual(src.Identity) {
		t.Fatalf("identity: want %v, got %v", src.Identity, token.Identity)
	}
	if token.Token != src.Token {
		t.Fatalf("token: want %v, got %v", src.Token, token.Token)
	}
}

// A failure past the accept gate arrives as an error object, not as a token.
func TestReadAccessToken_NodeRefusalIsReturnedNotSwallowed(t *testing.T) {
	token, err := readAccessToken(answer(t, astral.NewError("register policy refused")))
	if err == nil {
		t.Fatal("expected the node's refusal to surface")
	}
	if token != nil {
		t.Fatalf("expected no token after a refusal, got %v", token)
	}
}

// Registration has no empty success: an answer carrying nothing must not read
// as a token the caller can use.
func TestReadAccessToken_EmptyAnswerIsAProtocolError(t *testing.T) {
	token, err := readAccessToken(answer(t))
	if err == nil {
		t.Fatal("expected an empty answer to be an error")
	}
	if !errors.Is(err, apphost.ErrProtocolError) {
		t.Fatalf("want ErrProtocolError, got %v", err)
	}
	if token != nil {
		t.Fatalf("expected no token, got %v", token)
	}
}
