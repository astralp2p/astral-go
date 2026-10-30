package astrald

import (
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	libapphost "github.com/astralp2p/astral-go/lib/apphost"
	"github.com/astralp2p/astral-go/lib/query"
)

// Client wraps a Router with an optional fixed target identity, used to direct queries
// without the caller having to supply the target on every call.
type Client struct {
	Router
	targetID *astral.Identity
}

var defaultClient *Client

func New(router Router) *Client {
	return &Client{Router: router}
}

func Default() *Client {
	if defaultClient == nil {
		defaultClient = New(libapphost.DefaultRouter())
	}

	return defaultClient
}

func SetDefault(client *Client) {
	defaultClient = client
}

// Query routes an outbound query to client.targetID using the client's own guest identity as the caller.
func (client *Client) Query(ctx *astral.Context, method string, args any) (astral.Conn, error) {
	return client.RouteQuery(ctx, astral.Launch(query.New(client.guestID(ctx), client.targetID, method, args)))
}

// guestID is the caller of the client's queries, resolved within ctx when the
// router can resolve it so.
//
// why: lib/apphost's Router connects to learn the identity on first use, and
// GuestID connects with no context, so a host that never answered held a
// query past its ctx before the query was sent.
//
// A failure leaves the caller unset, as GuestID's nil does; RouteQuery then
// meets the same failure and returns it.
func (client *Client) guestID(ctx *astral.Context) *astral.Identity {
	id, _ := guestIDContext(ctx, client.Router)
	return id
}

// guestResolver is a Router that resolves its guest identity within a
// context, as lib/apphost's Router does.
type guestResolver interface {
	GuestIDContext(*astral.Context) (*astral.Identity, error)
}

// guestIDContext returns router's guest identity, resolved within ctx when
// router can resolve it so, and router.GuestID() otherwise.
func guestIDContext(ctx *astral.Context, router Router) (*astral.Identity, error) {
	if r, ok := router.(guestResolver); ok {
		return r.GuestIDContext(ctx)
	}
	return router.GuestID(), nil
}

func Query(ctx *astral.Context, method string, args any) (astral.Conn, error) {
	return Default().Query(ctx, method, args)
}

func (client *Client) QueryChannel(ctx *astral.Context, method string, args any, cfg ...channel.ConfigFunc) (*channel.Channel, error) {
	conn, err := client.Query(ctx, method, args)
	if err != nil {
		return nil, err
	}

	return channel.New(conn, cfg...), nil
}

func QueryChannel(ctx *astral.Context, method string, args any, cfg ...channel.ConfigFunc) (*channel.Channel, error) {
	return Default().QueryChannel(ctx, method, args, cfg...)
}

// WithTarget returns a shallow copy of the client with the target identity set; the original is not modified.
func (client *Client) WithTarget(identity *astral.Identity) *Client {
	c := *client
	c.targetID = identity
	return &c
}

func WithTarget(identity *astral.Identity) *Client {
	return Default().WithTarget(identity)
}

func GuestID() *astral.Identity {
	return Default().GuestID()
}

func HostID() *astral.Identity {
	return Default().HostID()
}

func RouteQuery(ctx *astral.Context, query *astral.InFlightQuery) (astral.Conn, error) {
	return Default().RouteQuery(ctx, query)
}
