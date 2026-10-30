package inference

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// The sizes v2 widened its text and media fields for: a long file read in one
// part, and a screenshot inline. v1 capped both at 64 KiB.
var (
	bigText  = astral.String32(strings.Repeat("words ", 1<<20/6+1))
	bigBytes = astral.Bytes32(bytes.Repeat([]byte{0xa5, 0x5a}, 5<<20/2))
)

func full() []astral.Object {
	temp := astral.Float32(0.7)
	image := &MediaRef{Kind: SourceInline, Data: bigBytes, MIME: "image/png"}
	stored := &MediaRef{Kind: SourceObject, ObjectID: astral.ObjectID{Size: 42, Hash: [32]byte{1, 2, 3}}, URL: "https://example.org/a.png", MIME: "image/jpeg"}
	call := &ToolCall{ID: "call_1", Name: "read_file", Arguments: bigText}
	return []astral.Object{
		&EffortLevel{Name: EffortOff, Description: "no thinking", TokenBudget: 1},
		&Model{ID: "m", DisplayName: "M", ContextWindow: 131072, MaxOutput: 8192,
			Modalities: []astral.String8{PartText, PartImage}, EffortLevels: []*EffortLevel{{Name: EffortOff}, {Name: EffortHigh, TokenBudget: 4096}},
			Streams: true, Description: "d", DefaultEffort: EffortLow, Tools: true},
		&Models{Models: []*Model{{ID: "a", Tools: true}, {ID: "b", DefaultEffort: EffortHigh}}},
		&Provider{Name: "ai0", Description: "the lab box"},
		&Tool{Name: "read_file", Description: "reads a file", Parameters: bigText},
		call,
		image,
		stored,
		&ContentPart{Kind: PartText, Text: bigText},
		&ContentPart{Kind: PartImage, Media: image},
		&Message{Role: RoleAssistant, ToolCalls: []*ToolCall{call, {ID: "call_2", Name: "ls", Arguments: "{}"}}},
		&Message{Role: RoleTool, Parts: []*ContentPart{{Kind: PartText, Text: "ok"}}, ToolCallID: "call_1"},
		&Request{Model: "m", Effort: EffortOff, MaxOutput: 100, Temperature: &temp,
			Messages: []*Message{
				{Role: RoleUser, Parts: []*ContentPart{{Kind: PartText, Text: bigText}, {Kind: PartImage, Media: image}}},
				{Role: RoleAssistant, ToolCalls: []*ToolCall{call}},
				{Role: RoleTool, Parts: []*ContentPart{{Kind: PartText, Text: "42"}}, ToolCallID: "call_1"},
			},
			System: bigText, Tools: []*Tool{{Name: "read_file", Parameters: `{"type":"object"}`}},
			ToolChoice: "read_file", Stream: true},
		&ToolCallDelta{Index: 3, ID: "call_3", Name: "ls", Arguments: bigText},
		&Chunk{Index: 9, Text: bigText, Reasoning: bigText,
			ToolCalls:    []*ToolCallDelta{{Index: 0, ID: "c0", Name: "a", Arguments: `{"x":`}, {Index: 1, Arguments: "1}"}},
			FinishReason: FinishToolCalls, Model: "m"},
		&Usage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3, CachedPromptTokens: 4, ReasoningTokens: 5},
		&Error{Kind: ErrorRateLimited, Status: 429, Message: "slow down", RetryAfter: 30},
	}
}

// Every type survives the wire, binary and JSON, with every field set and at
// the sizes v2 carries: decoding and encoding again gives the same bytes.
func TestFullTypesRoundTrip(t *testing.T) {
	for _, obj := range full() {
		var buf bytes.Buffer
		if err := channel.NewSender(&buf).Send(obj); err != nil {
			t.Fatalf("send %s: %v", obj.ObjectType(), err)
		}
		want := append([]byte(nil), buf.Bytes()...)
		got, err := channel.NewReceiver(&buf).Receive()
		if err != nil {
			t.Fatalf("receive %s: %v", obj.ObjectType(), err)
		}
		var again bytes.Buffer
		if err := channel.NewSender(&again).Send(got); err != nil {
			t.Fatalf("resend %s: %v", obj.ObjectType(), err)
		}
		if !bytes.Equal(want, again.Bytes()) {
			t.Fatalf("binary round trip changed %s", obj.ObjectType())
		}

		j, err := astral.Objectify(obj).MarshalJSON()
		if err != nil {
			t.Fatalf("marshal %s: %v", obj.ObjectType(), err)
		}
		fresh := reflect.New(reflect.TypeOf(obj).Elem()).Interface().(astral.Object)
		if err := astral.Objectify(fresh).UnmarshalJSON(j); err != nil {
			t.Fatalf("unmarshal %s: %v", obj.ObjectType(), err)
		}
		j2, err := astral.Objectify(fresh).MarshalJSON()
		if err != nil {
			t.Fatalf("remarshal %s: %v", obj.ObjectType(), err)
		}
		if !bytes.Equal(j, j2) {
			t.Fatalf("json round trip changed %s", obj.ObjectType())
		}
	}
}

// The zero value of every type is on the wire path too, since any query can
// make a node materialize one.
func TestZeroTypesRoundTrip(t *testing.T) {
	for _, obj := range Objects() {
		data, err := astral.EncodeBytes(obj)
		if err != nil {
			t.Fatalf("encode %s: %v", obj.ObjectType(), err)
		}
		got, err := astral.DecodeAs[astral.Object](data)
		if err != nil {
			t.Fatalf("decode %s: %v", obj.ObjectType(), err)
		}
		again, err := astral.EncodeBytes(got)
		if err != nil {
			t.Fatalf("re-encode %s: %v", obj.ObjectType(), err)
		}
		if !bytes.Equal(data, again) {
			t.Fatalf("zero %s changed on the wire", obj.ObjectType())
		}
	}
}

// The schema of inference v2. Blueprints compare field by field and in order,
// so a moved or retyped field changes the protocol even when the Go code
// still compiles.
var schema = []struct {
	name   string
	fields []string
}{
	{"inference.effort_level", []string{"Name string8", "Description string8", "TokenBudget uint32"}},
	{"inference.model", []string{"ID string8", "DisplayName string8", "ContextWindow uint32", "MaxOutput uint32",
		"Modalities []string8", "EffortLevels []inference.effort_level", "Streams bool", "Description string8",
		"DefaultEffort string8", "Tools bool"}},
	{"inference.models", []string{"Models []inference.model"}},
	{"inference.provider", []string{"Name string8", "Description string16"}},
	{"inference.tool", []string{"Name string8", "Description string16", "Parameters string32"}},
	{"inference.tool_call", []string{"ID string8", "Name string8", "Arguments string32"}},
	{"inference.media_ref", []string{"Kind string8", "Data bytes32", "ObjectID object_id.sha256", "URL string16", "MIME string8"}},
	{"inference.content_part", []string{"Kind string8", "Text string32", "Media *inference.media_ref"}},
	{"inference.message", []string{"Role string8", "Parts []inference.content_part", "ToolCalls []inference.tool_call", "ToolCallID string8"}},
	{"inference.request", []string{"Model string8", "Effort string8", "MaxOutput uint32", "Temperature *float32",
		"Messages []inference.message", "System string32", "Tools []inference.tool", "ToolChoice string8", "Stream bool"}},
	{"inference.tool_call_delta", []string{"Index uint32", "ID string8", "Name string8", "Arguments string32"}},
	{"inference.chunk", []string{"Index uint32", "Text string32", "Reasoning string32",
		"ToolCalls []inference.tool_call_delta", "FinishReason string8", "Model string8"}},
	{"inference.usage", []string{"PromptTokens uint32", "CompletionTokens uint32", "TotalTokens uint32",
		"CachedPromptTokens uint32", "ReasoningTokens uint32"}},
	{"inference.error", []string{"Kind string8", "Status uint32", "Message string16", "RetryAfter uint32"}},
}

func TestBlueprintsMatchTheSchema(t *testing.T) {
	bps, err := Blueprints()
	if err != nil {
		t.Fatal(err)
	}
	if len(bps) != len(schema) {
		t.Fatalf("%d blueprints; the schema has %d types", len(bps), len(schema))
	}
	for i, want := range schema {
		bp := bps[i]
		if bp.Type.String() != want.name {
			t.Fatalf("blueprint %d is %s; want %s", i, bp.Type, want.name)
		}
		var got []string
		for _, f := range bp.Fields {
			got = append(got, f.Name.String()+" "+specString(f.Spec))
		}
		if !reflect.DeepEqual(got, want.fields) {
			t.Errorf("%s fields:\n got %q\nwant %q", want.name, got, want.fields)
		}
	}
}

// A node registers a blueprint only after every type it references, so the
// push order must put dependencies first.
func TestBlueprintsListDependenciesFirst(t *testing.T) {
	bps, err := Blueprints()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, bp := range bps {
		for _, f := range bp.Fields {
			ref := f.Spec.ReferencedType()
			if strings.HasPrefix(ref, "inference.") && !seen[ref] {
				t.Errorf("%s.%s references %s before it is listed", bp.Type, f.Name, ref)
			}
		}
		seen[bp.Type.String()] = true
	}
}

// A type added to the package without a place in Objects would never be
// pushed, and a node without it could not decode the objects carrying it.
func TestObjectsListEveryInferenceType(t *testing.T) {
	listed := map[string]bool{}
	for _, o := range Objects() {
		listed[o.ObjectType()] = true
	}
	for _, name := range astral.DefaultBlueprints().OrderedBlueprints() {
		if strings.HasPrefix(name, "inference.") && !listed[name] {
			t.Errorf("%s is registered but not in Objects", name)
		}
	}
}

func TestErrorIsAGoError(t *testing.T) {
	var err error = fmt.Errorf("complete: %w", &Error{Kind: ErrorRateLimited, Status: 429, Message: "slow down", RetryAfter: 30})

	var e *Error
	if !errors.As(err, &e) || e.RetryAfter != 30 {
		t.Fatalf("errors.As found %+v", e)
	}
	if got, want := err.Error(), "complete: inference rate_limited (status 429): slow down"; got != want {
		t.Fatalf("message %q; want %q", got, want)
	}
	if got, want := (&Error{Kind: ErrorUnavailable}).Error(), "inference unavailable"; got != want {
		t.Fatalf("message %q; want %q", got, want)
	}
}

func specString(s astral.Spec) string {
	switch v := s.(type) {
	case *astral.PrimitiveSpec:
		return v.PrimitiveType.String()
	case *astral.RefSpec:
		return v.Type.String()
	case *astral.SliceSpec:
		return "[]" + v.Type.String()
	case *astral.PtrSpec:
		return "*" + v.Type.String()
	}
	return fmt.Sprintf("%T", s)
}
