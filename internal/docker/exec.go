package docker

import (
	"context"
	"io"
	"sync"

	"github.com/moby/moby/client"
)

var execShell = []string{"/bin/sh", "-c", "exec $(command -v bash || command -v sh)"}

type ExecSession struct {
	cli    *client.Client
	execID string
	resp   client.HijackedResponse
	closed sync.Once
}

func (d *Client) ContainerExec(ctx context.Context, containerID string) (*ExecSession, error) {
	created, err := d.cli.ExecCreate(ctx, containerID, client.ExecCreateOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          true,
		Cmd:          execShell,
	})
	if err != nil {
		return nil, err
	}

	attached, err := d.cli.ExecAttach(ctx, created.ID, client.ExecAttachOptions{TTY: true})
	if err != nil {
		return nil, err
	}

	return &ExecSession{cli: d.cli, execID: created.ID, resp: attached.HijackedResponse}, nil
}

func (s *ExecSession) Stdin() io.Writer {
	return s.resp.Conn
}

func (s *ExecSession) Stdout() io.Reader {
	return s.resp.Reader
}

func (s *ExecSession) Resize(ctx context.Context, rows, cols uint) error {
	_, err := s.cli.ExecResize(ctx, s.execID, client.ExecResizeOptions{Height: rows, Width: cols})
	return err
}

func (s *ExecSession) Close() {
	s.closed.Do(func() { s.resp.Close() })
}
