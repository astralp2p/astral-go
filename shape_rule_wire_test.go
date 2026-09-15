package pub_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/api/nat"
	"github.com/astralp2p/astral-go/api/services"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/log"
	"github.com/astralp2p/astral-go/lib/query"
	"github.com/astralp2p/astral-go/lib/routing"
)

// The reflection codec refuses a struct field no Blueprint describes, so every type that held
// one moved to an astral field type or to a hand-written codec. None of them was meant to
// change on the wire. The vectors below are what each type wrote — binary and JSON — at
// astral-go 6ea26c7, before the move.

type wireVector struct {
	name string
	hex  string
	json string
}

var wireVectors = []wireVector{
	{"query", "0102030405060708010282fee8775757cdd8fda8b220195f5b8611312cd145c5a1a3aa55df210e779b2c010282fee8775757cdd8fda8b220195f5b8611312cd145c5a1a3aa55df210e779b2c000000206f626a656374732e6765745f626c75657072696e743f747970653d7175657279", "{\"Caller\":\"0282fee8775757cdd8fda8b220195f5b8611312cd145c5a1a3aa55df210e779b2c\",\"Nonce\":\"102030405060708\",\"QueryString\":\"objects.get_blueprint?type=query\",\"Target\":\"0282fee8775757cdd8fda8b220195f5b8611312cd145c5a1a3aa55df210e779b2c\"}"},
	{"services.update", "0103737663010282fee8775757cdd8fda8b220195f5b8611312cd145c5a1a3aa55df210e779b2c0100000000", "{\"Available\":true,\"Name\":\"svc\",\"ProviderID\":\"0282fee8775757cdd8fda8b220195f5b8611312cd145c5a1a3aa55df210e779b2c\",\"Info\":null}"},
	{"nat.consume_hole_signal", "066c6f636b656401020304050607080100", "{\"Signal\":\"locked\",\"Pair\":\"102030405060708\",\"Ok\":true,\"Error\":\"\"}"},
	{"astrald.log.entry", "010282fee8775757cdd8fda8b220195f5b8611312cd145c5a1a3aa55df210e779b2c0317979cfe362a00000000000107737472696e6738026869", "{\"Level\":3,\"Objects\":[{\"Type\":\"string8\",\"Object\":\"hi\"}],\"Origin\":\"0282fee8775757cdd8fda8b220195f5b8611312cd145c5a1a3aa55df210e779b2c\",\"Time\":\"2023-11-14T22:13:20Z\"}"},
	{"routing.op_spec", "0000000d6765745f626c75657072696e740000000201000000047479706500000007737472696e67380101000000036f757400000007737472696e673800", "{\"Name\":\"get_blueprint\",\"Parameters\":[{\"Name\":\"type\",\"Required\":true,\"Type\":\"string8\"},{\"Name\":\"out\",\"Required\":false,\"Type\":\"string8\"}]}"},
	{"routing.op_spec/no_parameters", "000000047370656300000000", "{\"Name\":\"spec\",\"Parameters\":[]}"},
	{"mod.auth.see_objects_action", "0102030405060708010282fee8775757cdd8fda8b220195f5b8611312cd145c5a1a3aa55df210e779b2c00046d61696e", "{\"Nonce\":\"102030405060708\",\"ActorID\":\"0282fee8775757cdd8fda8b220195f5b8611312cd145c5a1a3aa55df210e779b2c\",\"ObjectID\":null,\"Repo\":\"main\"}"},
	{"mod.auth.admin_manage_apps_action/zero", "000000000000000000", "{\"Nonce\":\"0\",\"ActorID\":null}"},
	{"mod.auth.admin_network_action/zero", "000000000000000000", "{\"Nonce\":\"0\",\"ActorID\":null}"},
	{"mod.auth.admin_objects_action/zero", "000000000000000000000000", "{\"Nonce\":\"0\",\"ActorID\":null,\"ObjectID\":null,\"Repo\":\"\",\"Path\":\"\"}"},
	{"mod.auth.configure_node_state_action/zero", "000000000000000000", "{\"Nonce\":\"0\",\"ActorID\":null}"},
	{"mod.auth.see_node_state_action/zero", "000000000000000000", "{\"Nonce\":\"0\",\"ActorID\":null}"},
	{"mod.auth.see_objects_action/zero", "0000000000000000000000", "{\"Nonce\":\"0\",\"ActorID\":null,\"ObjectID\":null,\"Repo\":\"\"}"},
	{"mod.auth.serve_apps_action/zero", "000000000000000000", "{\"Nonce\":\"0\",\"ActorID\":null}"},
	{"mod.auth.serve_objects_action/zero", "00000000000000000000", "{\"Nonce\":\"0\",\"ActorID\":null,\"Role\":\"\"}"},
	{"mod.auth.store_objects_action/zero", "0000000000000000000000", "{\"Nonce\":\"0\",\"ActorID\":null,\"Repo\":\"\",\"Type\":\"\"}"},
	{"mod.auth.use_gateway_action/zero", "000000000000000000", "{\"Nonce\":\"0\",\"ActorID\":null}"},
	{"mod.coldcard.scan_action/zero", "000000000000000000", "{\"Nonce\":\"0\",\"ActorID\":null}"},
	{"mod.mcp.answer_agent_action/zero", "00000000000000000000", "{\"Action\":{\"ActorID\":null,\"Nonce\":\"0\"},\"FromID\":null}"},
	{"mod.mcp.call_agent_action/zero", "00000000000000000000", "{\"Action\":{\"ActorID\":null,\"Nonce\":\"0\"},\"ToID\":null}"},
	{"mod.nodes.relay_for_action/zero", "00000000000000000000", "{\"Nonce\":\"0\",\"ActorID\":null,\"ForID\":null}"},
	{"mod.objects.create_object_action/zero", "000000000000000000", "{\"Nonce\":\"0\",\"ActorID\":null}"},
	{"mod.user.admin_swarm_action/zero", "0000000000000000000000", "{\"Nonce\":\"0\",\"ActorID\":null,\"Subject\":null,\"ObjectID\":null}"},
	{"mod.user.see_swarm_action/zero", "000000000000000000", "{\"Nonce\":\"0\",\"ActorID\":null}"},
	{"mod.user.swarm_membership_action/zero", "000000000000000000", "{\"Nonce\":\"0\",\"ActorID\":null}"},
	{"query/zero", "0000000000000000000000000000", "{\"Caller\":null,\"Nonce\":\"0\",\"QueryString\":\"\",\"Target\":null}"},
	{"services.update/zero", "00000000", "{\"Available\":false,\"Name\":\"\",\"ProviderID\":null,\"Info\":null}"},
	{"nat.consume_hole_signal/zero", "0000000000000000000000", "{\"Signal\":\"\",\"Pair\":\"0\",\"Ok\":false,\"Error\":\"\"}"},
	{"astrald.log.entry/zero", "0000a1b203eb3d1a000000000000", "{\"Level\":0,\"Objects\":[],\"Origin\":null,\"Time\":\"0001-01-01T00:00:00Z\"}"},
	{"routing.op_spec/zero", "0000000000000000", "{\"Name\":\"\",\"Parameters\":[]}"},
}

// shapeRuleTypes are the registered types whose fields moved.
var shapeRuleTypes = []string{
	"mod.auth.admin_manage_apps_action", "mod.auth.admin_network_action", "mod.auth.admin_objects_action",
	"mod.auth.configure_node_state_action", "mod.auth.see_node_state_action", "mod.auth.see_objects_action",
	"mod.auth.serve_apps_action", "mod.auth.serve_objects_action", "mod.auth.store_objects_action",
	"mod.auth.use_gateway_action", "mod.coldcard.scan_action", "mod.mcp.answer_agent_action",
	"mod.mcp.call_agent_action", "mod.nodes.relay_for_action", "mod.objects.create_object_action",
	"mod.user.admin_swarm_action", "mod.user.see_swarm_action", "mod.user.swarm_membership_action",
	"query", "services.update", "nat.consume_hole_signal", "astrald.log.entry", "routing.op_spec",
}

func wireSamples(t *testing.T) map[string]astral.Object {
	id, err := astral.ParseIdentity("0282fee8775757cdd8fda8b220195f5b8611312cd145c5a1a3aa55df210e779b2c")
	if err != nil {
		t.Fatal(err)
	}
	nonce := astral.Nonce(0x0102030405060708)
	ts := astral.Time(time.Unix(1_700_000_000, 0).UTC())
	msg := astral.String8("hi")

	samples := map[string]astral.Object{
		"query":                   &astral.Query{Nonce: nonce, Caller: id, Target: id, QueryString: "objects.get_blueprint?type=query"},
		"services.update":         &services.Update{Available: true, Name: "svc", ProviderID: id, Info: astral.NewBundle()},
		"nat.consume_hole_signal": &nat.ConsumeHoleSignal{Signal: "locked", Pair: nonce, Ok: true},
		"astrald.log.entry":       &log.Entry{Origin: id, Level: 3, Time: ts, Objects: []astral.Object{&msg}},
		"routing.op_spec": &routing.OpSpec{Name: "get_blueprint", Parameters: []query.FieldSpec{
			{Name: "type", Type: "string8", Required: true}, {Name: "out", Type: "string8"}}},
		"routing.op_spec/no_parameters": &routing.OpSpec{Name: "spec"},
		"mod.auth.see_objects_action":   &auth.SeeObjectsAction{Action: auth.Action{Nonce: nonce, ActorID: id}, Repo: "main"},
	}
	for _, name := range shapeRuleTypes {
		samples[name+"/zero"] = astral.New(name)
	}
	return samples
}

func TestShapeRule_TypesKeepTheirWireBytes(t *testing.T) {
	samples := wireSamples(t)

	for _, v := range wireVectors {
		t.Run(v.name, func(t *testing.T) {
			obj := samples[v.name]
			if obj == nil {
				t.Fatalf("no sample named %s", v.name)
			}

			var buf bytes.Buffer
			if _, err := obj.WriteTo(&buf); err != nil {
				t.Fatalf("write: %v", err)
			}
			if got := hex.EncodeToString(buf.Bytes()); got != v.hex {
				t.Errorf("binary changed:\n want %s\n  got %s", v.hex, got)
			}

			j, err := json.Marshal(obj)
			if err != nil {
				t.Fatalf("json: %v", err)
			}
			if string(j) != v.json {
				t.Errorf("JSON changed:\n want %s\n  got %s", v.json, j)
			}
		})
	}
}

// The old bytes decode, and re-encode to themselves.
func TestShapeRule_OldBytesDecode(t *testing.T) {
	samples := wireSamples(t)

	for _, v := range wireVectors {
		t.Run(v.name, func(t *testing.T) {
			raw, _ := hex.DecodeString(v.hex)
			dst := reflect.New(reflect.TypeOf(samples[v.name]).Elem()).Interface().(astral.Object)

			if _, err := dst.ReadFrom(bytes.NewReader(raw)); err != nil {
				t.Fatalf("read: %v", err)
			}

			var buf bytes.Buffer
			if _, err := dst.WriteTo(&buf); err != nil {
				t.Fatalf("write: %v", err)
			}
			if got := hex.EncodeToString(buf.Bytes()); got != v.hex {
				t.Errorf("re-encoding changed:\n want %s\n  got %s", v.hex, got)
			}
		})
	}
}

// Every moved type derives a Blueprint, except routing.op_spec: a parameter entry has no
// registered Object Type, so op_spec encodes by hand and stays undescribable.
func TestShapeRule_MovedTypesDeriveABlueprint(t *testing.T) {
	for _, name := range append(shapeRuleTypes, "mod.auth.action") {
		t.Run(name, func(t *testing.T) {
			_, err := astral.BlueprintOf(astral.New(name))
			switch {
			case name == "routing.op_spec" && err == nil:
				t.Fatal("routing.op_spec derived a Blueprint; its parameter entry has no registered type")
			case name != "routing.op_spec" && err != nil:
				t.Fatalf("no Blueprint: %v", err)
			}
		})
	}
}
