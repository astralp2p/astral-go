package indexing

import (
	"errors"

	"github.com/astralp2p/astral-go/astral"
)

var ErrIndexNotFound = errors.New("index not found")
var ErrRepositoryNotFound = errors.New("repository not found")
var ErrAckMismatch = errors.New("ack does not match delivered change")
var ErrIndexingTemporarilyFailed = astral.NewError("indexing temporarily failed")

// IsIndexingTemporarilyFailed reports whether err is a temporary indexing failure.
// why: uses string comparison because astral.Error does not support errors.Is unwrapping.
func IsIndexingTemporarilyFailed(err error) bool {
	return err != nil && err.Error() == ErrIndexingTemporarilyFailed.Error()
}
