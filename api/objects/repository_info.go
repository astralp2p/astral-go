package objects

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

const (
	RepositoryKindRepository = "repository" // stores objects itself
	RepositoryKindGroup      = "group"      // delegates to its Children
)

type RepositoryInfo struct {
	Name  astral.String8
	Label astral.String8
	Free  astral.Uint64
	Kind  astral.String8
	// Children names the direct members of a group in lookup order.
	Children []astral.String8
	// Concurrent makes a group race member reads; otherwise members are read in order.
	Concurrent astral.Bool
}

var _ astral.Object = &RepositoryInfo{}

// astral

func (RepositoryInfo) ObjectType() string {
	return "mod.objects.repository_info"
}

func (info RepositoryInfo) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&info).WriteTo(w)
}

func (info *RepositoryInfo) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(info).ReadFrom(r)
}

// json

func (info RepositoryInfo) MarshalJSON() ([]byte, error) {
	type alias RepositoryInfo
	// why: a repository without members emits [], never null.
	if info.Children == nil {
		info.Children = []astral.String8{}
	}
	return json.Marshal(alias(info))
}

func (info *RepositoryInfo) UnmarshalJSON(bytes []byte) error {
	type alias RepositoryInfo
	return json.Unmarshal(bytes, (*alias)(info))
}

// ...

func init() {
	astral.MustAdd(&RepositoryInfo{})
}
