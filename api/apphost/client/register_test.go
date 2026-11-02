package apphost

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/api/services"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

func sampleToken() *apphost.AccessToken {
	return &apphost.AccessToken{
		Identity:  astral.GenerateIdentity(),
		Token:     astral.String8("s3cr3t"),
		ExpiresAt: astral.Time(time.Date(2036, time.May, 25, 12, 0, 0, 0, time.UTC)),
	}
}

// answer builds a channel over the objects the node sends back.
func answer(t *testing.T, objects ...astral.Object) *channel.Channel {
	t.Helper()

	var in, out bytes.Buffer
	s := channel.NewSender(&in)
	for _, o := range objects {
		if err := s.Send(o); err != nil {
			t.Fatalf("encode %s: %v", o.ObjectType(), err)
		}
	}

	return channel.New(channel.Join(&in, &out))
}

func discoveryPermit(t *testing.T, service astral.String8) *auth.Permit {
	t.Helper()
	scope := astral.NewBundle()
	if err := scope.Append(&services.DiscoveryScope{Rules: []*services.DiscoveryRule{{Services: []astral.String8{service}}}}); err != nil {
		t.Fatal(err)
	}
	return &auth.Permit{Action: astral.String8(services.ServiceDiscoveryAction{}.ObjectType()), Constraints: scope}
}

// noArguments requires the routed query to be method with no arguments. The
// op reads everything from the object on the channel.
func noArguments(t *testing.T, router *recordingRouter, method string) {
	t.Helper()
	path, params := query.Parse(router.queryString)
	if path != method {
		t.Fatalf("path: want %v, got %q", method, path)
	}
	if len(params) != 0 {
		t.Fatalf("want no arguments, got %v", params)
	}
}

// onlyObject requires the client to have sent exactly one object of type T.
func onlyObject[T astral.Object](t *testing.T, router *recordingRouter) T {
	t.Helper()
	sent := decodeAll(t, router.sent.Bytes())
	if len(sent) != 1 {
		t.Fatalf("sent %d objects; want one", len(sent))
	}
	obj, ok := sent[0].(T)
	if !ok {
		t.Fatalf("sent %s", sent[0].ObjectType())
	}
	return obj
}

func scopeServices(t *testing.T, p *auth.Permit) []astral.String8 {
	t.Helper()
	if p == nil || p.Constraints == nil {
		t.Fatalf("permit arrived without constraints: %+v", p)
	}
	scope, ok := astral.First[*services.DiscoveryScope](p.Constraints.Objects())
	if !ok || len(scope.Rules) != 1 {
		t.Fatalf("constraints arrived as %v", p.Constraints.Objects())
	}
	return scope.Rules[0].Services
}

func TestRegister_SendsOneRegisterRequestAndNoArguments(t *testing.T) {
	src := sampleToken()
	client, router := recordingClient(t, src)

	token, err := client.Register(astral.NewContext(nil), &auth.Permit{Action: "mod.user.see_swarm_action"}, &auth.Permit{Action: "mod.auth.see_objects_action"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if !token.Identity.IsEqual(src.Identity) {
		t.Fatalf("identity: want %v, got %v", src.Identity, token.Identity)
	}

	noArguments(t, router, "apphost.register")
	req := onlyObject[*apphost.RegisterRequest](t, router)
	if len(req.GrantPermits) != 2 || req.GrantPermits[0].Action != "mod.user.see_swarm_action" || req.GrantPermits[1].Action != "mod.auth.see_objects_action" {
		t.Fatalf("grant permits arrived as %+v", req.GrantPermits)
	}
	if req.Name != "" || len(req.ContractPermits) != 0 {
		t.Fatalf("Register sent %+v", req)
	}
}

func TestRegisterWith_SendsNameBothListsAndConstraints(t *testing.T) {
	client, router := recordingClient(t, sampleToken())

	_, err := client.RegisterWith(astral.NewContext(nil), &apphost.RegisterRequest{
		Name:            "smithd@dell /home/robzon/.smith/daemon",
		GrantPermits:    []*auth.Permit{discoveryPermit(t, "inference")},
		ContractPermits: []*auth.Permit{{Action: "mod.auth.store_objects_action"}},
	})
	if err != nil {
		t.Fatalf("RegisterWith: %v", err)
	}

	noArguments(t, router, "apphost.register")
	req := onlyObject[*apphost.RegisterRequest](t, router)
	if req.Name != "smithd@dell /home/robzon/.smith/daemon" {
		t.Fatalf("name: got %q", req.Name)
	}
	if len(req.GrantPermits) != 1 || req.GrantPermits[0].Action != "mod.services.service_discovery_action" {
		t.Fatalf("grant permits arrived as %+v", req.GrantPermits)
	}
	if got := scopeServices(t, req.GrantPermits[0]); len(got) != 1 || got[0] != "inference" {
		t.Fatalf("scope services: got %v", got)
	}
	if len(req.ContractPermits) != 1 || req.ContractPermits[0].Action != "mod.auth.store_objects_action" {
		t.Fatalf("contract permits arrived as %+v", req.ContractPermits)
	}
}

// The op reads one object after accept, so an app asking for nothing still
// sends one.
func TestRegister_WithNoPermitsStillSendsTheObject(t *testing.T) {
	client, router := recordingClient(t, sampleToken())

	if _, err := client.Register(astral.NewContext(nil)); err != nil {
		t.Fatalf("Register: %v", err)
	}

	noArguments(t, router, "apphost.register")
	req := onlyObject[*apphost.RegisterRequest](t, router)
	if req.Name != "" || len(req.GrantPermits) != 0 || len(req.ContractPermits) != 0 {
		t.Fatalf("Register with no permits sent %+v", req)
	}
}

func TestRegisterWith_NilRequestIsRefused(t *testing.T) {
	client, router := recordingClient(t, sampleToken())

	_, err := client.RegisterWith(astral.NewContext(nil), nil)
	if !errors.Is(err, apphost.ErrProtocolError) {
		t.Fatalf("want ErrProtocolError, got %v", err)
	}
	if router.queryString != "" {
		t.Fatalf("a nil request routed %q", router.queryString)
	}
}

func TestReadAccessToken_ReturnsTheMintedToken(t *testing.T) {
	src := sampleToken()

	token, err := readAccessToken(astral.NewContext(nil), answer(t, src))
	if err != nil {
		t.Fatalf("readAccessToken: %v", err)
	}

	if !token.Identity.IsEqual(src.Identity) {
		t.Fatalf("identity: want %v, got %v", src.Identity, token.Identity)
	}
	if token.Token != src.Token {
		t.Fatalf("token: want %v, got %v", src.Token, token.Token)
	}
}

// A failure past the accept gate arrives as an error object, not as a token.
func TestReadAccessToken_NodeRefusalIsReturnedNotSwallowed(t *testing.T) {
	token, err := readAccessToken(astral.NewContext(nil), answer(t, astral.NewError("register policy refused")))
	if err == nil {
		t.Fatal("expected the node's refusal to surface")
	}
	if token != nil {
		t.Fatalf("expected no token after a refusal, got %v", token)
	}
}

// Registration has no empty success: an answer carrying nothing must not read
// as a token the caller can use.
func TestReadAccessToken_EmptyAnswerIsAProtocolError(t *testing.T) {
	token, err := readAccessToken(astral.NewContext(nil), answer(t))
	if err == nil {
		t.Fatal("expected an empty answer to be an error")
	}
	if !errors.Is(err, apphost.ErrProtocolError) {
		t.Fatalf("want ErrProtocolError, got %v", err)
	}
	if token != nil {
		t.Fatalf("expected no token, got %v", token)
	}
}
