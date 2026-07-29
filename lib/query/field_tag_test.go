package query

import "testing"

// ParseTag recognises exactly three tokens: skip, required, and key:<name>.
// Every other token is filed into Other, which nothing reads. That silent
// discard is deliberate — it is the behaviour the drop-optional task left in
// place after query:"optional" reached 236 uses while doing nothing. These
// tests pin it so that any move to a loud parser is a conscious edit here.

func TestParseTag_RecognisedTokens(t *testing.T) {
	cases := []struct {
		name string
		tag  string
		want FieldTag
	}{
		{"skip", "skip", FieldTag{Skip: true}},
		{"required", "required", FieldTag{Required: true}},
		{"key", "key:custom_name", FieldTag{Key: "custom_name"}},
		{"key and required", "key:v;required", FieldTag{Key: "v", Required: true}},
		{"skip and required", "skip;required", FieldTag{Skip: true, Required: true}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ParseTag(c.tag)

			if got.Skip != c.want.Skip {
				t.Fatalf("Skip: want %v, got %v", c.want.Skip, got.Skip)
			}
			if got.Required != c.want.Required {
				t.Fatalf("Required: want %v, got %v", c.want.Required, got.Required)
			}
			if got.Key != c.want.Key {
				t.Fatalf("Key: want %q, got %q", c.want.Key, got.Key)
			}
		})
	}
}

// A key token without a value is skipped, leaving Key empty rather than
// producing a field named after the empty string.
func TestParseTag_KeyWithoutValueIsSkipped(t *testing.T) {
	got := ParseTag("key")

	if got.Key != "" {
		t.Fatalf("Key: want empty, got %q", got.Key)
	}
	if _, found := got.Other["key"]; found {
		t.Fatal("a valueless key token must not fall through to Other")
	}
}

// An unrecognised token is collected into Other and raises no error. This is
// the query:"optional" failure mode: the tag parses, the field is unaffected,
// and nothing reads Other.
func TestParseTag_UnknownTokenIsCollectedNotRejected(t *testing.T) {
	cases := []struct {
		name      string
		tag       string
		wantKey   string
		wantValue string
	}{
		{"bare token", "optional", "optional", ""},
		{"key value token", "min:3", "min", "3"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ParseTag(c.tag)

			value, found := got.Other[c.wantKey]
			if !found {
				t.Fatalf("Other[%q]: want present, got absent (Other=%v)", c.wantKey, got.Other)
			}
			if value != c.wantValue {
				t.Fatalf("Other[%q]: want %q, got %q", c.wantKey, c.wantValue, value)
			}

			// the unknown token must not have set a recognised field
			if got.Skip || got.Required || got.Key != "" {
				t.Fatalf("an unknown token must leave recognised fields untouched, got %+v", *got)
			}
		})
	}
}

// An unknown token alongside a recognised one leaves the recognised one intact.
func TestParseTag_UnknownTokenDoesNotDisturbRecognisedTokens(t *testing.T) {
	got := ParseTag("required;optional;key:name")

	if !got.Required {
		t.Fatal("Required: want true, got false")
	}
	if got.Key != "name" {
		t.Fatalf("Key: want %q, got %q", "name", got.Key)
	}
	if _, found := got.Other["optional"]; !found {
		t.Fatalf("Other: want the unknown token present, got %v", got.Other)
	}
}

// The empty tag yields a zero FieldTag with a usable Other map. The empty
// string splits to one empty token, which lands in Other under "".
func TestParseTag_EmptyTag(t *testing.T) {
	got := ParseTag("")

	if got.Skip || got.Required || got.Key != "" {
		t.Fatalf("want a zero FieldTag, got %+v", *got)
	}
	if got.Other == nil {
		t.Fatal("Other: want an initialised map, got nil")
	}
}
