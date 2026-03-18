package util

import (
	"fmt"
	"net"
	"time"
)

func WaitForPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	url := fmt.Sprintf(":%d", port)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", url, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("port %d did not respond in %v", port, timeout)
}
