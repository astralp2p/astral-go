package indexing

import (
	"github.com/astralp2p/astral-go/api/indexing"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// EnableRepo turns on indexing for the named object repository.
func (c *Client) EnableRepo(ctx *astral.Context, repo string) error {
	return c.setRepoIndexing(ctx, repo, false)
}

// DisableRepo turns off indexing for the named object repository.
func (c *Client) DisableRepo(ctx *astral.Context, repo string) error {
	return c.setRepoIndexing(ctx, repo, true)
}

func (c *Client) setRepoIndexing(ctx *astral.Context, repo string, disable bool) error {
	ch, err := c.queryCh(ctx, indexing.MethodEnableRepo, query.Args{
		"repo":    repo,
		"disable": disable,
	})
	if err != nil {
		return err
	}
	defer ch.Close()

	var ack *astral.Ack
	return ch.Switch(channel.Expect(&ack), channel.PassErrors)
}

func EnableRepo(ctx *astral.Context, repo string) error {
	return Default().EnableRepo(ctx, repo)
}

func DisableRepo(ctx *astral.Context, repo string) error {
	return Default().DisableRepo(ctx, repo)
}
