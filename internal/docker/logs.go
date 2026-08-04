package docker

import (
	"bytes"
	"context"
	"io"
	"strconv"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

type LogLine struct {
	Stream  string
	Message string
}

func (d *Client) ContainerLogs(ctx context.Context, containerID string, tail int, follow bool) (<-chan LogLine, error) {
	inspectCtx, cancel := context.WithTimeout(ctx, callTimeout)
	info, err := d.cli.ContainerInspect(inspectCtx, containerID, client.ContainerInspectOptions{})
	cancel()
	if err != nil {
		return nil, err
	}
	tty := info.Container.Config != nil && info.Container.Config.Tty

	tailValue := "all"
	if tail > 0 {
		tailValue = strconv.Itoa(tail)
	}
	res, err := d.cli.ContainerLogs(ctx, containerID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Tail:       tailValue,
	})
	if err != nil {
		return nil, err
	}

	out := make(chan LogLine)
	go func() {
		defer close(out)
		defer func() { _ = res.Close() }()
		if tty {
			w := &logWriter{ctx: ctx, out: out, stream: "stdout"}
			_, _ = io.Copy(w, res)
			w.flush()
			return
		}
		stdout := &logWriter{ctx: ctx, out: out, stream: "stdout"}
		stderr := &logWriter{ctx: ctx, out: out, stream: "stderr"}
		_, _ = stdcopy.StdCopy(stdout, stderr, res)
		stdout.flush()
		stderr.flush()
	}()
	return out, nil
}

type logWriter struct {
	ctx    context.Context
	out    chan<- LogLine
	stream string
	buf    []byte
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		line := bytes.TrimRight(w.buf[:i], "\r")
		w.buf = w.buf[i+1:]
		if !w.emit(string(line)) {
			return 0, context.Canceled
		}
	}
	return len(p), nil
}

func (w *logWriter) flush() {
	if len(w.buf) == 0 {
		return
	}
	w.emit(string(bytes.TrimRight(w.buf, "\r")))
	w.buf = nil
}

func (w *logWriter) emit(message string) bool {
	select {
	case <-w.ctx.Done():
		return false
	case w.out <- LogLine{Stream: w.stream, Message: message}:
		return true
	}
}
