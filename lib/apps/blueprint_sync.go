package apps

import (
	"errors"
	"log"
	"sync"

	objectsClient "github.com/astralp2p/astral-go/api/objects/client"
	"github.com/astralp2p/astral-go/astral"
)

var (
	blueprintCacheOnce sync.Once
	blueprintCache     []*astral.Blueprint
)

// SyncBlueprints pushes every local runtime Blueprint (struct kind + alias kind) to the
// node, skipping any already present. The cache is built once per process (reflection cost
// paid at first call); the remote list and diff/push run every invocation since the node
// may have lost state across reconnects.
//
// AllBlueprints yields alias-kind Blueprints first, then struct-kind ones, so a Blueprint
// with a RefSpec to an alias type passes the remote validateReferences check during replay.
func SyncBlueprints(ctx *astral.Context) error {
	blueprintCacheOnce.Do(func() {
		var err error
		blueprintCache, err = astral.DefaultBlueprints().AllBlueprints()

		// why: the previous comment here reasoned that every undescribable entry is a
		// built-in or a shared module type, so dropping the error was harmless. That
		// holds for astral-go's own types and stops holding the moment an app registers
		// one — a struct with a bare Go string field derives nothing, is silently left
		// out of the sync, and surfaces much later as a decode error on the peer. The
		// error is the only signal that happened, so it is reported rather than dropped.
		//
		// Reported, not returned: the undescribable set includes astral-go's own types,
		// so failing here would break every app for a condition none of them caused.
		if err != nil {
			log.Printf("blueprint sync: these types have no derivable blueprint and will "+
				"not be pushed, so a peer cannot decode them: %v", err)
		}
	})

	remote, err := objectsClient.Blueprints(ctx)
	if err != nil {
		return err
	}
	have := make(map[string]struct{}, len(remote))
	for _, n := range remote {
		have[n] = struct{}{}
	}

	for _, b := range blueprintCache {
		name := b.Type.String()
		if _, ok := have[name]; ok {
			continue
		}
		_, regErr := objectsClient.Register(ctx, b)
		if regErr == nil || errors.Is(regErr, astral.ErrAlreadyRegistered) {
			continue
		}
		log.Printf("blueprint sync: %s: %v", name, regErr)
	}
	return nil
}

// WithBlueprintSync installs SyncBlueprints as a registration hook so the
// app's local Blueprints are pushed on every (re)connect.
func WithBlueprintSync() ServeOption {
	return WithRegistrationHook(SyncBlueprints)
}
