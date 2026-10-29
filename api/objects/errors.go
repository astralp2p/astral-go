package objects

import "github.com/astralp2p/astral-go/astral"

// ErrNotFound refuses a read of an object no candidate delivers.
var ErrNotFound = astral.NewError("object not found")

// ErrRepositoryNotFound refuses a read that names a repository the node does not have.
var ErrRepositoryNotFound = astral.NewError("repository not found")
