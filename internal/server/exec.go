package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/rivly/rivly/internal/docker"
)

const execReadLimit = 1 << 20

type execControl struct {
	Type string `json:"type"`
	Cols uint   `json:"cols"`
	Rows uint   `json:"rows"`
}

func (s *Server) handleContainerExec(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}
	containerID := chi.URLParam(r, "containerID")

	env, err := s.queries.GetEnvironment(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, "environment not found")
		return
	}
	if err != nil {
		s.serverError(w, r, "could not load environment", err)
		return
	}

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.CloseNow() }()
	conn.SetReadLimit(execReadLimit)

	ctx, cancel := s.streamContext(r)
	defer cancel()
	session, err := s.docker.ContainerExec(ctx, env.ID, env.Url, containerID)
	if err != nil {
		_ = conn.Write(ctx, websocket.MessageText, execError("could not start a shell in this container"))
		_ = conn.Close(websocket.StatusInternalError, "exec failed")
		return
	}

	s.bridgeExec(ctx, conn, session)
	_ = conn.Close(websocket.StatusNormalClosure, "")
}

func (s *Server) bridgeExec(ctx context.Context, conn *websocket.Conn, session *docker.ExecSession) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer session.Close()

	go func() {
		<-ctx.Done()
		session.Close()
	}()

	go func() {
		defer cancel()
		for {
			typ, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			if typ == websocket.MessageText {
				handleExecControl(ctx, session, data)
				continue
			}
			if _, err := session.Stdin().Write(data); err != nil {
				return
			}
		}
	}()

	buf := make([]byte, 32*1024)
	for {
		n, err := session.Stdout().Read(buf)
		if n > 0 {
			if werr := conn.Write(ctx, websocket.MessageBinary, buf[:n]); werr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func handleExecControl(ctx context.Context, session *docker.ExecSession, data []byte) {
	var ctrl execControl
	if err := json.Unmarshal(data, &ctrl); err != nil {
		return
	}
	if ctrl.Type == "resize" && ctrl.Rows > 0 && ctrl.Cols > 0 {
		_ = session.Resize(ctx, ctrl.Rows, ctrl.Cols)
	}
}

func execError(message string) []byte {
	data, _ := json.Marshal(map[string]string{"type": "error", "message": message})
	return data
}
