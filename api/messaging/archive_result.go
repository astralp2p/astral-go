package messaging

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// ArchiveResult is what MethodArchive answers. Changed says this call moved the
// message; false means it was already where the call asked for, or the caller
// does not hold it.
//
// why the field is not called Archived: undo runs through the same operation,
// and there "archived: true" would name the opposite of what happened.
type ArchiveResult struct {
	Changed astral.Bool
}

// astral

var _ astral.Object = &ArchiveResult{}

func (r ArchiveResult) ObjectType() string { return "messaging.archive_result" }

func (r ArchiveResult) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *ArchiveResult) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

// json

func (r ArchiveResult) MarshalJSON() ([]byte, error) {
	type alias ArchiveResult
	return json.Marshal(alias(r))
}

func (r *ArchiveResult) UnmarshalJSON(bytes []byte) error {
	type alias ArchiveResult
	var v alias

	err := json.Unmarshal(bytes, &v)
	if err != nil {
		return err
	}

	*r = ArchiveResult(v)
	return nil
}

func init() {
	astral.MustAdd(&ArchiveResult{})
}
