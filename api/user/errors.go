package user

import "github.com/astralp2p/astral-go/astral"

var ErrRequestDeclined = astral.NewError("request declined")
var ErrNoActiveContract = astral.NewError("no active contract")
var ErrExpelled = astral.NewError("identity expelled from swarm")
var ErrAlreadyClaimed = astral.NewError("node already claimed")
var ErrJoinSelf = astral.NewError("cannot join self")
var ErrIntroduceSelf = astral.NewError("cannot introduce to self")
var ErrNoMnemonic = astral.NewError("no mnemonic stored")
