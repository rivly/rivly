package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGitCredentialsCRUD(t *testing.T) {
	srv := newTestServer(t)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if code := getStatus(t, &http.Client{}, ts.URL+"/api/v1/git-credentials"); code != http.StatusUnauthorized {
		t.Fatalf("list without auth: want 401, got %d", code)
	}

	client := authedClient(t, ts)

	var empty []gitCredentialResponse
	getJSON(t, client, ts.URL+"/api/v1/git-credentials", &empty)
	if len(empty) != 0 {
		t.Fatalf("credentials: want empty, got %d", len(empty))
	}

	var created gitCredentialResponse
	postJSONStatus(t, client, ts.URL+"/api/v1/git-credentials",
		`{"name":"GitHub","username":"bob","token":"ghp_secret"}`, http.StatusCreated, &created)
	if created.Name != "GitHub" || created.Username != "bob" {
		t.Fatalf("create: got %+v", created)
	}

	if code := postStatus(t, client, ts.URL+"/api/v1/git-credentials",
		`{"name":"","username":"x","token":"y"}`); code != http.StatusBadRequest {
		t.Fatalf("missing name: want 400, got %d", code)
	}

	if code := postStatus(t, client, ts.URL+"/api/v1/git-credentials",
		`{"name":"GitHub","username":"x","token":"y"}`); code != http.StatusConflict {
		t.Fatalf("duplicate name: want 409, got %d", code)
	}

	var list []gitCredentialResponse
	getJSON(t, client, ts.URL+"/api/v1/git-credentials", &list)
	if len(list) != 1 || list[0].Name != "GitHub" {
		t.Fatalf("credentials: got %+v", list)
	}

	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/git-credentials/1",
		strings.NewReader(`{"name":"GitHub","username":"alice","token":""}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update: want 200, got %d", resp.StatusCode)
	}

	delReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/git-credentials/1", nil)
	delResp, err := client.Do(delReq)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	_ = delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: want 204, got %d", delResp.StatusCode)
	}
}

func TestGitCredentialTest(t *testing.T) {
	srv := newTestServer(t)

	git := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, _ := r.BasicAuth()
		if user == "bob" && pass == "good" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer git.Close()

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := authedClient(t, ts)

	ok := fmt.Sprintf(`{"repositoryUrl":%q,"username":"bob","token":"good"}`, git.URL+"/acme/app")
	if code := postStatus(t, client, ts.URL+"/api/v1/git-credentials/test", ok); code != http.StatusOK {
		t.Fatalf("test ok: want 200, got %d", code)
	}

	bad := fmt.Sprintf(`{"repositoryUrl":%q,"username":"bob","token":"bad"}`, git.URL+"/acme/app")
	if code := postStatus(t, client, ts.URL+"/api/v1/git-credentials/test", bad); code != http.StatusBadGateway {
		t.Fatalf("test bad: want 502, got %d", code)
	}
}
