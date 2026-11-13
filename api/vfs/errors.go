package vfs

import (
	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// ErrNotPermitted is the value of auth.ErrNotPermitted: the text is the same.
var ErrNotPermitted = auth.ErrNotPermitted

var ErrNoSuchVFS = astral.NewError("no such vfs")
var ErrInvalidPath = astral.NewError("invalid path")
var ErrNotFound = astral.NewError("not found")
var ErrNotAFolder = astral.NewError("not a folder")
var ErrIsAFolder = astral.NewError("is a folder")
var ErrAlreadyExists = astral.NewError("already exists")
var ErrFolderNotEmpty = astral.NewError("folder not empty")
var ErrQuotaExceeded = astral.NewError("quota exceeded")
var ErrNoInodesLeft = astral.NewError("no inodes left")
var ErrInvalidTime = astral.NewError("invalid time")
var ErrNoStorage = astral.NewError("no storage")
