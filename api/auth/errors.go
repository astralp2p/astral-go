package auth

import (
	"errors"

	"github.com/astralp2p/astral-go/astral"
)

var ErrInvalidContract = errors.New("invalid contract")
var ErrContractExpired = errors.New("contract expired")
var ErrAlreadySigned = errors.New("already signed")

// ErrNotPermitted refuses a caller the node does not authorize for the action.
var ErrNotPermitted = astral.NewError("not permitted")
