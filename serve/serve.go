package serve

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
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

func ServeAssessment(distDir string) (int, func(), error) {
	port, err := GetFreePort()

	if err != nil {
		return -1, func() {}, err
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.FileServer(spaFS{http.Dir(distDir)}),
	}

	go server.ListenAndServe()

	stop := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}

	return port, stop, nil
}
