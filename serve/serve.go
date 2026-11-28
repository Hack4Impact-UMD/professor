package serve

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/Hack4Impact-UMD/professor/sandbox"
)

type spaFS struct {
	fs http.FileSystem
}

func (s spaFS) Open(name string) (http.File, error) {
	f, err := s.fs.Open(name)

	if os.IsNotExist(err) {
		return s.fs.Open("index.html")
	}

	return f, err
}

// get an available port
func GetFreePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return -1, err
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	return port, nil
}

func ServeAssessment(ctx context.Context, sb sandbox.Sandbox, distDir string) (int, func(), error) {
	ctx, cancel := context.WithCancel(ctx)
	out, _, wait, err := sb.ExecPipe(ctx, &sandbox.SandboxCommand{
		Args: []string{
			"/app/professor", "serve-static",
		},
		Dir: distDir,
	})

	if err != nil {
		return -1, func() {}, err
	}

	go func() {
		wait()
	}()

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		port, err := strconv.Atoi(scanner.Text())
		if err != nil {
			return port, cancel, nil
		}
	}

	return -1, cancel, nil
}

func ServeAssessmentBlocking(port int, distDir string) error {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.FileServer(spaFS{http.Dir(distDir)}),
	}
	err := server.ListenAndServe()
	return err
}
