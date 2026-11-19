package apps

import (
	"errors"

	"github.com/astralp2p/astral-go/api/services"
	servicesclient "github.com/astralp2p/astral-go/api/services/client"
	"github.com/astralp2p/astral-go/astral"
)

// WithDiscovery follows names with reach on the default node for as long as
// Serve runs. The follow's Watcher reopens it after its stream ends
// (servicesclient.WithReconnect with its defaults), and onChange runs after
// every Changed signal of the Watcher and once more when the Watcher stops.
// onChange runs on one goroutine.
//
// why not a registration hook: the follow is a query connection of its own,
// which can end while the app's registration stays, and an app with no
// handler runs no hooks.
func WithDiscovery(names []string, reach services.Reach, onChange func(*servicesclient.Watcher)) ServeOption {
	return func(cfg *serveConfig) error {
		if len(names) == 0 {
			return errors.New("no services to discover")
		}
		if onChange == nil {
			return errors.New("nil discovery callback")
		}
		cfg.discoveries = append(cfg.discoveries, discovery{names: names, reach: reach, onChange: onChange})
		return nil
	}
}

type discovery struct {
	names    []string
	reach    services.Reach
	onChange func(*servicesclient.Watcher)
}

// start opens d's Watcher on client and runs onChange until the Watcher
// stops. stop closes the Watcher and waits for the last onChange.
func (d discovery) start(ctx *astral.Context, client *servicesclient.Client) (stop func(), err error) {
	w, err := client.WatchIn(ctx, d.reach, d.names, servicesclient.WithReconnect(0, 0))
	if err != nil {
		return nil, err
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-w.Changed():
				d.onChange(w)
			case <-w.Done():
				d.onChange(w)
				return
			}
		}
	}()
	return func() {
		w.Close()
		<-done
	}, nil
}

// startDiscoveries starts every discovery on client. stop ends them all.
func startDiscoveries(ctx *astral.Context, list []discovery, client *servicesclient.Client) (stop func(), err error) {
	var stops []func()
	stop = func() {
		for _, s := range stops {
			s()
		}
	}
	for _, d := range list {
		s, err := d.start(ctx, client)
		if err != nil {
			stop()
			return nil, err
		}
		stops = append(stops, s)
	}
	return stop, nil
}
