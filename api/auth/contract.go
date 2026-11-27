package auth

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/astral"
)

// Contract is the unsigned body of an authorization grant from Issuer to Subject.
// Wrap it in SignedContract before indexing or verifying.
type Contract struct {
	Issuer  *astral.Identity
	Subject *astral.Identity
	Permits []*Permit

	ExpiresAt astral.Time

	// RequestID names the request this contract answers: the entry ID
	// (astral.ResolveObjectID) of the stored signed request, not the ID of the
	// request body: an api/user.SignedContractRequest for an offer and the
	// contract that carries it out, or an api/user.SignedJoinRequest for a
	// membership a carrier signs in the carrier session. An offer
	// (a contract only its issuer has signed) and the contract its subject
	// completes from it carry the same RequestID. Nil when the contract answers
	// no stored request.
	//
	// The link never adds or redirects authority: a contract has authority only
	// when both its issuer and its subject signed it.
	//
	// why a request, not its answer: an offer is itself the answer, and an
	// entry cannot name an entry that contains it.
	// why last: fields encode in order with no tags; appending keeps the
	// earlier fields at their offsets. New swarms only (D11): a contract
	// encoded with another field layout does not decode.
	RequestID *astral.ObjectID
}

type Permit struct {
	Action      astral.String8 // object type of an action
	Constraints *astral.Bundle // list of constraints
	Delegation  astral.Uint8   // hops allowed below a link carrying this permit; 0 = non-delegable
}

var _ astral.Object = &Permit{}
var _ crypto.SignableTextObject = &Contract{}

func (Permit) ObjectType() string { return "mod.auth.permit" }

func (p Permit) WriteTo(w io.Writer) (int64, error)   { return astral.Objectify(&p).WriteTo(w) }
func (p *Permit) ReadFrom(r io.Reader) (int64, error) { return astral.Objectify(p).ReadFrom(r) }

func (p Permit) MarshalJSON() ([]byte, error)  { return astral.Objectify(&p).MarshalJSON() }
func (p *Permit) UnmarshalJSON(b []byte) error { return astral.Objectify(p).UnmarshalJSON(b) }

// Allows reports whether p permits the action: the action type must match and,
// for Constrainable actions, the permit's constraints must pass.
func (p *Permit) Allows(action ActionObject) bool {
	if string(p.Action) != action.ObjectType() {
		return false
	}
	if ca, ok := action.(Constrainable); ok {
		return ca.ApplyConstraints(p.Constraints)
	}
	return true
}

// HasPermit returns all permits in this contract that match the given action type.
// Empty result means the contract grants no such permission.
func (c *Contract) HasPermit(action string) []*Permit {
	if c.Permits == nil {
		return nil
	}
	var result []*Permit
	for _, p := range c.Permits {
		if string(p.Action) == action {
			result = append(result, p)
		}
	}
	return result
}

func (Contract) ObjectType() string { return "mod.auth.contract" }

func (c Contract) WriteTo(w io.Writer) (int64, error)   { return astral.Objectify(&c).WriteTo(w) }
func (c *Contract) ReadFrom(r io.Reader) (int64, error) { return astral.Objectify(c).ReadFrom(r) }

func (c Contract) MarshalJSON() ([]byte, error)  { return astral.Objectify(&c).MarshalJSON() }
func (c *Contract) UnmarshalJSON(b []byte) error { return astral.Objectify(c).UnmarshalJSON(b) }

func (c *Contract) SignableHash() []byte {
	id, err := astral.ResolveObjectID(c)
	if err != nil {
		return nil
	}
	return id.Hash[:]
}

// MaxContractTextLen is the longest SignableText of a contract: a text signer
// such as a Coldcard signs at most 240 bytes, and the commitment prefix
// "[<20 base64 characters>] " takes 23 of them (astrald mod/crypto).
const MaxContractTextLen = 240 - 23

// SignableText is one fixed-shape ASCII line of at most MaxContractTextLen
// bytes:
//
//	<issuer> grants <subject> <actions> depth <d> until <YYYY-MM-DD hh:mm:ss>
//
// <actions> lists the distinct permit actions in permit order, comma
// separated, each by its actionWord. When the line has no room for every
// action, or an action has no actionWord, the list ends with "+<n>", the
// number of actions not shown; a contract without permits shows "none".
// <d> is the largest Delegation among the permits. Constraints are not shown;
// they only narrow a permit. So the text never shows a grant narrower than the
// one signed. The commitment the signer prefixes binds everything else,
// RequestID included.
//
// why not every permit as "<action type>:<depth>": with two 66-byte identities
// and the commitment prefix, that list leaves a Privileged membership at 255
// bytes ("mod.user.privileged_member_action:0"), so a Coldcard could not sign
// the one contract the user key must sign in a ceremony. A single depth never
// understates a permit's Delegation, and "+<n>" keeps every action counted.
func (c *Contract) SignableText() string {
	return contractText(c, c.summarize())
}

// PermitWorder is implemented by an action type whose permit names itself by
// a short word in a contract's signable text, e.g. the tier of a membership.
//
// A word longer than MaxPermitWordLen or outside [a-z_] is ignored.
type PermitWorder interface {
	PermitWord() string
}

// MaxPermitWordLen is the longest PermitWord a contract's signable text carries
// (len("privileged_member")).
const MaxPermitWordLen = 17

// actionWord returns how a contract's signable text names action: its
// PermitWord when its registered type declares a valid one, else
// "<module>.<name>" for an action type "mod.<module>.<name>_action" whose
// parts are in [a-z0-9_]. It returns false for any other action, which the
// text counts but does not name.
//
// why only these shapes: a word holds no '.' and a shortened type holds one,
// so a shortened type renders like no word and no other type; both use only
// [a-z0-9_.], so no action forges the line's separators.
func actionWord(action string) (string, bool) {
	if w, ok := astral.New(action).(PermitWorder); ok {
		if word := w.PermitWord(); validPermitWord(word) {
			return word, true
		}
	}
	rest, ok := strings.CutPrefix(action, "mod.")
	if !ok {
		return "", false
	}
	rest, ok = strings.CutSuffix(rest, "_action")
	if !ok {
		return "", false
	}
	module, name, ok := strings.Cut(rest, ".")
	if !ok || !actionPart(module) || !actionPart(name) {
		return "", false
	}
	return module + "." + name, true
}

func validPermitWord(w string) bool {
	if w == "" || len(w) > MaxPermitWordLen {
		return false
	}
	for i := 0; i < len(w); i++ {
		if (w[i] < 'a' || w[i] > 'z') && w[i] != '_' {
			return false
		}
	}
	return true
}

func actionPart(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if (s[i] < 'a' || s[i] > 'z') && (s[i] < '0' || s[i] > '9') && s[i] != '_' {
			return false
		}
	}
	return true
}

// permitSummary is what a contract's signable text shows of its permits.
type permitSummary struct {
	words  []string // the actionWord of each distinct action that has one, in permit order
	hidden uint64   // distinct actions without an actionWord
	depth  uint8    // the largest Delegation
}

func (c *Contract) summarize() permitSummary {
	var s permitSummary
	seen := make(map[string]bool, len(c.Permits))
	for _, p := range c.Permits {
		if p == nil {
			continue
		}
		s.depth = max(s.depth, uint8(p.Delegation))
		if seen[string(p.Action)] {
			continue
		}
		seen[string(p.Action)] = true
		if w, ok := actionWord(string(p.Action)); ok {
			s.words = append(s.words, w)
		} else {
			s.hidden++
		}
	}
	return s
}

// contractText takes the summary apart from c so a test can render the worst
// case, e.g. math.MaxUint32 hidden actions, without building that many permits.
func contractText(c *Contract, s permitSummary) string {
	head := c.Issuer.String() + " grants " + c.Subject.String() + " "
	tail := fmt.Sprintf(" depth %d until %s", s.depth, c.ExpiresAt.Time().Format("2006-01-02 15:04:05"))
	return head + actionList(s, MaxContractTextLen-len(head)-len(tail)) + tail
}

// actionList renders the longest prefix of s.words that fits in budget bytes
// together with the "+<n>" count of the actions left out.
func actionList(s permitSummary, budget int) string {
	if len(s.words) == 0 && s.hidden == 0 {
		return "none"
	}
	total := uint64(len(s.words)) + s.hidden
	shown, used := 0, 0
	for _, w := range s.words {
		next := used + len(w)
		if shown > 0 {
			next++ // the comma before w
		}
		if next+overflowLen(shown+1, total) > budget {
			break
		}
		shown, used = shown+1, next
	}
	list := strings.Join(s.words[:shown], ",")
	if left := total - uint64(shown); left > 0 {
		if shown > 0 {
			list += ","
		}
		list += "+" + strconv.FormatUint(left, 10)
	}
	return list
}

// overflowLen is the length of the ",+<n>" or "+<n>" a list of shown words out
// of total actions ends with, or 0 when every action is shown.
func overflowLen(shown int, total uint64) int {
	left := total - uint64(shown)
	if left == 0 {
		return 0
	}
	n := 1 + len(strconv.FormatUint(left, 10))
	if shown > 0 {
		n++
	}
	return n
}

func init() {
	astral.MustAdd(&Contract{})
	astral.MustAdd(&Permit{})
}
