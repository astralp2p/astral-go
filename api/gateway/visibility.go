package gateway

import "github.com/astralp2p/astral-go/astral"

// Visibility controls whether a registered gateway node is advertised publicly or kept private.
type Visibility = astral.String8

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)
