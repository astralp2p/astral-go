package services

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// ServiceDiscoveryAction requests permission for Actor to discover Service on
// the contributing node NodeID. Discovery submits one action per requested
// service before it evaluates any provider.
type ServiceDiscoveryAction struct {
	auth.Action
	Service astral.String8
	NodeID  *astral.Identity
}

func (ServiceDiscoveryAction) ObjectType() string {
	return "mod.services.service_discovery_action"
}

func (a ServiceDiscoveryAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *ServiceDiscoveryAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints covers the action only when cs holds exactly one
// DiscoveryScope that allows Service on NodeID.
//
// why: an unconstrained permit covers nothing. Discovery is granted per
// service, and an app can ask apphost.register for any permit by name, so an
// unconstrained permit honoured in full would let an app grant itself
// discovery of every service.
func (a ServiceDiscoveryAction) ApplyConstraints(cs *astral.Bundle) bool {
	if cs == nil {
		return false
	}

	objects := cs.Objects()
	if len(objects) != 1 {
		return false
	}

	scope, ok := objects[0].(*DiscoveryScope)
	if !ok {
		return false
	}

	return scope.Allows(a.Service, a.NodeID)
}

// DiscoveryScope narrows a discovery permit to pairs of service and
// contributing node. It allows a pair when one of its rules does.
type DiscoveryScope struct {
	Rules []*DiscoveryRule
}

func (DiscoveryScope) ObjectType() string { return "mod.services.discovery_scope" }

func (s DiscoveryScope) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&s).WriteTo(w)
}

func (s *DiscoveryScope) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(s).ReadFrom(r)
}

func (s DiscoveryScope) MarshalJSON() ([]byte, error)  { return astral.Objectify(&s).MarshalJSON() }
func (s *DiscoveryScope) UnmarshalJSON(b []byte) error { return astral.Objectify(s).UnmarshalJSON(b) }

// Allows reports whether one rule of s allows service on node.
func (s DiscoveryScope) Allows(service astral.String8, node *astral.Identity) bool {
	for _, rule := range s.Rules {
		if rule != nil && rule.Allows(service, node) {
			return true
		}
	}
	return false
}

// DiscoveryRule allows its Services on its Nodes. An empty Services list allows
// nothing; an empty Nodes list allows any node within reach.
type DiscoveryRule struct {
	Services []astral.String8
	Nodes    []*astral.Identity
}

func (DiscoveryRule) ObjectType() string { return "mod.services.discovery_rule" }

func (r DiscoveryRule) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *DiscoveryRule) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

func (r DiscoveryRule) MarshalJSON() ([]byte, error)  { return astral.Objectify(&r).MarshalJSON() }
func (r *DiscoveryRule) UnmarshalJSON(b []byte) error { return astral.Objectify(r).UnmarshalJSON(b) }

// Allows reports whether r allows service on node.
func (r DiscoveryRule) Allows(service astral.String8, node *astral.Identity) bool {
	return r.allowsService(service) && r.allowsNode(node)
}

func (r DiscoveryRule) allowsService(service astral.String8) bool {
	for _, s := range r.Services {
		if s == service {
			return true
		}
	}
	return false
}

func (r DiscoveryRule) allowsNode(node *astral.Identity) bool {
	if len(r.Nodes) == 0 {
		return true
	}
	for _, n := range r.Nodes {
		if n != nil && node != nil && n.IsEqual(node) {
			return true
		}
	}
	return false
}

func init() {
	astral.MustAdd(&ServiceDiscoveryAction{})
	astral.MustAdd(&DiscoveryScope{})
	astral.MustAdd(&DiscoveryRule{})
}
