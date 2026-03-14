package main

import (
	"log"
	"net/http"
	"os"
	"fmt"
)

func testHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello from Professor!")
}

func main() {
	http.HandleFunc("/", testHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
