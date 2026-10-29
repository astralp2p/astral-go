package user

import "github.com/astralp2p/astral-go/astral"

var ErrInvitationDeclined = astral.NewError("invitation declined")
var ErrRequestDeclined = astral.NewError("request declined")
var ErrContractDeclined = astral.NewError("contract declined")
var ErrNoActiveContract = astral.NewError("no active contract")
var ErrExpelled = astral.NewError("identity expelled from swarm")
var ErrAlreadyClaimed = astral.NewError("node already claimed")
var ErrJoinSelf = astral.NewError("cannot join self")
