package apphost

import (
	"errors"
)

const (
	MethodCreateToken     = "apphost.create_token"
	MethodListTokens      = "apphost.list_tokens"
	MethodRegisterHandler = "apphost.register_handler"
	MethodCancel          = "apphost.cancel"
	MethodBind            = "apphost.bind"
	MethodNewAppContract  = "apphost.new_app_contract"
	MethodSignAppContract = "apphost.sign_app_contract"
	MethodInstallApp      = "apphost.install_app"
	MethodHoldObject      = "apphost.hold_object"
	MethodUnholdObject    = "apphost.unhold_object"
	MethodListHeldObjects = "apphost.list_held_objects"
)

var ErrProtocolError = errors.New("protocol error")
