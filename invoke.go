package main

import (
	"log"

	"github.com/Hack4Impact-UMD/professor/cmd"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("warn: could not load .env")
	}

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
