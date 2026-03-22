package git

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetRepoSizeKB_Success(t *testing.T) {
	const wantSize = 42
	const wantPath = "/repos/some-org/some-repo"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Errorf("unexpected path: got %q, want %q", r.URL.Path, wantPath)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(githubRepoResponse{Id: 1, FullName: "some-org/some-repo", Size: wantSize})
	}))
	defer srv.Close()

	client := NewGitHubClient(srv.URL, srv.Client())
	size, err := client.GetRepoSizeKB("some-org/some-repo")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if size != wantSize {
		t.Errorf("expected size %d, got %d", wantSize, size)
	}
}

func TestGetRepoSizeKB_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	client := NewGitHubClient(srv.URL, srv.Client())
	size, err := client.GetRepoSizeKB("some-org/some-repo")

	if err == nil {
		t.Fatal("expected a decode error, got nil")
	}
	if size != -1 {
		t.Errorf("expected size -1 on error, got %d", size)
	}
}

func TestGetRepoSizeKB_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // close immediately so the request fails

	client := NewGitHubClient(srv.URL, srv.Client())
	size, err := client.GetRepoSizeKB("some-org/some-repo")

	if err == nil {
		t.Fatal("expected a network error, got nil")
	}
	if size != -1 {
		t.Errorf("expected size -1 on error, got %d", size)
	}
}
