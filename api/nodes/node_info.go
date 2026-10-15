package nodes

import (
	"bytes"
	"io"

	"github.com/astralp2p/astral-go/api/exonet"
	"github.com/astralp2p/astral-go/astral"
	"github.com/jxskiss/base62"
)

type NodeInfo struct {
	Alias     astral.String8
	Identity  *astral.Identity
	Endpoints []exonet.Endpoint
}

// astral

func (NodeInfo) ObjectType() string {
	return "mod.nodes.node_info"
}

func (info NodeInfo) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&info).WriteTo(w)
}

func (info *NodeInfo) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(info).ReadFrom(r)
}

// text

func (info NodeInfo) MarshalText() (text []byte, err error) {
	var buf = &bytes.Buffer{}

	_, err = info.WriteTo(buf)
	if err != nil {
		return
	}

	return base62.Encode(buf.Bytes()), nil
}

func (info *NodeInfo) UnmarshalText(text []byte) (err error) {
	data, err := base62.Decode(text)
	if err != nil {
		return
	}

	_, err = info.ReadFrom(bytes.NewReader(data))

	return
}

func init() {
	astral.MustAdd(&NodeInfo{})
}
