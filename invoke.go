package main

import (
	"log/slog"

	"github.com/Hack4Impact-UMD/professor/cmd"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("could not load .env", "err", err)
	}

	cmd.Execute()
}
