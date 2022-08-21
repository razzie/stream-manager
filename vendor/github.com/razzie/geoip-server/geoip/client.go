package geoip

import (
	"context"
)

// Client is used to retrieve location data from a provider
type Client interface {
	Provider() string
	GetLocation(ctx context.Context, hostname string) (*Location, error)
}

type client struct {
	provider *Provider
	params   []string
}

func (c *client) Provider() string {
	return c.provider.Name
}

func (c *client) GetLocation(ctx context.Context, hostname string) (*Location, error) {
	return c.provider.doRequest(ctx, hostname, c.params)
}

// NewClient returns a new client to the provider
func (p *Provider) NewClient(params ...string) Client {
	return &client{
		provider: p,
		params:   params,
	}
}
