package auth

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
)

// wordAction is an action type that names itself by a PermitWord.
type wordAction struct{ Action }

func (wordAction) ObjectType() string                     { return "mod.authtest.word_action" }
func (wordAction) PermitWord() string                     { return "tested" }
func (a wordAction) WriteTo(w io.Writer) (int64, error)   { return astral.Objectify(&a).WriteTo(w) }
func (a *wordAction) ReadFrom(r io.Reader) (int64, error) { return astral.Objectify(a).ReadFrom(r) }

// badWordAction declares a PermitWord the signable text refuses.
type badWordAction struct{ Action }

func (badWordAction) ObjectType() string { return "mod.authtest.bad_word_action" }
func (badWordAction) PermitWord() string { return "Bad Word" }
func (a badWordAction) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&a).WriteTo(w)
}
func (a *badWordAction) ReadFrom(r io.Reader) (int64, error) { return astral.Objectify(a).ReadFrom(r) }

func init() { astral.MustAdd(&wordAction{}, &badWordAction{}) }

// signableLine is the text a text signer such as a Coldcard signs: the 15-byte
// commitment in base64, then SignableText (astrald mod/crypto formatSignableText).
func signableLine(hash []byte, text string) string {
	return "[" + base64.StdEncoding.EncodeToString(hash[0:15]) + "] " + text
}

func textContract(permits ...*Permit) *Contract {
	return &Contract{
		Issuer:    astral.GenerateIdentity(),
		Subject:   astral.GenerateIdentity(),
		Permits:   permits,
		ExpiresAt: astral.Time(time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)),
	}
}

// actionsOf returns the "<actions> depth <d>" part of c's signable text.
func actionsOf(t *testing.T, c *Contract) string {
	t.Helper()
	text := c.SignableText()
	head := c.Issuer.String() + " grants " + c.Subject.String() + " "
	tail := " until " + c.ExpiresAt.Time().Format("2006-01-02 15:04:05")
	if !strings.HasPrefix(text, head) || !strings.HasSuffix(text, tail) {
		t.Fatalf("text has the wrong shape: %q", text)
	}
	return strings.TrimSuffix(strings.TrimPrefix(text, head), tail)
}

func TestContractSignableText_ActionsAndDepth(t *testing.T) {
	for name, tc := range map[string]struct {
		permits []*Permit
		want    string
	}{
		"no permits": {nil, "none depth 0"},
		"word":       {[]*Permit{{Action: "mod.authtest.word_action"}}, "tested depth 0"},
		"shortened": {[]*Permit{
			{Action: "mod.auth.sudo_action"},
			{Action: "mod.auth.see_objects_action", Delegation: 2},
		}, "auth.sudo,auth.see_objects depth 2"},
		"largest depth": {[]*Permit{
			{Action: "mod.auth.sudo_action", Delegation: 7},
			{Action: "mod.auth.see_objects_action", Delegation: 2},
		}, "auth.sudo,auth.see_objects depth 7"},
		"distinct actions": {[]*Permit{
			{Action: "mod.vfs.read_action"},
			{Action: "mod.vfs.read_action", Delegation: 1},
			nil,
		}, "vfs.read depth 1"},
		"unnamed actions are counted": {[]*Permit{
			{Action: "mod.vfs.read_action"},
			{Action: "test.action"},
			{Action: "mod.Auth.sudo_action"},
		}, "vfs.read,+2 depth 0"},
		"refused word falls back to the type": {[]*Permit{
			{Action: "mod.authtest.bad_word_action"},
		}, "authtest.bad_word depth 0"},
		"only unnamed": {[]*Permit{{Action: "test.action", Delegation: 255}}, "+1 depth 255"},
	} {
		if got := actionsOf(t, textContract(tc.permits...)); got != tc.want {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}

func TestActionWord(t *testing.T) {
	for action, want := range map[string]string{
		"mod.authtest.word_action":     "tested",
		"mod.auth.sudo_action":         "auth.sudo",
		"mod.user.admin_swarm_action":  "user.admin_swarm",
		"mod.x9.y_9_action":            "x9.y_9",
		"mod.authtest.bad_word_action": "authtest.bad_word",
		"mod.auth.action":              "",
		"mod.auth._action":             "",
		"mod..sudo_action":             "",
		"mod.a.b.c_action":             "",
		"mod.Auth.sudo_action":         "",
		"auth.sudo_action":             "",
		"mod.auth.sudo":                "",
		"mod.auth.sudo,x_action":       "",
		"mod.auth.sudo until_action":   "",
		"":                             "",
	} {
		got, ok := actionWord(action)
		if got != want || ok != (want != "") {
			t.Errorf("actionWord(%q) = %q %v, want %q", action, got, ok, want)
		}
	}
}

// TestContractSignableText_Overflow shows that the actions the line has no
// room for are counted, never dropped.
func TestContractSignableText_Overflow(t *testing.T) {
	var permits []*Permit
	for i := 0; i < 50; i++ {
		permits = append(permits, &Permit{Action: astral.String8(fmt.Sprintf("mod.module%02d.long_action_name_action", i))})
	}
	c := textContract(permits...)
	text := c.SignableText()
	if len(text) > MaxContractTextLen {
		t.Fatalf("%d bytes, over %d: %q", len(text), MaxContractTextLen, text)
	}

	list := strings.TrimSuffix(actionsOf(t, c), " depth 0")
	items := strings.Split(list, ",")
	last := items[len(items)-1]
	var left int
	if _, err := fmt.Sscanf(last, "+%d", &left); err != nil {
		t.Fatalf("list does not end with a count: %q", list)
	}
	if shown := len(items) - 1; shown+left != len(permits) || shown == 0 {
		t.Fatalf("shown %d + counted %d != %d actions: %q", shown, left, len(permits), list)
	}
}

func TestContractSignableText_WorstCase(t *testing.T) {
	c := sampleContract(&astral.ObjectID{Size: math.MaxUint64})
	c.ExpiresAt = astral.Time(time.Unix(0, math.MaxInt64))
	long := strings.Repeat("z", MaxPermitWordLen)
	var words []string
	for i := 0; i < 100; i++ {
		words = append(words, long)
	}

	for _, s := range []permitSummary{
		{words: words, hidden: math.MaxUint64, depth: math.MaxUint8},
		{words: words, depth: math.MaxUint8},
		{hidden: math.MaxUint64, depth: math.MaxUint8},
		{words: []string{strings.Repeat("y", 300)}, depth: math.MaxUint8},
	} {
		line := signableLine(c.SignableHash(), contractText(c, s))
		if len(line) > 240 {
			t.Fatalf("worst case is %d bytes, over 240: %q", len(line), line)
		}
		for i := 0; i < len(line); i++ {
			if line[i] < 0x20 || line[i] > 0x7e {
				t.Fatalf("byte %d is not printable ASCII: %q", i, line)
			}
		}
		t.Logf("contract worst case: %d bytes  %s", len(line), line)
	}
}

func TestValidPermitWord(t *testing.T) {
	for w, want := range map[string]bool{
		"privileged_member":      true,
		"member":                 true,
		"":                       false,
		"Member":                 false,
		"privileged_member_plus": false,
		"tier 1":                 false,
	} {
		if got := validPermitWord(w); got != want {
			t.Errorf("validPermitWord(%q) = %v, want %v", w, got, want)
		}
	}
}

// TestContractSignableText_BindsRequestID shows the text omits RequestID and
// the commitment binds it.
func TestContractSignableText_BindsRequestID(t *testing.T) {
	c := sampleContract(nil)
	text, hash := c.SignableText(), c.SignableHash()
	c.RequestID = sampleRequestID()
	if c.SignableText() != text || bytes.Equal(c.SignableHash(), hash) {
		t.Fatal("RequestID must change the commitment, not the text")
	}
}
