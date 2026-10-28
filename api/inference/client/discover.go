package inference

import (
	"github.com/astralp2p/astral-go/api/inference"
	"github.com/astralp2p/astral-go/api/services"
	servicescli "github.com/astralp2p/astral-go/api/services/client"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/sig"
)

// Offering is one provider's inference offering to the caller, read from its
// services.update.
type Offering struct {
	ProviderID *astral.Identity
	// Available is false when the provider withdrew the offering.
	Available bool
	// Provider names the provider; nil when the offering carries no
	// inference.provider.
	Provider *inference.Provider
	// Models is the caller's inventory; nil when the offering carries no
	// inference.models.
	Models *inference.Models
	// Operations lists the ops the offering's services.operations_list names.
	Operations []string
}

// Discovery is one item of an inference discovery. Exactly one field is set.
type Discovery struct {
	Offering *Offering
	// Removed lists the providers whose offerings were lost with their node.
	Removed []*astral.Identity
	// Initial ends the initial attempt, once.
	Initial *servicescli.InitialOutcome
	// Err is terminal; the channel closes after it.
	Err error
}

// Discover finds the inference providers that offer to the caller, as the
// target node sees them with reach. Without follow the channel closes after
// the Initial item; with follow it continues until ctx ends or the node
// closes the stream.
func (client *Client) Discover(ctx *astral.Context, reach services.Reach, follow bool) (<-chan Discovery, error) {
	events, err := servicescli.New(client.targetID, client.astral).DiscoverIn(ctx, reach, []string{inference.ServiceName}, follow)
	if err != nil {
		return nil, err
	}

	out := make(chan Discovery)
	go relayDiscovery(ctx, events, out)
	return out, nil
}

// Discover runs Discover on the default client.
func Discover(ctx *astral.Context, reach services.Reach, follow bool) (<-chan Discovery, error) {
	return Default().Discover(ctx, reach, follow)
}

// relayDiscovery passes the inference items of events to out until either
// ends or ctx ends.
//
// why RecvOk: a follow stream can stay silent for hours, and the caller
// that ends ctx must see out close without waiting for the next item.
func relayDiscovery(ctx *astral.Context, events <-chan servicescli.Event, out chan<- Discovery) {
	defer close(out)
	for {
		ev, open, err := sig.RecvOk(ctx, events)
		if err != nil || !open {
			return
		}
		d, ok := readDiscovery(ev)
		if !ok {
			continue
		}
		if sig.Send(ctx, out, d) != nil {
			return
		}
	}
}

// readDiscovery turns one services event into an inference item, and reports
// whether it concerns inference.
func readDiscovery(ev servicescli.Event) (Discovery, bool) {
	switch {
	case ev.Update != nil:
		if string(ev.Update.Name) != inference.ServiceName {
			return Discovery{}, false
		}
		return Discovery{Offering: readOffering(ev.Update)}, true
	case ev.Removed != nil:
		var lost []*astral.Identity
		for _, k := range ev.Removed {
			if string(k.Name) == inference.ServiceName {
				lost = append(lost, k.ProviderID)
			}
		}
		return Discovery{Removed: lost}, len(lost) > 0
	case ev.Initial != nil:
		return Discovery{Initial: ev.Initial}, true
	default:
		return Discovery{Err: ev.Err}, true
	}
}

func readOffering(u *services.Update) *Offering {
	o := &Offering{ProviderID: u.ProviderID, Available: bool(u.Available)}
	if u.Info == nil {
		return o
	}
	objs := u.Info.Objects()
	o.Provider, _ = astral.First[*inference.Provider](objs)
	o.Models, _ = astral.First[*inference.Models](objs)
	if ops, ok := astral.First[*services.OperationsList](objs); ok {
		for _, op := range ops.Operations {
			o.Operations = append(o.Operations, string(op))
		}
	}
	return o
}
