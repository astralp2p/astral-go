package inference

import (
	"errors"
	"testing"

	"github.com/astralp2p/astral-go/api/inference"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/routing"
)

func modelsProvider(t *testing.T, answer ...astral.Object) *Client {
	t.Helper()
	c, _ := clientFor(t, map[string]any{inference.OpModels: func(_ *astral.Context, q *routing.IncomingQuery) error {
		ch := q.Accept()
		defer ch.Close()
		for _, o := range answer {
			if err := ch.Send(o); err != nil {
				return err
			}
		}
		return nil
	}})
	return c
}

func TestModels(t *testing.T) {
	c := modelsProvider(t, &inference.Models{Models: []*inference.Model{{ID: "m", DefaultEffort: inference.EffortLow, Tools: true}}}, &astral.EOS{})

	list, err := c.Models(astral.NewContext(nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Models) != 1 || list.Models[0].ID != "m" || !bool(list.Models[0].Tools) {
		t.Fatalf("models %+v", list.Models)
	}
}

func TestModelsError(t *testing.T) {
	c := modelsProvider(t, &inference.Error{Kind: inference.ErrorUnavailable, Status: 503, Message: "loading"}, &astral.EOS{})

	list, err := c.Models(astral.NewContext(nil))
	var ierr *inference.Error
	if !errors.As(err, &ierr) || ierr.Kind != inference.ErrorUnavailable || ierr.Status != 503 || list != nil {
		t.Fatalf("got %v, %v; want the provider's inference.error", list, err)
	}
}

func TestModelsNoAnswer(t *testing.T) {
	c := modelsProvider(t)
	if list, err := c.Models(astral.NewContext(nil)); err == nil || list != nil {
		t.Fatalf("got %v, %v; want an error", list, err)
	}
}
