package docker

import (
	"context"

	"github.com/moby/moby/client"
)

func (m *Manager) RegistryLogin(ctx context.Context, id int64, host, server, username, password string) error {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()
	_, err = cli.RegistryLogin(ctx, client.RegistryLoginOptions{
		Username:      username,
		Password:      password,
		ServerAddress: server,
	})
	return err
}
