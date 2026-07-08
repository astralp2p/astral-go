package astral

import "testing"

// TestUnmarshalFieldJSON_ObjectSpec_NullPayload pins the nested analog of the
// JSONReceiver panic. An ObjectSpec field carrying a known type with an explicit
// null payload — {"Type":"ack","Object":null} — must decode to a non-nil Object.
//
// Before the fix the inner json.Unmarshal(env.Object, &obj) nils the interface on the
// null literal, so unmarshalFieldJSON returned (nil, nil): a nil field value that a
// downstream ObjectType()/WriteTo() call would panic on. Note this is distinct from a
// fully-null field ("null"), which the guard above resolves to &Nil{}.
func TestUnmarshalFieldJSON_ObjectSpec_NullPayload(t *testing.T) {
	obj, err := unmarshalFieldJSON(&ObjectSpec{}, []byte(`{"Type":"ack","Object":null}`))
	if err != nil {
		t.Fatalf("unmarshalFieldJSON returned error: %v", err)
	}
	if obj == nil {
		t.Fatal("null payload decoded to a nil field value — downstream ObjectType() would panic")
	}
	if got := obj.ObjectType(); got != "ack" {
		t.Fatalf("ObjectType: want %q, got %q", "ack", got)
	}
}

// TestUnmarshalFieldJSON_ObjectSpec_FullNullStillNil confirms the fix leaves the
// fully-null field semantics intact: a bare null resolves to the canonical &Nil{}.
func TestUnmarshalFieldJSON_ObjectSpec_FullNullStillNil(t *testing.T) {
	obj, err := unmarshalFieldJSON(&ObjectSpec{}, []byte(`null`))
	if err != nil {
		t.Fatalf("unmarshalFieldJSON returned error: %v", err)
	}
	if _, ok := obj.(*Nil); !ok {
		t.Fatalf("bare null: want *Nil, got %T", obj)
	}
}
