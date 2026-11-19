package nat

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// ReflectUDPOffer is the first answer of nat.reflect_udp: the UDP port the
// reflector listens on for this query, and the token a datagram to it must carry.
type ReflectUDPOffer struct {
	Port  astral.Uint16
	Token astral.Bytes8
}

func (ReflectUDPOffer) ObjectType() string { return "nat.reflect_udp_offer" }

func (o ReflectUDPOffer) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&o).WriteTo(w)
}

func (o *ReflectUDPOffer) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(o).ReadFrom(r)
}

func (o ReflectUDPOffer) MarshalJSON() ([]byte, error) {
	type alias ReflectUDPOffer
	return json.Marshal(alias(o))
}

func (o *ReflectUDPOffer) UnmarshalJSON(bytes []byte) error {
	type alias ReflectUDPOffer
	var a alias
	if err := json.Unmarshal(bytes, &a); err != nil {
		return err
	}
	*o = ReflectUDPOffer(a)
	return nil
}

func init() {
	astral.MustAdd(&ReflectUDPOffer{})
}
