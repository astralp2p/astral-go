package apphost

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// sampleToken returns a token whose expiry carries nanosecond precision.
func sampleToken() *AccessToken {
	return &AccessToken{
		Identity:  astral.GenerateIdentity(),
		Token:     astral.String8("s3cr3t"),
		ExpiresAt: astral.Time(time.Date(2027, time.August, 3, 17, 28, 16, 334633730, time.UTC)),
	}
}

// The expiry field uses the spec's text encoding for time: RFC 3339.
// See .ai/system/primitive-types/time.md.
func TestAccessToken_MarshalText_ExpiryIsRFC3339(t *testing.T) {
	src := sampleToken()

	text, err := src.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	parts := strings.SplitN(string(text), ",", 3)
	if len(parts) != 3 {
		t.Fatalf("want 3 fields, got %d: %q", len(parts), text)
	}

	var expiresAt astral.Time
	if err := expiresAt.UnmarshalText([]byte(parts[2])); err != nil {
		t.Fatalf("expiry field is not RFC 3339: %v", err)
	}
	if !expiresAt.Time().Equal(src.ExpiresAt.Time()) {
		t.Fatalf("expiry: want %v, got %v", src.ExpiresAt.Time(), expiresAt.Time())
	}
}

func TestAccessToken_TextRoundTrip(t *testing.T) {
	src := sampleToken()

	text, err := src.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	var dst AccessToken
	if err := dst.UnmarshalText(text); err != nil {
		t.Fatalf("unmarshal %q: %v", text, err)
	}

	if !dst.Identity.IsEqual(src.Identity) {
		t.Fatalf("identity: want %v, got %v", src.Identity, dst.Identity)
	}
	if dst.Token != src.Token {
		t.Fatalf("token: want %v, got %v", src.Token, dst.Token)
	}
	if !dst.ExpiresAt.Time().Equal(src.ExpiresAt.Time()) {
		t.Fatalf("expiry: want %v, got %v", src.ExpiresAt.Time(), dst.ExpiresAt.Time())
	}
}

// A token sent on a text channel is readable by the matching receiver. This is the
// path taken by apphost.create_token and apphost.list_tokens under out=text.
func TestAccessToken_TextChannelRoundTrip(t *testing.T) {
	src := sampleToken()

	var buf bytes.Buffer
	if err := channel.NewTextSender(&buf).Send(src); err != nil {
		t.Fatal(err)
	}

	obj, err := channel.NewTextReceiver(bytes.NewReader(buf.Bytes())).Receive()
	if err != nil {
		t.Fatalf("receive %q: %v", buf.String(), err)
	}

	dst, ok := obj.(*AccessToken)
	if !ok {
		t.Fatalf("want *AccessToken, got %T", obj)
	}
	if !dst.ExpiresAt.Time().Equal(src.ExpiresAt.Time()) {
		t.Fatalf("expiry: want %v, got %v", src.ExpiresAt.Time(), dst.ExpiresAt.Time())
	}
}
