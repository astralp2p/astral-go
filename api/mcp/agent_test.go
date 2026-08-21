package mcp

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// timeWithNanos is a fixed instant carrying sub-second precision: astral.Time
// encodes as UnixNano, so a lost nanosecond shows up as a changed expiry.
func timeWithNanos() time.Time {
	return time.Unix(1700000000, 123456789).UTC()
}

func sampleAgent() *Agent {
	return &Agent{
		Identity:  astral.GenerateIdentity(),
		Alias:     astral.String8("scout"),
		Token:     astral.String8("h4d8s2w6y1b9t3n7"),
		ExpiresAt: astral.Time(timeWithNanos()),
	}
}

func sampleAgentInfo() *AgentInfo {
	return &AgentInfo{
		Identity:  astral.GenerateIdentity(),
		Alias:     astral.String8("scout"),
		Visible:   astral.Bool(true),
		ExpiresAt: astral.Time(timeWithNanos()),
	}
}

func TestAgent_BinaryRoundTrip(t *testing.T) {
	src := sampleAgent()

	data, err := astral.EncodeBytes(src)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	obj, _, err := astral.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	dst, ok := obj.(*Agent)
	if !ok {
		t.Fatalf("want *Agent, got %T", obj)
	}
	if !dst.Identity.IsEqual(src.Identity) {
		t.Fatalf("identity: want %v, got %v", src.Identity, dst.Identity)
	}
	if dst.Alias != src.Alias {
		t.Fatalf("alias: want %v, got %v", src.Alias, dst.Alias)
	}
	if dst.Token != src.Token {
		t.Fatalf("token: want %v, got %v", src.Token, dst.Token)
	}
	if !dst.ExpiresAt.Time().Equal(src.ExpiresAt.Time()) {
		t.Fatalf("expiry: want %v, got %v", src.ExpiresAt.Time(), dst.ExpiresAt.Time())
	}
}

func TestAgentInfo_BinaryRoundTrip(t *testing.T) {
	src := sampleAgentInfo()

	data, err := astral.EncodeBytes(src)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	obj, _, err := astral.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	dst, ok := obj.(*AgentInfo)
	if !ok {
		t.Fatalf("want *AgentInfo, got %T", obj)
	}
	if !dst.Identity.IsEqual(src.Identity) {
		t.Fatalf("identity: want %v, got %v", src.Identity, dst.Identity)
	}
	if dst.Alias != src.Alias {
		t.Fatalf("alias: want %v, got %v", src.Alias, dst.Alias)
	}
	if dst.Visible != src.Visible {
		t.Fatalf("visible: want %v, got %v", src.Visible, dst.Visible)
	}
	if !dst.ExpiresAt.Time().Equal(src.ExpiresAt.Time()) {
		t.Fatalf("expiry: want %v, got %v", src.ExpiresAt.Time(), dst.ExpiresAt.Time())
	}
}

// The type split exists so a caller cannot read a withheld token as an agent
// that has none. AgentInfo must therefore carry no token field at all — not an
// empty one.
func TestAgentInfo_CarriesNoTokenField(t *testing.T) {
	data, err := json.Marshal(sampleAgentInfo())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, found := fields["Token"]; found {
		t.Fatalf("AgentInfo carries a Token field: %s", data)
	}
}

// A record sent on a JSON channel is readable by the matching receiver, which
// materializes it by object type. This is the path taken by mcp.create_agent
// and mcp.list_agents under out=json.
func TestAgent_JSONChannelRoundTrip(t *testing.T) {
	src := sampleAgent()

	var buf bytes.Buffer
	if err := channel.NewJSONSender(&buf).Send(src); err != nil {
		t.Fatal(err)
	}

	obj, err := channel.NewJSONReceiver(bytes.NewReader(buf.Bytes())).Receive()
	if err != nil {
		t.Fatalf("receive %q: %v", buf.String(), err)
	}

	dst, ok := obj.(*Agent)
	if !ok {
		t.Fatalf("want *Agent, got %T", obj)
	}
	if dst.Token != src.Token {
		t.Fatalf("token: want %v, got %v", src.Token, dst.Token)
	}
	if !dst.ExpiresAt.Time().Equal(src.ExpiresAt.Time()) {
		t.Fatalf("expiry: want %v, got %v", src.ExpiresAt.Time(), dst.ExpiresAt.Time())
	}
}
