package serve

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetFreePort(t *testing.T) {
	port, err := GetFreePort()
	if err != nil {
		t.Fatalf("GetFreePort() error = %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Errorf("GetFreePort() = %d, want valid port (1-65535)", port)
	}
}

func TestGetFreePortUnique(t *testing.T) {
	port1, err := GetFreePort()
	if err != nil {
		t.Fatalf("first GetFreePort() error = %v", err)
	}
	port2, err := GetFreePort()
	if err != nil {
		t.Fatalf("second GetFreePort() error = %v", err)
	}
	if port1 <= 0 || port2 <= 0 {
		t.Errorf("expected valid ports, got %d and %d", port1, port2)
	}
}

func TestServeAssessment(t *testing.T) {
	dir := t.TempDir()

	content := "<html><body>hello</body></html>"
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	port, stop, err := ServeAssessment(dir)
	if err != nil {
		t.Fatalf("ServeAssessment() error = %v", err)
	}
	defer stop()

	if port <= 0 || port > 65535 {
		t.Errorf("ServeAssessment() port = %d, want valid port", port)
	}

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/index.html", port))
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if string(body) != content {
		t.Errorf("body = %q, want %q", string(body), content)
	}
}

func TestServeAssessmentStop(t *testing.T) {
	dir := t.TempDir()

	port, stop, err := ServeAssessment(dir)
	if err != nil {
		t.Fatalf("ServeAssessment() error = %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	stop()
	time.Sleep(50 * time.Millisecond)

	_, err = http.Get(fmt.Sprintf("http://localhost:%d/", port))
	if err == nil {
		t.Error("expected request to fail after stop, but it succeeded")
	}
}

func TestServeAssessmentInvalidDir(t *testing.T) {
	// FileServer returns 404s for missing files rather than failing to start
	port, stop, err := ServeAssessment("/nonexistent/path/that/does/not/exist")
	if err != nil {
		return // acceptable to fail fast on bad dir
	}
	defer stop()

	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d for missing dir", resp.StatusCode, http.StatusNotFound)
	}
}
