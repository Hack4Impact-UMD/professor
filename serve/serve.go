package serve

import (
	"fmt"
	"net"
	"net/http"
)

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

func ServeAssessment(distDir string) (int, error) {
	port, err := GetFreePort()

	if err != nil {
		return -1, err
	}

	server := http.NewServeMux()

	server.Handle("/", http.FileServer(http.Dir(distDir)))

	http.ListenAndServe(fmt.Sprintf(":%d", port), server)

	return port, nil
}
