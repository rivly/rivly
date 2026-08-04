package docker

import (
	"context"

	"github.com/moby/moby/client"
)

func (d *Client) RegistryLogin(ctx context.Context, server, username, password string) error {
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()
	_, err := d.cli.RegistryLogin(ctx, client.RegistryLoginOptions{
		Username:      username,
		Password:      password,
		ServerAddress: server,
	})
	return err
}
