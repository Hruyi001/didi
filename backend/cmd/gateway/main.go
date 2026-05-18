package main

import (
	"log"
	"os"

	"didi/backend/internal/gateway"
)

func main() {
	apiBase := os.Getenv("API_BASE")
	if apiBase == "" {
		apiBase = "http://localhost:18080"
	}
	authBase := os.Getenv("AUTH_BASE")
	r := gateway.NewRouter(&gateway.Clients{APIBase: apiBase, AuthBase: authBase})
	log.Fatal(r.Run(":8080"))
}
