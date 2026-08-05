package apphost

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

// cancelTimeout bounds the dial that delivers apphost.cancel. The cancel is best
// effort: the host ends the query on its own deadline if it never arrives.
const cancelTimeout = 5 * time.Second

// Router manages connections to an apphost endpoint, caching resolved identities
// across calls and handling context-driven query cancellation.
type Router struct {
	endpoint string
	token    string

	// mu guards the cached identities: connect writes them from whichever goroutine
	// dials, including the cancel watcher, while GuestID and HostID read them.
	mu      sync.Mutex
	guestID *astral.Identity
	hostID  *astral.Identity
}

var defaultRouter = newDefaultRouter()

func NewRouter(endpoint string, token string) *Router {
	return &Router{endpoint: endpoint, token: token}
}

func DefaultRouter() *Router {
	return defaultRouter
}

func SetDefaultRouter(router *Router) {
	defaultRouter = router
}

// RouteQuery routes a query via the host. Returns ErrNodeUnavailable if the
// IPC connection cannot be established (query was never sent, safe to retry).
func (router *Router) RouteQuery(ctx *astral.Context, q *astral.InFlightQuery) (astral.Conn, error) {
	host, err := router.connect(ctx)
	if err != nil {
		return nil, err
	}

	// cancel the query while it is still en route. The host only holds a cancellable
	// entry between accepting the query and routing it, which is exactly the window
	// this call blocks in; afterwards the caller cancels by closing the returned conn.
	var done = make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-done:
			return
		case <-ctx.Done():
		}

		// why: both channels can be ready at once and select picks at random, so an
		// accepted query would be chased with a cancel the host answers "query not
		// found". Prefer done.
		select {
		case <-done:
			return
		default:
		}

		// why: ctx is already cancelled here, so dialling on it fails before a byte is
		// written and the cancel was never sent at all. The cancel needs a context that
		// outlives the one whose end triggered it.
		cancelCtx, stop := astral.NewContext(nil).WithTimeout(cancelTimeout)
		defer stop()

		cancelHost, err := router.connect(cancelCtx)
		if err != nil {
			// best effort: the query still ends on the host's own deadline
			return
		}
		defer cancelHost.Close()

		conn, _ := cancelHost.RouteQuery(
			astral.Launch(query.New(nil, nil, apphost.MethodCancel, query.Args{"id": q.Nonce})),
			astral.ZoneDevice,
			nil,
		)
		if conn != nil {
			conn.Close()
		}
	}()

	return host.RouteQuery(q, ctx.Zone(), ctx.Filters())
}

// GuestID returns the authenticated guest identity, connecting to the host to
// resolve it on first call; returns nil if the connection or auth fails.
func (router *Router) GuestID() *astral.Identity {
	if id := router.cachedGuestID(); id != nil {
		return id
	}

	host, err := router.connect(astral.NewContext(nil))
	if err != nil {
		return nil
	}
	defer host.Close()

	return router.cachedGuestID()
}

// HostID returns the host node's identity, connecting to resolve it on first
// call; returns nil if the connection fails.
func (router *Router) HostID() *astral.Identity {
	if id := router.cachedHostID(); id != nil {
		return id
	}

	host, err := router.connect(astral.NewContext(nil))
	if err != nil {
		return nil
	}
	defer host.Close()

	return router.cachedHostID()
}

func (router *Router) cachedGuestID() *astral.Identity {
	router.mu.Lock()
	defer router.mu.Unlock()
	return router.guestID
}

func (router *Router) cachedHostID() *astral.Identity {
	router.mu.Lock()
	defer router.mu.Unlock()
	return router.hostID
}

func (router *Router) Endpoint() string {
	return router.endpoint
}

func (router *Router) Protocol() string {
	split := strings.SplitN(router.endpoint, ":", 2)
	return split[0]
}

// connect makes a single attempt to connect and authenticate with the host.
// Returns ErrNodeUnavailable if the IPC dial fails.
func (router *Router) connect(ctx *astral.Context) (*Host, error) {
	host, err := Connect(ctx, router.endpoint)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNodeUnavailable, err)
	}

	router.mu.Lock()
	router.hostID = host.HostID()
	router.mu.Unlock()

	if len(router.token) == 0 {
		return host, nil
	}

	err = host.AuthToken(router.token)
	if err != nil {
		host.Close()
		return nil, err
	}

	router.mu.Lock()
	router.guestID = host.GuestID()
	router.mu.Unlock()

	return host, nil
}

func newDefaultRouter() *Router {
	// TODO: find an optimal endpoint (unix, then tcp)
	return NewRouter(DefaultEndpoint, os.Getenv(AuthTokenEnv))
}
