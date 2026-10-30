package inference

import (
	"errors"
	"fmt"

	"github.com/astralp2p/astral-go/api/inference"
	objectscli "github.com/astralp2p/astral-go/api/objects/client"
	"github.com/astralp2p/astral-go/astral"
)

// StaleBlueprintError reports a node that holds an inference blueprint other
// than this module's. It loaded an older shape since it started, and a node
// keeps a blueprint until it stops, so only a restart clears it.
type StaleBlueprintError struct {
	Node *astral.Identity
	Type string
}

func (e *StaleBlueprintError) Error() string {
	return fmt.Sprintf("node %v holds an older %s; restart the node", e.Node, e.Type)
}

// PushBlueprints registers every inference blueprint on the target node,
// dependencies first, with objects.register_blueprint. A type the node already
// holds counts as registered, whatever its shape there; CheckBlueprints tells.
// It needs mod.auth.store_objects_action.
//
// why: a node that hosts no inference app cannot decode an offering or an
// answer that crosses it until some app pushes the types.
func (client *Client) PushBlueprints(ctx *astral.Context) error {
	bps, err := inference.Blueprints()
	if err != nil {
		return err
	}
	objects := objectscli.New(client.targetID, client.astral)
	for _, bp := range bps {
		_, err := objects.Register(ctx, bp)
		if err != nil && !errors.Is(err, astral.ErrAlreadyRegistered) {
			return fmt.Errorf("register %s: %w", bp.Type, err)
		}
	}
	return nil
}

// CheckBlueprints compares every inference blueprint the target node holds
// with this module's, by Object ID, through objects.get_blueprint. A node that
// holds another shape fails with *StaleBlueprintError. It needs
// mod.auth.see_objects_action.
//
// why: objects.register_blueprint answers "already registered" whether the
// shapes match or not, and a node holding an older shape decodes every
// inference object with it: fields go missing or the stream breaks, and no
// error names the cause.
func (client *Client) CheckBlueprints(ctx *astral.Context) error {
	bps, err := inference.Blueprints()
	if err != nil {
		return err
	}
	objects := objectscli.New(client.targetID, client.astral)
	for _, local := range bps {
		remote, err := objects.GetBlueprint(ctx, local.Type.String())
		if err != nil {
			return fmt.Errorf("node %v: get blueprint %s: %w", client.node(), local.Type, err)
		}
		same, err := sameBlueprint(local, remote)
		if err != nil {
			return err
		}
		if !same {
			return &StaleBlueprintError{Node: client.node(), Type: local.Type.String()}
		}
	}
	return nil
}

// PushBlueprints runs PushBlueprints on the default client.
func PushBlueprints(ctx *astral.Context) error {
	return Default().PushBlueprints(ctx)
}

// CheckBlueprints runs CheckBlueprints on the default client.
func CheckBlueprints(ctx *astral.Context) error {
	return Default().CheckBlueprints(ctx)
}

// node is the identity of the target node.
func (client *Client) node() *astral.Identity {
	if client.targetID != nil {
		return client.targetID
	}
	return client.astral.HostID()
}

func sameBlueprint(a, b *astral.Blueprint) (bool, error) {
	aID, err := astral.ResolveObjectID(a)
	if err != nil {
		return false, err
	}
	bID, err := astral.ResolveObjectID(b)
	if err != nil {
		return false, err
	}
	return aID.IsEqual(bID), nil
}
