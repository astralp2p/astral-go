package user

import (
	"encoding/json"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// swarmEntryEvents returns one populated value of every SwarmState event
// type, and one with its optional IDs left nil.
func swarmEntryEvents() []astral.Object {
	issuer, subject := astral.GenerateIdentity(), astral.GenerateIdentity()
	return []astral.Object{
		&SwarmRequestFiled{RequestID: sampleRequestID(), Kind: astral.String8((&SignedJoinRequest{}).ObjectType())},
		&SwarmRequestApproved{RequestID: sampleRequestID(), ApprovalID: sampleRequestID()},
		&SwarmRequestApproved{RequestID: sampleRequestID()},
		&SwarmRequestDeclined{RequestID: sampleRequestID()},
		&SwarmRequestExpired{RequestID: sampleRequestID()},
		&SwarmContractAdded{ContractID: sampleRequestID(), Issuer: issuer, Subject: subject, ApprovalID: sampleRequestID()},
		&SwarmContractAdded{ContractID: sampleRequestID(), Issuer: issuer, Subject: subject},
	}
}

func TestSwarmEntryEvents_BinaryRoundTrip(t *testing.T) {
	for _, orig := range swarmEntryEvents() {
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

func TestSwarmEntryEvents_JSONRoundTrip(t *testing.T) {
	for _, orig := range swarmEntryEvents() {
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
