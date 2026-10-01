package apps

import (
	"errors"

	servicesclient "github.com/astralp2p/astral-go/api/services/client"
	"github.com/astralp2p/astral-go/astral"
)

// WithServices advertises p's services on every registration and reconnect,
// keeping one binding open while the app is registered.
//
// why the hook retries itself: a hook error on the first registration stops the
// registrar, and on reconnect the registrar retries without delay. The node may
// still hold the previous binding for a moment after a reconnect, so
// Provider.Serve backs off on that one refusal.
func WithServices(p *servicesclient.Provider) ServeOption {
	return func(cfg *serveConfig) error {
		if p == nil || len(p.Names()) == 0 {
			return errors.New("no services to advertise")
		}
		cfg.hooks = append(cfg.hooks, func(ctx *astral.Context) error {
			return p.Serve(ctx, servicesclient.Default())
		})
		return nil
	}
}
