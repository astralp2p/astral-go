package user

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/astral"
)

func dummySig() *crypto.Signature {
	return &crypto.Signature{Scheme: crypto.SchemeASN1, Data: astral.Bytes16("dummy-signature-bytes")}
}

func sampleRequestID() *astral.ObjectID {
	id := &astral.ObjectID{Size: 77}
	copy(id.Hash[:], bytes.Repeat([]byte{0x5a}, 32))
	return id
}

// signedEntries returns one populated value of every new signed entry type.
func signedEntries() []astral.Object {
	filer, subject := astral.GenerateIdentity(), astral.GenerateIdentity()
	cr := NewContractRequest(filer, subject, []*auth.Permit{{Action: "test.action", Delegation: 2}})
	return []astral.Object{
		&SignedContractRequest{ContractRequest: cr, FiledBySig: dummySig()},
		&SignedJoinRequest{JoinRequest: NewJoinRequest(subject, filer, TierTrusted), JoinerSig: dummySig()},
		&SignedExpelRequest{ExpelRequest: NewExpelRequest(filer, subject), FiledBySig: dummySig()},
		&SignedApproval{Approval: NewApproval(filer, sampleRequestID()), IssuerSig: dummySig()},
		&SignedDecline{Decline: NewDecline(filer, sampleRequestID()), IssuerSig: dummySig()},
	}
}

// sameObject compares two objects by ObjectID, which covers every field the
// codec writes, and by their JSON form, which shows a readable diff.
func sameObject(t *testing.T, want, got astral.Object) {
	t.Helper()
	wantID, err := astral.ResolveObjectID(want)
	if err != nil {
		t.Fatalf("resolve want: %v", err)
	}
	gotID, err := astral.ResolveObjectID(got)
	if err != nil {
		t.Fatalf("resolve got: %v", err)
	}
	if *wantID != *gotID {
		wj, _ := json.Marshal(want)
		gj, _ := json.Marshal(got)
		t.Fatalf("%s changed across round trip:\nwant %s\ngot  %s", want.ObjectType(), wj, gj)
	}
}

func TestSwarmEntries_BinaryRoundTrip(t *testing.T) {
	for _, orig := range signedEntries() {
		data, err := astral.EncodeBytes(orig)
		if err != nil {
			t.Fatalf("%s encode: %v", orig.ObjectType(), err)
		}
		got, err := astral.DecodeAs[astral.Object](data)
		if err != nil {
			t.Fatalf("%s decode: %v", orig.ObjectType(), err)
		}
		sameObject(t, orig, got)
	}
}

func TestSwarmEntries_JSONRoundTrip(t *testing.T) {
	for _, orig := range signedEntries() {
		data, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("%s marshal: %v", orig.ObjectType(), err)
		}
		got := astral.New(orig.ObjectType())
		if got == nil {
			t.Fatalf("%s not registered", orig.ObjectType())
		}
		if err := json.Unmarshal(data, got); err != nil {
			t.Fatalf("%s unmarshal: %v", orig.ObjectType(), err)
		}
		sameObject(t, orig, got)
	}
}

// TestSwarmEntries_BodyRoundTrip covers the unsigned bodies, which carry the
// signable hash.
func TestSwarmEntries_BodyRoundTrip(t *testing.T) {
	for _, e := range signedEntries() {
		body := e.(interface{ SignableHash() []byte })
		var obj astral.Object
		switch v := e.(type) {
		case *SignedContractRequest:
			obj = v.ContractRequest
		case *SignedJoinRequest:
			obj = v.JoinRequest
		case *SignedExpelRequest:
			obj = v.ExpelRequest
		case *SignedApproval:
			obj = v.Approval
		case *SignedDecline:
			obj = v.Decline
		}
		data, err := astral.EncodeBytes(obj)
		if err != nil {
			t.Fatalf("%s encode: %v", obj.ObjectType(), err)
		}
		bin, err := astral.DecodeAs[astral.Object](data)
		if err != nil {
			t.Fatalf("%s decode: %v", obj.ObjectType(), err)
		}
		sameObject(t, obj, bin)

		j, err := json.Marshal(obj)
		if err != nil {
			t.Fatalf("%s marshal: %v", obj.ObjectType(), err)
		}
		js := astral.New(obj.ObjectType())
		if err := json.Unmarshal(j, js); err != nil {
			t.Fatalf("%s unmarshal: %v", obj.ObjectType(), err)
		}
		sameObject(t, obj, js)

		if !bytes.Equal(body.SignableHash(), bin.(interface{ SignableHash() []byte }).SignableHash()) {
			t.Fatalf("%s signable hash changed", obj.ObjectType())
		}
	}
}

func TestSwarmEnums_RoundTrip(t *testing.T) {
	values := []astral.Object{}
	for _, v := range Tiers {
		v := v
		values = append(values, &v)
	}
	for _, v := range []RequestStatus{RequestPending, RequestApproved, RequestDeclined, RequestExpired} {
		v := v
		values = append(values, &v)
	}
	for _, v := range []EntryVerdict{EntryAccept, EntryDrop, EntryWait} {
		v := v
		values = append(values, &v)
	}
	for _, v := range EntryKinds {
		v := v
		values = append(values, &v)
	}

	for _, orig := range values {
		data, err := astral.EncodeBytes(orig)
		if err != nil {
			t.Fatalf("%s encode: %v", orig.ObjectType(), err)
		}
		bin, err := astral.DecodeAs[astral.Object](data)
		if err != nil {
			t.Fatalf("%s decode: %v", orig.ObjectType(), err)
		}
		sameObject(t, orig, bin)

		j, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("%s marshal: %v", orig.ObjectType(), err)
		}
		js := astral.New(orig.ObjectType())
		if err := json.Unmarshal(j, js); err != nil {
			t.Fatalf("%s unmarshal: %v", orig.ObjectType(), err)
		}
		sameObject(t, orig, js)
	}
}

func TestSwarmEnums_IsValid(t *testing.T) {
	if Tier("").IsValid() || Tier("admin").IsValid() || RequestStatus("").IsValid() ||
		EntryVerdict("").IsValid() || EntryKind("tmp").IsValid() {
		t.Fatal("an undeclared value reports valid")
	}
	if !TierPrivilegedMember.IsValid() || !RequestExpired.IsValid() || !EntryWait.IsValid() || !EntryExpels.IsValid() {
		t.Fatal("a declared value reports invalid")
	}
}

func TestEntryKindOf(t *testing.T) {
	want := []EntryKind{EntryRequests, EntryRequests, EntryRequests, EntryApprovals, EntryDeclines}
	for i, e := range signedEntries() {
		if got, ok := EntryKindOf(e); !ok || got != want[i] {
			t.Errorf("%s: got %q %v, want %q", e.ObjectType(), got, ok, want[i])
		}
	}
	if k, ok := EntryKindOf(&auth.SignedContract{}); !ok || k != EntryContracts {
		t.Errorf("signed contract: got %q %v", k, ok)
	}
	if k, ok := EntryKindOf(sampleSignedExpulsion()); !ok || k != EntryExpels {
		t.Errorf("expulsion: got %q %v", k, ok)
	}
	if _, ok := EntryKindOf(&Approval{}); ok {
		t.Error("an unsigned body is not an entry")
	}
}

func TestRequestTimeout(t *testing.T) {
	filed := astral.Time(time.Unix(1700000000, 0))
	at := func(d time.Duration) astral.Time { return astral.Time(filed.Time().Add(d)) }

	for d, want := range map[time.Duration]bool{
		0:                      false,
		-time.Second:           false,
		time.Second:            true,
		RequestTimeout:         true,
		RequestTimeout + 1:     false,
		2 * RequestTimeout:     false,
		RequestTimeout / 2:     true,
		-RequestTimeout - 1:    false,
		RequestTimeout - 1:     true,
		time.Nanosecond:        true,
		time.Duration(1 << 62): false,
	} {
		if got := ValidRequestTimeout(filed, at(d)); got != want {
			t.Errorf("ValidRequestTimeout(+%v) = %v, want %v", d, got, want)
		}
	}
	if !RequestDeadline(filed).Time().Equal(at(RequestTimeout).Time()) {
		t.Error("RequestDeadline is not FiledAt + RequestTimeout")
	}

	r := NewJoinRequest(astral.GenerateIdentity(), astral.GenerateIdentity(), TierMember)
	if !r.ValidTimeout() || r.Expired(r.FiledAt.Time()) || !r.Expired(r.Timeout.Time().Add(time.Nanosecond)) {
		t.Error("a new request has the default timeout")
	}
}

func TestNewMembership(t *testing.T) {
	issuer, subject := astral.GenerateIdentity(), astral.GenerateIdentity()
	for _, tier := range Tiers {
		for _, approval := range []*astral.ObjectID{nil, sampleRequestID()} {
			c, err := NewMembership(issuer, subject, tier, approval, time.Hour)
			if err != nil {
				t.Fatalf("%s: %v", tier, err)
			}
			if got, ok := ContractTier(c); !ok || got != tier {
				t.Fatalf("ContractTier: got %q %v, want %q", got, ok, tier)
			}
			if c.ApprovalID != approval {
				t.Fatalf("approvalID not carried")
			}
			if err := auth.ValidatePermit(c.Permits[0]); err != nil {
				t.Fatalf("%s permit invalid: %v", tier, err)
			}
			if !strings.Contains(c.SignableText(), " "+string(tier)+" permits (1) until ") {
				t.Fatalf("signable text lacks the tier word: %q", c.SignableText())
			}

			m, ok := MembershipOf(&auth.SignedContract{Contract: c})
			if !ok || m.Tier != tier || !m.Subject.IsEqual(subject) {
				t.Fatalf("MembershipOf: %+v %v", m, ok)
			}
		}
	}

	for _, bad := range []struct {
		issuer, subject *astral.Identity
		tier            Tier
		d               time.Duration
	}{
		{nil, subject, TierMember, time.Hour},
		{issuer, nil, TierMember, time.Hour},
		{issuer, subject, "admin", time.Hour},
		{issuer, subject, TierMember, 0},
	} {
		if _, err := NewMembership(bad.issuer, bad.subject, bad.tier, nil, bad.d); err == nil {
			t.Errorf("NewMembership(%v, %v, %q, %v) succeeded", bad.issuer, bad.subject, bad.tier, bad.d)
		}
	}
}

func TestContractTier_RejectsNonMemberships(t *testing.T) {
	two := &auth.Contract{Permits: []*auth.Permit{
		{Action: astral.String8(TierMember.Action())},
		{Action: astral.String8(TierTrusted.Action())},
	}}
	node, _ := NewNodeContract(astral.GenerateIdentity(), astral.GenerateIdentity(), true, time.Hour)
	for _, c := range []*auth.Contract{nil, {}, two, node} {
		if tier, ok := ContractTier(c); ok {
			t.Errorf("ContractTier(%+v) = %q", c, tier)
		}
	}
	if _, ok := MembershipOf(nil); ok {
		t.Error("MembershipOf(nil) succeeded")
	}
	if TierMember.Action() == "" || Tier("admin").Action() != "" {
		t.Error("Tier.Action")
	}
}

func TestTierActions_RefuseConstraints(t *testing.T) {
	cs := astral.NewBundle()
	if err := cs.Append(astral.NewString8("x")); err != nil {
		t.Fatal(err)
	}
	for _, a := range []auth.Constrainable{MemberAction{}, TrustedAction{}, PrivilegedMemberAction{}} {
		if !a.ApplyConstraints(nil) || a.ApplyConstraints(cs) {
			t.Errorf("%T: constraints are refused, none are allowed", a)
		}
	}
}

// --- signable text: the worst case of every signable type fits a Coldcard ---

// signableLine is the text a text signer such as a Coldcard signs: the 15-byte
// commitment in base64, then SignableText (astrald mod/crypto formatSignableText).
func signableLine(t *testing.T, obj crypto.SignableTextObject, text string) string {
	t.Helper()
	hash := obj.SignableHash()
	if len(hash) < 15 {
		t.Fatalf("%T: no signable hash", obj)
	}
	return "[" + base64.StdEncoding.EncodeToString(hash[0:15]) + "] " + text
}

// worstIdentity renders like every identity, as 66 hex characters.
func worstIdentity() *astral.Identity { return astral.GenerateIdentity() }

// worstObjectID renders at full length: no leading zero digit to trim.
func worstObjectID() *astral.ObjectID {
	id := &astral.ObjectID{Size: math.MaxUint64}
	copy(id.Hash[:], bytes.Repeat([]byte{0xff}, 32))
	return id
}

// worstTime is the latest astral.Time (UnixNano in 64 bits).
var worstTime = astral.Time(time.Unix(0, math.MaxInt64))

func TestSignableText_WorstCase(t *testing.T) {
	cr := &ContractRequest{FiledBy: worstIdentity(), Identity: worstIdentity(), Timeout: worstTime}
	jr := &JoinRequest{Joiner: worstIdentity(), Carrier: worstIdentity(), Tier: TierPrivilegedMember, Timeout: worstTime}
	er := &ExpelRequest{FiledBy: worstIdentity(), Subject: worstIdentity(), Timeout: worstTime}
	ap := &Approval{Issuer: worstIdentity(), RequestID: worstObjectID()}
	de := &Decline{Issuer: worstIdentity(), RequestID: worstObjectID()}
	ex := &Expulsion{Issuer: worstIdentity(), Subject: worstIdentity(), ExpelledAt: worstTime}
	ms, err := NewMembership(worstIdentity(), worstIdentity(), TierPrivilegedMember, worstObjectID(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	ms.ExpiresAt = worstTime

	for _, tc := range []struct {
		obj  crypto.SignableTextObject
		text string
	}{
		{cr, contractRequestText(cr, math.MaxUint32)},
		{jr, jr.SignableText()},
		{&JoinRequest{Tier: Tier(strings.Repeat("x", 255)), Timeout: worstTime}, (&JoinRequest{Tier: Tier(strings.Repeat("x", 255))}).SignableText()},
		{er, er.SignableText()},
		{ap, ap.SignableText()},
		{de, de.SignableText()},
		{ex, ex.SignableText()},
		{ms, ms.SignableText()}, // one tier permit; auth tests the permit-count worst case
	} {
		line := signableLine(t, tc.obj, tc.text)
		if len(line) > 240 {
			t.Errorf("%T: %d bytes, over 240: %q", tc.obj, len(line), line)
		}
		for i := 0; i < len(line); i++ {
			if line[i] < 0x20 || line[i] > 0x7e {
				t.Errorf("%T: byte %d is not printable ASCII: %q", tc.obj, i, line)
				break
			}
		}
		t.Logf("%-22T %3d bytes  %s", tc.obj, len(line), line)
	}
}
