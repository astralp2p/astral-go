package apphost

import (
	"errors"
)

const (
	MethodRegister        = "apphost.register"
	MethodCreateToken     = "apphost.create_token"
	MethodDeleteToken     = "apphost.delete_token"
	MethodListTokens      = "apphost.list_tokens"
	MethodRegisterHandler = "apphost.register_handler"
	MethodCancel          = "apphost.cancel"
	MethodBind            = "apphost.bind"
	MethodNewAppContract  = "apphost.new_app_contract"
	MethodSignAppContract = "apphost.sign_app_contract"
	MethodHoldObject      = "apphost.hold_object"
	MethodUnholdObject    = "apphost.unhold_object"
	MethodListHeldObjects = "apphost.list_held_objects"
)

var ErrProtocolError = errors.New("protocol error")
