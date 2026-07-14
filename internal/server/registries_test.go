package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegistriesCRUD(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := getStatus(t, &http.Client{}, ts.URL+"/api/v1/registries"); code != http.StatusUnauthorized {
		t.Fatalf("list without auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)

	var empty []registryResponse
	getJSON(t, client, ts.URL+"/api/v1/registries", &empty)
	if len(empty) != 0 {
		t.Fatalf("registries: want empty, got %d", len(empty))
	}

	var created registryResponse
	postJSONStatus(t, client, ts.URL+"/api/v1/registries",
		`{"name":"GitHub","server":"https://ghcr.io/","username":"bob","password":"s3cret"}`, http.StatusCreated, &created)
	if created.Name != "GitHub" || created.Server != "ghcr.io" || created.Username != "bob" {
		t.Fatalf("create: got %+v", created)
	}

	if code := postStatus(t, client, ts.URL+"/api/v1/registries",
		`{"name":"","server":"docker.io","username":"x","password":"y"}`); code != http.StatusBadRequest {
		t.Fatalf("missing name: want 400, got %d", code)
	}

	var list []registryResponse
	getJSON(t, client, ts.URL+"/api/v1/registries", &list)
	if len(list) != 1 || list[0].Server != "ghcr.io" {
		t.Fatalf("registries: got %+v", list)
	}

	// duplicate server -> 409
	if code := postStatus(t, client, ts.URL+"/api/v1/registries",
		`{"name":"Dup","server":"ghcr.io","username":"x","password":"y"}`); code != http.StatusConflict {
		t.Fatalf("duplicate: want 409, got %d", code)
	}

	// missing fields -> 400
	if code := postStatus(t, client, ts.URL+"/api/v1/registries",
		`{"name":"X","server":"","username":"x","password":"y"}`); code != http.StatusBadRequest {
		t.Fatalf("missing server: want 400, got %d", code)
	}

	// update
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/registries/1",
		strings.NewReader(`{"name":"GitHub","server":"ghcr.io","username":"alice","password":"newpass"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update: want 200, got %d", resp.StatusCode)
	}

	// delete
	delReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/registries/1", nil)
	delResp, err := client.Do(delReq)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	_ = delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: want 204, got %d", delResp.StatusCode)
	}
}

func TestRegistryTest(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	seedEnvironment(t, srv)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)

	if code := postStatus(t, client, ts.URL+"/api/v1/registries/test",
		`{"server":"ghcr.io","username":"bob","password":"good"}`); code != http.StatusOK {
		t.Fatalf("test ok: want 200, got %d", code)
	}

	srv.docker = fakeDocker{registryLoginErr: errors.New("unauthorized")}
	if code := postStatus(t, client, ts.URL+"/api/v1/registries/test",
		`{"server":"ghcr.io","username":"bob","password":"bad"}`); code != http.StatusBadGateway {
		t.Fatalf("test bad: want 502, got %d", code)
	}
}
