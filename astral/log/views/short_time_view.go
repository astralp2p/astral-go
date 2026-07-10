package views

import (
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/fmt"
)

type ShortTimeView struct {
	*astral.Time
}

func (v ShortTimeView) Render() string {
	return v.Time.Time().Format("15:04:05.000")
}

func init() {
	fmt.SetView(func(o *astral.Time) fmt.View {
		return &ShortTimeView{Time: o}
	})
}
